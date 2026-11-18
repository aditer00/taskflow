package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aditya/taskflow-go/internal/queue"
	"github.com/aditya/taskflow-go/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	Ping(ctx context.Context) error
	CreateUser(ctx context.Context, name, email, passwordHash string) (store.User, error)
	GetUserByEmail(ctx context.Context, email string) (store.User, error)
	GetUserByID(ctx context.Context, id int64) (store.User, error)
	ListProjects(ctx context.Context, userID int64) ([]store.Project, error)
	CreateProject(ctx context.Context, userID int64, name, description string) (store.Project, error)
	GetProject(ctx context.Context, id int64) (store.Project, error)
	DeleteProject(ctx context.Context, id, userID int64) error
	ListTasks(ctx context.Context, projectID int64) ([]store.Task, error)
	CreateTask(ctx context.Context, t store.Task) (store.Task, error)
	GetTask(ctx context.Context, id int64) (store.Task, error)
	UpdateTaskStatus(ctx context.Context, id, userID int64, status string) (store.Task, error)
	DeleteTask(ctx context.Context, id, userID int64) (string, error)
}

type SessionStore interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, sid string) (int64, error)
	Set(ctx context.Context, sid string, userID int64, ttl time.Duration) error
	Delete(ctx context.Context, sid string) error
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Delete(ctx context.Context, key string) error
}

type QueuePublisher interface {
	Enqueue(ctx context.Context, msg queue.TaskNotification) error
}

type Server struct {
	repo      Repository
	sessions  SessionStore
	storage   ObjectStorage
	queue     QueuePublisher
	logger    *slog.Logger
	templates map[string]*template.Template
}

func NewServer(repo Repository, sessions SessionStore, storage ObjectStorage, queue QueuePublisher, logger *slog.Logger) (*Server, error) {
	pages := []string{"landing.html", "login.html", "register.html", "projects.html", "tasks.html"}
	tpls := make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t, err := template.ParseFS(TemplateFS, "templates/layout.html", "templates/"+p)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		tpls[p] = t
	}

	return &Server{
		repo:      repo,
		sessions:  sessions,
		storage:   storage,
		queue:     queue,
		logger:    logger,
		templates: tpls,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	staticSub, _ := fs.Sub(StaticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /", s.handleLanding)

	mux.HandleFunc("GET /login", s.handleLoginForm)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /register", s.handleRegisterForm)
	mux.HandleFunc("POST /register", s.handleRegister)
	mux.HandleFunc("POST /logout", s.requireAuth(s.handleLogout))

	mux.HandleFunc("GET /projects", s.requireAuth(s.handleProjectsList))
	mux.HandleFunc("POST /projects", s.requireAuth(s.handleProjectCreate))
	mux.HandleFunc("GET /projects/{id}", s.requireAuth(s.handleProjectShow))
	mux.HandleFunc("POST /projects/{id}/delete", s.requireAuth(s.handleProjectDelete))

	mux.HandleFunc("POST /projects/{id}/tasks", s.requireAuth(s.handleTaskCreate))
	mux.HandleFunc("POST /tasks/{id}/status", s.requireAuth(s.handleTaskStatus))
	mux.HandleFunc("POST /tasks/{id}/delete", s.requireAuth(s.handleTaskDelete))
	mux.HandleFunc("GET /tasks/{id}/attachment", s.requireAuth(s.handleTaskAttachment))

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{
		"database": "ok",
		"redis":    "ok",
	}
	status := http.StatusOK
	overall := "ok"

	if err := s.repo.Ping(r.Context()); err != nil {
		checks["database"] = "error"
		overall = "degraded"
		status = http.StatusServiceUnavailable
		s.logger.Error("healthcheck database failure", "error", err)
	}

	if err := s.sessions.Ping(r.Context()); err != nil {
		checks["redis"] = "error"
		overall = "degraded"
		status = http.StatusServiceUnavailable
		s.logger.Error("healthcheck redis failure", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": overall,
		"checks": checks,
	})
}

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := map[string]any{"Title": "TaskFlow - 12-Factor Task Management"}
	if user, ok := s.currentUser(r); ok {
		data["User"] = user
	}
	s.render(w, "landing.html", data)
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(r); ok {
		http.Redirect(w, r, "/projects", http.StatusFound)
		return
	}
	s.render(w, "login.html", map[string]any{"Title": "Sign in - TaskFlow"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	user, err := s.repo.GetUserByEmail(r.Context(), email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		s.renderStatus(w, http.StatusUnprocessableEntity, "login.html", map[string]any{
			"Title": "Sign in - TaskFlow",
			"Email": email,
			"Error": "Invalid email or password.",
		})
		return
	}

	if err := s.startSession(w, r.Context(), user.ID); err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}
	s.logger.Info("user logged in", "user_id", user.ID, "email", user.Email)
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (s *Server) handleRegisterForm(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(r); ok {
		http.Redirect(w, r, "/projects", http.StatusFound)
		return
	}
	s.render(w, "register.html", map[string]any{"Title": "Create account - TaskFlow"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirmation")

	if name == "" || email == "" || len(password) < 8 || password != confirm {
		s.renderStatus(w, http.StatusUnprocessableEntity, "register.html", map[string]any{
			"Title": "Create account - TaskFlow",
			"Name":  name,
			"Email": email,
			"Error": "Check name, email, and matching password (min 8 chars).",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Password error", http.StatusInternalServerError)
		return
	}

	user, err := s.repo.CreateUser(r.Context(), name, email, string(hash))
	if err != nil {
		s.renderStatus(w, http.StatusUnprocessableEntity, "register.html", map[string]any{
			"Title": "Create account - TaskFlow",
			"Name":  name,
			"Email": email,
			"Error": "Email already registered.",
		})
		return
	}

	if err := s.startSession(w, r.Context(), user.ID); err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}
	s.logger.Info("user registered", "user_id", user.ID)
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request, _ store.User) {
	if c, err := r.Cookie("taskflow_session"); err == nil {
		_ = s.sessions.Delete(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "taskflow_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleProjectsList(w http.ResponseWriter, r *http.Request, user store.User) {
	projects, err := s.repo.ListProjects(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	s.render(w, "projects.html", map[string]any{
		"Title":    "Projects - TaskFlow",
		"User":     user,
		"Projects": projects,
	})
}

func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request, user store.User) {
	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("description"))
	if name == "" || len(name) > 120 {
		projects, _ := s.repo.ListProjects(r.Context(), user.ID)
		s.renderStatus(w, http.StatusUnprocessableEntity, "projects.html", map[string]any{
			"Title":    "Projects - TaskFlow",
			"User":     user,
			"Projects": projects,
			"Error":    "Project name is required (max 120 chars).",
		})
		return
	}

	p, err := s.repo.CreateProject(r.Context(), user.ID, name, desc)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	s.logger.Info("project created", "project_id", p.ID, "user_id", user.ID)
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (s *Server) handleProjectShow(w http.ResponseWriter, r *http.Request, user store.User) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	project, err := s.repo.GetProject(r.Context(), projectID)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if project.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	tasks, err := s.repo.ListTasks(r.Context(), project.ID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	s.renderTaskBoard(w, http.StatusOK, user, project, tasks, "")
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request, user store.User) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.repo.DeleteProject(r.Context(), projectID, user.ID); err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	s.logger.Info("project deleted", "project_id", projectID, "user_id", user.ID)
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request, user store.User) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	project, err := s.repo.GetProject(r.Context(), projectID)
	if err != nil || project.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		tasks, _ := s.repo.ListTasks(r.Context(), project.ID)
		s.renderTaskBoard(w, http.StatusUnprocessableEntity, user, project, tasks, "Invalid form or file exceeds 5MB.")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	desc := strings.TrimSpace(r.FormValue("description"))
	priority := r.FormValue("priority")
	dueDate := strings.TrimSpace(r.FormValue("due_date"))

	if title == "" {
		tasks, _ := s.repo.ListTasks(r.Context(), project.ID)
		s.renderTaskBoard(w, http.StatusUnprocessableEntity, user, project, tasks, "Task title is required.")
		return
	}
	if priority != "low" && priority != "medium" && priority != "high" {
		priority = "medium"
	}

	var attachmentKey string
	file, header, err := r.FormFile("attachment")
	if err == nil {
		defer file.Close()
		if header.Size > 5<<20 {
			tasks, _ := s.repo.ListTasks(r.Context(), project.ID)
			s.renderTaskBoard(w, http.StatusUnprocessableEntity, user, project, tasks, "Attachment must be 5MB or smaller.")
			return
		}
		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowed := map[string]bool{".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".txt": true, ".md": true}
		if !allowed[ext] {
			tasks, _ := s.repo.ListTasks(r.Context(), project.ID)
			s.renderTaskBoard(w, http.StatusUnprocessableEntity, user, project, tasks, "Allowed attachment types: pdf, png, jpg, jpeg, txt, md.")
			return
		}
		attachmentKey = fmt.Sprintf("attachments/%d-%s%s", time.Now().UnixNano(), randomHex(6), ext)
		ct := header.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		if err := s.storage.Put(r.Context(), attachmentKey, file, ct); err != nil {
			s.logger.Error("s3 upload failed", "error", err)
			http.Error(w, "Storage error", http.StatusInternalServerError)
			return
		}
	}

	task, err := s.repo.CreateTask(r.Context(), store.Task{
		ProjectID:      project.ID,
		UserID:         user.ID,
		Title:          title,
		Description:    desc,
		Status:         "todo",
		Priority:       priority,
		DueDate:        dueDate,
		AttachmentPath: attachmentKey,
	})
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	s.logger.Info("task created", "task_id", task.ID, "project_id", project.ID, "has_attachment", attachmentKey != "")

	_ = s.queue.Enqueue(r.Context(), queue.TaskNotification{
		TaskID:      task.ID,
		ProjectName: project.Name,
		Title:       task.Title,
		Status:      task.Status,
		Priority:    task.Priority,
		Event:       "created",
		Recipient:   user.Email,
	})

	http.Redirect(w, r, fmt.Sprintf("/projects/%d", project.ID), http.StatusSeeOther)
}

func (s *Server) handleTaskStatus(w http.ResponseWriter, r *http.Request, user store.User) {
	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	status := r.FormValue("status")
	if status != "todo" && status != "in_progress" && status != "done" {
		http.Error(w, "Invalid status", http.StatusUnprocessableEntity)
		return
	}

	task, err := s.repo.UpdateTaskStatus(r.Context(), taskID, user.ID, status)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	s.logger.Info("task status updated", "task_id", task.ID, "status", status)

	if status == "done" {
		project, _ := s.repo.GetProject(r.Context(), task.ProjectID)
		_ = s.queue.Enqueue(r.Context(), queue.TaskNotification{
			TaskID:      task.ID,
			ProjectName: project.Name,
			Title:       task.Title,
			Status:      task.Status,
			Priority:    task.Priority,
			Event:       "completed",
			Recipient:   user.Email,
		})
	}

	http.Redirect(w, r, fmt.Sprintf("/projects/%d", task.ProjectID), http.StatusSeeOther)
}

func (s *Server) handleTaskDelete(w http.ResponseWriter, r *http.Request, user store.User) {
	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	task, err := s.repo.GetTask(r.Context(), taskID)
	if err != nil || task.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	path, err := s.repo.DeleteTask(r.Context(), taskID, user.ID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if path != "" {
		_ = s.storage.Delete(r.Context(), path)
	}
	s.logger.Info("task deleted", "task_id", taskID)
	http.Redirect(w, r, fmt.Sprintf("/projects/%d", task.ProjectID), http.StatusSeeOther)
}

func (s *Server) handleTaskAttachment(w http.ResponseWriter, r *http.Request, user store.User) {
	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	task, err := s.repo.GetTask(r.Context(), taskID)
	if err != nil || task.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if task.AttachmentPath == "" {
		http.NotFound(w, r)
		return
	}

	body, ct, err := s.storage.Get(r.Context(), task.AttachmentPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(task.AttachmentPath)))
	_, _ = io.Copy(w, body)
}

func (s *Server) renderTaskBoard(w http.ResponseWriter, code int, user store.User, project store.Project, tasks []store.Task, errMsg string) {
	openCount := 0
	doneCount := 0
	for _, t := range tasks {
		if t.Status == "done" {
			doneCount++
		} else {
			openCount++
		}
	}
	s.renderStatus(w, code, "tasks.html", map[string]any{
		"Title":      project.Name + " - TaskFlow",
		"User":       user,
		"Project":    project,
		"Tasks":      tasks,
		"TotalCount": len(tasks),
		"OpenCount":  openCount,
		"DoneCount":  doneCount,
		"Error":      errMsg,
	})
}

func (s *Server) requireAuth(next func(http.ResponseWriter, *http.Request, store.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.currentUser(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r, user)
	}
}

func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	c, err := r.Cookie("taskflow_session")
	if err != nil || c.Value == "" {
		return store.User{}, false
	}
	uid, err := s.sessions.Get(r.Context(), c.Value)
	if err != nil || uid <= 0 {
		return store.User{}, false
	}
	u, err := s.repo.GetUserByID(r.Context(), uid)
	if err != nil {
		return store.User{}, false
	}
	return u, true
}

func (s *Server) startSession(w http.ResponseWriter, ctx context.Context, userID int64) error {
	sid := randomHex(24)
	if err := s.sessions.Set(ctx, sid, userID, 24*time.Hour); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "taskflow_session",
		Value:    sid,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *Server) render(w http.ResponseWriter, name string, data map[string]any) {
	s.renderStatus(w, http.StatusOK, name, data)
}

func (s *Server) renderStatus(w http.ResponseWriter, code int, name string, data map[string]any) {
	tpl, ok := s.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
		s.logger.Error("template render error", "template", name, "error", err)
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

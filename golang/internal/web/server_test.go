package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aditya/taskflow-go/internal/queue"
	"github.com/aditya/taskflow-go/internal/store"
)

type fakeRepo struct {
	pingErr  error
	users    map[int64]store.User
	projects map[int64]store.Project
	tasks    map[int64]store.Task
}

func (f *fakeRepo) Ping(context.Context) error { return f.pingErr }
func (f *fakeRepo) CreateUser(_ context.Context, name, email, hash string) (store.User, error) {
	u := store.User{ID: int64(len(f.users) + 1), Name: name, Email: email, PasswordHash: hash}
	f.users[u.ID] = u
	return u, nil
}
func (f *fakeRepo) GetUserByEmail(_ context.Context, email string) (store.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return store.User{}, store.ErrNotFound
}
func (f *fakeRepo) GetUserByID(_ context.Context, id int64) (store.User, error) {
	u, ok := f.users[id]
	if !ok {
		return u, store.ErrNotFound
	}
	return u, nil
}
func (f *fakeRepo) ListProjects(_ context.Context, userID int64) ([]store.Project, error) {
	var out []store.Project
	for _, p := range f.projects {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (f *fakeRepo) CreateProject(_ context.Context, userID int64, name, desc string) (store.Project, error) {
	p := store.Project{ID: int64(len(f.projects) + 1), UserID: userID, Name: name, Description: desc}
	f.projects[p.ID] = p
	return p, nil
}
func (f *fakeRepo) GetProject(_ context.Context, id int64) (store.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return p, store.ErrNotFound
	}
	return p, nil
}
func (f *fakeRepo) DeleteProject(_ context.Context, id, userID int64) error {
	p, ok := f.projects[id]
	if !ok || p.UserID != userID {
		return store.ErrNotFound
	}
	delete(f.projects, id)
	return nil
}
func (f *fakeRepo) ListTasks(_ context.Context, projectID int64) ([]store.Task, error) {
	var out []store.Task
	for _, t := range f.tasks {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (f *fakeRepo) CreateTask(_ context.Context, t store.Task) (store.Task, error) {
	t.ID = int64(len(f.tasks) + 1)
	f.tasks[t.ID] = t
	return t, nil
}
func (f *fakeRepo) GetTask(_ context.Context, id int64) (store.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return t, store.ErrNotFound
	}
	return t, nil
}
func (f *fakeRepo) UpdateTaskStatus(_ context.Context, id, userID int64, status string) (store.Task, error) {
	t, ok := f.tasks[id]
	if !ok || t.UserID != userID {
		return t, store.ErrNotFound
	}
	t.Status = status
	f.tasks[id] = t
	return t, nil
}
func (f *fakeRepo) DeleteTask(_ context.Context, id, userID int64) (string, error) {
	t, ok := f.tasks[id]
	if !ok || t.UserID != userID {
		return "", store.ErrNotFound
	}
	delete(f.tasks, id)
	return t.AttachmentPath, nil
}

type fakeSessions struct {
	pingErr error
	data    map[string]int64
}

func (f *fakeSessions) Ping(context.Context) error { return f.pingErr }
func (f *fakeSessions) Get(_ context.Context, sid string) (int64, error) {
	uid, ok := f.data[sid]
	if !ok {
		return 0, errors.New("missing")
	}
	return uid, nil
}
func (f *fakeSessions) Set(_ context.Context, sid string, uid int64, _ time.Duration) error {
	f.data[sid] = uid
	return nil
}
func (f *fakeSessions) Delete(_ context.Context, sid string) error {
	delete(f.data, sid)
	return nil
}

type fakeStorage struct{ files map[string]string }

func (f *fakeStorage) Put(_ context.Context, key string, body io.Reader, _ string) error {
	b, _ := io.ReadAll(body)
	f.files[key] = string(b)
	return nil
}
func (f *fakeStorage) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	v, ok := f.files[key]
	if !ok {
		return nil, "", errors.New("not found")
	}
	return io.NopCloser(strings.NewReader(v)), "text/plain", nil
}
func (f *fakeStorage) Delete(_ context.Context, key string) error {
	delete(f.files, key)
	return nil
}

type fakeQueue struct{ items []queue.TaskNotification }

func (f *fakeQueue) Enqueue(_ context.Context, msg queue.TaskNotification) error {
	f.items = append(f.items, msg)
	return nil
}

func newTestServer(repo *fakeRepo, sess *fakeSessions, stor *fakeStorage, q *fakeQueue) http.Handler {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv, err := NewServer(repo, sess, stor, q, logger)
	if err != nil {
		panic(err)
	}
	return srv.Handler()
}

func TestHealthEndpoint(t *testing.T) {
	repo := &fakeRepo{users: map[int64]store.User{}, projects: map[int64]store.Project{}, tasks: map[int64]store.Task{}}
	sess := &fakeSessions{data: map[string]int64{}}
	h := newTestServer(repo, sess, &fakeStorage{files: map[string]string{}}, &fakeQueue{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	sess.pingErr = errors.New("redis down")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec2.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &body)
	if body["status"] != "degraded" {
		t.Fatalf("expected degraded status, got %v", body["status"])
	}
}

func TestLandingPageReturns200(t *testing.T) {
	repo := &fakeRepo{users: map[int64]store.User{}, projects: map[int64]store.Project{}, tasks: map[int64]store.Task{}}
	sess := &fakeSessions{data: map[string]int64{}}
	h := newTestServer(repo, sess, &fakeStorage{files: map[string]string{}}, &fakeQueue{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "TaskFlow") {
		t.Fatalf("expected landing page to contain TaskFlow")
	}
}

func TestUnauthenticatedRedirectsToLogin(t *testing.T) {
	repo := &fakeRepo{users: map[int64]store.User{}, projects: map[int64]store.Project{}, tasks: map[int64]store.Task{}}
	sess := &fakeSessions{data: map[string]int64{}}
	h := newTestServer(repo, sess, &fakeStorage{files: map[string]string{}}, &fakeQueue{})

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got %d (%s)", rec.Code, rec.Header().Get("Location"))
	}
}

func TestForbiddenOtherUsersProject(t *testing.T) {
	repo := &fakeRepo{
		users: map[int64]store.User{
			1: {ID: 1, Name: "Owner", Email: "owner@example.com"},
			2: {ID: 2, Name: "Other", Email: "other@example.com"},
		},
		projects: map[int64]store.Project{
			10: {ID: 10, UserID: 1, Name: "Secret"},
		},
		tasks: map[int64]store.Task{},
	}
	sess := &fakeSessions{data: map[string]int64{"sid-other": 2}}
	h := newTestServer(repo, sess, &fakeStorage{files: map[string]string{}}, &fakeQueue{})

	req := httptest.NewRequest(http.MethodGet, "/projects/10", nil)
	req.AddCookie(&http.Cookie{Name: "taskflow_session", Value: "sid-other"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRejectDisallowedAttachment(t *testing.T) {
	repo := &fakeRepo{
		users:    map[int64]store.User{1: {ID: 1, Name: "User", Email: "u@example.com"}},
		projects: map[int64]store.Project{1: {ID: 1, UserID: 1, Name: "Proj"}},
		tasks:    map[int64]store.Task{},
	}
	sess := &fakeSessions{data: map[string]int64{"sid": 1}}
	stor := &fakeStorage{files: map[string]string{}}
	h := newTestServer(repo, sess, stor, &fakeQueue{})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("title", "Bad upload")
	fw, _ := mw.CreateFormFile("attachment", "malware.exe")
	_, _ = fw.Write([]byte("MZ"))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/projects/1/tasks", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "taskflow_session", Value: "sid"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
	if len(stor.files) != 0 || len(repo.tasks) != 0 {
		t.Fatalf("expected no stored file or task on validation failure")
	}
}

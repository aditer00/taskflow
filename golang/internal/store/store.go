package store

import (
	"context"
	"embed"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

var ErrNotFound = errors.New("not found")

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
}

type Project struct {
	ID          int64
	UserID      int64
	Name        string
	Description string
	TasksCount  int
	CreatedAt   time.Time
}

type Task struct {
	ID             int64
	ProjectID      int64
	UserID         int64
	Title          string
	Description    string
	Status         string
	Priority       string
	DueDate        string
	AttachmentPath string
	CreatedAt      time.Time
}

type TaskStats struct {
	Total   int
	Open    int
	Overdue int
}

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) Migrate(ctx context.Context) error {
	sqlBytes, err := migrationFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, string(sqlBytes))
	return err
}

func (s *Store) Seed(ctx context.Context) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) ON CONFLICT (email) DO NOTHING`,
		"Test User", "test@example.com", string(hash),
	)
	return err
}

func (s *Store) CreateUser(ctx context.Context, name, email, passwordHash string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id, name, email, password_hash`,
		name, email, passwordHash,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash)
	return u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) ListProjects(ctx context.Context, userID int64) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.user_id, p.name, p.description, COUNT(t.id) AS tasks_count, p.created_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.user_id = $1
		GROUP BY p.id
		ORDER BY p.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.TasksCount, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreateProject(ctx context.Context, userID int64, name, description string) (Project, error) {
	var p Project
	err := s.pool.QueryRow(ctx,
		`INSERT INTO projects (user_id, name, description) VALUES ($1, $2, $3) RETURNING id, user_id, name, description, created_at`,
		userID, name, description,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt)
	return p, err
}

func (s *Store) GetProject(ctx context.Context, id int64) (Project, error) {
	var p Project
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, name, description, created_at FROM projects WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Store) DeleteProject(ctx context.Context, id, userID int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListTasks(ctx context.Context, projectID int64) ([]Task, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, user_id, title, description, status, priority, COALESCE(TO_CHAR(due_date, 'YYYY-MM-DD'), ''), attachment_path, created_at
		FROM tasks
		WHERE project_id = $1
		ORDER BY id DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.AttachmentPath, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateTask(ctx context.Context, t Task) (Task, error) {
	var due *string
	if t.DueDate != "" {
		due = &t.DueDate
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tasks (project_id, user_id, title, description, status, priority, due_date, attachment_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`, t.ProjectID, t.UserID, t.Title, t.Description, t.Status, t.Priority, due, t.AttachmentPath).Scan(&t.ID, &t.CreatedAt)
	return t, err
}

func (s *Store) GetTask(ctx context.Context, id int64) (Task, error) {
	var t Task
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, user_id, title, description, status, priority, COALESCE(TO_CHAR(due_date, 'YYYY-MM-DD'), ''), attachment_path, created_at
		FROM tasks WHERE id = $1
	`, id).Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.AttachmentPath, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) UpdateTaskStatus(ctx context.Context, id, userID int64, status string) (Task, error) {
	var t Task
	err := s.pool.QueryRow(ctx, `
		UPDATE tasks SET status = $1 WHERE id = $2 AND user_id = $3
		RETURNING id, project_id, user_id, title, description, status, priority, COALESCE(TO_CHAR(due_date, 'YYYY-MM-DD'), ''), attachment_path, created_at
	`, status, id, userID).Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.AttachmentPath, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) DeleteTask(ctx context.Context, id, userID int64) (string, error) {
	var path string
	err := s.pool.QueryRow(ctx,
		`DELETE FROM tasks WHERE id = $1 AND user_id = $2 RETURNING attachment_path`,
		id, userID,
	).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return path, err
}

func (s *Store) GetTaskStats(ctx context.Context) (TaskStats, error) {
	var st TaskStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status != 'done'),
			COUNT(*) FILTER (WHERE status != 'done' AND due_date IS NOT NULL AND due_date < CURRENT_DATE)
		FROM tasks
	`).Scan(&st.Total, &st.Open, &st.Overdue)
	return st, err
}

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/miuratouya/go-rest-api/internal/task"
)

type Repository struct {
	db *sql.DB
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create tasks table: %w", err)
	}

	return nil
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, filter task.Filter) ([]task.Task, error) {
	normalizedFilter := filter.Normalize()

	query := `
SELECT id, title, description, status, created_at, updated_at
FROM tasks
`
	args := make([]any, 0, 2)

	if normalizedFilter.Status != nil {
		query += "WHERE status = ?\n"
		args = append(args, string(*normalizedFilter.Status))
	}

	query += "ORDER BY id DESC\nLIMIT ?"
	args = append(args, normalizedFilter.Limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		currentTask, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, currentTask)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (task.Task, error) {
	const query = `
SELECT id, title, description, status, created_at, updated_at
FROM tasks
WHERE id = ?
`

	row := r.db.QueryRowContext(ctx, query, id)
	currentTask, err := scanTask(row)
	if err != nil {
		return task.Task{}, err
	}

	return currentTask, nil
}

func (r *Repository) Create(ctx context.Context, value task.Task) (task.Task, error) {
	const query = `
INSERT INTO tasks (title, description, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
`

	result, err := r.db.ExecContext(
		ctx,
		query,
		value.Title,
		value.Description,
		string(value.Status),
		formatTime(value.CreatedAt),
		formatTime(value.UpdatedAt),
	)
	if err != nil {
		return task.Task{}, fmt.Errorf("insert task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return task.Task{}, fmt.Errorf("get last insert id: %w", err)
	}

	value.ID = id
	return value, nil
}

func (r *Repository) Update(ctx context.Context, value task.Task) (task.Task, error) {
	const query = `
UPDATE tasks
SET title = ?, description = ?, status = ?, updated_at = ?
WHERE id = ?
`

	result, err := r.db.ExecContext(
		ctx,
		query,
		value.Title,
		value.Description,
		string(value.Status),
		formatTime(value.UpdatedAt),
		value.ID,
	)
	if err != nil {
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return task.Task{}, fmt.Errorf("count updated rows: %w", err)
	}

	if rowsAffected == 0 {
		return task.Task{}, task.ErrTaskNotFound
	}

	return value, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `
DELETE FROM tasks
WHERE id = ?
`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted rows: %w", err)
	}

	if rowsAffected == 0 {
		return task.ErrTaskNotFound
	}

	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTask(scanner scannable) (task.Task, error) {
	var currentTask task.Task
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&currentTask.ID,
		&currentTask.Title,
		&currentTask.Description,
		&currentTask.Status,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return task.Task{}, task.ErrTaskNotFound
		}
		return task.Task{}, fmt.Errorf("scan task: %w", err)
	}

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return task.Task{}, err
	}

	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return task.Task{}, err
	}

	currentTask.CreatedAt = parsedCreatedAt
	currentTask.UpdatedAt = parsedUpdatedAt

	return currentTask, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse stored time %q: %w", value, err)
	}

	return parsedTime, nil
}

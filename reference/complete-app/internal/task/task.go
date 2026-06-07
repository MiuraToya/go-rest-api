package task

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrInvalidTask  = errors.New("invalid task")
)

type Status string

const (
	StatusTodo  Status = "todo"
	StatusDoing Status = "doing"
	StatusDone  Status = "done"
)

var validStatuses = map[Status]struct{}{
	StatusTodo:  {},
	StatusDoing: {},
	StatusDone:  {},
}

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Filter struct {
	Status *Status
	Limit  int
}

type CreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *Status `json:"status"`
}

func (s Status) Validate() error {
	if _, ok := validStatuses[s]; ok {
		return nil
	}

	return fmt.Errorf("%w: unknown status %q", ErrInvalidTask, s)
}

func (t Task) Validate() error {
	switch {
	case strings.TrimSpace(t.Title) == "":
		return fmt.Errorf("%w: title is required", ErrInvalidTask)
	case len([]rune(t.Title)) > 100:
		return fmt.Errorf("%w: title must be 100 characters or fewer", ErrInvalidTask)
	case len([]rune(t.Description)) > 500:
		return fmt.Errorf("%w: description must be 500 characters or fewer", ErrInvalidTask)
	}

	if err := t.Status.Validate(); err != nil {
		return err
	}

	return nil
}

func (f Filter) Normalize() Filter {
	switch {
	case f.Limit <= 0:
		f.Limit = 20
	case f.Limit > 100:
		f.Limit = 100
	}

	return f
}

func normalizeTitle(title string) string {
	return strings.TrimSpace(title)
}

func normalizeDescription(description string) string {
	return strings.TrimSpace(description)
}

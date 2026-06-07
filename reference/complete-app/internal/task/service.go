package task

import (
	"context"
	"fmt"
	"time"
)

type Repository interface {
	List(ctx context.Context, filter Filter) ([]Task, error)
	GetByID(ctx context.Context, id int64) (Task, error)
	Create(ctx context.Context, value Task) (Task, error)
	Update(ctx context.Context, value Task) (Task, error)
	Delete(ctx context.Context, id int64) error
}

type Service interface {
	List(ctx context.Context, filter Filter) ([]Task, error)
	Get(ctx context.Context, id int64) (Task, error)
	Create(ctx context.Context, input CreateInput) (Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (Task, error)
	Delete(ctx context.Context, id int64) error
}

type Manager struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Manager {
	return newService(repo, time.Now)
}

func newService(repo Repository, now func() time.Time) *Manager {
	return &Manager{
		repo: repo,
		now:  now,
	}
}

func (m *Manager) List(ctx context.Context, filter Filter) ([]Task, error) {
	tasks, err := m.repo.List(ctx, filter.Normalize())
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

func (m *Manager) Get(ctx context.Context, id int64) (Task, error) {
	if err := validateID(id); err != nil {
		return Task{}, err
	}

	task, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("get task: %w", err)
	}

	return task, nil
}

func (m *Manager) Create(ctx context.Context, input CreateInput) (Task, error) {
	now := m.now().UTC()
	newTask := Task{
		Title:       normalizeTitle(input.Title),
		Description: normalizeDescription(input.Description),
		Status:      StatusTodo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := newTask.Validate(); err != nil {
		return Task{}, err
	}

	createdTask, err := m.repo.Create(ctx, newTask)
	if err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}

	return createdTask, nil
}

func (m *Manager) Update(ctx context.Context, id int64, input UpdateInput) (Task, error) {
	if err := validateID(id); err != nil {
		return Task{}, err
	}

	currentTask, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("get task for update: %w", err)
	}

	if input.Title != nil {
		currentTask.Title = normalizeTitle(*input.Title)
	}

	if input.Description != nil {
		currentTask.Description = normalizeDescription(*input.Description)
	}

	if input.Status != nil {
		currentTask.Status = *input.Status
	}

	currentTask.UpdatedAt = m.now().UTC()

	if err := currentTask.Validate(); err != nil {
		return Task{}, err
	}

	updatedTask, err := m.repo.Update(ctx, currentTask)
	if err != nil {
		return Task{}, fmt.Errorf("update task: %w", err)
	}

	return updatedTask, nil
}

func (m *Manager) Delete(ctx context.Context, id int64) error {
	if err := validateID(id); err != nil {
		return err
	}

	if err := m.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	return nil
}

func validateID(id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidTask)
	}

	return nil
}

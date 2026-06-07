package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateTask_ValidInput_ReturnsCreatedTask(t *testing.T) {
	// 有効な作成入力のとき、前後の空白を削除し `todo` 状態で作成されることを確認する。
	// Python の dict より struct を使うことで、入力の形をコンパイル時に固定できる。
	// Arrange
	fixedNow := time.Date(2026, 6, 7, 1, 2, 3, 0, time.UTC)
	repository := &stubRepository{
		createFn: func(_ context.Context, value Task) (Task, error) {
			if value.Title != "Write Go tutorial" {
				t.Fatalf("expected trimmed title, got %q", value.Title)
			}
			if value.Description != "Cover structs and interfaces" {
				t.Fatalf("expected trimmed description, got %q", value.Description)
			}
			if value.Status != StatusTodo {
				t.Fatalf("expected default status %q, got %q", StatusTodo, value.Status)
			}

			value.ID = 10
			return value, nil
		},
	}
	service := newService(repository, func() time.Time { return fixedNow })

	// Act
	createdTask, err := service.Create(context.Background(), CreateInput{
		Title:       "  Write Go tutorial  ",
		Description: "  Cover structs and interfaces  ",
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if createdTask.ID != 10 {
		t.Fatalf("expected ID 10, got %d", createdTask.ID)
	}
	if !createdTask.CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected CreatedAt %v, got %v", fixedNow, createdTask.CreatedAt)
	}
}

func TestUpdateTask_InvalidStatus_ReturnsValidationError(t *testing.T) {
	// 不正な status を更新しようとしたとき、保存前にバリデーションエラーになることを確認する。
	// 入力検証を service 層に寄せることで、handler と repository の責務を薄く保てる。
	// Arrange
	repository := &stubRepository{
		getByIDFn: func(_ context.Context, id int64) (Task, error) {
			return Task{
				ID:          id,
				Title:       "Review code",
				Description: "Check the API tutorial",
				Status:      StatusTodo,
				CreatedAt:   time.Date(2026, 6, 7, 1, 2, 3, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 6, 7, 1, 2, 3, 0, time.UTC),
			}, nil
		},
		updateFn: func(_ context.Context, value Task) (Task, error) {
			t.Fatalf("repository.Update should not be called for invalid input, got %+v", value)
			return Task{}, nil
		},
	}
	service := newService(repository, time.Now)
	invalidStatus := Status("blocked")

	// Act
	_, err := service.Update(context.Background(), 1, UpdateInput{Status: &invalidStatus})

	// Assert
	if !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("expected ErrInvalidTask, got %v", err)
	}
}

func TestListTasks_EmptyLimit_UsesDefaultLimit(t *testing.T) {
	// limit 未指定の一覧取得では、service が既定値 20 を repository に渡すことを確認する。
	// 既定値を一箇所に寄せると、handler が増えても振る舞いを揃えやすい。
	// Arrange
	repository := &stubRepository{
		listFn: func(_ context.Context, filter Filter) ([]Task, error) {
			if filter.Limit != 20 {
				t.Fatalf("expected default limit 20, got %d", filter.Limit)
			}
			return []Task{}, nil
		},
	}
	service := newService(repository, time.Now)

	// Act
	_, err := service.List(context.Background(), Filter{})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

type stubRepository struct {
	listFn    func(ctx context.Context, filter Filter) ([]Task, error)
	getByIDFn func(ctx context.Context, id int64) (Task, error)
	createFn  func(ctx context.Context, value Task) (Task, error)
	updateFn  func(ctx context.Context, value Task) (Task, error)
	deleteFn  func(ctx context.Context, id int64) error
}

func (s *stubRepository) List(ctx context.Context, filter Filter) ([]Task, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx, filter)
}

func (s *stubRepository) GetByID(ctx context.Context, id int64) (Task, error) {
	if s.getByIDFn == nil {
		return Task{}, nil
	}
	return s.getByIDFn(ctx, id)
}

func (s *stubRepository) Create(ctx context.Context, value Task) (Task, error) {
	if s.createFn == nil {
		return value, nil
	}
	return s.createFn(ctx, value)
}

func (s *stubRepository) Update(ctx context.Context, value Task) (Task, error) {
	if s.updateFn == nil {
		return value, nil
	}
	return s.updateFn(ctx, value)
}

func (s *stubRepository) Delete(ctx context.Context, id int64) error {
	if s.deleteFn == nil {
		return nil
	}
	return s.deleteFn(ctx, id)
}

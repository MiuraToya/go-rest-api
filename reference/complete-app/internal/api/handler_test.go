package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/miuratouya/go-rest-api/internal/task"
)

func TestCreateTask_UnknownField_ReturnsBadRequest(t *testing.T) {
	// 想定外フィールドを含む JSON のとき、400 を返して誤った入力を早く検知することを確認する。
	// `DisallowUnknownFields` はスキーマが曖昧になりやすい API を防ぐ実務的な設定。
	// Arrange
	handler := &Handler{taskService: stubTaskService{}}
	body := bytes.NewBufferString(`{"title":"Write tests","unexpected":"field"}`)
	request := httptest.NewRequest(http.MethodPost, "/tasks", body)
	recorder := httptest.NewRecorder()

	// Act
	handler.createTask(recorder, request)

	// Assert
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestListTasks_ValidQuery_ReturnsTasks(t *testing.T) {
	// 正しい query parameter のとき、service の結果が JSON で返ることを確認する。
	// handler の unit test では DB をつながず、HTTP 入出力の変換責務に絞って検証する。
	// Arrange
	now := time.Date(2026, 6, 7, 1, 2, 3, 0, time.UTC)
	service := stubTaskService{
		listFn: func(_ context.Context, filter task.Filter) ([]task.Task, error) {
			if filter.Status == nil || *filter.Status != task.StatusDoing {
				t.Fatalf("expected status filter %q, got %+v", task.StatusDoing, filter.Status)
			}
			return []task.Task{
				{
					ID:          7,
					Title:       "Ship tutorial",
					Description: "Explain the Go API structure",
					Status:      task.StatusDoing,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			}, nil
		},
	}
	handler := &Handler{taskService: service}
	request := httptest.NewRequest(http.MethodGet, "/tasks?status=doing&limit=10", nil)
	recorder := httptest.NewRecorder()

	// Act
	handler.listTasks(recorder, request)

	// Assert
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		Tasks []task.Task `json:"tasks"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(payload.Tasks))
	}
	if payload.Tasks[0].ID != 7 {
		t.Fatalf("expected task ID 7, got %d", payload.Tasks[0].ID)
	}
}

type stubTaskService struct {
	listFn   func(ctx context.Context, filter task.Filter) ([]task.Task, error)
	getFn    func(ctx context.Context, id int64) (task.Task, error)
	createFn func(ctx context.Context, input task.CreateInput) (task.Task, error)
	updateFn func(ctx context.Context, id int64, input task.UpdateInput) (task.Task, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (s stubTaskService) List(ctx context.Context, filter task.Filter) ([]task.Task, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx, filter)
}

func (s stubTaskService) Get(ctx context.Context, id int64) (task.Task, error) {
	if s.getFn == nil {
		return task.Task{}, nil
	}
	return s.getFn(ctx, id)
}

func (s stubTaskService) Create(ctx context.Context, input task.CreateInput) (task.Task, error) {
	if s.createFn == nil {
		return task.Task{}, nil
	}
	return s.createFn(ctx, input)
}

func (s stubTaskService) Update(ctx context.Context, id int64, input task.UpdateInput) (task.Task, error) {
	if s.updateFn == nil {
		return task.Task{}, nil
	}
	return s.updateFn(ctx, id, input)
}

func (s stubTaskService) Delete(ctx context.Context, id int64) error {
	if s.deleteFn == nil {
		return nil
	}
	return s.deleteFn(ctx, id)
}

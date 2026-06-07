//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/miuratouya/go-rest-api/internal/api"
	"github.com/miuratouya/go-rest-api/internal/store/sqlite"
	"github.com/miuratouya/go-rest-api/internal/task"
)

func TestTaskLifecycle_SQLiteBackedAPI_WorksEndToEnd(t *testing.T) {
	// SQLite をつないだ API 全体で、作成・取得・更新・削除が一貫して動くことを確認する。
	// IT では HTTP, service, repository, SQLite をまとめて検証し、結合不整合を早めに見つける。
	// Arrange
	router := newIntegrationRouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	createdTask := createTaskThroughAPI(t, server.URL, map[string]string{
		"title":       "Learn Go",
		"description": "Build a REST API with SQLite",
	})

	// Act
	updatedTask := updateTaskThroughAPI(t, server.URL, createdTask.ID, map[string]string{
		"status": "done",
	})
	listedTasks := listTasksThroughAPI(t, server.URL, "?status=done&limit=5")
	getResponse := getTaskThroughAPI(t, server.URL, createdTask.ID)
	deleteTaskThroughAPI(t, server.URL, createdTask.ID)
	finalStatusCode := getTaskStatusCode(t, server.URL, createdTask.ID)

	// Assert
	if updatedTask.Status != task.StatusDone {
		t.Fatalf("expected status %q, got %q", task.StatusDone, updatedTask.Status)
	}
	if len(listedTasks) != 1 {
		t.Fatalf("expected 1 listed task, got %d", len(listedTasks))
	}
	if listedTasks[0].ID != createdTask.ID {
		t.Fatalf("expected listed task ID %d, got %d", createdTask.ID, listedTasks[0].ID)
	}
	if getResponse.ID != createdTask.ID {
		t.Fatalf("expected fetched task ID %d, got %d", createdTask.ID, getResponse.ID)
	}
	if finalStatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404 after delete, got %d", finalStatusCode)
	}
}

func newIntegrationRouter(t *testing.T) http.Handler {
	t.Helper()

	db := newSQLiteDatabase(t)
	repository := sqlite.NewRepository(db)
	service := task.NewService(repository)
	logger := slog.New(slog.NewTextHandler(ioDiscard{}, nil))

	return api.NewRouter(api.RouterDependencies{
		Logger:      logger,
		TaskService: service,
	})
}

func newSQLiteDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "integration.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return db
}

func createTaskThroughAPI(t *testing.T, serverURL string, payload map[string]string) task.Task {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	response, err := http.Post(serverURL+"/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create task through API: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}

	var createdTask task.Task
	if err := json.NewDecoder(response.Body).Decode(&createdTask); err != nil {
		t.Fatalf("decode created task: %v", err)
	}

	return createdTask
}

func updateTaskThroughAPI(t *testing.T, serverURL string, taskID int64, payload map[string]string) task.Task {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/tasks/%d", serverURL, taskID), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build update request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("update task through API: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var updatedTask task.Task
	if err := json.NewDecoder(response.Body).Decode(&updatedTask); err != nil {
		t.Fatalf("decode updated task: %v", err)
	}

	return updatedTask
}

func listTasksThroughAPI(t *testing.T, serverURL, query string) []task.Task {
	t.Helper()

	response, err := http.Get(serverURL + "/tasks" + query)
	if err != nil {
		t.Fatalf("list tasks through API: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload struct {
		Tasks []task.Task `json:"tasks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	return payload.Tasks
}

func getTaskThroughAPI(t *testing.T, serverURL string, taskID int64) task.Task {
	t.Helper()

	response, err := http.Get(fmt.Sprintf("%s/tasks/%d", serverURL, taskID))
	if err != nil {
		t.Fatalf("get task through API: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var currentTask task.Task
	if err := json.NewDecoder(response.Body).Decode(&currentTask); err != nil {
		t.Fatalf("decode get response: %v", err)
	}

	return currentTask
}

func deleteTaskThroughAPI(t *testing.T, serverURL string, taskID int64) {
	t.Helper()

	request, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/tasks/%d", serverURL, taskID), nil)
	if err != nil {
		t.Fatalf("build delete request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("delete task through API: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.StatusCode)
	}
}

func getTaskStatusCode(t *testing.T, serverURL string, taskID int64) int {
	t.Helper()

	response, err := http.Get(fmt.Sprintf("%s/tasks/%d", serverURL, taskID))
	if err != nil {
		t.Fatalf("get task status code through API: %v", err)
	}
	defer response.Body.Close()

	return response.StatusCode
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

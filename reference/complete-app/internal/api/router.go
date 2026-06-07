package api

import (
	"log/slog"
	"net/http"

	"github.com/miuratouya/go-rest-api/internal/task"
)

type RouterDependencies struct {
	Logger      *slog.Logger
	TaskService task.Service
}

func NewRouter(dependencies RouterDependencies) http.Handler {
	handler := &Handler{
		taskService: dependencies.TaskService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.healthz)
	mux.HandleFunc("GET /tasks", handler.listTasks)
	mux.HandleFunc("GET /tasks/{id}", handler.getTask)
	mux.HandleFunc("POST /tasks", handler.createTask)
	mux.HandleFunc("PATCH /tasks/{id}", handler.updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", handler.deleteTask)

	withRequestID := withRequestID(mux)
	withRecover := withRecover(withRequestID, dependencies.Logger)
	return withLogging(withRecover, dependencies.Logger)
}

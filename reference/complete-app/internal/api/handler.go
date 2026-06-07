package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/miuratouya/go-rest-api/internal/task"
)

type Handler struct {
	taskService task.Service
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	filter, err := readFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	tasks, err := h.taskService.List(r.Context(), filter)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	currentTask, err := h.taskService.Get(r.Context(), id)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, currentTask)
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var input task.CreateInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	createdTask, err := h.taskService.Create(r.Context(), input)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, createdTask)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var input task.UpdateInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	updatedTask, err := h.taskService.Update(r.Context(), id, input)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updatedTask)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.taskService.Delete(r.Context(), id); err != nil {
		handleTaskError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, task.ErrInvalidTask):
		writeError(w, http.StatusBadRequest, "invalid_task", err.Error())
	case errors.Is(err, task.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "task_not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
	}
}

func readFilter(r *http.Request) (task.Filter, error) {
	query := r.URL.Query()
	filter := task.Filter{}

	if rawStatus := query.Get("status"); rawStatus != "" {
		status := task.Status(rawStatus)
		if err := status.Validate(); err != nil {
			return task.Filter{}, err
		}
		filter.Status = &status
	}

	if rawLimit := query.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return task.Filter{}, fmt.Errorf("limit must be an integer: %w", err)
		}
		filter.Limit = limit
	}

	return filter, nil
}

func parseTaskID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("path parameter id must be an integer: %w", err)
	}

	return id, nil
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("request body must contain a single JSON object: %w", err)
	}

	return errors.New("request body must contain a single JSON object")
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":{"code":"encode_error","message":"failed to encode response"}}`, http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

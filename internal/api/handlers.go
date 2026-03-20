package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"task-queue-mcp/internal/queue"
)

// Handler provides REST API handlers
type Handler struct {
	manager *queue.Manager
}

// NewHandler creates a new API handler
func NewHandler(manager *queue.Manager) *Handler {
	return &Handler{manager: manager}
}

// RegisterRoutes registers API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Queue endpoints
	mux.HandleFunc("GET /api/queues", h.ListQueues)
	mux.HandleFunc("POST /api/queues", h.CreateQueue)
	mux.HandleFunc("GET /api/queues/{id}", h.GetQueue)
	mux.HandleFunc("DELETE /api/queues/{id}", h.DeleteQueue)
	mux.HandleFunc("GET /api/queues/{id}/tasks", h.GetQueueTasks)
	mux.HandleFunc("GET /api/queues/{id}/stats", h.GetQueueStats)

	// Task endpoints
	mux.HandleFunc("POST /api/tasks", h.CreateTask)
	mux.HandleFunc("GET /api/tasks/{id}", h.GetTask)
	mux.HandleFunc("PATCH /api/tasks/{id}", h.UpdateTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.DeleteTask)
	mux.HandleFunc("POST /api/tasks/{id}/prioritize", h.PrioritizeTask)
	mux.HandleFunc("POST /api/tasks/{id}/start", h.StartTask)
	mux.HandleFunc("POST /api/tasks/{id}/finish", h.FinishTask)
}

// Queue handlers

func (h *Handler) ListQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := h.manager.ListQueues(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Add stats to each queue
	type QueueWithStats struct {
		*queue.Queue
		Stats *queue.QueueStats `json:"stats"`
	}

	result := make([]QueueWithStats, len(queues))
	for i, q := range queues {
		stats, err := h.manager.GetQueueStats(r.Context(), q.ID)
		if err != nil {
			stats = &queue.QueueStats{}
		}
		result[i] = QueueWithStats{Queue: q, Stats: stats}
	}

	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CreateQueue(w http.ResponseWriter, r *http.Request) {
	var input queue.CreateQueueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	q, err := h.manager.CreateQueue(r.Context(), input)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, q)
}

func (h *Handler) GetQueue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid queue ID")
		return
	}

	q, err := h.manager.GetQueue(r.Context(), id)
	if err != nil {
		if err == queue.ErrQueueNotFound {
			h.writeError(w, http.StatusNotFound, "Queue not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stats, err := h.manager.GetQueueStats(r.Context(), id)
	if err != nil {
		stats = &queue.QueueStats{}
	}

	result := struct {
		*queue.Queue
		Stats *queue.QueueStats `json:"stats"`
	}{
		Queue: q,
		Stats: stats,
	}

	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid queue ID")
		return
	}

	if err := h.manager.DeleteQueue(r.Context(), id); err != nil {
		if err == queue.ErrQueueNotFound {
			h.writeError(w, http.StatusNotFound, "Queue not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetQueueTasks(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid queue ID")
		return
	}

	var status *queue.TaskStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		s := queue.TaskStatus(statusStr)
		status = &s
	}

	tasks, err := h.manager.ListTasks(r.Context(), id, status)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) GetQueueStats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid queue ID")
		return
	}

	stats, err := h.manager.GetQueueStats(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// Task handlers

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var input queue.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task, err := h.manager.CreateTask(r.Context(), input)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := h.manager.GetTask(r.Context(), id)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	var input queue.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task, err := h.manager.UpdateTask(r.Context(), id, input)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	if err := h.manager.DeleteTask(r.Context(), id); err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PrioritizeTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	var input struct {
		Position int `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		input.Position = 1 // Default to front
	}

	task, err := h.manager.PrioritizeTask(r.Context(), id, input.Position)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) StartTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := h.manager.StartTask(r.Context(), id)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) FinishTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := h.manager.FinishTask(r.Context(), id)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

// Helper functions

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

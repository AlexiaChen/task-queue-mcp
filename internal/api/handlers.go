package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexiaChen/issue-kanban-mcp/internal/memory"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
)

// Handler provides REST API handlers
type Handler struct {
	manager       *queue.Manager
	memoryManager *memory.MemoryManager
	tripleManager *memory.TripleManager
}

// NewHandler creates a new API handler
func NewHandler(manager *queue.Manager) *Handler {
	return &Handler{manager: manager}
}

// SetMemoryManager sets the memory manager for memory API endpoints
func (h *Handler) SetMemoryManager(mm *memory.MemoryManager) {
	h.memoryManager = mm
}

// SetTripleManager sets the triple manager for knowledge graph API endpoints
func (h *Handler) SetTripleManager(tm *memory.TripleManager) {
	h.tripleManager = tm
}

// RegisterRoutes registers API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Project endpoints
	mux.HandleFunc("GET /api/projects", h.ListProjects)
	mux.HandleFunc("POST /api/projects", h.CreateProject)
	mux.HandleFunc("GET /api/projects/{id}", h.GetProject)
	mux.HandleFunc("DELETE /api/projects/{id}", h.DeleteProject)
	mux.HandleFunc("GET /api/projects/{id}/issues", h.GetProjectIssues)
	mux.HandleFunc("GET /api/projects/{id}/stats", h.GetProjectStats)

	// Issue endpoints
	mux.HandleFunc("POST /api/issues", h.CreateIssue)
	mux.HandleFunc("GET /api/issues/{id}", h.GetIssue)
	mux.HandleFunc("PATCH /api/issues/{id}", h.UpdateIssue)
	mux.HandleFunc("PUT /api/issues/{id}", h.EditIssue)
	mux.HandleFunc("DELETE /api/issues/{id}", h.DeleteIssue)
	mux.HandleFunc("POST /api/issues/{id}/prioritize", h.PrioritizeIssue)
	mux.HandleFunc("POST /api/issues/{id}/start", h.StartIssue)
	mux.HandleFunc("POST /api/issues/{id}/finish", h.FinishIssue)

	// Memory endpoints (only if memory manager is configured)
	if h.memoryManager != nil {
		mux.HandleFunc("POST /api/projects/{id}/memories", h.StoreMemory)
		mux.HandleFunc("GET /api/projects/{id}/memories", h.ListMemories)
		mux.HandleFunc("GET /api/projects/{id}/memories/search", h.SearchMemories)
		mux.HandleFunc("DELETE /api/projects/{id}/memories/{mid}", h.DeleteMemory)
	}

	// Triple endpoints (only if triple manager is configured)
	if h.tripleManager != nil {
		mux.HandleFunc("POST /api/projects/{id}/triples", h.StoreTriple)
		mux.HandleFunc("GET /api/projects/{id}/triples", h.QueryTriples)
		mux.HandleFunc("GET /api/projects/{id}/triples/{tid}", h.GetTriple)
		mux.HandleFunc("PATCH /api/projects/{id}/triples/{tid}", h.InvalidateTriple)
		mux.HandleFunc("DELETE /api/projects/{id}/triples/{tid}", h.DeleteTriple)
	}
}

// Queue handlers

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	queues, err := h.manager.ListProjects(r.Context())
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
		stats, err := h.manager.GetProjectStats(r.Context(), q.ID)
		if err != nil {
			stats = &queue.QueueStats{}
		}
		result[i] = QueueWithStats{Queue: q, Stats: stats}
	}

	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var input queue.CreateQueueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	q, err := h.manager.CreateProject(r.Context(), input)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, q)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	q, err := h.manager.GetProject(r.Context(), id)
	if err != nil {
		if err == queue.ErrQueueNotFound {
			h.writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stats, err := h.manager.GetProjectStats(r.Context(), id)
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

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	if err := h.manager.DeleteProject(r.Context(), id); err != nil {
		if err == queue.ErrQueueNotFound {
			h.writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		if err == queue.ErrCannotDeleteGlobalProject {
			h.writeError(w, http.StatusForbidden, err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetProjectIssues(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	var status *queue.TaskStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		s := queue.TaskStatus(statusStr)
		status = &s
	}

	tasks, err := h.manager.ListIssues(r.Context(), id, status)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) GetProjectStats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	stats, err := h.manager.GetProjectStats(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// Task handlers

func (h *Handler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	var input queue.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task, err := h.manager.CreateIssue(r.Context(), input)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) GetIssue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	task, err := h.manager.GetIssue(r.Context(), id)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Issue not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) UpdateIssue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	var input queue.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task, err := h.manager.UpdateIssue(r.Context(), id, input)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Issue not found")
			return
		}
		if err == queue.ErrInvalidStatus {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

// EditIssue handles PUT /api/issues/{id} — updates content fields of a pending issue.
func (h *Handler) EditIssue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	var input queue.EditTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task, err := h.manager.EditIssue(r.Context(), id, input)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Issue not found")
			return
		}
		if err == queue.ErrCannotEditNonPending {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) DeleteIssue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	if err := h.manager.DeleteIssue(r.Context(), id); err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Issue not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PrioritizeIssue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	task, err := h.manager.PrioritizeIssue(r.Context(), id)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Issue not found")
			return
		}
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) StartIssue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	task, err := h.manager.StartIssue(r.Context(), id)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Issue not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

func (h *Handler) FinishIssue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	task, err := h.manager.FinishIssue(r.Context(), id)
	if err != nil {
		if err == queue.ErrTaskNotFound {
			h.writeError(w, http.StatusNotFound, "Issue not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

// Memory handlers

func (h *Handler) StoreMemory(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var input memory.StoreMemoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.ProjectID = projectID

	mem, err := h.memoryManager.Store(r.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case memory.ErrEmptyContent, memory.ErrContentTooLong, memory.ErrInvalidCategory, memory.ErrInvalidImportance:
			status = http.StatusBadRequest
		}
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, mem)
}

func (h *Handler) ListMemories(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	opts := memory.ListOptions{
		Category: r.URL.Query().Get("category"),
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if opts.Limit, err = strconv.Atoi(limitStr); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if opts.Offset, err = strconv.Atoi(offsetStr); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}

	mems, err := h.memoryManager.List(r.Context(), projectID, opts)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if mems == nil {
		mems = []*memory.Memory{}
	}

	h.writeJSON(w, http.StatusOK, mems)
}

func (h *Handler) SearchMemories(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		h.writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	opts := memory.SearchOptions{
		Category: r.URL.Query().Get("category"),
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if opts.Limit, err = strconv.Atoi(limitStr); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}

	results, err := h.memoryManager.Search(r.Context(), projectID, query, opts)
	if err != nil {
		status := http.StatusInternalServerError
		if err == memory.ErrEmptyQuery || err == memory.ErrInvalidCategory {
			status = http.StatusBadRequest
		}
		h.writeError(w, status, err.Error())
		return
	}
	if results == nil {
		results = []memory.MemorySearchResult{}
	}

	h.writeJSON(w, http.StatusOK, results)
}

func (h *Handler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	memoryID, err := strconv.ParseInt(r.PathValue("mid"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid memory ID")
		return
	}

	if err := h.memoryManager.Delete(r.Context(), projectID, memoryID); err != nil {
		status := http.StatusInternalServerError
		switch err {
		case memory.ErrMemoryNotFound:
			status = http.StatusNotFound
		case memory.ErrMemoryNotInProject:
			status = http.StatusForbidden
		}
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "memory deleted"})
}

// Triple handlers

func (h *Handler) StoreTriple(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var input memory.StoreTripleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.ProjectID = projectID

	triple, err := h.tripleManager.Store(r.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case memory.ErrEmptySubject, memory.ErrEmptyPredicate, memory.ErrEmptyObject,
			memory.ErrSubjectTooLong, memory.ErrPredicateTooLong, memory.ErrObjectTooLong,
			memory.ErrInvalidConfidence, memory.ErrInvalidTimeRange:
			status = http.StatusBadRequest
		}
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, triple)
}

func (h *Handler) GetTriple(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	tripleID, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid triple ID")
		return
	}

	triple, err := h.tripleManager.Get(r.Context(), projectID, tripleID)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case memory.ErrTripleNotFound:
			status = http.StatusNotFound
		case memory.ErrTripleNotInProject:
			status = http.StatusForbidden
		}
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, triple)
}

func (h *Handler) QueryTriples(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	opts := memory.QueryTripleOptions{
		Subject:     r.URL.Query().Get("subject"),
		Predicate:   r.URL.Query().Get("predicate"),
		Object:      r.URL.Query().Get("object"),
		PointInTime: r.URL.Query().Get("point_in_time"),
	}
	if r.URL.Query().Get("active_only") == "true" {
		opts.ActiveOnly = true
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if opts.Limit, err = strconv.Atoi(limitStr); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if opts.Offset, err = strconv.Atoi(offsetStr); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}

	triples, err := h.tripleManager.Query(r.Context(), projectID, opts)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, triples)
}

func (h *Handler) InvalidateTriple(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	tripleID, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid triple ID")
		return
	}

	var body struct {
		ValidTo string `json:"valid_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validTo := time.Now().UTC()
	if body.ValidTo != "" {
		parsed, parseErr := time.Parse(time.RFC3339, body.ValidTo)
		if parseErr != nil {
			h.writeError(w, http.StatusBadRequest, "invalid valid_to format (use RFC3339)")
			return
		}
		validTo = parsed.UTC()
	}

	if err := h.tripleManager.Invalidate(r.Context(), projectID, tripleID, validTo); err != nil {
		status := http.StatusInternalServerError
		switch err {
		case memory.ErrTripleNotFound:
			status = http.StatusNotFound
		case memory.ErrTripleNotInProject:
			status = http.StatusForbidden
		}
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "triple invalidated"})
}

func (h *Handler) DeleteTriple(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	tripleID, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid triple ID")
		return
	}

	if err := h.tripleManager.Delete(r.Context(), projectID, tripleID); err != nil {
		status := http.StatusInternalServerError
		switch err {
		case memory.ErrTripleNotFound:
			status = http.StatusNotFound
		case memory.ErrTripleNotInProject:
			status = http.StatusForbidden
		}
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "triple deleted"})
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

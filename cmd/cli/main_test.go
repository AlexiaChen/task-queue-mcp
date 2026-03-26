package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"task-queue-mcp/internal/apiclient"
	"task-queue-mcp/internal/queue"
)

// mockQueue is the shared fixture used across tests.
var mockQueue = apiclient.QueueWithStats{
	Queue: queue.Queue{
		ID:          1,
		Name:        "Test Queue",
		Description: "A test queue",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	},
	Stats: queue.QueueStats{Total: 2, Pending: 1, Doing: 1, Finished: 0},
}

var mockTask = queue.Task{
	ID:          10,
	QueueID:     1,
	Title:       "Test Task",
	Description: "A test task",
	Status:      queue.StatusPending,
	Priority:    0,
	CreatedAt:   time.Now(),
	UpdatedAt:   time.Now(),
}

// runCmd executes a root command with the given args against ts and returns stdout.
func runCmd(t *testing.T, ts *httptest.Server, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"--server", ts.URL}, args...))
	err := root.Execute()
	return out.String(), err
}

func TestQueuesList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/projects" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]apiclient.QueueWithStats{mockQueue})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "projects", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Test Queue") {
		t.Errorf("expected 'Test Queue' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header row in output, got:\n%s", out)
	}
}

func TestQueuesCreate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/projects" {
			var input queue.CreateQueueInput
			json.NewDecoder(r.Body).Decode(&input)
			q := queue.Queue{ID: 99, Name: input.Name, Description: input.Description}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(q)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "projects", "create", "--name", "New Queue", "--desc", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created project") {
		t.Errorf("expected 'Created project' in output, got:\n%s", out)
	}
}

func TestQueuesStats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/projects/1/stats" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockQueue.Stats)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "projects", "stats", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Pending") {
		t.Errorf("expected stats output, got:\n%s", out)
	}
}

func TestQueuesDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/projects/1" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "projects", "delete", "--yes", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Deleted project") {
		t.Errorf("expected 'Deleted project' in output, got:\n%s", out)
	}
}

func TestTasksList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/projects/1/issues" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]queue.Task{mockTask})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "issues", "list", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Test Task") {
		t.Errorf("expected 'Test Task' in output, got:\n%s", out)
	}
}

func TestTasksListWithStatusFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/projects/1/issues" {
			if r.URL.Query().Get("status") != "pending" {
				http.Error(w, "expected status=pending", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]queue.Task{mockTask})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	_, err := runCmd(t, ts, "issues", "list", "--status", "pending", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTasksGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/issues/10" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockTask)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "issues", "get", "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Test Task") {
		t.Errorf("expected task details, got:\n%s", out)
	}
}

func TestTasksCreate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/issues" {
			var input queue.CreateTaskInput
			json.NewDecoder(r.Body).Decode(&input)
			t := queue.Task{ID: 42, QueueID: input.QueueID, Title: input.Title, Status: queue.StatusPending}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(t)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "issues", "create", "1", "--title", "New Task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created issue") {
		t.Errorf("expected 'Created issue', got:\n%s", out)
	}
}

func TestTasksEdit(t *testing.T) {
	edited := mockTask
	edited.Title = "Updated Title"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/api/issues/10" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(edited)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "issues", "edit", "10", "--title", "Updated Title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("expected 'updated', got:\n%s", out)
	}
}

func TestTasksDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/issues/10" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "issues", "delete", "--yes", "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Deleted issue") {
		t.Errorf("expected 'Deleted issue', got:\n%s", out)
	}
}

func TestTasksPrioritize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/issues/10/prioritize" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockTask)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out, err := runCmd(t, ts, "issues", "prioritize", "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "moved to front") {
		t.Errorf("expected 'moved to front', got:\n%s", out)
	}
}

func TestServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "queue not found"})
	}))
	defer ts.Close()

	_, err := runCmd(t, ts, "projects", "stats", "999")
	if err == nil {
		t.Fatal("expected error for not found, got nil")
	}
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
)

func setupTestAPI(t *testing.T) (*Handler, *queue.Manager, *queue.MockStorage) {
	storage := queue.NewMockStorage()
	manager := queue.NewManager(storage)
	handler := NewHandler(manager)
	return handler, manager, storage
}

func TestAPI_ListQueues(t *testing.T) {
	handler, _, _ := setupTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/queues", nil)
	rec := httptest.NewRecorder()

	handler.ListProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestAPI_CreateQueue(t *testing.T) {
	handler, _, _ := setupTestAPI(t)

	body := bytes.NewBufferString(`{"name":"Test Queue","description":"Test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/queues", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateProject(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var q map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &q); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if q["name"] != "Test Queue" {
		t.Errorf("Expected name 'Test Queue', got %v", q["name"])
	}
}

func TestAPI_CreateQueue_InvalidBody(t *testing.T) {
	handler, _, _ := setupTestAPI(t)

	body := bytes.NewBufferString(`invalid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/queues", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestAPI_GetQueue(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create a queue first
	manager.CreateProject(context.Background(), queue.CreateQueueInput{
		Name: "Test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/queues/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.GetProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if result["name"] != "Test" {
		t.Errorf("Expected name 'Test', got %v", result["name"])
	}
}

func TestAPI_GetQueue_InvalidID(t *testing.T) {
	handler, _, _ := setupTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/queues/invalid", nil)
	req.SetPathValue("id", "invalid")
	rec := httptest.NewRecorder()

	handler.GetProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestAPI_GetQueue_NotFound(t *testing.T) {
	handler, _, _ := setupTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/queues/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()

	handler.GetProject(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestAPI_DeleteQueue(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create a queue first
	manager.CreateProject(context.Background(), queue.CreateQueueInput{
		Name: "Test",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/queues/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.DeleteProject(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

func TestAPI_CreateTask(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create a queue first
	manager.CreateProject(context.Background(), queue.CreateQueueInput{
		Name: "Test",
	})

	body := bytes.NewBufferString(`{"project_id":1,"title":"Test Task","description":"Desc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateIssue(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var task map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if task["title"] != "Test Task" {
		t.Errorf("Expected title 'Test Task', got %v", task["title"])
	}
}

func TestAPI_UpdateTask(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create queue and task
	q, _ := manager.CreateProject(context.Background(), queue.CreateQueueInput{Name: "Test"})
	manager.CreateIssue(context.Background(), queue.CreateTaskInput{
		ProjectID: q.ID,
		Title:   "Task",
	})

	body := bytes.NewBufferString(`{"status":"doing"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.UpdateIssue(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if result["status"] != "doing" {
		t.Errorf("Expected status 'doing', got %v", result["status"])
	}
}

func TestAPI_DeleteTask(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create queue and task
	q, _ := manager.CreateProject(context.Background(), queue.CreateQueueInput{Name: "Test"})
	manager.CreateIssue(context.Background(), queue.CreateTaskInput{
		ProjectID: q.ID,
		Title:   "Task",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.DeleteIssue(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

func TestAPI_PrioritizeTask(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create queue and task
	q, _ := manager.CreateProject(context.Background(), queue.CreateQueueInput{Name: "Test"})
	manager.CreateIssue(context.Background(), queue.CreateTaskInput{
		ProjectID: q.ID,
		Title:   "Task",
	})

	body := bytes.NewBufferString(`{"position":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/prioritize", body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.PrioritizeIssue(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_StartTask(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create queue and task
	q, _ := manager.CreateProject(context.Background(), queue.CreateQueueInput{Name: "Test"})
	manager.CreateIssue(context.Background(), queue.CreateTaskInput{
		ProjectID: q.ID,
		Title:   "Task",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/start", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.StartIssue(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if result["status"] != "doing" {
		t.Errorf("Expected status 'doing', got %v", result["status"])
	}
}

func TestAPI_FinishTask(t *testing.T) {
	handler, manager, _ := setupTestAPI(t)

	// Create queue and task, start it first
	q, _ := manager.CreateProject(context.Background(), queue.CreateQueueInput{Name: "Test"})
	task, _ := manager.CreateIssue(context.Background(), queue.CreateTaskInput{
		ProjectID: q.ID,
		Title:   "Task",
	})
	manager.StartIssue(context.Background(), task.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/finish", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.FinishIssue(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if result["status"] != "finished" {
		t.Errorf("Expected status 'finished', got %v", result["status"])
	}
}

func TestAPI_DeleteGlobalProject_Forbidden(t *testing.T) {
handler, _, _ := setupTestAPI(t)

req := httptest.NewRequest(http.MethodDelete, "/api/projects/0", nil)
req.SetPathValue("id", "0")
rec := httptest.NewRecorder()

handler.DeleteProject(rec, req)

if rec.Code != http.StatusForbidden {
t.Errorf("expected status 403 when deleting global project, got %d: %s", rec.Code, rec.Body.String())
}
}

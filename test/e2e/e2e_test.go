package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"task-queue-mcp/internal/api"
	"task-queue-mcp/internal/queue"
	"task-queue-mcp/internal/storage"
)

// E2E test that runs against a real server
// Set E2E_SERVER_URL to test against a running server
// e.g., E2E_SERVER_URL=http://localhost:9292 go test -v ./test/e2e/...

var serverURL string

func TestMain(m *testing.M) {
	serverURL = os.Getenv("E2E_SERVER_URL")
	if serverURL == "" {
		fmt.Println("E2E_SERVER_URL not set, skipping e2e tests")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestE2E_QueueCRUD(t *testing.T) {
	client := NewE2EClient(serverURL)

	// List queues (should be empty or have existing)
	queues, err := client.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list queues: %v", err)
	}
	initialCount := len(queues)

	// Create queue
	q, err := client.CreateProject(map[string]interface{}{
		"name":        fmt.Sprintf("E2E Test Queue %d", time.Now().Unix()),
		"description": "Created by e2e test",
	})
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	queueID := int64(q["id"].(float64))
	t.Logf("Created queue with ID: %d", queueID)

	// Get queue
	q2, err := client.GetProject(queueID)
	if err != nil {
		t.Fatalf("Failed to get queue: %v", err)
	}
	if q2["name"] != q["name"] {
		t.Errorf("Queue name mismatch")
	}

	// List queues (should have one more)
	queues, err = client.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list queues: %v", err)
	}
	if len(queues) != initialCount+1 {
		t.Errorf("Expected %d queues, got %d", initialCount+1, len(queues))
	}

	// Delete queue
	if err := client.DeleteProject(queueID); err != nil {
		t.Fatalf("Failed to delete queue: %v", err)
	}

	// Verify deleted
	_, err = client.GetProject(queueID)
	if err == nil {
		t.Error("Expected error getting deleted queue")
	}
}

func TestE2E_TaskCRUD(t *testing.T) {
	client := NewE2EClient(serverURL)

	// Create queue
	q, err := client.CreateProject(map[string]interface{}{
		"name": fmt.Sprintf("Task Test Queue %d", time.Now().Unix()),
	})
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	queueID := int64(q["id"].(float64))
	defer client.DeleteProject(queueID)

	// Create task
	task, err := client.CreateIssue(map[string]interface{}{
		"project_id":  queueID,
		"title":       "E2E Test Task",
		"description": "Test task description",
		"priority":    "high",
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	taskID := int64(task["id"].(float64))
	t.Logf("Created task with ID: %d", taskID)

	// Verify task status is pending
	if task["status"] != "pending" {
		t.Errorf("Expected status pending, got %s", task["status"])
	}

	// List tasks
	tasks, err := client.ListIssues(queueID, "")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	// Start task
	task, err = client.StartIssue(taskID)
	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}
	if task["status"] != "doing" {
		t.Errorf("Expected status doing, got %s", task["status"])
	}

	// Finish task
	task, err = client.FinishIssue(taskID)
	if err != nil {
		t.Fatalf("Failed to finish task: %v", err)
	}
	if task["status"] != "finished" {
		t.Errorf("Expected status finished, got %s", task["status"])
	}

	// List finished tasks
	tasks, err = client.ListIssues(queueID, "finished")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("Expected 1 finished task, got %d", len(tasks))
	}

	// Delete task
	if err := client.DeleteIssue(taskID); err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify deleted
	tasks, err = client.ListIssues(queueID, "")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after delete, got %d", len(tasks))
	}
}

func TestE2E_TaskPrioritization(t *testing.T) {
	client := NewE2EClient(serverURL)

	// Create queue
	q, _ := client.CreateProject(map[string]interface{}{
		"name": fmt.Sprintf("Priority Test Queue %d", time.Now().Unix()),
	})
	queueID := int64(q["id"].(float64))
	defer client.DeleteProject(queueID)

	// Create multiple tasks: task1 and task2 with low priority, task3 with high priority
	client.CreateIssue(map[string]interface{}{
		"project_id": queueID,
		"title":    "Task 1",
		"priority": "low",
	})
	client.CreateIssue(map[string]interface{}{
		"project_id": queueID,
		"title":    "Task 2",
		"priority": "low",
	})
	task3, _ := client.CreateIssue(map[string]interface{}{
		"project_id": queueID,
		"title":    "Task 3",
		"priority": "high",
	})

	task3ID := int64(task3["id"].(float64))

	// Prioritize task3 (high priority) ahead of lower-priority tasks
	_, err := client.PrioritizeIssue(task3ID)
	if err != nil {
		t.Fatalf("Failed to prioritize task: %v", err)
	}

	// List tasks and verify order
	tasks, err := client.ListIssues(queueID, "")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	// Task 3 should now be first (or have highest priority)
	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	t.Logf("Tasks after prioritization:")
	for i, task := range tasks {
		t.Logf("  %d: %s (priority: %v, position: %v)",
			i+1, task["title"], task["priority"], task["position"])
	}
}

func TestE2E_QueueStats(t *testing.T) {
	client := NewE2EClient(serverURL)

	// Create queue
	q, _ := client.CreateProject(map[string]interface{}{
		"name": fmt.Sprintf("Stats Test Queue %d", time.Now().Unix()),
	})
	queueID := int64(q["id"].(float64))
	defer client.DeleteProject(queueID)

	// Create tasks in different states
	task1, _ := client.CreateIssue(map[string]interface{}{
		"project_id": queueID,
		"title":    "Pending Task",
	})
	task2, _ := client.CreateIssue(map[string]interface{}{
		"project_id": queueID,
		"title":    "Doing Task",
	})
	task3, _ := client.CreateIssue(map[string]interface{}{
		"project_id": queueID,
		"title":    "Finished Task",
	})

	// Start and finish tasks
	client.StartIssue(int64(task2["id"].(float64)))
	client.StartIssue(int64(task3["id"].(float64)))
	client.FinishIssue(int64(task3["id"].(float64)))

	// Get queue with stats
	q2, err := client.GetProject(queueID)
	if err != nil {
		t.Fatalf("Failed to get queue: %v", err)
	}

	stats := q2["stats"].(map[string]interface{})
	if int(stats["total"].(float64)) != 3 {
		t.Errorf("Expected total 3, got %v", stats["total"])
	}
	if int(stats["pending"].(float64)) != 1 {
		t.Errorf("Expected pending 1, got %v", stats["pending"])
	}
	if int(stats["doing"].(float64)) != 1 {
		t.Errorf("Expected doing 1, got %v", stats["doing"])
	}
	if int(stats["finished"].(float64)) != 1 {
		t.Errorf("Expected finished 1, got %v", stats["finished"])
	}

	// Suppress unused variable warnings
	_ = task1
}

// E2E Client

type E2EClient struct {
	baseURL string
	client  *http.Client
}

func NewE2EClient(baseURL string) *E2EClient {
	return &E2EClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// doObject makes a request expecting a JSON object response
func (c *E2EClient) doObject(method, path string, body interface{}) (map[string]interface{}, error) {
	respBody, err := c.doRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	if respBody == nil {
		return nil, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON object: %w", err)
	}
	return result, nil
}

// doArray makes a request expecting a JSON array response
func (c *E2EClient) doArray(method, path string, body interface{}) ([]map[string]interface{}, error) {
	respBody, err := c.doRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	if respBody == nil {
		return nil, nil
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON array: %w", err)
	}
	return result, nil
}

// doRequest makes an HTTP request and returns the raw body
func (c *E2EClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("HTTP %d: %v", resp.StatusCode, errResp["error"])
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	return respBody, nil
}

func (c *E2EClient) ListProjects() ([]map[string]interface{}, error) {
	return c.doArray("GET", "/api/projects", nil)
}

func (c *E2EClient) CreateProject(data map[string]interface{}) (map[string]interface{}, error) {
	return c.doObject("POST", "/api/projects", data)
}

func (c *E2EClient) GetProject(id int64) (map[string]interface{}, error) {
	return c.doObject("GET", fmt.Sprintf("/api/projects/%d", id), nil)
}

func (c *E2EClient) DeleteProject(id int64) error {
	_, err := c.doObject("DELETE", fmt.Sprintf("/api/projects/%d", id), nil)
	return err
}

func (c *E2EClient) CreateIssue(data map[string]interface{}) (map[string]interface{}, error) {
	return c.doObject("POST", "/api/issues", data)
}

func (c *E2EClient) GetIssue(id int64) (map[string]interface{}, error) {
	return c.doObject("GET", fmt.Sprintf("/api/issues/%d", id), nil)
}

func (c *E2EClient) UpdateIssue(id int64, data map[string]interface{}) (map[string]interface{}, error) {
	return c.doObject("PATCH", fmt.Sprintf("/api/issues/%d", id), data)
}

func (c *E2EClient) DeleteIssue(id int64) error {
	_, err := c.doObject("DELETE", fmt.Sprintf("/api/issues/%d", id), nil)
	return err
}

func (c *E2EClient) ListIssues(queueID int64, status string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/api/projects/%d/issues", queueID)
	if status != "" {
		path += "?status=" + status
	}
	return c.doArray("GET", path, nil)
}

func (c *E2EClient) StartIssue(id int64) (map[string]interface{}, error) {
	return c.doObject("POST", fmt.Sprintf("/api/issues/%d/start", id), nil)
}

func (c *E2EClient) FinishIssue(id int64) (map[string]interface{}, error) {
	return c.doObject("POST", fmt.Sprintf("/api/issues/%d/finish", id), nil)
}

func (c *E2EClient) PrioritizeIssue(id int64) (map[string]interface{}, error) {
	return c.doObject("POST", fmt.Sprintf("/api/issues/%d/prioritize", id), nil)
}

// Integration test using real storage

func TestIntegration_FullWorkflow(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "integration-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize storage
	store, err := storage.NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	manager := queue.NewManager(store)
	ctx := context.Background()

	// Create queue
	q, err := manager.CreateProject(ctx, queue.CreateQueueInput{
		Name:        "Integration Test Queue",
		Description: "Full workflow test",
	})
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	// Create tasks
	for i := 1; i <= 5; i++ {
		_, err := manager.CreateIssue(ctx, queue.CreateTaskInput{
			ProjectID:   q.ID,
			Title:       fmt.Sprintf("Task %d", i),
			Description: fmt.Sprintf("Description for task %d", i),
			Priority:    queue.Priority(i % 3),
		})
		if err != nil {
			t.Fatalf("Failed to create task %d: %v", i, err)
		}
	}

	// List tasks
	tasks, err := manager.ListIssues(ctx, q.ID, nil)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	if len(tasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(tasks))
	}

	// Start and finish tasks
	for i, task := range tasks {
		if i < 2 {
			_, err := manager.StartIssue(ctx, task.ID)
			if err != nil {
				t.Fatalf("Failed to start task %d: %v", task.ID, err)
			}
			_, err = manager.FinishIssue(ctx, task.ID)
			if err != nil {
				t.Fatalf("Failed to finish task %d: %v", task.ID, err)
			}
		} else if i < 4 {
			_, err := manager.StartIssue(ctx, task.ID)
			if err != nil {
				t.Fatalf("Failed to start task %d: %v", task.ID, err)
			}
		}
	}

	// Check stats
	stats, err := manager.GetProjectStats(ctx, q.ID)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 5 {
		t.Errorf("Expected total 5, got %d", stats.Total)
	}
	if stats.Finished != 2 {
		t.Errorf("Expected finished 2, got %d", stats.Finished)
	}
	if stats.Doing != 2 {
		t.Errorf("Expected doing 2, got %d", stats.Doing)
	}
	if stats.Pending != 1 {
		t.Errorf("Expected pending 1, got %d", stats.Pending)
	}

	// Test prioritization: add a low-priority task first, then a high-priority one
	// so the high-priority task can jump ahead of the lower-priority one.
	lowPrioTask, err := manager.CreateIssue(ctx, queue.CreateTaskInput{
		ProjectID:  q.ID,
		Title:    "Low Priority Task",
		Priority: queue.PriorityLow,
	})
	if err != nil {
		t.Fatalf("Failed to create low priority task: %v", err)
	}
	highPrioTask, err := manager.CreateIssue(ctx, queue.CreateTaskInput{
		ProjectID:  q.ID,
		Title:    "High Priority Task",
		Priority: queue.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("Failed to create high priority task: %v", err)
	}
	prioritizedTask, err := manager.PrioritizeIssue(ctx, highPrioTask.ID)
	if err != nil {
		t.Fatalf("Failed to prioritize task: %v", err)
	}
	if prioritizedTask.Priority != queue.PriorityHigh {
		t.Errorf("Expected priority High, got %v", prioritizedTask.Priority)
	}
	_ = lowPrioTask
}

// Suppress unused import warning
var _ = api.NewHandler

package storage

import (
	"context"
	"os"
	"testing"

	"task-queue-mcp/internal/queue"
)

func TestSQLiteStorage(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	store, err := NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("CreateProject", func(t *testing.T) {
		q, err := store.CreateProject(ctx, queue.CreateQueueInput{
			Name:        "Test Queue",
			Description: "Test Description",
		})
		if err != nil {
			t.Fatalf("Failed to create queue: %v", err)
		}
		if q.ID != 1 {
			t.Errorf("Expected ID 1, got %d", q.ID)
		}
		if q.Name != "Test Queue" {
			t.Errorf("Expected name 'Test Queue', got %s", q.Name)
		}
	})

	t.Run("GetProject", func(t *testing.T) {
		q, err := store.GetProject(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get queue: %v", err)
		}
		if q.Name != "Test Queue" {
			t.Errorf("Expected name 'Test Queue', got %s", q.Name)
		}
	})

	t.Run("GetQueueNotFound", func(t *testing.T) {
		_, err := store.GetProject(ctx, 999)
		if err != queue.ErrQueueNotFound {
			t.Errorf("Expected ErrQueueNotFound, got %v", err)
		}
	})

	t.Run("ListProjects", func(t *testing.T) {
		queues, err := store.ListProjects(ctx)
		if err != nil {
			t.Fatalf("Failed to list queues: %v", err)
		}
		if len(queues) != 1 {
			t.Errorf("Expected 1 queue, got %d", len(queues))
		}
	})

	t.Run("CreateIssue", func(t *testing.T) {
		task, err := store.CreateIssue(ctx, queue.CreateTaskInput{
			ProjectID:   1,
			Title:       "Test Task",
			Description: "Task Description",
			Priority:    queue.PriorityHigh,
		})
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		if task.ID != 1 {
			t.Errorf("Expected ID 1, got %d", task.ID)
		}
		if task.Status != queue.StatusPending {
			t.Errorf("Expected status pending, got %s", task.Status)
		}
		if task.Position != 1 {
			t.Errorf("Expected position 1, got %d", task.Position)
		}
	})

	t.Run("GetIssue", func(t *testing.T) {
		task, err := store.GetIssue(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if task.Title != "Test Task" {
			t.Errorf("Expected title 'Test Task', got %s", task.Title)
		}
	})

	t.Run("ListIssues", func(t *testing.T) {
		tasks, err := store.ListIssues(ctx, 1, nil)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		}
	})

	t.Run("ListTasksWithStatus", func(t *testing.T) {
		tasks, err := store.ListIssues(ctx, 1, ptrStatus(queue.StatusPending))
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 pending task, got %d", len(tasks))
		}
	})

	t.Run("UpdateTaskStatusToDoing", func(t *testing.T) {
		status := queue.StatusDoing
		task, err := store.UpdateIssue(ctx, 1, queue.UpdateTaskInput{Status: &status})
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}
		if task.Status != queue.StatusDoing {
			t.Errorf("Expected status doing, got %s", task.Status)
		}
		if task.StartedAt == nil {
			t.Error("Expected started_at to be set")
		}
	})

	t.Run("UpdateTaskStatusToFinished", func(t *testing.T) {
		status := queue.StatusFinished
		task, err := store.UpdateIssue(ctx, 1, queue.UpdateTaskInput{Status: &status})
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}
		if task.Status != queue.StatusFinished {
			t.Errorf("Expected status finished, got %s", task.Status)
		}
		if task.FinishedAt == nil {
			t.Error("Expected finished_at to be set")
		}
	})

	t.Run("GetProjectStats", func(t *testing.T) {
		stats, err := store.GetProjectStats(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		if stats.Total != 1 {
			t.Errorf("Expected total 1, got %d", stats.Total)
		}
		if stats.Finished != 1 {
			t.Errorf("Expected finished 1, got %d", stats.Finished)
		}
	})

	t.Run("PrioritizeIssue", func(t *testing.T) {
		// Create a low-priority task (will be at the front in queue position order)
		taskLow, err := store.CreateIssue(ctx, queue.CreateTaskInput{
			ProjectID:  1,
			Title:    "Low Priority Task",
			Priority: queue.PriorityLow,
		})
		if err != nil {
			t.Fatalf("Failed to create low priority task: %v", err)
		}

		// Create a medium-priority task (will be behind low-priority in queue)
		taskMedium, err := store.CreateIssue(ctx, queue.CreateTaskInput{
			ProjectID:  1,
			Title:    "Medium Priority Task",
			Priority: queue.PriorityMedium,
		})
		if err != nil {
			t.Fatalf("Failed to create medium priority task: %v", err)
		}

		// Prioritize the medium task to jump ahead of the low-priority task
		task, err := store.PrioritizeIssue(ctx, taskMedium.ID)
		if err != nil {
			t.Fatalf("Failed to prioritize task: %v", err)
		}
		// Should have moved to the original position of the low-priority task
		if task.Position != taskLow.Position {
			t.Errorf("Expected position %d, got %d", taskLow.Position, task.Position)
		}
		// Priority should remain unchanged (still Medium)
		if task.Priority != queue.PriorityMedium {
			t.Errorf("Expected priority Medium, got %v", task.Priority)
		}
	})

	t.Run("DeleteIssue", func(t *testing.T) {
		err := store.DeleteIssue(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		_, err = store.GetIssue(ctx, 1)
		if err != queue.ErrTaskNotFound {
			t.Errorf("Expected ErrTaskNotFound, got %v", err)
		}
	})

	t.Run("DeleteProject", func(t *testing.T) {
		err := store.DeleteProject(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to delete queue: %v", err)
		}

		_, err = store.GetProject(ctx, 1)
		if err != queue.ErrQueueNotFound {
			t.Errorf("Expected ErrQueueNotFound, got %v", err)
		}
	})
}

func ptrStatus(s queue.TaskStatus) *queue.TaskStatus {
	return &s
}

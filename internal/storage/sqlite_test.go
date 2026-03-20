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

	t.Run("CreateQueue", func(t *testing.T) {
		q, err := store.CreateQueue(ctx, queue.CreateQueueInput{
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

	t.Run("GetQueue", func(t *testing.T) {
		q, err := store.GetQueue(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get queue: %v", err)
		}
		if q.Name != "Test Queue" {
			t.Errorf("Expected name 'Test Queue', got %s", q.Name)
		}
	})

	t.Run("GetQueueNotFound", func(t *testing.T) {
		_, err := store.GetQueue(ctx, 999)
		if err != queue.ErrQueueNotFound {
			t.Errorf("Expected ErrQueueNotFound, got %v", err)
		}
	})

	t.Run("ListQueues", func(t *testing.T) {
		queues, err := store.ListQueues(ctx)
		if err != nil {
			t.Fatalf("Failed to list queues: %v", err)
		}
		if len(queues) != 1 {
			t.Errorf("Expected 1 queue, got %d", len(queues))
		}
	})

	t.Run("CreateTask", func(t *testing.T) {
		task, err := store.CreateTask(ctx, queue.CreateTaskInput{
			QueueID:     1,
			Title:       "Test Task",
			Description: "Task Description",
			Priority:    5,
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

	t.Run("GetTask", func(t *testing.T) {
		task, err := store.GetTask(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if task.Title != "Test Task" {
			t.Errorf("Expected title 'Test Task', got %s", task.Title)
		}
	})

	t.Run("ListTasks", func(t *testing.T) {
		tasks, err := store.ListTasks(ctx, 1, nil)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		}
	})

	t.Run("ListTasksWithStatus", func(t *testing.T) {
		tasks, err := store.ListTasks(ctx, 1, ptrStatus(queue.StatusPending))
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 pending task, got %d", len(tasks))
		}
	})

	t.Run("UpdateTaskStatusToDoing", func(t *testing.T) {
		status := queue.StatusDoing
		task, err := store.UpdateTask(ctx, 1, queue.UpdateTaskInput{Status: &status})
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
		task, err := store.UpdateTask(ctx, 1, queue.UpdateTaskInput{Status: &status})
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

	t.Run("GetQueueStats", func(t *testing.T) {
		stats, err := store.GetQueueStats(ctx, 1)
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

	t.Run("PrioritizeTask", func(t *testing.T) {
		// Create another task
		task2, err := store.CreateTask(ctx, queue.CreateTaskInput{
			QueueID: 1,
			Title:   "Second Task",
		})
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Prioritize task2 to front
		task, err := store.PrioritizeTask(ctx, task2.ID, 1)
		if err != nil {
			t.Fatalf("Failed to prioritize task: %v", err)
		}
		if task.Position != 1 {
			t.Errorf("Expected position 1, got %d", task.Position)
		}
		if task.Priority != 1000 {
			t.Errorf("Expected priority 1000, got %d", task.Priority)
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		err := store.DeleteTask(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		_, err = store.GetTask(ctx, 1)
		if err != queue.ErrTaskNotFound {
			t.Errorf("Expected ErrTaskNotFound, got %v", err)
		}
	})

	t.Run("DeleteQueue", func(t *testing.T) {
		err := store.DeleteQueue(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to delete queue: %v", err)
		}

		_, err = store.GetQueue(ctx, 1)
		if err != queue.ErrQueueNotFound {
			t.Errorf("Expected ErrQueueNotFound, got %v", err)
		}
	})
}

func ptrStatus(s queue.TaskStatus) *queue.TaskStatus {
	return &s
}

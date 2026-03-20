package queue

import (
	"context"
	"testing"
)

func TestManager_CreateQueue(t *testing.T) {
	m := NewManager(NewMockStorage())

	q, err := m.CreateQueue(context.Background(), CreateQueueInput{
		Name:        "Test",
		Description: "Desc",
	})
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if q.Name != "Test" {
		t.Errorf("Expected name 'Test', got %s", q.Name)
	}
}

func TestManager_CreateQueue_EmptyName(t *testing.T) {
	m := NewManager(NewMockStorage())

	_, err := m.CreateQueue(context.Background(), CreateQueueInput{Name: ""})
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestManager_CreateTask_EmptyTitle(t *testing.T) {
	m := NewManager(NewMockStorage())

	_, err := m.CreateTask(context.Background(), CreateTaskInput{
		QueueID: 1,
		Title:   "",
	})
	if err == nil {
		t.Error("Expected error for empty title")
	}
}

func TestManager_TaskStatusTransitions(t *testing.T) {
	m := NewManager(NewMockStorage())

	// Create queue and task
	q, _ := m.CreateQueue(context.Background(), CreateQueueInput{Name: "Test"})
	task, _ := m.CreateTask(context.Background(), CreateTaskInput{
		QueueID: q.ID,
		Title:   "Task",
	})

	// Start task
	task, err := m.StartTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}
	if task.Status != StatusDoing {
		t.Errorf("Expected status doing, got %s", task.Status)
	}

	// Finish task
	task, err = m.FinishTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Failed to finish task: %v", err)
	}
	if task.Status != StatusFinished {
		t.Errorf("Expected status finished, got %s", task.Status)
	}

	// Reset task
	task, err = m.ResetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Failed to reset task: %v", err)
	}
	if task.Status != StatusPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}
}

func TestManager_InvalidStatus(t *testing.T) {
	m := NewManager(NewMockStorage())

	q, _ := m.CreateQueue(context.Background(), CreateQueueInput{Name: "Test"})
	task, _ := m.CreateTask(context.Background(), CreateTaskInput{
		QueueID: q.ID,
		Title:   "Task",
	})

	invalidStatus := TaskStatus("invalid")
	_, err := m.UpdateTask(context.Background(), task.ID, UpdateTaskInput{Status: &invalidStatus})
	if err != ErrInvalidStatus {
		t.Errorf("Expected ErrInvalidStatus, got %v", err)
	}
}

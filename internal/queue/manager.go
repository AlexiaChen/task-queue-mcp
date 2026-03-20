package queue

import (
	"context"
	"errors"
	"time"
)

var (
	ErrQueueNotFound = errors.New("queue not found")
	ErrTaskNotFound  = errors.New("task not found")
	ErrInvalidStatus = errors.New("invalid task status")
)

// Storage defines the interface for queue persistence
type Storage interface {
	// Queue operations
	CreateQueue(ctx context.Context, input CreateQueueInput) (*Queue, error)
	GetQueue(ctx context.Context, id int64) (*Queue, error)
	ListQueues(ctx context.Context) ([]*Queue, error)
	DeleteQueue(ctx context.Context, id int64) error
	GetQueueStats(ctx context.Context, id int64) (*QueueStats, error)

	// Task operations
	CreateTask(ctx context.Context, input CreateTaskInput) (*Task, error)
	GetTask(ctx context.Context, id int64) (*Task, error)
	ListTasks(ctx context.Context, queueID int64, status *TaskStatus) ([]*Task, error)
	UpdateTask(ctx context.Context, id int64, input UpdateTaskInput) (*Task, error)
	DeleteTask(ctx context.Context, id int64) error
	PrioritizeTask(ctx context.Context, taskID int64, position int) (*Task, error)
}

// Manager provides business logic for queue management
type Manager struct {
	storage Storage
}

// NewManager creates a new queue manager
func NewManager(storage Storage) *Manager {
	return &Manager{storage: storage}
}

// CreateQueue creates a new queue
func (m *Manager) CreateQueue(ctx context.Context, input CreateQueueInput) (*Queue, error) {
	if input.Name == "" {
		return nil, errors.New("queue name is required")
	}
	return m.storage.CreateQueue(ctx, input)
}

// GetQueue retrieves a queue by ID
func (m *Manager) GetQueue(ctx context.Context, id int64) (*Queue, error) {
	return m.storage.GetQueue(ctx, id)
}

// ListQueues returns all queues
func (m *Manager) ListQueues(ctx context.Context) ([]*Queue, error) {
	return m.storage.ListQueues(ctx)
}

// DeleteQueue deletes a queue and all its tasks
func (m *Manager) DeleteQueue(ctx context.Context, id int64) error {
	return m.storage.DeleteQueue(ctx, id)
}

// GetQueueStats returns statistics for a queue
func (m *Manager) GetQueueStats(ctx context.Context, id int64) (*QueueStats, error) {
	return m.storage.GetQueueStats(ctx, id)
}

// CreateTask creates a new task in a queue
func (m *Manager) CreateTask(ctx context.Context, input CreateTaskInput) (*Task, error) {
	if input.Title == "" {
		return nil, errors.New("task title is required")
	}
	return m.storage.CreateTask(ctx, input)
}

// GetTask retrieves a task by ID
func (m *Manager) GetTask(ctx context.Context, id int64) (*Task, error) {
	return m.storage.GetTask(ctx, id)
}

// ListTasks returns tasks in a queue, optionally filtered by status
func (m *Manager) ListTasks(ctx context.Context, queueID int64, status *TaskStatus) ([]*Task, error) {
	return m.storage.ListTasks(ctx, queueID, status)
}

// UpdateTask updates a task's status
func (m *Manager) UpdateTask(ctx context.Context, id int64, input UpdateTaskInput) (*Task, error) {
	if input.Status != nil {
		validStatuses := map[TaskStatus]bool{
			StatusPending:  true,
			StatusDoing:    true,
			StatusFinished: true,
		}
		if !validStatuses[*input.Status] {
			return nil, ErrInvalidStatus
		}
	}
	return m.storage.UpdateTask(ctx, id, input)
}

// DeleteTask deletes a task
func (m *Manager) DeleteTask(ctx context.Context, id int64) error {
	return m.storage.DeleteTask(ctx, id)
}

// PrioritizeTask moves a task to a specific position in the queue
func (m *Manager) PrioritizeTask(ctx context.Context, taskID int64, position int) (*Task, error) {
	return m.storage.PrioritizeTask(ctx, taskID, position)
}

// StartTask moves a task to "doing" status
func (m *Manager) StartTask(ctx context.Context, id int64) (*Task, error) {
	status := StatusDoing
	return m.UpdateTask(ctx, id, UpdateTaskInput{Status: &status})
}

// FinishTask moves a task to "finished" status
func (m *Manager) FinishTask(ctx context.Context, id int64) (*Task, error) {
	status := StatusFinished
	return m.UpdateTask(ctx, id, UpdateTaskInput{Status: &status})
}

// ResetTask moves a task back to "pending" status
func (m *Manager) ResetTask(ctx context.Context, id int64) (*Task, error) {
	status := StatusPending
	return m.UpdateTask(ctx, id, UpdateTaskInput{Status: &status})
}

// Helper function to get current time pointer
func timeNow() *time.Time {
	now := time.Now()
	return &now
}

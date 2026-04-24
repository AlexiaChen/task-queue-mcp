package queue

import (
	"context"
	"errors"
	"time"
)

var (
	ErrQueueNotFound              = errors.New("queue not found")
	ErrTaskNotFound               = errors.New("task not found")
	ErrInvalidStatus              = errors.New("invalid task status")
	ErrCannotEditNonPending       = errors.New("task can only be edited when pending")
	ErrCannotDeleteGlobalProject  = errors.New("cannot delete global project (project_id=0 is reserved)")
)

// Storage defines the interface for queue persistence
type Storage interface {
	// Queue operations
	CreateProject(ctx context.Context, input CreateQueueInput) (*Queue, error)
	GetProject(ctx context.Context, id int64) (*Queue, error)
	ListProjects(ctx context.Context) ([]*Queue, error)
	DeleteProject(ctx context.Context, id int64) error
	GetProjectStats(ctx context.Context, id int64) (*QueueStats, error)

	// Task operations
	CreateIssue(ctx context.Context, input CreateTaskInput) (*Task, error)
	GetIssue(ctx context.Context, id int64) (*Task, error)
	ListIssues(ctx context.Context, projectID int64, status *TaskStatus) ([]*Task, error)
	UpdateIssue(ctx context.Context, id int64, input UpdateTaskInput) (*Task, error)
	EditIssue(ctx context.Context, id int64, input EditTaskInput) (*Task, error)
	DeleteIssue(ctx context.Context, id int64) error
	PrioritizeIssue(ctx context.Context, taskID int64) (*Task, error)
}

// Manager provides business logic for queue management
type Manager struct {
	storage Storage
}

// NewManager creates a new queue manager
func NewManager(storage Storage) *Manager {
	return &Manager{storage: storage}
}

// CreateProject creates a new queue
func (m *Manager) CreateProject(ctx context.Context, input CreateQueueInput) (*Queue, error) {
	if input.Name == "" {
		return nil, errors.New("queue name is required")
	}
	return m.storage.CreateProject(ctx, input)
}

// GetProject retrieves a queue by ID
func (m *Manager) GetProject(ctx context.Context, id int64) (*Queue, error) {
	return m.storage.GetProject(ctx, id)
}

// ListProjects returns all queues
func (m *Manager) ListProjects(ctx context.Context) ([]*Queue, error) {
	return m.storage.ListProjects(ctx)
}

// DeleteProject deletes a queue and all its tasks
func (m *Manager) DeleteProject(ctx context.Context, id int64) error {
	if id == GlobalProjectID {
		return ErrCannotDeleteGlobalProject
	}
	return m.storage.DeleteProject(ctx, id)
}

// GetProjectStats returns statistics for a queue
func (m *Manager) GetProjectStats(ctx context.Context, id int64) (*QueueStats, error) {
	return m.storage.GetProjectStats(ctx, id)
}

// CreateIssue creates a new task in a queue
func (m *Manager) CreateIssue(ctx context.Context, input CreateTaskInput) (*Task, error) {
	if input.Title == "" {
		return nil, errors.New("task title is required")
	}
	return m.storage.CreateIssue(ctx, input)
}

// GetIssue retrieves a task by ID
func (m *Manager) GetIssue(ctx context.Context, id int64) (*Task, error) {
	return m.storage.GetIssue(ctx, id)
}

// ListIssues returns tasks in a queue, optionally filtered by status
func (m *Manager) ListIssues(ctx context.Context, projectID int64, status *TaskStatus) ([]*Task, error) {
	return m.storage.ListIssues(ctx, projectID, status)
}

// UpdateIssue updates a task's status.
func (m *Manager) UpdateIssue(ctx context.Context, id int64, input UpdateTaskInput) (*Task, error) {
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
	return m.storage.UpdateIssue(ctx, id, input)
}

// EditIssue updates the content (title, description, priority) of a pending task.
func (m *Manager) EditIssue(ctx context.Context, id int64, input EditTaskInput) (*Task, error) {
	if input.Title != nil && *input.Title == "" {
		return nil, errors.New("task title cannot be empty")
	}
	if input.Priority != nil && *input.Priority < PriorityLow {
		return nil, errors.New("task priority cannot be negative")
	}
	task, err := m.storage.GetIssue(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.Status != StatusPending {
		return nil, ErrCannotEditNonPending
	}
	return m.storage.EditIssue(ctx, id, input)
}

// DeleteIssue deletes a task
func (m *Manager) DeleteIssue(ctx context.Context, id int64) error {
	return m.storage.DeleteIssue(ctx, id)
}

// PrioritizeIssue moves a pending task ahead of lower-priority pending tasks.
func (m *Manager) PrioritizeIssue(ctx context.Context, taskID int64) (*Task, error) {
	return m.storage.PrioritizeIssue(ctx, taskID)
}

// StartIssue moves a task to "doing" status
func (m *Manager) StartIssue(ctx context.Context, id int64) (*Task, error) {
	status := StatusDoing
	return m.UpdateIssue(ctx, id, UpdateTaskInput{Status: &status})
}

// FinishIssue moves a task to "finished" status
func (m *Manager) FinishIssue(ctx context.Context, id int64) (*Task, error) {
	status := StatusFinished
	return m.UpdateIssue(ctx, id, UpdateTaskInput{Status: &status})
}

// ResetIssue moves a task back to "pending" status
func (m *Manager) ResetIssue(ctx context.Context, id int64) (*Task, error) {
	status := StatusPending
	return m.UpdateIssue(ctx, id, UpdateTaskInput{Status: &status})
}

// Helper function to get current time pointer
func timeNow() *time.Time {
	now := time.Now()
	return &now
}

package queue

import (
	"context"
)

// MockStorage implements Storage interface for testing
type MockStorage struct {
	queues    map[int64]*Queue
	tasks     map[int64]*Task
	nextQueue int64
	nextTask  int64
}

// NewMockStorage creates a new mock storage
func NewMockStorage() *MockStorage {
	return &MockStorage{
		queues:    make(map[int64]*Queue),
		tasks:     make(map[int64]*Task),
		nextQueue: 1,
		nextTask:  1,
	}
}

func (m *MockStorage) CreateProject(ctx context.Context, input CreateQueueInput) (*Queue, error) {
	q := &Queue{
		ID:          m.nextQueue,
		Name:        input.Name,
		Description: input.Description,
	}
	m.queues[q.ID] = q
	m.nextQueue++
	return q, nil
}

func (m *MockStorage) GetProject(ctx context.Context, id int64) (*Queue, error) {
	q, ok := m.queues[id]
	if !ok {
		return nil, ErrQueueNotFound
	}
	return q, nil
}

func (m *MockStorage) ListProjects(ctx context.Context) ([]*Queue, error) {
	queues := make([]*Queue, 0)
	for _, q := range m.queues {
		queues = append(queues, q)
	}
	return queues, nil
}

func (m *MockStorage) DeleteProject(ctx context.Context, id int64) error {
	if _, ok := m.queues[id]; !ok {
		return ErrQueueNotFound
	}
	delete(m.queues, id)
	for taskID, task := range m.tasks {
		if task.ProjectID == id {
			delete(m.tasks, taskID)
		}
	}
	return nil
}

func (m *MockStorage) GetProjectStats(ctx context.Context, id int64) (*QueueStats, error) {
	stats := &QueueStats{}
	for _, task := range m.tasks {
		if task.ProjectID == id {
			stats.Total++
			switch task.Status {
			case StatusPending:
				stats.Pending++
			case StatusDoing:
				stats.Doing++
			case StatusFinished:
				stats.Finished++
			}
		}
	}
	return stats, nil
}

func (m *MockStorage) CreateIssue(ctx context.Context, input CreateTaskInput) (*Task, error) {
	maxPos := 0
	for _, t := range m.tasks {
		if t.ProjectID == input.ProjectID && t.Position > maxPos {
			maxPos = t.Position
		}
	}

	task := &Task{
		ID:          m.nextTask,
		ProjectID:   input.ProjectID,
		Title:       input.Title,
		Description: input.Description,
		Status:      StatusPending,
		Priority:    input.Priority,
		Position:    maxPos + 1,
	}
	m.tasks[task.ID] = task
	m.nextTask++
	return task, nil
}

func (m *MockStorage) GetIssue(ctx context.Context, id int64) (*Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t, nil
}

func (m *MockStorage) ListIssues(ctx context.Context, projectID int64, status *TaskStatus) ([]*Task, error) {
	tasks := make([]*Task, 0)
	for _, t := range m.tasks {
		if t.ProjectID == projectID {
			if status == nil || t.Status == *status {
				tasks = append(tasks, t)
			}
		}
	}
	return tasks, nil
}

func (m *MockStorage) UpdateIssue(ctx context.Context, id int64, input UpdateTaskInput) (*Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	if input.Status != nil {
		t.Status = *input.Status
	}
	return t, nil
}

func (m *MockStorage) EditIssue(ctx context.Context, id int64, input EditTaskInput) (*Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	if input.Title != nil {
		t.Title = *input.Title
	}
	if input.Description != nil {
		t.Description = *input.Description
	}
	if input.Priority != nil {
		t.Priority = *input.Priority
	}
	return t, nil
}

func (m *MockStorage) DeleteIssue(ctx context.Context, id int64) error {
	if _, ok := m.tasks[id]; !ok {
		return ErrTaskNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *MockStorage) PrioritizeIssue(ctx context.Context, taskID int64) (*Task, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}
	t.Position = 1
	return t, nil
}

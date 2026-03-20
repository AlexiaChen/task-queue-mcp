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

func (m *MockStorage) CreateQueue(ctx context.Context, input CreateQueueInput) (*Queue, error) {
	q := &Queue{
		ID:          m.nextQueue,
		Name:        input.Name,
		Description: input.Description,
	}
	m.queues[q.ID] = q
	m.nextQueue++
	return q, nil
}

func (m *MockStorage) GetQueue(ctx context.Context, id int64) (*Queue, error) {
	q, ok := m.queues[id]
	if !ok {
		return nil, ErrQueueNotFound
	}
	return q, nil
}

func (m *MockStorage) ListQueues(ctx context.Context) ([]*Queue, error) {
	var queues []*Queue
	for _, q := range m.queues {
		queues = append(queues, q)
	}
	return queues, nil
}

func (m *MockStorage) DeleteQueue(ctx context.Context, id int64) error {
	if _, ok := m.queues[id]; !ok {
		return ErrQueueNotFound
	}
	delete(m.queues, id)
	for taskID, task := range m.tasks {
		if task.QueueID == id {
			delete(m.tasks, taskID)
		}
	}
	return nil
}

func (m *MockStorage) GetQueueStats(ctx context.Context, id int64) (*QueueStats, error) {
	stats := &QueueStats{}
	for _, task := range m.tasks {
		if task.QueueID == id {
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

func (m *MockStorage) CreateTask(ctx context.Context, input CreateTaskInput) (*Task, error) {
	maxPos := 0
	for _, t := range m.tasks {
		if t.QueueID == input.QueueID && t.Position > maxPos {
			maxPos = t.Position
		}
	}

	task := &Task{
		ID:          m.nextTask,
		QueueID:     input.QueueID,
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

func (m *MockStorage) GetTask(ctx context.Context, id int64) (*Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t, nil
}

func (m *MockStorage) ListTasks(ctx context.Context, queueID int64, status *TaskStatus) ([]*Task, error) {
	var tasks []*Task
	for _, t := range m.tasks {
		if t.QueueID == queueID {
			if status == nil || t.Status == *status {
				tasks = append(tasks, t)
			}
		}
	}
	return tasks, nil
}

func (m *MockStorage) UpdateTask(ctx context.Context, id int64, input UpdateTaskInput) (*Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	if input.Status != nil {
		t.Status = *input.Status
	}
	return t, nil
}

func (m *MockStorage) DeleteTask(ctx context.Context, id int64) error {
	if _, ok := m.tasks[id]; !ok {
		return ErrTaskNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *MockStorage) PrioritizeTask(ctx context.Context, taskID int64, position int) (*Task, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}
	t.Position = position
	t.Priority = 1000
	return t, nil
}

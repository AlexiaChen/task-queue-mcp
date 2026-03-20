package queue

import "time"

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusPending  TaskStatus = "pending"
	StatusDoing    TaskStatus = "doing"
	StatusFinished TaskStatus = "finished"
)

// Queue represents a task queue
type Queue struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Task represents a task in a queue
type Task struct {
	ID          int64      `json:"id"`
	QueueID     int64      `json:"queue_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      TaskStatus `json:"status"`
	Priority    int        `json:"priority"`
	Position    int        `json:"position"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

// CreateQueueInput represents input for creating a queue
type CreateQueueInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateTaskInput represents input for creating a task
type CreateTaskInput struct {
	QueueID     int64  `json:"queue_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

// UpdateTaskInput represents input for updating a task
type UpdateTaskInput struct {
	Status *TaskStatus `json:"status,omitempty"`
}

// QueueStats represents statistics for a queue
type QueueStats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Doing    int `json:"doing"`
	Finished int `json:"finished"`
}

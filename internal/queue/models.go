package queue

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Priority represents the priority level of a task.
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityMedium Priority = 1
	PriorityHigh   Priority = 2
)

// String returns the string representation of a Priority.
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	default:
		return "low"
	}
}

// MarshalJSON serializes Priority as a string ("low", "medium", "high").
func (p Priority) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON deserializes Priority from a string or numeric value (backward compat).
func (p *Priority) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		prio, err := ParsePriority(s)
		if err != nil {
			return err
		}
		*p = prio
		return nil
	}
	// Numeric fallback for backward compatibility
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("invalid priority: %s", string(data))
	}
	switch {
	case n <= 0:
		*p = PriorityLow
	case n == 1:
		*p = PriorityMedium
	default:
		*p = PriorityHigh
	}
	return nil
}

// ParsePriority parses a priority string into a Priority value.
// Accepts "low"/"medium"/"high" (case-insensitive), aliases "l"/"m"/"h", and "0"/"1"/"2".
func ParsePriority(s string) (Priority, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low", "l", "0":
		return PriorityLow, nil
	case "medium", "m", "1":
		return PriorityMedium, nil
	case "high", "h", "2":
		return PriorityHigh, nil
	default:
		return PriorityLow, fmt.Errorf("invalid priority %q: must be low, medium, or high", s)
	}
}

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
	ProjectID   int64      `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      TaskStatus `json:"status"`
	Priority    Priority   `json:"priority"`
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
	ProjectID   int64    `json:"project_id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    Priority `json:"priority,omitempty"`
}

// UpdateTaskInput represents input for updating a task's status
type UpdateTaskInput struct {
	Status *TaskStatus `json:"status,omitempty"`
}

// EditTaskInput represents input for editing a pending task's content
type EditTaskInput struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Priority    *Priority `json:"priority,omitempty"`
}

// QueueStats represents statistics for a queue
type QueueStats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Doing    int `json:"doing"`
	Finished int `json:"finished"`
}

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task-queue-mcp/internal/queue"

	_ "modernc.org/sqlite"
)

// SQLiteStorage implements queue.Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// CreateQueue creates a new queue
func (s *SQLiteStorage) CreateQueue(ctx context.Context, input queue.CreateQueueInput) (*queue.Queue, error) {
	now := time.Now()
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO queues (name, description, created_at, updated_at) VALUES (?, ?, ?, ?)",
		input.Name, input.Description, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetQueue(ctx, id)
}

// GetQueue retrieves a queue by ID
func (s *SQLiteStorage) GetQueue(ctx context.Context, id int64) (*queue.Queue, error) {
	q := &queue.Queue{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM queues WHERE id = ?",
		id,
	).Scan(&q.ID, &q.Name, &q.Description, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, queue.ErrQueueNotFound
		}
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	return q, nil
}

// ListQueues returns all queues
func (s *SQLiteStorage) ListQueues(ctx context.Context) ([]*queue.Queue, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM queues ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}
	defer rows.Close()

	var queues []*queue.Queue
	for rows.Next() {
		q := &queue.Queue{}
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan queue: %w", err)
		}
		queues = append(queues, q)
	}
	return queues, nil
}

// DeleteQueue deletes a queue and all its tasks
func (s *SQLiteStorage) DeleteQueue(ctx context.Context, id int64) error {
	// First delete all tasks in the queue
	_, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE queue_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete queue tasks: %w", err)
	}

	// Then delete the queue
	result, err := s.db.ExecContext(ctx, "DELETE FROM queues WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete queue: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return queue.ErrQueueNotFound
	}
	return nil
}

// GetQueueStats returns statistics for a queue
func (s *SQLiteStorage) GetQueueStats(ctx context.Context, id int64) (*queue.QueueStats, error) {
	stats := &queue.QueueStats{}
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE queue_id = ?",
		id,
	).Scan(&stats.Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE queue_id = ? AND status = ?",
		id, queue.StatusPending,
	).Scan(&stats.Pending)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE queue_id = ? AND status = ?",
		id, queue.StatusDoing,
	).Scan(&stats.Doing)
	if err != nil {
		return nil, fmt.Errorf("failed to get doing count: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE queue_id = ? AND status = ?",
		id, queue.StatusFinished,
	).Scan(&stats.Finished)
	if err != nil {
		return nil, fmt.Errorf("failed to get finished count: %w", err)
	}

	return stats, nil
}

// CreateTask creates a new task in a queue
func (s *SQLiteStorage) CreateTask(ctx context.Context, input queue.CreateTaskInput) (*queue.Task, error) {
	now := time.Now()

	// Get the max position for the queue
	var maxPosition int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(position), 0) FROM tasks WHERE queue_id = ?",
		input.QueueID,
	).Scan(&maxPosition)
	if err != nil {
		return nil, fmt.Errorf("failed to get max position: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (queue_id, title, description, status, priority, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.QueueID, input.Title, input.Description, queue.StatusPending, input.Priority, maxPosition+1, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetTask(ctx, id)
}

// GetTask retrieves a task by ID
func (s *SQLiteStorage) GetTask(ctx context.Context, id int64) (*queue.Task, error) {
	t := &queue.Task{}
	var startedAt, finishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, queue_id, title, description, status, priority, position, created_at, updated_at, started_at, finished_at
		 FROM tasks WHERE id = ?`,
		id,
	).Scan(&t.ID, &t.QueueID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Position,
		&t.CreatedAt, &t.UpdatedAt, &startedAt, &finishedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, queue.ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		t.FinishedAt = &finishedAt.Time
	}
	return t, nil
}

// ListTasks returns tasks in a queue, optionally filtered by status
func (s *SQLiteStorage) ListTasks(ctx context.Context, queueID int64, status *queue.TaskStatus) ([]*queue.Task, error) {
	var query string
	var args []interface{}

	if status != nil {
		query = `SELECT id, queue_id, title, description, status, priority, position, created_at, updated_at, started_at, finished_at
				 FROM tasks WHERE queue_id = ? AND status = ? ORDER BY priority DESC, position ASC`
		args = []interface{}{queueID, *status}
	} else {
		query = `SELECT id, queue_id, title, description, status, priority, position, created_at, updated_at, started_at, finished_at
				 FROM tasks WHERE queue_id = ? ORDER BY priority DESC, position ASC`
		args = []interface{}{queueID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*queue.Task
	for rows.Next() {
		t := &queue.Task{}
		var startedAt, finishedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.QueueID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Position,
			&t.CreatedAt, &t.UpdatedAt, &startedAt, &finishedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		if startedAt.Valid {
			t.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			t.FinishedAt = &finishedAt.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// UpdateTask updates a task
func (s *SQLiteStorage) UpdateTask(ctx context.Context, id int64, input queue.UpdateTaskInput) (*queue.Task, error) {
	now := time.Now()

	if input.Status != nil {
		var startedAt, finishedAt interface{}
		var query string

		switch *input.Status {
		case queue.StatusDoing:
			startedAt = now
			query = `UPDATE tasks SET status = ?, started_at = ?, updated_at = ? WHERE id = ?`
		case queue.StatusFinished:
			finishedAt = now
			if _, err := s.GetTask(ctx, id); err != nil {
				return nil, err
			}
			query = `UPDATE tasks SET status = ?, finished_at = ?, updated_at = ? WHERE id = ?`
		case queue.StatusPending:
			query = `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`
		default:
			return nil, queue.ErrInvalidStatus
		}

		var result sql.Result
		var err error

		switch *input.Status {
		case queue.StatusDoing:
			result, err = s.db.ExecContext(ctx, query, *input.Status, startedAt, now, id)
		case queue.StatusFinished:
			result, err = s.db.ExecContext(ctx, query, *input.Status, finishedAt, now, id)
		case queue.StatusPending:
			result, err = s.db.ExecContext(ctx, query, *input.Status, now, id)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}
		if affected == 0 {
			return nil, queue.ErrTaskNotFound
		}
	}

	return s.GetTask(ctx, id)
}

// DeleteTask deletes a task
func (s *SQLiteStorage) DeleteTask(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return queue.ErrTaskNotFound
	}
	return nil
}

// PrioritizeTask moves a task to a specific position
func (s *SQLiteStorage) PrioritizeTask(ctx context.Context, taskID int64, position int) (*queue.Task, error) {
	// Get the task
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// If position is 0 or negative, move to front (position 1)
	if position <= 0 {
		position = 1
	}

	// Shift tasks to make room
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET position = position + 1, updated_at = ?
		 WHERE queue_id = ? AND position >= ? AND id != ?`,
		time.Now(), task.QueueID, position, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to shift tasks: %w", err)
	}

	// Update the task's position
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET position = ?, priority = ?, updated_at = ? WHERE id = ?`,
		position, 1000, // High priority for prioritized tasks
		time.Now(), taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update task position: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.GetTask(ctx, taskID)
}

// runMigrations runs database migrations
func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS queues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			queue_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			priority INTEGER DEFAULT 0,
			position INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			started_at DATETIME,
			finished_at DATETIME,
			FOREIGN KEY (queue_id) REFERENCES queues(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_tasks_queue_id ON tasks(queue_id);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(queue_id, position);
	`)
	return err
}

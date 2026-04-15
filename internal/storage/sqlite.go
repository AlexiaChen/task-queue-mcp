package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlexiaChen/issue-kanban-mcp/internal/memory"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"

	_ "modernc.org/sqlite"
)

// SQLiteStorage implements queue.Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

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

// CreateProject creates a new queue
func (s *SQLiteStorage) CreateProject(ctx context.Context, input queue.CreateQueueInput) (*queue.Queue, error) {
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

	return s.GetProject(ctx, id)
}

// GetProject retrieves a queue by ID
func (s *SQLiteStorage) GetProject(ctx context.Context, id int64) (*queue.Queue, error) {
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

// ListProjects returns all queues
func (s *SQLiteStorage) ListProjects(ctx context.Context) ([]*queue.Queue, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM queues ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}
	defer rows.Close()

	queues := make([]*queue.Queue, 0)
	for rows.Next() {
		q := &queue.Queue{}
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan queue: %w", err)
		}
		queues = append(queues, q)
	}
	return queues, nil
}

// DeleteProject deletes a queue and all its tasks, triples, and memories
func (s *SQLiteStorage) DeleteProject(ctx context.Context, id int64) error {
	// Delete triples first (references memories via source_memory_id FK)
	_, err := s.db.ExecContext(ctx, "DELETE FROM memory_triples WHERE project_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete project triples: %w", err)
	}

	// Delete memories (triggers handle FTS cleanup)
	_, err = s.db.ExecContext(ctx, "DELETE FROM memories WHERE project_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete project memories: %w", err)
	}

	// Then delete all tasks in the queue
	_, err = s.db.ExecContext(ctx, "DELETE FROM tasks WHERE project_id = ?", id)
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

// GetProjectStats returns statistics for a queue
func (s *SQLiteStorage) GetProjectStats(ctx context.Context, id int64) (*queue.QueueStats, error) {
	stats := &queue.QueueStats{}
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE project_id = ?",
		id,
	).Scan(&stats.Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status = ?",
		id, queue.StatusPending,
	).Scan(&stats.Pending)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status = ?",
		id, queue.StatusDoing,
	).Scan(&stats.Doing)
	if err != nil {
		return nil, fmt.Errorf("failed to get doing count: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status = ?",
		id, queue.StatusFinished,
	).Scan(&stats.Finished)
	if err != nil {
		return nil, fmt.Errorf("failed to get finished count: %w", err)
	}

	return stats, nil
}

// CreateIssue creates a new task in a queue
func (s *SQLiteStorage) CreateIssue(ctx context.Context, input queue.CreateTaskInput) (*queue.Task, error) {
	now := time.Now()

	// Get the max position for the queue
	var maxPosition int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(position), 0) FROM tasks WHERE project_id = ?",
		input.ProjectID,
	).Scan(&maxPosition)
	if err != nil {
		return nil, fmt.Errorf("failed to get max position: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (project_id, title, description, status, priority, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.ProjectID, input.Title, input.Description, queue.StatusPending, input.Priority, maxPosition+1, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetIssue(ctx, id)
}

// GetIssue retrieves a task by ID
func (s *SQLiteStorage) GetIssue(ctx context.Context, id int64) (*queue.Task, error) {
	t := &queue.Task{}
	var startedAt, finishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, title, description, status, priority, position, created_at, updated_at, started_at, finished_at
		 FROM tasks WHERE id = ?`,
		id,
	).Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Position,
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

// ListIssues returns tasks in a queue, optionally filtered by status
func (s *SQLiteStorage) ListIssues(ctx context.Context, projectID int64, status *queue.TaskStatus) ([]*queue.Task, error) {
	var query string
	var args []interface{}

	if status != nil {
		query = `SELECT id, project_id, title, description, status, priority, position, created_at, updated_at, started_at, finished_at
				 FROM tasks WHERE project_id = ? AND status = ? ORDER BY priority DESC, position ASC`
		args = []interface{}{projectID, *status}
	} else {
		query = `SELECT id, project_id, title, description, status, priority, position, created_at, updated_at, started_at, finished_at
				 FROM tasks WHERE project_id = ? ORDER BY priority DESC, position ASC`
		args = []interface{}{projectID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]*queue.Task, 0)
	for rows.Next() {
		t := &queue.Task{}
		var startedAt, finishedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Position,
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

// UpdateIssue updates a task's status
func (s *SQLiteStorage) UpdateIssue(ctx context.Context, id int64, input queue.UpdateTaskInput) (*queue.Task, error) {
	now := time.Now()

	if input.Status == nil {
		return nil, fmt.Errorf("status is required")
	}

	var query string
	switch *input.Status {
	case queue.StatusDoing:
		query = `UPDATE tasks SET status = ?, started_at = ?, updated_at = ? WHERE id = ?`
		result, err := s.db.ExecContext(ctx, query, *input.Status, now, now, id)
		if err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return nil, queue.ErrTaskNotFound
		}
	case queue.StatusFinished:
		query = `UPDATE tasks SET status = ?, finished_at = ?, updated_at = ? WHERE id = ?`
		result, err := s.db.ExecContext(ctx, query, *input.Status, now, now, id)
		if err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return nil, queue.ErrTaskNotFound
		}
	case queue.StatusPending:
		query = `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`
		result, err := s.db.ExecContext(ctx, query, *input.Status, now, id)
		if err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return nil, queue.ErrTaskNotFound
		}
	default:
		return nil, queue.ErrInvalidStatus
	}

	return s.GetIssue(ctx, id)
}

// EditIssue updates the content fields (title, description, priority) of a task
func (s *SQLiteStorage) EditIssue(ctx context.Context, id int64, input queue.EditTaskInput) (*queue.Task, error) {
	now := time.Now()

	setClauses := []string{}
	args := []interface{}{}

	if input.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *input.Title)
	}
	if input.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *input.Description)
	}
	if input.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *input.Priority)
	}
	if len(setClauses) == 0 {
		return s.GetIssue(ctx, id)
	}
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, now)
	args = append(args, id)

	query := "UPDATE tasks SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to edit task: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, queue.ErrTaskNotFound
	}
	return s.GetIssue(ctx, id)
}

// DeleteIssue deletes a task
func (s *SQLiteStorage) DeleteIssue(ctx context.Context, id int64) error {
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

// PrioritizeIssue moves a pending task ahead of lower-priority pending tasks.
func (s *SQLiteStorage) PrioritizeIssue(ctx context.Context, taskID int64) (*queue.Task, error) {
	task, err := s.GetIssue(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.Status != queue.StatusPending {
		return nil, errors.New("only pending issues can be prioritized")
	}

	if task.Priority == queue.PriorityLow {
		return nil, errors.New("low priority issues cannot jump the queue")
	}

	// Count lower-priority pending tasks in the same queue
	var count int
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE project_id=? AND status='pending' AND priority < ? AND id != ?",
		task.ProjectID, int(task.Priority), taskID,
	).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count lower-priority tasks: %w", err)
	}
	if count == 0 {
		return nil, errors.New("no lower-priority pending issues exist to jump ahead of")
	}

	// Get the earliest position among lower-priority pending tasks
	var minPos int
	err = s.db.QueryRowContext(ctx,
		"SELECT MIN(position) FROM tasks WHERE project_id=? AND status='pending' AND priority < ?",
		task.ProjectID, int(task.Priority),
	).Scan(&minPos)
	if err != nil {
		return nil, fmt.Errorf("failed to get min position: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Shift all tasks at position >= minPos (except this task) back by one
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET position = position + 1, updated_at = ?
		 WHERE project_id = ? AND position >= ? AND id != ?`,
		time.Now(), task.ProjectID, minPos, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to shift tasks: %w", err)
	}

	// Move this task to minPos (do NOT change its priority)
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET position = ?, updated_at = ? WHERE id = ?`,
		minPos, time.Now(), taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update task position: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.GetIssue(ctx, taskID)
}

// --- Memory Storage Implementation ---

// StoreMemory persists a new memory. Returns the existing memory if a
// duplicate (same project_id + content_hash) already exists.
func (s *SQLiteStorage) StoreMemory(ctx context.Context, input memory.StoreMemoryInput) (*memory.Memory, error) {
	now := time.Now().UTC()

	category := input.Category
	if category == "" {
		category = string(memory.CategoryGeneral)
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO memories (project_id, content, summary, category, tags, source, importance, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, content_hash) DO NOTHING`,
		input.ProjectID, input.Content, input.Summary, category,
		input.Tags, input.Source, input.Importance, input.ContentHash,
		now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store memory: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Duplicate — return existing
		var mem memory.Memory
		err := s.db.QueryRowContext(ctx,
			`SELECT id, project_id, content, summary, category, tags, source, importance, content_hash, created_at, updated_at
			 FROM memories WHERE project_id = ? AND content_hash = ?`,
			input.ProjectID, input.ContentHash,
		).Scan(&mem.ID, &mem.ProjectID, &mem.Content, &mem.Summary,
			&mem.Category, &mem.Tags, &mem.Source, &mem.Importance,
			&mem.ContentHash, &mem.CreatedAt, &mem.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch existing memory: %w", err)
		}
		return &mem, nil
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return s.GetMemory(ctx, id)
}

// GetMemory retrieves a memory by its ID.
func (s *SQLiteStorage) GetMemory(ctx context.Context, id int64) (*memory.Memory, error) {
	var mem memory.Memory
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, content, summary, category, tags, source, importance, content_hash, created_at, updated_at
		 FROM memories WHERE id = ?`, id,
	).Scan(&mem.ID, &mem.ProjectID, &mem.Content, &mem.Summary,
		&mem.Category, &mem.Tags, &mem.Source, &mem.Importance,
		&mem.ContentHash, &mem.CreatedAt, &mem.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, memory.ErrMemoryNotFound
		}
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}
	return &mem, nil
}

// SearchMemories performs FTS5 full-text search scoped to a project.
func (s *SQLiteStorage) SearchMemories(ctx context.Context, projectID int64, query string, opts memory.SearchOptions) ([]memory.MemorySearchResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = memory.DefaultSearchLimit
	}

	// Build query with optional category filter
	sqlQuery := `
		SELECT m.id, m.project_id, m.content, m.summary, m.category, m.tags,
			   m.source, m.importance, m.content_hash, m.created_at, m.updated_at,
			   rank
		FROM memories_fts fts
		JOIN memories m ON m.id = fts.rowid
		WHERE memories_fts MATCH ?
		  AND m.project_id = ?`

	args := []interface{}{query, projectID}

	if opts.Category != "" {
		sqlQuery += ` AND m.category = ?`
		args = append(args, opts.Category)
	}

	sqlQuery += ` ORDER BY rank, m.importance DESC, m.updated_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		// FTS5 MATCH can fail on invalid syntax — return empty instead of error
		if strings.Contains(err.Error(), "fts5") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	var results []memory.MemorySearchResult
	for rows.Next() {
		var r memory.MemorySearchResult
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.Content, &r.Summary, &r.Category,
			&r.Tags, &r.Source, &r.Importance, &r.ContentHash,
			&r.CreatedAt, &r.UpdatedAt, &r.Rank,
		); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// ListMemories returns memories for a project ordered by created_at DESC.
func (s *SQLiteStorage) ListMemories(ctx context.Context, projectID int64, opts memory.ListOptions) ([]*memory.Memory, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = memory.DefaultListLimit
	}

	sqlQuery := `
		SELECT id, project_id, content, summary, category, tags, source, importance, content_hash, created_at, updated_at
		FROM memories WHERE project_id = ?`
	args := []interface{}{projectID}

	if opts.Category != "" {
		sqlQuery += ` AND category = ?`
		args = append(args, opts.Category)
	}

	sqlQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	defer rows.Close()

	var mems []*memory.Memory
	for rows.Next() {
		var m memory.Memory
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Content, &m.Summary,
			&m.Category, &m.Tags, &m.Source, &m.Importance,
			&m.ContentHash, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		mems = append(mems, &m)
	}
	return mems, nil
}

// DeleteMemory removes a memory. Returns ErrMemoryNotInProject if the
// memory does not belong to the specified project.
func (s *SQLiteStorage) DeleteMemory(ctx context.Context, projectID int64, memoryID int64) error {
	// Check existence and project ownership
	var ownerProject int64
	err := s.db.QueryRowContext(ctx,
		`SELECT project_id FROM memories WHERE id = ?`, memoryID,
	).Scan(&ownerProject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memory.ErrMemoryNotFound
		}
		return fmt.Errorf("failed to check memory: %w", err)
	}

	if ownerProject != projectID {
		return memory.ErrMemoryNotInProject
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, memoryID)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}
	return nil
}

// DeleteMemoriesByProject removes all memories for a project.
func (s *SQLiteStorage) DeleteMemoriesByProject(ctx context.Context, projectID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM memories WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project memories: %w", err)
	}
	return nil
}

// --- Triple Storage Implementation ---

// StoreTriple persists a new triple. If ReplaceExisting is true,
// active triples with the same (project_id, subject, predicate) and a different
// object will be invalidated within the same transaction.
func (s *SQLiteStorage) StoreTriple(ctx context.Context, input memory.StoreTripleInput) (*memory.Triple, error) {
	now := time.Now().UTC()

	validFrom := now
	if input.ValidFrom != "" {
		parsed, err := time.Parse(time.RFC3339, input.ValidFrom)
		if err != nil {
			return nil, fmt.Errorf("invalid valid_from: %w", err)
		}
		validFrom = parsed.UTC()
	}

	var validTo *time.Time
	if input.ValidTo != "" {
		parsed, err := time.Parse(time.RFC3339, input.ValidTo)
		if err != nil {
			return nil, fmt.Errorf("invalid valid_to: %w", err)
		}
		utc := parsed.UTC()
		validTo = &utc
	}

	confidence := input.Confidence
	if confidence == 0 {
		confidence = 1.0
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Auto-invalidation: if ReplaceExisting, invalidate active triples with same
	// (project, subject, predicate) but different object
	if input.ReplaceExisting {
		_, err = tx.ExecContext(ctx, `
			UPDATE memory_triples
			SET valid_to = ?
			WHERE project_id = ? AND subject = ? AND predicate = ?
			  AND object != ? AND valid_to IS NULL`,
			validFrom, input.ProjectID, input.Subject, input.Predicate, input.Object,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to invalidate existing triples: %w", err)
		}
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO memory_triples (project_id, subject, predicate, object, valid_from, valid_to, confidence, source_memory_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.ProjectID, input.Subject, input.Predicate, input.Object,
		validFrom, validTo, confidence, input.SourceMemoryID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store triple: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.GetTriple(ctx, input.ProjectID, id)
}

// GetTriple retrieves a triple by project and ID.
func (s *SQLiteStorage) GetTriple(ctx context.Context, projectID int64, id int64) (*memory.Triple, error) {
	var t memory.Triple
	var validTo sql.NullTime
	var sourceMemoryID sql.NullInt64

	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, subject, predicate, object, valid_from, valid_to,
		       confidence, source_memory_id, created_at
		FROM memory_triples WHERE id = ?`, id,
	).Scan(&t.ID, &t.ProjectID, &t.Subject, &t.Predicate, &t.Object,
		&t.ValidFrom, &validTo, &t.Confidence, &sourceMemoryID, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, memory.ErrTripleNotFound
		}
		return nil, fmt.Errorf("failed to get triple: %w", err)
	}

	if t.ProjectID != projectID {
		return nil, memory.ErrTripleNotInProject
	}

	if validTo.Valid {
		t.ValidTo = &validTo.Time
	}
	if sourceMemoryID.Valid {
		v := sourceMemoryID.Int64
		t.SourceMemoryID = &v
	}

	return &t, nil
}

// QueryTriples returns triples matching the given filters.
func (s *SQLiteStorage) QueryTriples(ctx context.Context, projectID int64, opts memory.QueryTripleOptions) ([]*memory.Triple, error) {
	query := `SELECT id, project_id, subject, predicate, object, valid_from, valid_to,
	                 confidence, source_memory_id, created_at
	          FROM memory_triples WHERE project_id = ?`
	args := []interface{}{projectID}

	if opts.Subject != "" {
		query += " AND subject LIKE ?"
		args = append(args, "%"+opts.Subject+"%")
	}
	if opts.Predicate != "" {
		query += " AND predicate LIKE ?"
		args = append(args, "%"+opts.Predicate+"%")
	}
	if opts.Object != "" {
		query += " AND object LIKE ?"
		args = append(args, "%"+opts.Object+"%")
	}

	if opts.PointInTime != "" {
		pit, err := time.Parse(time.RFC3339, opts.PointInTime)
		if err != nil {
			return nil, fmt.Errorf("invalid point_in_time: %w", err)
		}
		// [valid_from, valid_to) semantics: valid_from <= pit AND (valid_to IS NULL OR valid_to > pit)
		query += " AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)"
		args = append(args, pit.UTC(), pit.UTC())
	} else if opts.ActiveOnly {
		query += " AND valid_to IS NULL"
	}

	query += " ORDER BY valid_from DESC, id DESC"

	limit := opts.Limit
	if limit <= 0 {
		limit = memory.DefaultTripleQueryLimit
	}
	query += " LIMIT ?"
	args = append(args, limit)

	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query triples: %w", err)
	}
	defer rows.Close()

	var results []*memory.Triple
	for rows.Next() {
		var t memory.Triple
		var validTo sql.NullTime
		var sourceMemoryID sql.NullInt64

		err := rows.Scan(&t.ID, &t.ProjectID, &t.Subject, &t.Predicate, &t.Object,
			&t.ValidFrom, &validTo, &t.Confidence, &sourceMemoryID, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan triple: %w", err)
		}

		if validTo.Valid {
			t.ValidTo = &validTo.Time
		}
		if sourceMemoryID.Valid {
			v := sourceMemoryID.Int64
			t.SourceMemoryID = &v
		}
		results = append(results, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return results, nil
}

// InvalidateTriple sets valid_to on a triple, marking it as no longer active.
func (s *SQLiteStorage) InvalidateTriple(ctx context.Context, projectID int64, tripleID int64, validTo time.Time) error {
	// First check the triple exists and belongs to project
	_, err := s.GetTriple(ctx, projectID, tripleID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `UPDATE memory_triples SET valid_to = ? WHERE id = ?`,
		validTo.UTC(), tripleID)
	if err != nil {
		return fmt.Errorf("failed to invalidate triple: %w", err)
	}
	return nil
}

// DeleteTriple hard-deletes a triple.
func (s *SQLiteStorage) DeleteTriple(ctx context.Context, projectID int64, tripleID int64) error {
	// First check the triple exists and belongs to project
	_, err := s.GetTriple(ctx, projectID, tripleID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM memory_triples WHERE id = ?`, tripleID)
	if err != nil {
		return fmt.Errorf("failed to delete triple: %w", err)
	}
	return nil
}

// DeleteTriplesByProject removes all triples for a project.
func (s *SQLiteStorage) DeleteTriplesByProject(ctx context.Context, projectID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM memory_triples WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project triples: %w", err)
	}
	return nil
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
			project_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			priority INTEGER DEFAULT 0,
			position INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			started_at DATETIME,
			finished_at DATETIME,
			FOREIGN KEY (project_id) REFERENCES queues(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(project_id, position);
	`)
	if err != nil {
		return err
	}

	// Memory system tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			summary TEXT DEFAULT '',
			category TEXT NOT NULL DEFAULT 'general'
				CHECK(category IN ('decision','fact','event','preference','advice','general')),
			tags TEXT DEFAULT '',
			source TEXT DEFAULT '',
			importance INTEGER NOT NULL DEFAULT 3 CHECK(importance BETWEEN 1 AND 5),
			content_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (project_id) REFERENCES queues(id) ON DELETE CASCADE,
			UNIQUE(project_id, content_hash)
		);

		CREATE INDEX IF NOT EXISTS idx_memories_project_id ON memories(project_id);
		CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(project_id, category);
		CREATE INDEX IF NOT EXISTS idx_memories_hash ON memories(project_id, content_hash);
	`)
	if err != nil {
		return err
	}

	// FTS5 virtual table for full-text search on memory content + summary
	_, err = db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
			content,
			summary,
			tags,
			content='memories',
			content_rowid='id',
			tokenize='unicode61'
		);
	`)
	if err != nil {
		return err
	}

	// Triggers to keep FTS index in sync
	_, err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
			INSERT INTO memories_fts(rowid, content, summary, tags)
			VALUES (new.id, new.content, new.summary, new.tags);
		END;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, content, summary, tags)
			VALUES ('delete', old.id, old.content, old.summary, old.tags);
		END;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, content, summary, tags)
			VALUES ('delete', old.id, old.content, old.summary, old.tags);
			INSERT INTO memories_fts(rowid, content, summary, tags)
			VALUES (new.id, new.content, new.summary, new.tags);
		END;
	`)
	if err != nil {
		return err
	}

	// Temporal knowledge graph table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS memory_triples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			subject TEXT NOT NULL,
			predicate TEXT NOT NULL,
			object TEXT NOT NULL,
			valid_from DATETIME NOT NULL,
			valid_to DATETIME,
			confidence REAL NOT NULL DEFAULT 1.0 CHECK(confidence BETWEEN 0.0 AND 1.0),
			source_memory_id INTEGER,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (project_id) REFERENCES queues(id),
			FOREIGN KEY (source_memory_id) REFERENCES memories(id)
		);

		CREATE INDEX IF NOT EXISTS idx_triples_project ON memory_triples(project_id);
		CREATE INDEX IF NOT EXISTS idx_triples_subject ON memory_triples(project_id, subject);
		CREATE INDEX IF NOT EXISTS idx_triples_predicate ON memory_triples(project_id, subject, predicate);
		CREATE INDEX IF NOT EXISTS idx_triples_active ON memory_triples(project_id, valid_to);
	`)
	if err != nil {
		return err
	}

	return nil
}

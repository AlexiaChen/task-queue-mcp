package memory

import "context"

// Storage defines the persistence interface for the memory system.
// Implementations must ensure project-level isolation for all operations.
type Storage interface {
	// StoreMemory persists a new memory. Returns the existing memory if
	// a duplicate (same project_id + content_hash) already exists.
	StoreMemory(ctx context.Context, input StoreMemoryInput) (*Memory, error)

	// GetMemory retrieves a memory by its ID.
	GetMemory(ctx context.Context, id int64) (*Memory, error)

	// SearchMemories performs FTS5 full-text search scoped to a project.
	// Results are ranked by BM25 relevance, importance, and recency.
	SearchMemories(ctx context.Context, projectID int64, query string, opts SearchOptions) ([]MemorySearchResult, error)

	// ListMemories returns memories for a project ordered by created_at DESC.
	ListMemories(ctx context.Context, projectID int64, opts ListOptions) ([]*Memory, error)

	// DeleteMemory removes a memory. Returns ErrMemoryNotInProject if the
	// memory does not belong to the specified project.
	DeleteMemory(ctx context.Context, projectID int64, memoryID int64) error

	// DeleteMemoriesByProject removes all memories for a project.
	// Called during project deletion to maintain referential integrity.
	DeleteMemoriesByProject(ctx context.Context, projectID int64) error
}

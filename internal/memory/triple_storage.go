package memory

import (
	"context"
	"time"
)

// TripleStorage defines the persistence interface for the temporal knowledge graph.
// All operations are project-scoped to enforce isolation.
type TripleStorage interface {
	// StoreTriple persists a new triple. If ReplaceExisting is true on input,
	// active triples with the same (project_id, subject, predicate) and a
	// different object will be invalidated (valid_to = new triple's valid_from)
	// within the same transaction.
	StoreTriple(ctx context.Context, input StoreTripleInput) (*Triple, error)

	// GetTriple retrieves a triple by project and ID.
	// Returns ErrTripleNotInProject if the triple belongs to a different project.
	GetTriple(ctx context.Context, projectID int64, id int64) (*Triple, error)

	// QueryTriples returns triples matching the given filters.
	// Supports filtering by subject/predicate/object (LIKE), active-only,
	// point-in-time, and pagination. Results ordered by valid_from DESC, id DESC.
	QueryTriples(ctx context.Context, projectID int64, opts QueryTripleOptions) ([]*Triple, error)

	// InvalidateTriple sets valid_to on a triple, marking it as no longer active.
	// Returns ErrTripleNotInProject if the triple doesn't belong to the project.
	InvalidateTriple(ctx context.Context, projectID int64, tripleID int64, validTo time.Time) error

	// DeleteTriple hard-deletes a triple.
	// Returns ErrTripleNotInProject if the triple doesn't belong to the project.
	DeleteTriple(ctx context.Context, projectID int64, tripleID int64) error

	// DeleteTriplesByProject removes all triples for a project.
	// Called during project deletion for cleanup.
	DeleteTriplesByProject(ctx context.Context, projectID int64) error
}

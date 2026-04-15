package memory

import (
	"context"
	"strings"
	"time"
)

// TripleManager handles business logic for the temporal knowledge graph.
type TripleManager struct {
	storage TripleStorage
}

// NewTripleManager creates a new TripleManager.
func NewTripleManager(storage TripleStorage) *TripleManager {
	return &TripleManager{storage: storage}
}

// Store validates input and stores a new triple.
// If ReplaceExisting is true and an active triple with the same
// (project, subject, predicate) but different object exists,
// it will be invalidated with valid_to = new triple's valid_from.
func (tm *TripleManager) Store(ctx context.Context, input StoreTripleInput) (*Triple, error) {
	// Trim whitespace
	input.Subject = strings.TrimSpace(input.Subject)
	input.Predicate = strings.TrimSpace(input.Predicate)
	input.Object = strings.TrimSpace(input.Object)

	// Validate required fields
	if input.Subject == "" {
		return nil, ErrEmptySubject
	}
	if input.Predicate == "" {
		return nil, ErrEmptyPredicate
	}
	if input.Object == "" {
		return nil, ErrEmptyObject
	}

	// Validate field lengths
	if len(input.Subject) > MaxTripleFieldLength {
		return nil, ErrSubjectTooLong
	}
	if len(input.Predicate) > MaxTripleFieldLength {
		return nil, ErrPredicateTooLong
	}
	if len(input.Object) > MaxTripleFieldLength {
		return nil, ErrObjectTooLong
	}

	// Default confidence to 1.0
	if input.Confidence == 0 {
		input.Confidence = 1.0
	}
	if input.Confidence < 0 || input.Confidence > 1.0 {
		return nil, ErrInvalidConfidence
	}

	// Validate time range if both are provided
	if input.ValidFrom != "" && input.ValidTo != "" {
		from, err := time.Parse(time.RFC3339, input.ValidFrom)
		if err != nil {
			return nil, ErrInvalidTimeRange
		}
		to, err := time.Parse(time.RFC3339, input.ValidTo)
		if err != nil {
			return nil, ErrInvalidTimeRange
		}
		if !to.After(from) {
			return nil, ErrInvalidTimeRange
		}
	}

	return tm.storage.StoreTriple(ctx, input)
}

// Query returns triples matching the given filters.
func (tm *TripleManager) Query(ctx context.Context, projectID int64, opts QueryTripleOptions) ([]*Triple, error) {
	if opts.Limit <= 0 {
		opts.Limit = DefaultTripleQueryLimit
	}
	return tm.storage.QueryTriples(ctx, projectID, opts)
}

// Get retrieves a triple by project and ID.
func (tm *TripleManager) Get(ctx context.Context, projectID int64, tripleID int64) (*Triple, error) {
	return tm.storage.GetTriple(ctx, projectID, tripleID)
}

// Invalidate marks a triple as no longer active by setting valid_to.
func (tm *TripleManager) Invalidate(ctx context.Context, projectID int64, tripleID int64, validTo time.Time) error {
	return tm.storage.InvalidateTriple(ctx, projectID, tripleID, validTo.UTC())
}

// Delete hard-deletes a triple.
func (tm *TripleManager) Delete(ctx context.Context, projectID int64, tripleID int64) error {
	return tm.storage.DeleteTriple(ctx, projectID, tripleID)
}

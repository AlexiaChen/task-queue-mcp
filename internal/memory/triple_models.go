package memory

import (
	"errors"
	"time"
)

// Triple represents a (subject, predicate, object) fact with temporal validity.
// Interval semantics: [valid_from, valid_to) — closed start, open end.
// valid_to == nil means the triple is still active (no known end).
type Triple struct {
	ID             int64      `json:"id"`
	ProjectID      int64      `json:"project_id"`
	Subject        string     `json:"subject"`
	Predicate      string     `json:"predicate"`
	Object         string     `json:"object"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to,omitempty"`
	Confidence     float64    `json:"confidence"`
	SourceMemoryID *int64     `json:"source_memory_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// IsActive returns whether the triple is currently valid.
// Uses [valid_from, valid_to) semantics: active if valid_to is nil or strictly after now.
func (t *Triple) IsActive() bool {
	return t.IsActiveAt(time.Now())
}

// IsActiveAt returns whether the triple was valid at a specific point in time.
// Uses [valid_from, valid_to) semantics: valid_from <= at AND (valid_to IS NULL OR valid_to > at).
func (t *Triple) IsActiveAt(at time.Time) bool {
	if at.Before(t.ValidFrom) {
		return false
	}
	if t.ValidTo == nil {
		return true
	}
	return t.ValidTo.After(at)
}

// StoreTripleInput is the input for storing a new triple.
type StoreTripleInput struct {
	ProjectID      int64   `json:"project_id"`
	Subject        string  `json:"subject"`
	Predicate      string  `json:"predicate"`
	Object         string  `json:"object"`
	ValidFrom      string  `json:"valid_from,omitempty"`
	ValidTo        string  `json:"valid_to,omitempty"`
	Confidence     float64 `json:"confidence,omitempty"`
	SourceMemoryID *int64  `json:"source_memory_id,omitempty"`
	// ReplaceExisting controls auto-invalidation behavior.
	// When true: if an active triple with the same (project, subject, predicate) exists
	// and has a different object, it will be invalidated (valid_to = new.valid_from).
	// When false (default): the triple is inserted without affecting existing triples.
	// Use true for single-valued predicates (e.g., "status", "assigned_to").
	// Use false for multi-valued predicates (e.g., "has_label", "depends_on").
	ReplaceExisting bool `json:"replace_existing,omitempty"`
}

// QueryTripleOptions configures triple query behavior.
type QueryTripleOptions struct {
	Subject     string `json:"subject,omitempty"`
	Predicate   string `json:"predicate,omitempty"`
	Object      string `json:"object,omitempty"`
	ActiveOnly  bool   `json:"active_only,omitempty"`
	PointInTime string `json:"point_in_time,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// MaxTripleFieldLength is the maximum length for subject/predicate/object fields.
const MaxTripleFieldLength = 1024

// DefaultTripleQueryLimit is the default number of results for triple queries.
const DefaultTripleQueryLimit = 50

// Triple-related errors
var (
	ErrTripleNotFound     = errors.New("triple not found")
	ErrTripleNotInProject = errors.New("triple does not belong to this project")
	ErrEmptySubject       = errors.New("triple subject cannot be empty")
	ErrEmptyPredicate     = errors.New("triple predicate cannot be empty")
	ErrEmptyObject        = errors.New("triple object cannot be empty")
	ErrSubjectTooLong     = errors.New("triple subject exceeds maximum length")
	ErrPredicateTooLong   = errors.New("triple predicate exceeds maximum length")
	ErrObjectTooLong      = errors.New("triple object exceeds maximum length")
	ErrInvalidConfidence  = errors.New("confidence must be between 0.0 and 1.0")
	ErrInvalidTimeRange   = errors.New("valid_to must be after valid_from")
)

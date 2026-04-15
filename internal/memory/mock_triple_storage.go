package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MockTripleStorage is an in-memory implementation of TripleStorage for testing.
type MockTripleStorage struct {
	mu      sync.RWMutex
	triples map[int64]*Triple
	nextID  int64
}

// NewMockTripleStorage creates a new MockTripleStorage.
func NewMockTripleStorage() *MockTripleStorage {
	return &MockTripleStorage{
		triples: make(map[int64]*Triple),
		nextID:  1,
	}
}

func (m *MockTripleStorage) StoreTriple(ctx context.Context, input StoreTripleInput) (*Triple, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	validFrom := time.Now().UTC()
	if input.ValidFrom != "" {
		t, err := time.Parse(time.RFC3339, input.ValidFrom)
		if err != nil {
			return nil, fmt.Errorf("invalid valid_from: %w", err)
		}
		validFrom = t.UTC()
	}

	var validTo *time.Time
	if input.ValidTo != "" {
		t, err := time.Parse(time.RFC3339, input.ValidTo)
		if err != nil {
			return nil, fmt.Errorf("invalid valid_to: %w", err)
		}
		utc := t.UTC()
		validTo = &utc
	}

	// Auto-invalidation for replace_existing
	if input.ReplaceExisting {
		for _, existing := range m.triples {
			if existing.ProjectID == input.ProjectID &&
				existing.Subject == input.Subject &&
				existing.Predicate == input.Predicate &&
				existing.Object != input.Object &&
				existing.ValidTo == nil {
				existing.ValidTo = &validFrom
			}
		}
	}

	confidence := input.Confidence
	if confidence == 0 {
		confidence = 1.0
	}

	triple := &Triple{
		ID:             m.nextID,
		ProjectID:      input.ProjectID,
		Subject:        input.Subject,
		Predicate:      input.Predicate,
		Object:         input.Object,
		ValidFrom:      validFrom,
		ValidTo:        validTo,
		Confidence:     confidence,
		SourceMemoryID: input.SourceMemoryID,
		CreatedAt:      time.Now().UTC(),
	}
	m.triples[m.nextID] = triple
	m.nextID++
	return triple, nil
}

func (m *MockTripleStorage) GetTriple(ctx context.Context, projectID int64, id int64) (*Triple, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.triples[id]
	if !ok {
		return nil, ErrTripleNotFound
	}
	if t.ProjectID != projectID {
		return nil, ErrTripleNotInProject
	}
	return t, nil
}

func (m *MockTripleStorage) QueryTriples(ctx context.Context, projectID int64, opts QueryTripleOptions) ([]*Triple, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*Triple
	now := time.Now().UTC()

	for _, t := range m.triples {
		if t.ProjectID != projectID {
			continue
		}
		if opts.Subject != "" && !strings.Contains(strings.ToLower(t.Subject), strings.ToLower(opts.Subject)) {
			continue
		}
		if opts.Predicate != "" && !strings.Contains(strings.ToLower(t.Predicate), strings.ToLower(opts.Predicate)) {
			continue
		}
		if opts.Object != "" && !strings.Contains(strings.ToLower(t.Object), strings.ToLower(opts.Object)) {
			continue
		}
		if opts.ActiveOnly {
			if t.ValidTo != nil && !t.ValidTo.After(now) {
				continue
			}
		}
		if opts.PointInTime != "" {
			pit, err := time.Parse(time.RFC3339, opts.PointInTime)
			if err == nil {
				if !t.IsActiveAt(pit) {
					continue
				}
			}
		}
		results = append(results, t)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultTripleQueryLimit
	}
	offset := opts.Offset
	if offset > len(results) {
		return nil, nil
	}
	results = results[offset:]
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (m *MockTripleStorage) InvalidateTriple(ctx context.Context, projectID int64, tripleID int64, validTo time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.triples[tripleID]
	if !ok {
		return ErrTripleNotFound
	}
	if t.ProjectID != projectID {
		return ErrTripleNotInProject
	}
	utc := validTo.UTC()
	t.ValidTo = &utc
	return nil
}

func (m *MockTripleStorage) DeleteTriple(ctx context.Context, projectID int64, tripleID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.triples[tripleID]
	if !ok {
		return ErrTripleNotFound
	}
	if t.ProjectID != projectID {
		return ErrTripleNotInProject
	}
	delete(m.triples, tripleID)
	return nil
}

func (m *MockTripleStorage) DeleteTriplesByProject(ctx context.Context, projectID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, t := range m.triples {
		if t.ProjectID == projectID {
			delete(m.triples, id)
		}
	}
	return nil
}

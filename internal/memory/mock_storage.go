package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MockMemoryStorage is an in-memory implementation of memory.Storage for testing.
type MockMemoryStorage struct {
	mu        sync.RWMutex
	memories  map[int64]*Memory
	nextID    int64
	hashIndex map[string]int64 // "projectID:contentHash" → memoryID
}

// NewMockMemoryStorage creates a new mock memory storage.
func NewMockMemoryStorage() *MockMemoryStorage {
	return &MockMemoryStorage{
		memories:  make(map[int64]*Memory),
		nextID:    1,
		hashIndex: make(map[string]int64),
	}
}

func (m *MockMemoryStorage) hashKey(projectID int64, contentHash string) string {
	return fmt.Sprintf("%d:%s", projectID, contentHash)
}

func (m *MockMemoryStorage) StoreMemory(ctx context.Context, input StoreMemoryInput) (*Memory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.hashKey(input.ProjectID, input.ContentHash)
	if existingID, ok := m.hashIndex[key]; ok {
		return m.memories[existingID], nil
	}

	now := time.Now()
	category := MemoryCategory(input.Category)
	if category == "" {
		category = CategoryGeneral
	}
	importance := input.Importance
	if importance == 0 {
		importance = 1
	}

	mem := &Memory{
		ID:          m.nextID,
		ProjectID:   input.ProjectID,
		Content:     input.Content,
		Summary:     input.Summary,
		Category:    category,
		Tags:        input.Tags,
		Source:      input.Source,
		Importance:  importance,
		ContentHash: input.ContentHash,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m.memories[m.nextID] = mem
	m.hashIndex[key] = m.nextID
	m.nextID++

	return mem, nil
}

func (m *MockMemoryStorage) GetMemory(ctx context.Context, id int64) (*Memory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mem, ok := m.memories[id]
	if !ok {
		return nil, ErrMemoryNotFound
	}
	return mem, nil
}

func (m *MockMemoryStorage) SearchMemories(ctx context.Context, projectID int64, query string, opts SearchOptions) ([]MemorySearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queryLower := strings.ToLower(query)
	var results []MemorySearchResult

	for _, mem := range m.memories {
		if mem.ProjectID != projectID {
			continue
		}
		if opts.Category != "" && string(mem.Category) != opts.Category {
			continue
		}
		// Simple substring match for mock
		contentLower := strings.ToLower(mem.Content + " " + mem.Summary + " " + mem.Tags)
		if strings.Contains(contentLower, queryLower) {
			results = append(results, MemorySearchResult{
				Memory: *mem,
				Rank:   float64(mem.Importance),
			})
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (m *MockMemoryStorage) ListMemories(ctx context.Context, projectID int64, opts ListOptions) ([]*Memory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*Memory
	for _, mem := range m.memories {
		if mem.ProjectID != projectID {
			continue
		}
		if opts.Category != "" && string(mem.Category) != opts.Category {
			continue
		}
		results = append(results, mem)
	}

	// Apply offset
	if opts.Offset > 0 && opts.Offset < len(results) {
		results = results[opts.Offset:]
	} else if opts.Offset >= len(results) {
		return nil, nil
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (m *MockMemoryStorage) DeleteMemory(ctx context.Context, projectID int64, memoryID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mem, ok := m.memories[memoryID]
	if !ok {
		return ErrMemoryNotFound
	}
	if mem.ProjectID != projectID {
		return ErrMemoryNotInProject
	}

	key := m.hashKey(mem.ProjectID, mem.ContentHash)
	delete(m.hashIndex, key)
	delete(m.memories, memoryID)
	return nil
}

func (m *MockMemoryStorage) DeleteMemoriesByProject(ctx context.Context, projectID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, mem := range m.memories {
		if mem.ProjectID == projectID {
			key := m.hashKey(mem.ProjectID, mem.ContentHash)
			delete(m.hashIndex, key)
			delete(m.memories, id)
		}
	}
	return nil
}

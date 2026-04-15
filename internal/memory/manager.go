package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// MemoryManager handles business logic for the memory system.
type MemoryManager struct {
	storage Storage
}

// NewMemoryManager creates a new MemoryManager.
func NewMemoryManager(storage Storage) *MemoryManager {
	return &MemoryManager{storage: storage}
}

// NormalizeContent trims whitespace, normalizes line endings, and collapses
// multiple blank lines for consistent content hashing.
func NormalizeContent(content string) string {
	s := strings.TrimSpace(content)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	re := regexp.MustCompile(`\n{3,}`)
	s = re.ReplaceAllString(s, "\n\n")
	return s
}

// ContentHash computes a SHA-256 hash of the normalized content.
func ContentHash(content string) string {
	normalized := NormalizeContent(content)
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

// Store validates input and stores a new memory.
// Returns the existing memory if a duplicate is found (same project + content hash).
func (m *MemoryManager) Store(ctx context.Context, input StoreMemoryInput) (*Memory, error) {
	if strings.TrimSpace(input.Content) == "" {
		return nil, ErrEmptyContent
	}
	if len(input.Content) > MaxContentLength {
		return nil, ErrContentTooLong
	}

	if input.Category == "" {
		input.Category = string(CategoryGeneral)
	}
	if !IsValidCategory(input.Category) {
		return nil, ErrInvalidCategory
	}

	if input.Importance == 0 {
		input.Importance = 1
	}
	if input.Importance < 1 || input.Importance > 5 {
		return nil, ErrInvalidImportance
	}

	input.Content = NormalizeContent(input.Content)
	input.ContentHash = ContentHash(input.Content)

	return m.storage.StoreMemory(ctx, input)
}

// Search performs a full-text search for memories within a project.
func (m *MemoryManager) Search(ctx context.Context, projectID int64, query string, opts SearchOptions) ([]MemorySearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	if opts.Category != "" && !IsValidCategory(opts.Category) {
		return nil, ErrInvalidCategory
	}
	if opts.Limit <= 0 {
		opts.Limit = DefaultSearchLimit
	}

	return m.storage.SearchMemories(ctx, projectID, query, opts)
}

// List returns memories for a project, ordered by creation time descending.
func (m *MemoryManager) List(ctx context.Context, projectID int64, opts ListOptions) ([]*Memory, error) {
	if opts.Category != "" && !IsValidCategory(opts.Category) {
		return nil, ErrInvalidCategory
	}
	if opts.Limit <= 0 {
		opts.Limit = DefaultListLimit
	}

	return m.storage.ListMemories(ctx, projectID, opts)
}

// Delete removes a memory, verifying it belongs to the specified project.
func (m *MemoryManager) Delete(ctx context.Context, projectID int64, memoryID int64) error {
	return m.storage.DeleteMemory(ctx, projectID, memoryID)
}

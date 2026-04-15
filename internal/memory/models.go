package memory

import (
	"errors"
	"time"
)

// MemoryCategory represents the type of memory being stored.
type MemoryCategory string

const (
	CategoryDecision   MemoryCategory = "decision"
	CategoryFact       MemoryCategory = "fact"
	CategoryEvent      MemoryCategory = "event"
	CategoryPreference MemoryCategory = "preference"
	CategoryAdvice     MemoryCategory = "advice"
	CategoryGeneral    MemoryCategory = "general"
)

// ValidCategories lists all valid memory categories.
var ValidCategories = []MemoryCategory{
	CategoryDecision, CategoryFact, CategoryEvent,
	CategoryPreference, CategoryAdvice, CategoryGeneral,
}

// IsValidCategory checks whether a category string is valid.
func IsValidCategory(c string) bool {
	for _, valid := range ValidCategories {
		if string(valid) == c {
			return true
		}
	}
	return false
}

// Memory represents a stored memory entry.
type Memory struct {
	ID          int64          `json:"id"`
	ProjectID   int64          `json:"project_id"`
	Content     string         `json:"content"`
	Summary     string         `json:"summary,omitempty"`
	Category    MemoryCategory `json:"category"`
	Tags        string         `json:"tags,omitempty"`
	Source      string         `json:"source,omitempty"`
	Importance  int            `json:"importance"`
	ContentHash string         `json:"content_hash"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// StoreMemoryInput is the input for storing a new memory.
// ContentHash is computed by the manager before calling storage.
type StoreMemoryInput struct {
	ProjectID   int64  `json:"project_id"`
	Content     string `json:"content"`
	Summary     string `json:"summary,omitempty"`
	Category    string `json:"category,omitempty"`
	Tags        string `json:"tags,omitempty"`
	Source      string `json:"source,omitempty"`
	Importance  int    `json:"importance,omitempty"`
	ContentHash string `json:"-"`
}

// SearchOptions configures memory search behavior.
type SearchOptions struct {
	Category string `json:"category,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// ListOptions configures memory listing behavior.
type ListOptions struct {
	Category string `json:"category,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// MemorySearchResult wraps a Memory with its BM25 search rank.
type MemorySearchResult struct {
	Memory
	Rank float64 `json:"rank"`
}

// MaxContentLength is the maximum allowed length for memory content (50KB).
const MaxContentLength = 50 * 1024

// DefaultSearchLimit is the default number of results returned by search.
const DefaultSearchLimit = 20

// DefaultListLimit is the default number of results returned by list.
const DefaultListLimit = 50

// Errors
var (
	ErrMemoryNotFound      = errors.New("memory not found")
	ErrProjectNotFound     = errors.New("project not found for memory operation")
	ErrEmptyContent        = errors.New("memory content cannot be empty")
	ErrContentTooLong      = errors.New("memory content exceeds maximum length")
	ErrInvalidCategory     = errors.New("invalid memory category")
	ErrInvalidImportance   = errors.New("importance must be between 1 and 5")
	ErrEmptyQuery          = errors.New("search query cannot be empty")
	ErrMemoryNotInProject  = errors.New("memory does not belong to this project")
	ErrDuplicateMemory     = errors.New("duplicate memory already exists")
)

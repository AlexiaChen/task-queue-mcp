package memory

import (
	"context"
	"testing"
)

func TestMemoryManager_Store(t *testing.T) {
	ctx := context.Background()
	store := NewMockMemoryStorage()
	mgr := NewMemoryManager(store)

	t.Run("store basic memory", func(t *testing.T) {
		mem, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "We decided to use PostgreSQL for the database.",
			Category:  "decision",
			Tags:      "database,architecture",
			Source:    "issue-42",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mem.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if mem.ProjectID != 1 {
			t.Errorf("expected project_id 1, got %d", mem.ProjectID)
		}
		if mem.Category != CategoryDecision {
			t.Errorf("expected category 'decision', got %q", mem.Category)
		}
		if mem.Importance != 1 {
			t.Errorf("expected default importance 1, got %d", mem.Importance)
		}
		if mem.ContentHash == "" {
			t.Error("expected non-empty content hash")
		}
	})

	t.Run("store with importance", func(t *testing.T) {
		mem, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID:  1,
			Content:    "Critical: never use raw SQL without parameterized queries.",
			Category:   "advice",
			Importance: 5,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mem.Importance != 5 {
			t.Errorf("expected importance 5, got %d", mem.Importance)
		}
	})

	t.Run("default category", func(t *testing.T) {
		mem, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "Some general note without category.",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mem.Category != CategoryGeneral {
			t.Errorf("expected default category 'general', got %q", mem.Category)
		}
	})

	t.Run("content dedup returns existing", func(t *testing.T) {
		content := "Unique content for dedup test."
		mem1, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   content,
		})
		if err != nil {
			t.Fatalf("first store failed: %v", err)
		}

		mem2, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   content,
		})
		if err != nil {
			t.Fatalf("second store failed: %v", err)
		}

		if mem1.ID != mem2.ID {
			t.Errorf("expected same ID for duplicate, got %d and %d", mem1.ID, mem2.ID)
		}
	})

	t.Run("dedup ignores whitespace differences", func(t *testing.T) {
		mem1, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "\n\ncontent with whitespace\n\n\n\nextra lines\n\n",
		})
		if err != nil {
			t.Fatalf("first store failed: %v", err)
		}

		mem2, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "content with whitespace\n\nextra lines",
		})
		if err != nil {
			t.Fatalf("second store failed: %v", err)
		}

		if mem1.ID != mem2.ID {
			t.Errorf("expected same ID for normalized duplicate, got %d and %d", mem1.ID, mem2.ID)
		}
	})

	t.Run("same content different projects not deduped", func(t *testing.T) {
		content := "Cross-project content test."
		mem1, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 10,
			Content:   content,
		})
		if err != nil {
			t.Fatalf("store in project 10 failed: %v", err)
		}

		mem2, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 20,
			Content:   content,
		})
		if err != nil {
			t.Fatalf("store in project 20 failed: %v", err)
		}

		if mem1.ID == mem2.ID {
			t.Error("expected different IDs for different projects")
		}
	})
}

func TestMemoryManager_Store_Validation(t *testing.T) {
	ctx := context.Background()
	store := NewMockMemoryStorage()
	mgr := NewMemoryManager(store)

	t.Run("empty content", func(t *testing.T) {
		_, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "",
		})
		if err != ErrEmptyContent {
			t.Errorf("expected ErrEmptyContent, got %v", err)
		}
	})

	t.Run("whitespace-only content", func(t *testing.T) {
		_, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "   \n\t  ",
		})
		if err != ErrEmptyContent {
			t.Errorf("expected ErrEmptyContent, got %v", err)
		}
	})

	t.Run("content too long", func(t *testing.T) {
		longContent := make([]byte, MaxContentLength+1)
		for i := range longContent {
			longContent[i] = 'a'
		}
		_, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   string(longContent),
		})
		if err != ErrContentTooLong {
			t.Errorf("expected ErrContentTooLong, got %v", err)
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		_, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "Some memory",
			Category:  "invalid_cat",
		})
		if err != ErrInvalidCategory {
			t.Errorf("expected ErrInvalidCategory, got %v", err)
		}
	})

	t.Run("importance too low", func(t *testing.T) {
		_, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID:  1,
			Content:    "Some memory",
			Importance: -1,
		})
		if err != ErrInvalidImportance {
			t.Errorf("expected ErrInvalidImportance, got %v", err)
		}
	})

	t.Run("importance too high", func(t *testing.T) {
		_, err := mgr.Store(ctx, StoreMemoryInput{
			ProjectID:  1,
			Content:    "Some memory",
			Importance: 6,
		})
		if err != ErrInvalidImportance {
			t.Errorf("expected ErrInvalidImportance, got %v", err)
		}
	})
}

func TestMemoryManager_Search(t *testing.T) {
	ctx := context.Background()
	store := NewMockMemoryStorage()
	mgr := NewMemoryManager(store)

	// Seed data
	mgr.Store(ctx, StoreMemoryInput{ProjectID: 1, Content: "PostgreSQL is our primary database.", Category: "decision"})
	mgr.Store(ctx, StoreMemoryInput{ProjectID: 1, Content: "Redis is used for caching.", Category: "fact"})
	mgr.Store(ctx, StoreMemoryInput{ProjectID: 2, Content: "PostgreSQL setup for project 2.", Category: "decision"})

	t.Run("basic search", func(t *testing.T) {
		results, err := mgr.Search(ctx, 1, "PostgreSQL", SearchOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].ProjectID != 1 {
			t.Error("result should be from project 1")
		}
	})

	t.Run("search respects project isolation", func(t *testing.T) {
		results, err := mgr.Search(ctx, 2, "PostgreSQL", SearchOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result from project 2, got %d", len(results))
		}
		if results[0].ProjectID != 2 {
			t.Error("result should be from project 2")
		}
	})

	t.Run("search with category filter", func(t *testing.T) {
		results, err := mgr.Search(ctx, 1, "PostgreSQL", SearchOptions{Category: "fact"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results (category mismatch), got %d", len(results))
		}
	})

	t.Run("empty query error", func(t *testing.T) {
		_, err := mgr.Search(ctx, 1, "", SearchOptions{})
		if err != ErrEmptyQuery {
			t.Errorf("expected ErrEmptyQuery, got %v", err)
		}
	})

	t.Run("invalid category error", func(t *testing.T) {
		_, err := mgr.Search(ctx, 1, "test", SearchOptions{Category: "bogus"})
		if err != ErrInvalidCategory {
			t.Errorf("expected ErrInvalidCategory, got %v", err)
		}
	})
}

func TestMemoryManager_List(t *testing.T) {
	ctx := context.Background()
	store := NewMockMemoryStorage()
	mgr := NewMemoryManager(store)

	// Seed data
	mgr.Store(ctx, StoreMemoryInput{ProjectID: 1, Content: "Memory A", Category: "decision"})
	mgr.Store(ctx, StoreMemoryInput{ProjectID: 1, Content: "Memory B", Category: "fact"})
	mgr.Store(ctx, StoreMemoryInput{ProjectID: 1, Content: "Memory C", Category: "decision"})
	mgr.Store(ctx, StoreMemoryInput{ProjectID: 2, Content: "Memory D", Category: "fact"})

	t.Run("list all for project", func(t *testing.T) {
		results, err := mgr.List(ctx, 1, ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("expected 3 memories for project 1, got %d", len(results))
		}
	})

	t.Run("list with category filter", func(t *testing.T) {
		results, err := mgr.List(ctx, 1, ListOptions{Category: "decision"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 decisions, got %d", len(results))
		}
	})

	t.Run("list respects project isolation", func(t *testing.T) {
		results, err := mgr.List(ctx, 2, ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 memory for project 2, got %d", len(results))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		results, err := mgr.List(ctx, 1, ListOptions{Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 with limit, got %d", len(results))
		}
	})

	t.Run("invalid category error", func(t *testing.T) {
		_, err := mgr.List(ctx, 1, ListOptions{Category: "invalid"})
		if err != ErrInvalidCategory {
			t.Errorf("expected ErrInvalidCategory, got %v", err)
		}
	})
}

func TestMemoryManager_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMockMemoryStorage()
	mgr := NewMemoryManager(store)

	mem, _ := mgr.Store(ctx, StoreMemoryInput{
		ProjectID: 1,
		Content:   "Memory to delete.",
	})

	t.Run("delete existing memory", func(t *testing.T) {
		err := mgr.Delete(ctx, 1, mem.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deleted
		results, _ := mgr.List(ctx, 1, ListOptions{})
		if len(results) != 0 {
			t.Errorf("expected 0 memories after delete, got %d", len(results))
		}
	})

	t.Run("delete non-existent memory", func(t *testing.T) {
		err := mgr.Delete(ctx, 1, 9999)
		if err != ErrMemoryNotFound {
			t.Errorf("expected ErrMemoryNotFound, got %v", err)
		}
	})

	t.Run("delete memory from wrong project", func(t *testing.T) {
		mem2, _ := mgr.Store(ctx, StoreMemoryInput{
			ProjectID: 1,
			Content:   "Project 1 memory.",
		})
		err := mgr.Delete(ctx, 2, mem2.ID)
		if err != ErrMemoryNotInProject {
			t.Errorf("expected ErrMemoryNotInProject, got %v", err)
		}
	})
}

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"trim whitespace", "  hello  ", "hello"},
		{"normalize CRLF", "line1\r\nline2", "line1\nline2"},
		{"normalize CR", "line1\rline2", "line1\nline2"},
		{"collapse blank lines", "a\n\n\n\n\nb", "a\n\nb"},
		{"combined", "  hello\r\n\r\n\r\nworld  ", "hello\n\nworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeContent(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeContent(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	t.Run("same content same hash", func(t *testing.T) {
		h1 := ContentHash("hello world")
		h2 := ContentHash("hello world")
		if h1 != h2 {
			t.Error("same content should produce same hash")
		}
	})

	t.Run("different content different hash", func(t *testing.T) {
		h1 := ContentHash("hello world")
		h2 := ContentHash("goodbye world")
		if h1 == h2 {
			t.Error("different content should produce different hash")
		}
	})

	t.Run("whitespace normalization in hash", func(t *testing.T) {
		// CRLF normalization: \r\n → \n
		h1 := ContentHash("hello\r\nworld")
		h2 := ContentHash("hello\nworld")
		if h1 != h2 {
			t.Error("CRLF-normalized content should produce same hash")
		}
		// Outer trim + blank line collapse
		h3 := ContentHash("\n\nhello\n\n\n\nworld\n\n")
		h4 := ContentHash("hello\n\nworld")
		if h3 != h4 {
			t.Error("trimmed and collapsed content should produce same hash")
		}
	})
}

func TestIsValidCategory(t *testing.T) {
	valid := []string{"decision", "fact", "event", "preference", "advice", "general"}
	for _, c := range valid {
		if !IsValidCategory(c) {
			t.Errorf("expected %q to be valid", c)
		}
	}

	invalid := []string{"", "invalid", "DECISION", "Decision"}
	for _, c := range invalid {
		if IsValidCategory(c) {
			t.Errorf("expected %q to be invalid", c)
		}
	}
}

func TestMemoryManager_GlobalProject(t *testing.T) {
ctx := context.Background()
store := NewMockMemoryStorage()
mgr := NewMemoryManager(store)

const globalProjectID = int64(0)

// Seed: one global memory, one project-local memory
globalMem, err := mgr.Store(ctx, StoreMemoryInput{
ProjectID: globalProjectID,
Content:   "Global cross-project fact about the deployment pipeline.",
Category:  "fact",
})
if err != nil {
t.Fatalf("expected no error storing global memory, got: %v", err)
}
if globalMem.ProjectID != globalProjectID {
t.Errorf("expected project_id 0, got %d", globalMem.ProjectID)
}

_, err = mgr.Store(ctx, StoreMemoryInput{
ProjectID: 1,
Content:   "Project-specific note about deployment.",
Category:  "fact",
})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

t.Run("search global project returns only global memories", func(t *testing.T) {
results, err := mgr.Search(ctx, globalProjectID, "deployment", SearchOptions{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(results) != 1 {
t.Fatalf("expected 1 global result, got %d", len(results))
}
if results[0].ProjectID != globalProjectID {
t.Errorf("expected project_id 0, got %d", results[0].ProjectID)
}
})

t.Run("list global project returns only global memories", func(t *testing.T) {
memories, err := mgr.List(ctx, globalProjectID, ListOptions{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(memories) != 1 {
t.Fatalf("expected 1 global memory, got %d", len(memories))
}
if memories[0].ProjectID != globalProjectID {
t.Errorf("expected project_id 0, got %d", memories[0].ProjectID)
}
})

t.Run("project-local search does not return global memories", func(t *testing.T) {
results, err := mgr.Search(ctx, 1, "deployment", SearchOptions{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
for _, r := range results {
if r.ProjectID == globalProjectID {
t.Error("project-local search must not return global memories (project_id=0)")
}
}
})
}

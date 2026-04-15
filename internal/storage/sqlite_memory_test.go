package storage

import (
	"context"
	"os"
	"testing"

	"github.com/AlexiaChen/issue-kanban-mcp/internal/memory"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
)

func setupMemoryTestDB(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "memtest-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	store, err := NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create storage: %v", err)
	}

	return store, func() {
		store.Close()
		os.Remove(tmpFile.Name())
	}
}

func createTestProject(t *testing.T, store *SQLiteStorage, name string) *queue.Queue {
	t.Helper()
	ctx := context.Background()
	q, err := store.CreateProject(ctx, queue.CreateQueueInput{
		Name:        name,
		Description: "test project for memory",
	})
	if err != nil {
		t.Fatalf("failed to create project %q: %v", name, err)
	}
	return q
}

func TestSQLiteMemory_StoreAndGet(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj := createTestProject(t, store, "mem-test-1")

	input := memory.StoreMemoryInput{
		ProjectID:   int64(proj.ID),
		Content:     "Go uses goroutines for concurrency",
		Summary:     "Go concurrency model",
		Category:    "fact",
		Tags:        "go,concurrency",
		Source:      "documentation",
		Importance:  4,
		ContentHash: "abc123hash",
	}

	mem, err := store.StoreMemory(ctx, input)
	if err != nil {
		t.Fatalf("StoreMemory failed: %v", err)
	}
	if mem.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if mem.Content != input.Content {
		t.Errorf("content mismatch: got %q", mem.Content)
	}
	if string(mem.Category) != input.Category {
		t.Errorf("category mismatch: got %q", mem.Category)
	}
	if mem.Importance != 4 {
		t.Errorf("importance mismatch: got %d", mem.Importance)
	}
	if mem.ContentHash != "abc123hash" {
		t.Errorf("hash mismatch: got %q", mem.ContentHash)
	}

	// GetMemory
	got, err := store.GetMemory(ctx, mem.ID)
	if err != nil {
		t.Fatalf("GetMemory failed: %v", err)
	}
	if got.ID != mem.ID {
		t.Errorf("expected ID %d, got %d", mem.ID, got.ID)
	}
	if got.Content != input.Content {
		t.Errorf("content mismatch on get")
	}
}

func TestSQLiteMemory_Dedup(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj := createTestProject(t, store, "dedup-test")

	input := memory.StoreMemoryInput{
		ProjectID:   int64(proj.ID),
		Content:     "unique content",
		Category:    "fact",
		Importance:  3,
		ContentHash: "dedup-hash-001",
	}

	mem1, err := store.StoreMemory(ctx, input)
	if err != nil {
		t.Fatalf("first store failed: %v", err)
	}

	// Same hash, same project → should return existing
	mem2, err := store.StoreMemory(ctx, input)
	if err != nil {
		t.Fatalf("second store failed: %v", err)
	}

	if mem1.ID != mem2.ID {
		t.Errorf("dedup failed: got IDs %d and %d", mem1.ID, mem2.ID)
	}
}

func TestSQLiteMemory_DedupCrossProject(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj1 := createTestProject(t, store, "cross-proj-1")
	proj2 := createTestProject(t, store, "cross-proj-2")

	hash := "same-hash-across-projects"

	mem1, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
		ProjectID:   int64(proj1.ID),
		Content:     "content in project 1",
		Category:    "fact",
		Importance:  3,
		ContentHash: hash,
	})
	if err != nil {
		t.Fatalf("store in proj1 failed: %v", err)
	}

	mem2, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
		ProjectID:   int64(proj2.ID),
		Content:     "content in project 2",
		Category:    "fact",
		Importance:  3,
		ContentHash: hash,
	})
	if err != nil {
		t.Fatalf("store in proj2 failed: %v", err)
	}

	if mem1.ID == mem2.ID {
		t.Error("same hash in different projects should create separate memories")
	}
}

func TestSQLiteMemory_Search(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj := createTestProject(t, store, "search-test")
	pid := int64(proj.ID)

	memories := []memory.StoreMemoryInput{
		{ProjectID: pid, Content: "Go goroutines are lightweight threads", Category: "fact", Importance: 4, ContentHash: "s1"},
		{ProjectID: pid, Content: "Python uses the GIL for thread safety", Category: "fact", Importance: 3, ContentHash: "s2"},
		{ProjectID: pid, Content: "Use context.WithTimeout for HTTP requests", Category: "advice", Importance: 5, ContentHash: "s3"},
	}
	for _, m := range memories {
		if _, err := store.StoreMemory(ctx, m); err != nil {
			t.Fatalf("store failed: %v", err)
		}
	}

	t.Run("basic search", func(t *testing.T) {
		results, err := store.SearchMemories(ctx, pid, "goroutines", memory.SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least 1 result for 'goroutines'")
		}
		if results[0].Content != "Go goroutines are lightweight threads" {
			t.Errorf("unexpected first result: %q", results[0].Content)
		}
	})

	t.Run("search with category filter", func(t *testing.T) {
		results, err := store.SearchMemories(ctx, pid, "context", memory.SearchOptions{
			Category: "advice",
			Limit:    10,
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected result for 'context' in advice category")
		}
	})

	t.Run("search isolation", func(t *testing.T) {
		proj2 := createTestProject(t, store, "search-isolated")
		results, err := store.SearchMemories(ctx, int64(proj2.ID), "goroutines", memory.SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results in different project, got %d", len(results))
		}
	})

	t.Run("search no match", func(t *testing.T) {
		results, err := store.SearchMemories(ctx, pid, "kubernetes", memory.SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

func TestSQLiteMemory_List(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj := createTestProject(t, store, "list-test")
	pid := int64(proj.ID)

	for i := 0; i < 5; i++ {
		cat := "fact"
		if i%2 == 0 {
			cat = "advice"
		}
		_, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
			ProjectID:   pid,
			Content:     "memory content " + string(rune('A'+i)),
			Category:    cat,
			Importance:  3,
			ContentHash: "list-hash-" + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("store %d failed: %v", i, err)
		}
	}

	t.Run("list all", func(t *testing.T) {
		mems, err := store.ListMemories(ctx, pid, memory.ListOptions{Limit: 50})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if len(mems) != 5 {
			t.Errorf("expected 5 memories, got %d", len(mems))
		}
	})

	t.Run("list with category filter", func(t *testing.T) {
		mems, err := store.ListMemories(ctx, pid, memory.ListOptions{
			Category: "advice",
			Limit:    50,
		})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if len(mems) != 3 {
			t.Errorf("expected 3 advice memories, got %d", len(mems))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		mems, err := store.ListMemories(ctx, pid, memory.ListOptions{Limit: 2})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if len(mems) != 2 {
			t.Errorf("expected 2 memories, got %d", len(mems))
		}
	})

	t.Run("list isolation", func(t *testing.T) {
		proj2 := createTestProject(t, store, "list-isolated")
		mems, err := store.ListMemories(ctx, int64(proj2.ID), memory.ListOptions{Limit: 50})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if len(mems) != 0 {
			t.Errorf("expected 0 memories in new project, got %d", len(mems))
		}
	})
}

func TestSQLiteMemory_Delete(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj := createTestProject(t, store, "delete-test")
	pid := int64(proj.ID)

	mem, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
		ProjectID:   pid,
		Content:     "to be deleted",
		Category:    "fact",
		Importance:  3,
		ContentHash: "del-hash",
	})
	if err != nil {
		t.Fatalf("store failed: %v", err)
	}

	t.Run("delete existing", func(t *testing.T) {
		err := store.DeleteMemory(ctx, pid, mem.ID)
		if err != nil {
			t.Fatalf("delete failed: %v", err)
		}
		_, err = store.GetMemory(ctx, mem.ID)
		if err != memory.ErrMemoryNotFound {
			t.Errorf("expected ErrMemoryNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent", func(t *testing.T) {
		err := store.DeleteMemory(ctx, pid, 99999)
		if err != memory.ErrMemoryNotFound {
			t.Errorf("expected ErrMemoryNotFound, got %v", err)
		}
	})

	t.Run("delete wrong project", func(t *testing.T) {
		mem2, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
			ProjectID:   pid,
			Content:     "project scoped",
			Category:    "fact",
			Importance:  3,
			ContentHash: "proj-scope-hash",
		})
		if err != nil {
			t.Fatalf("store failed: %v", err)
		}

		proj2 := createTestProject(t, store, "delete-wrong-proj")
		err = store.DeleteMemory(ctx, int64(proj2.ID), mem2.ID)
		if err != memory.ErrMemoryNotInProject {
			t.Errorf("expected ErrMemoryNotInProject, got %v", err)
		}
	})
}

func TestSQLiteMemory_DeleteByProject(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj := createTestProject(t, store, "bulk-delete-test")
	pid := int64(proj.ID)

	for i := 0; i < 3; i++ {
		_, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
			ProjectID:   pid,
			Content:     "bulk content " + string(rune('A'+i)),
			Category:    "fact",
			Importance:  3,
			ContentHash: "bulk-hash-" + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("store %d failed: %v", i, err)
		}
	}

	err := store.DeleteMemoriesByProject(ctx, pid)
	if err != nil {
		t.Fatalf("DeleteMemoriesByProject failed: %v", err)
	}

	mems, err := store.ListMemories(ctx, pid, memory.ListOptions{Limit: 50})
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(mems) != 0 {
		t.Errorf("expected 0 memories after bulk delete, got %d", len(mems))
	}
}

func TestSQLiteMemory_DeleteProjectCascade(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	proj := createTestProject(t, store, "cascade-test")
	pid := int64(proj.ID)

	_, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
		ProjectID:   pid,
		Content:     "will be cascade deleted",
		Category:    "fact",
		Importance:  3,
		ContentHash: "cascade-hash",
	})
	if err != nil {
		t.Fatalf("store failed: %v", err)
	}

	// Delete project should also delete memories
	err = store.DeleteProject(ctx, pid)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	mems, err := store.ListMemories(ctx, pid, memory.ListOptions{Limit: 50})
	if err != nil {
		t.Fatalf("list after project delete failed: %v", err)
	}
	if len(mems) != 0 {
		t.Errorf("expected 0 memories after project delete, got %d", len(mems))
	}
}

func TestSQLiteMemory_GetNotFound(t *testing.T) {
	store, cleanup := setupMemoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := store.GetMemory(ctx, 99999)
	if err != memory.ErrMemoryNotFound {
		t.Errorf("expected ErrMemoryNotFound, got %v", err)
	}
}

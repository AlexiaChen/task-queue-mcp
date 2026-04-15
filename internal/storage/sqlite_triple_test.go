package storage

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"github.com/AlexiaChen/issue-kanban-mcp/internal/memory"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
)

func setupTripleTestDB(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "triple_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	storage, err := NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	_, err = storage.CreateProject(ctx, queue.CreateQueueInput{Name: "test-project", Description: "test"})
	if err != nil {
		storage.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create project: %v", err)
	}

	return storage, func() {
		storage.Close()
		os.Remove(tmpFile.Name())
	}
}

func TestSQLiteTriple_StoreAndGet(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	triple, err := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID:  1,
		Subject:    "auth-module",
		Predicate:  "uses",
		Object:     "JWT tokens",
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if triple.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if triple.Subject != "auth-module" {
		t.Errorf("subject = %q, want %q", triple.Subject, "auth-module")
	}
	if triple.Confidence != 0.9 {
		t.Errorf("confidence = %f, want 0.9", triple.Confidence)
	}
	if triple.ValidTo != nil {
		t.Error("valid_to should be nil for active triple")
	}

	// Get by ID
	fetched, err := store.GetTriple(ctx, 1, triple.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fetched.Subject != "auth-module" {
		t.Errorf("fetched subject = %q, want %q", fetched.Subject, "auth-module")
	}
	if fetched.Predicate != "uses" {
		t.Errorf("fetched predicate = %q, want %q", fetched.Predicate, "uses")
	}
}

func TestSQLiteTriple_GetNotFound(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := store.GetTriple(ctx, 1, 9999)
	if err != memory.ErrTripleNotFound {
		t.Errorf("err = %v, want ErrTripleNotFound", err)
	}
}

func TestSQLiteTriple_GetWrongProject(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	triple, _ := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1,
		Subject:   "test",
		Predicate: "is",
		Object:    "isolated",
	})

	_, err := store.GetTriple(ctx, 999, triple.ID)
	if err != memory.ErrTripleNotInProject {
		t.Errorf("err = %v, want ErrTripleNotInProject", err)
	}
}

func TestSQLiteTriple_StoreWithTimeRange(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	triple, err := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1,
		Subject:   "Max",
		Predicate: "works_at",
		Object:    "Google",
		ValidFrom: "2020-01-01T00:00:00Z",
		ValidTo:   "2024-06-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	expectedFrom, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	expectedTo, _ := time.Parse(time.RFC3339, "2024-06-01T00:00:00Z")

	if !triple.ValidFrom.Equal(expectedFrom) {
		t.Errorf("valid_from = %v, want %v", triple.ValidFrom, expectedFrom)
	}
	if triple.ValidTo == nil {
		t.Fatal("valid_to should not be nil")
	}
	if !triple.ValidTo.Equal(expectedTo) {
		t.Errorf("valid_to = %v, want %v", triple.ValidTo, expectedTo)
	}
}

func TestSQLiteTriple_StoreWithSourceMemory(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create a memory first
	mem, err := store.StoreMemory(ctx, memory.StoreMemoryInput{
		ProjectID:   1,
		Content:     "We decided to use JWT for authentication",
		Category:    "decision",
		Importance:  3,
		ContentHash: "abc123",
	})
	if err != nil {
		t.Fatalf("store memory: %v", err)
	}

	// Store triple referencing the memory
	triple, err := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID:      1,
		Subject:        "auth",
		Predicate:      "uses",
		Object:         "JWT",
		SourceMemoryID: &mem.ID,
	})
	if err != nil {
		t.Fatalf("store triple: %v", err)
	}
	if triple.SourceMemoryID == nil || *triple.SourceMemoryID != mem.ID {
		t.Errorf("source_memory_id = %v, want %d", triple.SourceMemoryID, mem.ID)
	}
}

func TestSQLiteTriple_ReplaceExisting(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Store initial fact
	old, err := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1,
		Subject:   "database",
		Predicate: "engine",
		Object:    "PostgreSQL",
		ValidFrom: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("store old: %v", err)
	}

	// Replace with new fact
	newTriple, err := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID:       1,
		Subject:         "database",
		Predicate:       "engine",
		Object:          "SQLite",
		ValidFrom:       "2026-03-15T00:00:00Z",
		ReplaceExisting: true,
	})
	if err != nil {
		t.Fatalf("store new: %v", err)
	}
	if newTriple.Object != "SQLite" {
		t.Errorf("new object = %q, want %q", newTriple.Object, "SQLite")
	}

	// Verify old triple was invalidated
	oldFetched, err := store.GetTriple(ctx, 1, old.ID)
	if err != nil {
		t.Fatalf("get old: %v", err)
	}
	if oldFetched.ValidTo == nil {
		t.Fatal("old triple should be invalidated")
	}
	expectedInvalidation, _ := time.Parse(time.RFC3339, "2026-03-15T00:00:00Z")
	if !oldFetched.ValidTo.Equal(expectedInvalidation) {
		t.Errorf("old valid_to = %v, want %v", oldFetched.ValidTo, expectedInvalidation)
	}
}

func TestSQLiteTriple_ReplaceExisting_SameObjectNoInvalidation(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	old, _ := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1,
		Subject:   "db",
		Predicate: "engine",
		Object:    "SQLite",
		ValidFrom: "2026-01-01T00:00:00Z",
	})

	// Store same object with ReplaceExisting=true
	store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID:       1,
		Subject:         "db",
		Predicate:       "engine",
		Object:          "SQLite",
		ValidFrom:       "2026-03-15T00:00:00Z",
		ReplaceExisting: true,
	})

	oldFetched, _ := store.GetTriple(ctx, 1, old.ID)
	if oldFetched.ValidTo != nil {
		t.Error("old triple should NOT be invalidated when same object is stored")
	}
}

func TestSQLiteTriple_QueryBySubject(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "auth-module", Predicate: "uses", Object: "JWT"})
	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "api-module", Predicate: "uses", Object: "REST"})
	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "auth-module", Predicate: "version", Object: "2.0"})

	results, err := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{Subject: "auth"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestSQLiteTriple_QueryByPredicate(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "auth", Predicate: "uses", Object: "JWT"})
	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "api", Predicate: "uses", Object: "REST"})
	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "auth", Predicate: "version", Object: "2.0"})

	results, err := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{Predicate: "uses"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestSQLiteTriple_QueryActiveOnly(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Active triple
	store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "db", Predicate: "engine", Object: "SQLite",
	})
	// Expired triple
	store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "db", Predicate: "engine", Object: "PostgreSQL",
		ValidFrom: "2020-01-01T00:00:00Z", ValidTo: "2024-01-01T00:00:00Z",
	})

	results, err := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{ActiveOnly: true})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Object != "SQLite" {
		t.Errorf("active triple = %q, want %q", results[0].Object, "SQLite")
	}
}

func TestSQLiteTriple_QueryPointInTime(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Max worked at Google 2020-2024
	store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "Max", Predicate: "works_at", Object: "Google",
		ValidFrom: "2020-01-01T00:00:00Z", ValidTo: "2024-06-01T00:00:00Z",
	})
	// Max works at OpenAI from 2024
	store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "Max", Predicate: "works_at", Object: "OpenAI",
		ValidFrom: "2024-06-01T00:00:00Z",
	})

	// At 2022 → Google
	results, err := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{
		PointInTime: "2022-06-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("query 2022: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d at 2022, want 1", len(results))
	}
	if results[0].Object != "Google" {
		t.Errorf("at 2022 = %q, want %q", results[0].Object, "Google")
	}

	// At 2025 → OpenAI
	results, err = store.QueryTriples(ctx, 1, memory.QueryTripleOptions{
		PointInTime: "2025-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("query 2025: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d at 2025, want 1", len(results))
	}
	if results[0].Object != "OpenAI" {
		t.Errorf("at 2025 = %q, want %q", results[0].Object, "OpenAI")
	}

	// At boundary 2024-06-01 (closed-open: Google ends, OpenAI starts)
	results, err = store.QueryTriples(ctx, 1, memory.QueryTripleOptions{
		PointInTime: "2024-06-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("query boundary: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d at boundary, want 1 (only OpenAI, Google's valid_to is exclusive)", len(results))
	}
	if results[0].Object != "OpenAI" {
		t.Errorf("at boundary = %q, want %q", results[0].Object, "OpenAI")
	}
}

func TestSQLiteTriple_QueryProjectIsolation(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create second project
	store.CreateProject(ctx, queue.CreateQueueInput{Name: "project-2", Description: "test2"})

	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "a", Predicate: "b", Object: "c"})
	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 2, Subject: "d", Predicate: "e", Object: "f"})

	results, err := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results for project 1, want 1", len(results))
	}
}

func TestSQLiteTriple_QueryOrdering(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "old", Predicate: "p", Object: "o",
		ValidFrom: "2020-01-01T00:00:00Z",
	})
	store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "new", Predicate: "p", Object: "o",
		ValidFrom: "2026-01-01T00:00:00Z",
	})

	results, err := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Newest first
	if results[0].Subject != "new" {
		t.Errorf("first result = %q, want %q (newest first)", results[0].Subject, "new")
	}
}

func TestSQLiteTriple_Invalidate(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	triple, _ := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "project", Predicate: "status", Object: "active",
	})

	invalidateAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	err := store.InvalidateTriple(ctx, 1, triple.ID, invalidateAt)
	if err != nil {
		t.Fatalf("invalidate: %v", err)
	}

	fetched, _ := store.GetTriple(ctx, 1, triple.ID)
	if fetched.ValidTo == nil {
		t.Fatal("should be invalidated")
	}
	if !fetched.ValidTo.Equal(invalidateAt) {
		t.Errorf("valid_to = %v, want %v", fetched.ValidTo, invalidateAt)
	}
}

func TestSQLiteTriple_InvalidateNotFound(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	err := store.InvalidateTriple(ctx, 1, 9999, time.Now())
	if err != memory.ErrTripleNotFound {
		t.Errorf("err = %v, want ErrTripleNotFound", err)
	}
}

func TestSQLiteTriple_Delete(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	triple, _ := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "test", Predicate: "is", Object: "deleted",
	})

	err := store.DeleteTriple(ctx, 1, triple.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = store.GetTriple(ctx, 1, triple.ID)
	if err != memory.ErrTripleNotFound {
		t.Errorf("err = %v, want ErrTripleNotFound after delete", err)
	}
}

func TestSQLiteTriple_DeleteWrongProject(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	triple, _ := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "test", Predicate: "is", Object: "protected",
	})

	err := store.DeleteTriple(ctx, 999, triple.ID)
	if err != memory.ErrTripleNotInProject {
		t.Errorf("err = %v, want ErrTripleNotInProject", err)
	}
}

func TestSQLiteTriple_DeleteByProject(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "a", Predicate: "b", Object: "c"})
	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "d", Predicate: "e", Object: "f"})

	err := store.DeleteTriplesByProject(ctx, 1)
	if err != nil {
		t.Fatalf("delete by project: %v", err)
	}

	results, _ := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{})
	if len(results) != 0 {
		t.Errorf("got %d results after project delete, want 0", len(results))
	}
}

func TestSQLiteTriple_DeleteProjectCascade(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	store.StoreTriple(ctx, memory.StoreTripleInput{ProjectID: 1, Subject: "a", Predicate: "b", Object: "c"})

	// Delete the project — should cascade to triples
	err := store.DeleteProject(ctx, 1)
	if err != nil {
		t.Fatalf("delete project: %v", err)
	}

	// Triples should be gone (can't query by project since project is gone, but DB should be clean)
	results, _ := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{})
	if len(results) != 0 {
		t.Errorf("got %d triples after project delete, want 0", len(results))
	}
}

func TestSQLiteTriple_DefaultConfidence(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	triple, _ := store.StoreTriple(ctx, memory.StoreTripleInput{
		ProjectID: 1, Subject: "a", Predicate: "b", Object: "c",
	})

	if math.Abs(triple.Confidence-1.0) > 1e-9 {
		t.Errorf("default confidence = %f, want 1.0", triple.Confidence)
	}
}

func TestSQLiteTriple_QueryPagination(t *testing.T) {
	store, cleanup := setupTripleTestDB(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		store.StoreTriple(ctx, memory.StoreTripleInput{
			ProjectID: 1,
			Subject:   "item",
			Predicate: "index",
			Object:    string(rune('A' + i)),
			ValidFrom: time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
	}

	// First page
	results, err := store.QueryTriples(ctx, 1, memory.QueryTripleOptions{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("query page 1: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("page 1: got %d, want 3", len(results))
	}

	// Second page
	results, err = store.QueryTriples(ctx, 1, memory.QueryTripleOptions{Limit: 3, Offset: 3})
	if err != nil {
		t.Fatalf("query page 2: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("page 2: got %d, want 3", len(results))
	}
}

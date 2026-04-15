package memory_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AlexiaChen/issue-kanban-mcp/internal/memory"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/storage"
)

func TestIntegration_TripleWorkflow(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "triple-integration-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	store, err := storage.NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	queueMgr := queue.NewManager(store)
	tripleMgr := memory.NewTripleManager(store)
	ctx := context.Background()

	// Create project
	proj, err := queueMgr.CreateProject(ctx, queue.CreateQueueInput{
		Name:        "Triple Integration Project",
		Description: "Testing temporal knowledge graph end-to-end",
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	pid := proj.ID

	// 1. Store triples representing project history
	t1, err := tripleMgr.Store(ctx, memory.StoreTripleInput{
		ProjectID: pid, Subject: "auth-module", Predicate: "uses", Object: "session cookies",
		ValidFrom: "2024-01-01T00:00:00Z", Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("store t1: %v", err)
	}

	_, err = tripleMgr.Store(ctx, memory.StoreTripleInput{
		ProjectID: pid, Subject: "auth-module", Predicate: "uses", Object: "JWT tokens",
		ValidFrom: "2024-06-01T00:00:00Z", Confidence: 0.95, ReplaceExisting: true,
	})
	if err != nil {
		t.Fatalf("store t2: %v", err)
	}

	// 2. Verify auto-invalidation: old triple should have valid_to set
	oldTriple, err := tripleMgr.Get(ctx, pid, t1.ID)
	if err != nil {
		t.Fatalf("get old triple: %v", err)
	}
	if oldTriple.ValidTo == nil {
		t.Fatal("old triple should be invalidated after replace_existing")
	}
	expectedInvalidation, _ := time.Parse(time.RFC3339, "2024-06-01T00:00:00Z")
	if !oldTriple.ValidTo.Equal(expectedInvalidation) {
		t.Errorf("old valid_to = %v, want %v", oldTriple.ValidTo, expectedInvalidation)
	}

	// 3. Store more triples
	_, err = tripleMgr.Store(ctx, memory.StoreTripleInput{
		ProjectID: pid, Subject: "database", Predicate: "engine", Object: "SQLite",
	})
	if err != nil {
		t.Fatalf("store db triple: %v", err)
	}

	_, err = tripleMgr.Store(ctx, memory.StoreTripleInput{
		ProjectID: pid, Subject: "auth-module", Predicate: "has_label", Object: "security",
	})
	if err != nil {
		t.Fatalf("store label triple: %v", err)
	}

	// 4. Query active-only: should exclude invalidated old triple
	active, err := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{ActiveOnly: true})
	if err != nil {
		t.Fatalf("query active: %v", err)
	}
	if len(active) != 3 {
		t.Errorf("active count = %d, want 3", len(active))
	}

	// 5. Point-in-time query: at 2024-03-01, auth used session cookies
	march24, err := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{
		Subject: "auth", PointInTime: "2024-03-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("query march 2024: %v", err)
	}
	if len(march24) != 1 {
		t.Fatalf("march 2024 count = %d, want 1", len(march24))
	}
	if march24[0].Object != "session cookies" {
		t.Errorf("march 2024 object = %q, want %q", march24[0].Object, "session cookies")
	}

	// 6. Point-in-time query: at 2024-09-01, auth uses JWT
	sept24, err := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{
		Subject: "auth", Predicate: "uses", PointInTime: "2024-09-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("query sept 2024: %v", err)
	}
	if len(sept24) != 1 {
		t.Fatalf("sept 2024 count = %d, want 1", len(sept24))
	}
	if sept24[0].Object != "JWT tokens" {
		t.Errorf("sept 2024 object = %q, want %q", sept24[0].Object, "JWT tokens")
	}

	// 7. Query by predicate
	usesTriples, err := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{Predicate: "uses"})
	if err != nil {
		t.Fatalf("query by predicate: %v", err)
	}
	if len(usesTriples) != 2 {
		t.Errorf("'uses' predicate count = %d, want 2 (both old and new)", len(usesTriples))
	}

	// 8. Invalidate a triple
	dbTriples, _ := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{Subject: "database", ActiveOnly: true})
	if len(dbTriples) != 1 {
		t.Fatalf("expected 1 active db triple, got %d", len(dbTriples))
	}
	err = tripleMgr.Invalidate(ctx, pid, dbTriples[0].ID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("invalidate: %v", err)
	}

	activeAfter, _ := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{ActiveOnly: true})
	if len(activeAfter) != 2 {
		t.Errorf("active after invalidation = %d, want 2", len(activeAfter))
	}

	// 9. Delete a triple
	labelTriples, _ := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{Predicate: "has_label"})
	if len(labelTriples) != 1 {
		t.Fatalf("expected 1 label triple, got %d", len(labelTriples))
	}
	err = tripleMgr.Delete(ctx, pid, labelTriples[0].ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	all, _ := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{})
	if len(all) != 3 {
		t.Errorf("total after delete = %d, want 3", len(all))
	}

	// 10. Project deletion cascades to triples
	err = queueMgr.DeleteProject(ctx, pid)
	if err != nil {
		t.Fatalf("delete project: %v", err)
	}
	remaining, _ := tripleMgr.Query(ctx, pid, memory.QueryTripleOptions{})
	if len(remaining) != 0 {
		t.Errorf("triples after project delete = %d, want 0", len(remaining))
	}
}

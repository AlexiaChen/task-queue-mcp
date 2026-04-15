package memory

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTripleManager_Store_ValidInput(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	triple, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "auth-module",
		Predicate: "uses",
		Object:    "JWT tokens",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triple.Subject != "auth-module" {
		t.Errorf("subject = %q, want %q", triple.Subject, "auth-module")
	}
	if triple.Predicate != "uses" {
		t.Errorf("predicate = %q, want %q", triple.Predicate, "uses")
	}
	if triple.Object != "JWT tokens" {
		t.Errorf("object = %q, want %q", triple.Object, "JWT tokens")
	}
	if triple.Confidence != 1.0 {
		t.Errorf("confidence = %f, want 1.0", triple.Confidence)
	}
	if triple.ValidTo != nil {
		t.Error("valid_to should be nil for active triple")
	}
}

func TestTripleManager_Store_EmptySubject(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "",
		Predicate: "uses",
		Object:    "Go",
	})
	if err != ErrEmptySubject {
		t.Errorf("err = %v, want ErrEmptySubject", err)
	}
}

func TestTripleManager_Store_EmptyPredicate(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: "",
		Object:    "Go",
	})
	if err != ErrEmptyPredicate {
		t.Errorf("err = %v, want ErrEmptyPredicate", err)
	}
}

func TestTripleManager_Store_EmptyObject(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: "uses",
		Object:    "",
	})
	if err != ErrEmptyObject {
		t.Errorf("err = %v, want ErrEmptyObject", err)
	}
}

func TestTripleManager_Store_WhitespaceOnlyFields(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "  \t  ",
		Predicate: "uses",
		Object:    "Go",
	})
	if err != ErrEmptySubject {
		t.Errorf("err = %v, want ErrEmptySubject for whitespace-only subject", err)
	}
}

func TestTripleManager_Store_SubjectTooLong(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   strings.Repeat("x", MaxTripleFieldLength+1),
		Predicate: "uses",
		Object:    "Go",
	})
	if err != ErrSubjectTooLong {
		t.Errorf("err = %v, want ErrSubjectTooLong", err)
	}
}

func TestTripleManager_Store_PredicateTooLong(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: strings.Repeat("x", MaxTripleFieldLength+1),
		Object:    "Go",
	})
	if err != ErrPredicateTooLong {
		t.Errorf("err = %v, want ErrPredicateTooLong", err)
	}
}

func TestTripleManager_Store_ObjectTooLong(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: "uses",
		Object:    strings.Repeat("x", MaxTripleFieldLength+1),
	})
	if err != ErrObjectTooLong {
		t.Errorf("err = %v, want ErrObjectTooLong", err)
	}
}

func TestTripleManager_Store_InvalidConfidence(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	tests := []struct {
		name       string
		confidence float64
	}{
		{"negative", -0.1},
		{"too_high", 1.1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mgr.Store(ctx, StoreTripleInput{
				ProjectID:  1,
				Subject:    "project",
				Predicate:  "uses",
				Object:     "Go",
				Confidence: tc.confidence,
			})
			if err != ErrInvalidConfidence {
				t.Errorf("err = %v, want ErrInvalidConfidence", err)
			}
		})
	}
}

func TestTripleManager_Store_DefaultConfidence(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	triple, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: "uses",
		Object:    "Go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triple.Confidence != 1.0 {
		t.Errorf("confidence = %f, want 1.0 (default)", triple.Confidence)
	}
}

func TestTripleManager_Store_CustomConfidence(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	triple, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID:  1,
		Subject:    "project",
		Predicate:  "uses",
		Object:     "Go",
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triple.Confidence != 0.8 {
		t.Errorf("confidence = %f, want 0.8", triple.Confidence)
	}
}

func TestTripleManager_Store_InvalidTimeRange(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	_, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: "uses",
		Object:    "Go",
		ValidFrom: "2026-01-15T00:00:00Z",
		ValidTo:   "2025-12-01T00:00:00Z",
	})
	if err != ErrInvalidTimeRange {
		t.Errorf("err = %v, want ErrInvalidTimeRange", err)
	}
}

func TestTripleManager_Store_ValidTimeRange(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	triple, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "Max",
		Predicate: "works_at",
		Object:    "Google",
		ValidFrom: "2020-01-01T00:00:00Z",
		ValidTo:   "2024-06-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triple.ValidTo == nil {
		t.Fatal("valid_to should not be nil")
	}
	expectedTo, _ := time.Parse(time.RFC3339, "2024-06-01T00:00:00Z")
	if !triple.ValidTo.Equal(expectedTo) {
		t.Errorf("valid_to = %v, want %v", triple.ValidTo, expectedTo)
	}
}

func TestTripleManager_Store_TrimsWhitespace(t *testing.T) {
	mgr := NewTripleManager(NewMockTripleStorage())
	ctx := context.Background()

	triple, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "  auth module  ",
		Predicate: "  uses  ",
		Object:    "  JWT  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triple.Subject != "auth module" {
		t.Errorf("subject = %q, want %q (trimmed)", triple.Subject, "auth module")
	}
	if triple.Predicate != "uses" {
		t.Errorf("predicate = %q, want %q (trimmed)", triple.Predicate, "uses")
	}
	if triple.Object != "JWT" {
		t.Errorf("object = %q, want %q (trimmed)", triple.Object, "JWT")
	}
}

// --- Temporal Semantics Tests ---

func TestTriple_IsActiveAt_ClosedOpenInterval(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	triple := Triple{
		ValidFrom: from,
		ValidTo:   &to,
	}

	tests := []struct {
		name   string
		at     time.Time
		expect bool
	}{
		{"before_start", from.Add(-time.Second), false},
		{"at_start", from, true},
		{"middle", from.Add(time.Hour * 24 * 90), true},
		{"just_before_end", to.Add(-time.Second), true},
		{"at_end_exclusive", to, false},
		{"after_end", to.Add(time.Second), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := triple.IsActiveAt(tc.at)
			if got != tc.expect {
				t.Errorf("IsActiveAt(%v) = %v, want %v", tc.at, got, tc.expect)
			}
		})
	}
}

func TestTriple_IsActiveAt_NilValidTo(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	triple := Triple{
		ValidFrom: from,
		ValidTo:   nil,
	}

	if !triple.IsActiveAt(from) {
		t.Error("should be active at valid_from")
	}
	if !triple.IsActiveAt(from.Add(time.Hour * 24 * 365 * 100)) {
		t.Error("should be active far in the future when valid_to is nil")
	}
	if triple.IsActiveAt(from.Add(-time.Second)) {
		t.Error("should not be active before valid_from")
	}
}

// --- Replace Existing Tests ---

func TestTripleManager_Store_ReplaceExisting_InvalidatesOldTriple(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	// Store initial triple
	old, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "database",
		Predicate: "uses",
		Object:    "PostgreSQL",
		ValidFrom: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("store old: %v", err)
	}
	if old.ValidTo != nil {
		t.Fatal("old triple should be active initially")
	}

	// Store replacement with ReplaceExisting=true
	newTriple, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID:       1,
		Subject:         "database",
		Predicate:       "uses",
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
		t.Fatal("old triple should be invalidated after replace")
	}
	expectedInvalidation, _ := time.Parse(time.RFC3339, "2026-03-15T00:00:00Z")
	if !oldFetched.ValidTo.Equal(expectedInvalidation) {
		t.Errorf("old valid_to = %v, want %v (new triple's valid_from)", oldFetched.ValidTo, expectedInvalidation)
	}
}

func TestTripleManager_Store_ReplaceExisting_DoesNotInvalidateSameObject(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	// Store initial triple
	old, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "database",
		Predicate: "uses",
		Object:    "SQLite",
		ValidFrom: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("store old: %v", err)
	}

	// Store same object with ReplaceExisting=true (should NOT invalidate)
	_, err = mgr.Store(ctx, StoreTripleInput{
		ProjectID:       1,
		Subject:         "database",
		Predicate:       "uses",
		Object:          "SQLite",
		ValidFrom:       "2026-03-15T00:00:00Z",
		ReplaceExisting: true,
	})
	if err != nil {
		t.Fatalf("store new: %v", err)
	}

	// Old triple should remain active (same object, no invalidation)
	oldFetched, err := store.GetTriple(ctx, 1, old.ID)
	if err != nil {
		t.Fatalf("get old: %v", err)
	}
	if oldFetched.ValidTo != nil {
		t.Error("old triple should remain active when same object is stored with replace_existing")
	}
}

func TestTripleManager_Store_NoReplaceExisting_DoesNotInvalidate(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	// Store initial triple
	old, err := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "issue-123",
		Predicate: "has_label",
		Object:    "bug",
		ValidFrom: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("store first label: %v", err)
	}

	// Store another label with ReplaceExisting=false (default, multi-valued predicate)
	_, err = mgr.Store(ctx, StoreTripleInput{
		ProjectID:       1,
		Subject:         "issue-123",
		Predicate:       "has_label",
		Object:          "backend",
		ReplaceExisting: false,
	})
	if err != nil {
		t.Fatalf("store second label: %v", err)
	}

	// Old triple should still be active
	oldFetched, err := store.GetTriple(ctx, 1, old.ID)
	if err != nil {
		t.Fatalf("get old: %v", err)
	}
	if oldFetched.ValidTo != nil {
		t.Error("old triple should remain active when ReplaceExisting=false")
	}
}

// --- Query Tests ---

func TestTripleManager_Query_EmptyOptions(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	mgr.Store(ctx, StoreTripleInput{ProjectID: 1, Subject: "a", Predicate: "b", Object: "c"})
	mgr.Store(ctx, StoreTripleInput{ProjectID: 1, Subject: "d", Predicate: "e", Object: "f"})
	mgr.Store(ctx, StoreTripleInput{ProjectID: 2, Subject: "g", Predicate: "h", Object: "i"})

	results, err := mgr.Query(ctx, 1, QueryTripleOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2 (project 1 only)", len(results))
	}
}

func TestTripleManager_Query_BySubject(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	mgr.Store(ctx, StoreTripleInput{ProjectID: 1, Subject: "auth-module", Predicate: "uses", Object: "JWT"})
	mgr.Store(ctx, StoreTripleInput{ProjectID: 1, Subject: "api-module", Predicate: "uses", Object: "REST"})

	results, err := mgr.Query(ctx, 1, QueryTripleOptions{Subject: "auth"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestTripleManager_Query_ActiveOnly(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	// Active triple
	mgr.Store(ctx, StoreTripleInput{ProjectID: 1, Subject: "db", Predicate: "uses", Object: "SQLite"})

	// Expired triple
	mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "db",
		Predicate: "uses",
		Object:    "PostgreSQL",
		ValidFrom: "2020-01-01T00:00:00Z",
		ValidTo:   "2024-01-01T00:00:00Z",
	})

	results, err := mgr.Query(ctx, 1, QueryTripleOptions{ActiveOnly: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d active results, want 1", len(results))
	}
	if results[0].Object != "SQLite" {
		t.Errorf("active triple object = %q, want %q", results[0].Object, "SQLite")
	}
}

func TestTripleManager_Query_PointInTime(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	// Triple valid 2020-2024
	mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "Max",
		Predicate: "works_at",
		Object:    "Google",
		ValidFrom: "2020-01-01T00:00:00Z",
		ValidTo:   "2024-06-01T00:00:00Z",
	})

	// Triple valid from 2024 onwards
	mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "Max",
		Predicate: "works_at",
		Object:    "OpenAI",
		ValidFrom: "2024-06-01T00:00:00Z",
	})

	// Query at 2022 → should see Google
	results, err := mgr.Query(ctx, 1, QueryTripleOptions{PointInTime: "2022-06-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results at 2022, want 1", len(results))
	}
	if results[0].Object != "Google" {
		t.Errorf("at 2022: object = %q, want %q", results[0].Object, "Google")
	}

	// Query at 2025 → should see OpenAI
	results, err = mgr.Query(ctx, 1, QueryTripleOptions{PointInTime: "2025-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results at 2025, want 1", len(results))
	}
	if results[0].Object != "OpenAI" {
		t.Errorf("at 2025: object = %q, want %q", results[0].Object, "OpenAI")
	}
}

// --- Invalidate Tests ---

func TestTripleManager_Invalidate(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	triple, _ := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: "status",
		Object:    "active",
	})

	invalidateAt := time.Now().UTC()
	err := mgr.Invalidate(ctx, 1, triple.ID, invalidateAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fetched, _ := store.GetTriple(ctx, 1, triple.ID)
	if fetched.ValidTo == nil {
		t.Fatal("triple should be invalidated")
	}
}

func TestTripleManager_Invalidate_WrongProject(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	triple, _ := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "project",
		Predicate: "status",
		Object:    "active",
	})

	err := mgr.Invalidate(ctx, 999, triple.ID, time.Now())
	if err != ErrTripleNotInProject {
		t.Errorf("err = %v, want ErrTripleNotInProject", err)
	}
}

// --- Delete Tests ---

func TestTripleManager_Delete(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	triple, _ := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "test",
		Predicate: "is",
		Object:    "deleted",
	})

	err := mgr.Delete(ctx, 1, triple.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.GetTriple(ctx, 1, triple.ID)
	if err != ErrTripleNotFound {
		t.Errorf("err = %v, want ErrTripleNotFound", err)
	}
}

func TestTripleManager_Delete_WrongProject(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	triple, _ := mgr.Store(ctx, StoreTripleInput{
		ProjectID: 1,
		Subject:   "test",
		Predicate: "is",
		Object:    "protected",
	})

	err := mgr.Delete(ctx, 999, triple.ID)
	if err != ErrTripleNotInProject {
		t.Errorf("err = %v, want ErrTripleNotInProject", err)
	}
}

func TestTripleManager_Query_DefaultLimit(t *testing.T) {
	store := NewMockTripleStorage()
	mgr := NewTripleManager(store)
	ctx := context.Background()

	// Store 60 triples
	for i := 0; i < 60; i++ {
		mgr.Store(ctx, StoreTripleInput{
			ProjectID: 1,
			Subject:   "item",
			Predicate: "index",
			Object:    strings.Repeat("x", i+1),
		})
	}

	results, err := mgr.Query(ctx, 1, QueryTripleOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != DefaultTripleQueryLimit {
		t.Errorf("got %d results, want %d (default limit)", len(results), DefaultTripleQueryLimit)
	}
}

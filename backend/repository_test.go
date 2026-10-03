package backend

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

// Integration tests against a real PostgreSQL with migrations applied.
// Skipped unless TEST_DATABASE_URL is set. Use a separate test database:
// these tests insert rows (with a unique postal code per run).
func testRepo(t *testing.T) (*Repository, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return NewRepository(pool), ctx
}

func TestRepository_CreateLeadMarksDuplicate(t *testing.T) {
	repo, ctx := testRepo(t)
	city := "Testort " + time.Now().Format("150405.000000")

	first, err := repo.CreateLead(ctx, lead("dup@example.com", "040 123456", "Hauptstraße", "14", "01067", city))
	if err != nil {
		t.Fatal(err)
	}
	if first.DuplicateOf != nil {
		t.Fatalf("first lead must not be a duplicate")
	}

	second, err := repo.CreateLead(ctx, lead(" DUP@example.com", "0170 1", "Hauptstr.", "14", "01067", city))
	if err != nil {
		t.Fatal(err)
	}
	if second.DuplicateOf == nil || *second.DuplicateOf != first.ID {
		t.Fatalf("second lead: duplicate_of = %v, want %s", second.DuplicateOf, first.ID)
	}
}

// Simulates a double-clicked submit button: both requests arrive at the same
// time. Exactly one of them must be the original.
func TestRepository_ConcurrentSubmissions(t *testing.T) {
	repo, ctx := testRepo(t)
	city := "Testort " + time.Now().Format("150405.000000")

	var wg sync.WaitGroup
	results := make([]Lead, 2)
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = repo.CreateLead(ctx, lead("race@example.com", "040 123456", "Lindenweg", "3", "12345", city))
		}()
	}
	wg.Wait()

	originals := 0
	for i := range 2 {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if results[i].DuplicateOf == nil {
			originals++
		}
	}
	if originals != 1 {
		t.Fatalf("expected exactly 1 original, got %d", originals)
	}
}

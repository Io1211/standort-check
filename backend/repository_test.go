package backend

import (
	"context"
	"os"
	"strings"
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
	t.Cleanup(func() { _ = repo.DeleteLead(ctx, first.ID) })
	if first.DuplicateOf != nil {
		t.Fatalf("first lead must not be a duplicate")
	}

	second, err := repo.CreateLead(ctx, lead(" DUP@example.com", "0170 1", "Hauptstr.", "14", "01067", city))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.DeleteLead(ctx, second.ID) })
	if second.DuplicateOf == nil || *second.DuplicateOf != first.ID {
		t.Fatalf("second lead: duplicate_of = %v, want %s", second.DuplicateOf, first.ID)
	}
}

// Deleting an original re-links its duplicates with the normal rule.
// Same address for all:
//
//	A (a@, phone 1)  original
//	B (a@, phone 2)  → A  (same email)
//	C (c@, phone 2)  → B  (same phone as B, nothing in common with A)
//	D (a@, phone 3)  → A  (same email)
//
// After deleting A: B has no older match (original), C stays → B, D → B.
func TestRepository_DeleteRelinksDuplicates(t *testing.T) {
	repo, ctx := testRepo(t)
	city := "Testort " + time.Now().Format("150405.000000")

	a, err := repo.CreateLead(ctx, lead("a@example.com", "040 111111", "Ringstraße", "1", "12345", city))
	if err != nil {
		t.Fatal(err)
	}
	b, err := repo.CreateLead(ctx, lead("a@example.com", "040 222222", "Ringstraße", "1", "12345", city))
	if err != nil {
		t.Fatal(err)
	}
	c, err := repo.CreateLead(ctx, lead("c@example.com", "040 222222", "Ringstraße", "1", "12345", city))
	if err != nil {
		t.Fatal(err)
	}
	d, err := repo.CreateLead(ctx, lead("a@example.com", "040 333333", "Ringstraße", "1", "12345", city))
	if err != nil {
		t.Fatal(err)
	}
	if *b.DuplicateOf != a.ID || *c.DuplicateOf != b.ID || *d.DuplicateOf != a.ID {
		t.Fatalf("setup: b→%v c→%v d→%v", *b.DuplicateOf, *c.DuplicateOf, *d.DuplicateOf)
	}

	if err := repo.DeleteLead(ctx, a.ID); err != nil {
		t.Fatalf("delete original: %v", err)
	}

	get := func(id string) LeadListItem {
		detail, err := repo.GetLead(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return detail.Lead
	}
	if got := get(b.ID); got.DuplicateOf != nil {
		t.Errorf("b should now be an original, duplicate_of = %v", *got.DuplicateOf)
	}
	if got := get(c.ID); got.DuplicateOf == nil || *got.DuplicateOf != b.ID {
		t.Errorf("c should still point to b, got %v", got.DuplicateOf)
	}
	if got := get(d.ID); got.DuplicateOf == nil || *got.DuplicateOf != b.ID {
		t.Errorf("d (same email as b) should now point to b, got %v", got.DuplicateOf)
	}

	if err := repo.DeleteLead(ctx, a.ID); err != ErrNotFound {
		t.Errorf("deleting twice: %v, want ErrNotFound", err)
	}
	for _, id := range []string{b.ID, c.ID, d.ID} {
		if err := repo.DeleteLead(ctx, id); err != nil {
			t.Errorf("cleanup %s: %v", id, err)
		}
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
		id := results[i].ID
		t.Cleanup(func() { _ = repo.DeleteLead(ctx, id) })
	}
	if originals != 1 {
		t.Fatalf("expected exactly 1 original, got %d", originals)
	}
}

// Placeholder contents are filled by the ad platform on click. A campaign
// link with "video_{creative}" must collect the leads whose content starts
// with "video_" and nothing else; a pure placeholder collects every lead of
// the campaign.
func TestRepository_CampaignMatchesPlaceholderContent(t *testing.T) {
	repo, ctx := testRepo(t)
	run := time.Now().Format("150405.000000")
	campaign := "placeholder-test-" + strings.ReplaceAll(run, ".", "-")

	newLead := func(email, content string) {
		l := lead(email, "040 "+email[:6], "Teststraße", "", "12345", "Testort "+run)
		l.UTMSource, l.UTMMedium, l.UTMCampaign, l.UTMContent = "google", "cpc", campaign, content
		saved, err := repo.CreateLead(ctx, l)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.DeleteLead(ctx, saved.ID) })
	}
	newLead("video1@example.com", "video_111")
	newLead("video2@example.com", "video_222")
	newLead("image1@example.com", "image_333")
	newLead("videox@example.com", "videoX") // "_" must not act as a wildcard
	newLead("nocont@example.com", "")

	newCampaign := func(content string) Campaign {
		c, err := repo.CreateCampaign(ctx, CampaignRequest{
			Name: "Test " + content, UTMSource: "google", UTMMedium: "cpc", UTMCampaign: campaign, UTMContent: content,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.DeleteCampaign(ctx, c.ID) })
		return c
	}
	for content, want := range map[string]int{
		"video_{creative}": 2,
		"image_{creative}": 1,
		"{creative}":       5,
		"{{ad.name}}":      5,
		"video_111":        1,
		"":                 5,
	} {
		if got := newCampaign(content).Leads; got != want {
			t.Errorf("content %q: %d leads, want %d", content, got, want)
		}
	}
}

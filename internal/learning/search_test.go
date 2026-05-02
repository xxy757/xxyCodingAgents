package learning

import "testing"

func TestSearchInsightsByPhaseAndQuery(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	mustAppend := func(e Entry) {
		t.Helper()
		if err := store.Append("demo", e); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}

	mustAppend(Entry{
		TS:         "2026-05-01T10:00:00Z",
		Skill:      "qa",
		Type:       "pitfall",
		Key:        "qa-screenshot",
		Insight:    "Use deterministic screenshot filename and verify file exists.",
		Confidence: 9,
	})
	mustAppend(Entry{
		TS:         "2026-05-01T11:00:00Z",
		Skill:      "review",
		Type:       "pattern",
		Key:        "review-diff",
		Insight:    "Check rollback path for every behavior change.",
		Confidence: 7,
	})

	searcher := NewSearcher(dir)
	got, err := searcher.SearchInsights(SearchOptions{
		ProjectSlug: "demo",
		Phase:       "browser-qa",
		QueryText:   "screenshot smoke check",
		Limit:       3,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected at least one insight")
	}
	if got[0] == "" {
		t.Fatalf("expected non-empty insight text")
	}
}

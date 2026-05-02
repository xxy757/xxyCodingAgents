package learning

import (
	"strings"
	"testing"
)

func TestStoreAppendAndReadAll(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	err := store.Append("my-project", Entry{
		Skill:      "qa",
		Type:       "pitfall",
		Key:        "screenshot-path",
		Insight:    "Always use absolute path for screenshot in smoke checks.",
		Confidence: 9,
		Source:     "observed",
		Files:      []string{"scripts/smoke-browse-qa.sh"},
	})
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}

	entries, err := store.ReadAll("my-project")
	if err != nil {
		t.Fatalf("read all failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].TS == "" {
		t.Fatalf("expected ts to be auto populated")
	}
	if entries[0].Confidence != 9 {
		t.Fatalf("expected confidence=9, got %d", entries[0].Confidence)
	}
	if entries[0].Insight == "" {
		t.Fatalf("expected insight to be preserved")
	}
}

func TestSanitizeSlug(t *testing.T) {
	got := SanitizeSlug("  HTTPS://GitHub.com/Org/My Repo.git  ")
	if got == "" {
		t.Fatalf("expected non-empty slug")
	}
	if strings.Contains(got, " ") {
		t.Fatalf("expected no spaces in slug, got %q", got)
	}
	if strings.Contains(got, "/") {
		t.Fatalf("expected no slash in slug, got %q", got)
	}
}

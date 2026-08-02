package domain_test

import (
	"testing"

	"github.com/abteilung6/assetagent/internal/domain"
)

func TestSystemCategorySlugs(t *testing.T) {
	want := []string{
		"income",
		"housing",
		"insurance",
		"mobility",
		"groceries",
		"leisure",
		"health",
		"saving_investing",
		"transfer",
		"other",
	}
	if len(domain.SystemCategorySlugs) != len(want) {
		t.Fatalf("len = %d, want %d", len(domain.SystemCategorySlugs), len(want))
	}
	seen := make(map[string]bool, len(domain.SystemCategorySlugs))
	for i, slug := range domain.SystemCategorySlugs {
		if slug != want[i] {
			t.Fatalf("slug[%d] = %q, want %q", i, slug, want[i])
		}
		if seen[slug] {
			t.Fatalf("duplicate slug %q", slug)
		}
		seen[slug] = true
	}
}

package classify_test

import (
	"testing"

	"github.com/abteilung6/assetagent/internal/classify"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

func TestSuggestCategory_transfer(t *testing.T) {
	slug, source, conf := classify.SuggestCategory(decimal.RequireFromString("-100"), "", true)
	if slug != "transfer" || source != domain.ClassificationSourceHeuristic || conf != domain.ClassificationConfidenceHigh {
		t.Fatalf("got %s %s %s", slug, source, conf)
	}
}

func TestSuggestCategory_paypalAndIncome(t *testing.T) {
	slug, source, conf := classify.SuggestCategory(decimal.RequireFromString("-12"), "PAYPAL", false)
	if slug != "other" || source != domain.ClassificationSourceMerchant || conf != domain.ClassificationConfidenceLow {
		t.Fatalf("paypal = %s %s %s", slug, source, conf)
	}
	slug, source, _ = classify.SuggestCategory(decimal.RequireFromString("2000"), "", false)
	if slug != "income" || source != domain.ClassificationSourceHeuristic {
		t.Fatalf("income = %s %s", slug, source)
	}
}

func TestMatchPattern_salaryAndHousing(t *testing.T) {
	rules := []classify.PatternRule{
		{Pattern: "SALARY", Slug: "income", Priority: 50, Confidence: "high"},
		{Pattern: "HAUSGELD", Slug: "housing", Priority: 50, Confidence: "high"},
		{Pattern: "REWE", Slug: "groceries", Priority: 40, Confidence: "high"},
	}

	match := classify.MatchPattern(
		"Azimo Stichting Third Party Account",
		"Aiven Deutschland GmbH/Salary 0726/Design Offices",
		rules,
	)
	if match == nil || match.Slug != "income" || match.Pattern != "SALARY" {
		t.Fatalf("salary match = %#v", match)
	}

	match = classify.MatchPattern(
		"WEG Hermann-Hesse-Str.",
		"50203.1700 Kathleen Moeller Hausgeld",
		rules,
	)
	if match == nil || match.Slug != "housing" {
		t.Fatalf("hausgeld match = %#v", match)
	}

	if !classify.ShouldAutoApply(match) {
		t.Fatal("housing high should auto-apply")
	}
}

func TestMatchPattern_priorityOrder(t *testing.T) {
	rules := []classify.PatternRule{
		{Pattern: "REWE", Slug: "groceries", Priority: 40, Confidence: "high"},
		{Pattern: "REWE SAGT", Slug: "other", Priority: 90, Confidence: "low"},
	}
	match := classify.MatchPattern("REWE SAGT DANKE", "", rules)
	if match == nil || match.Slug != "groceries" {
		t.Fatalf("want groceries first by priority, got %#v", match)
	}
}

func TestShouldAutoApply_rejectsOtherAndLow(t *testing.T) {
	if classify.ShouldAutoApply(&classify.PatternMatch{Slug: "other", Confidence: "high"}) {
		t.Fatal("other must not auto-apply")
	}
	if classify.ShouldAutoApply(&classify.PatternMatch{Slug: "leisure", Confidence: "low"}) {
		t.Fatal("low must not auto-apply")
	}
	if !classify.ShouldAutoApply(&classify.PatternMatch{Slug: "leisure", Confidence: "medium"}) {
		t.Fatal("medium leisure should auto-apply")
	}
}

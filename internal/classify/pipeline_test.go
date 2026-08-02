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

func TestSuggestCategory_merchantAndIncome(t *testing.T) {
	slug, source, _ := classify.SuggestCategory(decimal.RequireFromString("-12"), "REWE", false)
	if slug != "groceries" || source != domain.ClassificationSourceMerchant {
		t.Fatalf("rewe = %s %s", slug, source)
	}
	slug, source, _ = classify.SuggestCategory(decimal.RequireFromString("2000"), "", false)
	if slug != "income" || source != domain.ClassificationSourceHeuristic {
		t.Fatalf("income = %s %s", slug, source)
	}
}

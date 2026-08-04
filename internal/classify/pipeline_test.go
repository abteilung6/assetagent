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

func TestMatchPattern_telcoAndDining(t *testing.T) {
	rules := []classify.PatternRule{
		{Pattern: "TELEFONICA", Slug: "subscriptions", Priority: 50, Confidence: "high"},
		{Pattern: "TRATTORIA", Slug: "dining", Priority: 45, Confidence: "high"},
		{Pattern: "GA ", Slug: "cash_atm", Priority: 55, Confidence: "medium"},
		{Pattern: "MOONPAY", Slug: "saving_investing", Priority: 40, Confidence: "high"},
		{Pattern: "ENTGELT", Slug: "taxes_fees", Priority: 50, Confidence: "medium"},
	}

	match := classify.MatchPattern("Telefonica Germany GmbH + Co. OHG", "Rechnung", rules)
	if match == nil || match.Slug != "subscriptions" {
		t.Fatalf("telco = %#v", match)
	}

	match = classify.MatchPattern("Trattoria Pasta Degli//Berlin/DE/0", "Debitk", rules)
	if match == nil || match.Slug != "dining" {
		t.Fatalf("dining = %#v", match)
	}

	match = classify.MatchPattern("GA 7244/Alte Schoenhauser Strasse 45/Berlin/DE", "Fremdentgelt", rules)
	if match == nil || match.Slug != "cash_atm" {
		t.Fatalf("atm = %#v", match)
	}

	match = classify.MatchPattern("PayPal Europe", "MoonPay Technology Serv", rules)
	if match == nil || match.Slug != "saving_investing" {
		t.Fatalf("moonpay = %#v", match)
	}

	match = classify.MatchPattern("", "Entgeltabrechnung siehe Anlage", rules)
	if match == nil || match.Slug != "taxes_fees" {
		t.Fatalf("entgelt = %#v", match)
	}
}

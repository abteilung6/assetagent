package classify

import (
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

// SuggestCategory picks a system category slug for a booking without user rules.
func SuggestCategory(
	amount decimal.Decimal,
	merchantPattern string,
	isConfirmedTransfer bool,
) (slug, source, confidence string) {
	if isConfirmedTransfer {
		return "transfer", domain.ClassificationSourceHeuristic, domain.ClassificationConfidenceHigh
	}

	switch merchantPattern {
	case "REWE":
		return "groceries", domain.ClassificationSourceMerchant, domain.ClassificationConfidenceHigh
	case "NETFLIX", "AMAZON":
		return "leisure", domain.ClassificationSourceMerchant, domain.ClassificationConfidenceMedium
	case "PAYPAL":
		return "other", domain.ClassificationSourceMerchant, domain.ClassificationConfidenceLow
	}

	if amount.IsPositive() {
		return "income", domain.ClassificationSourceHeuristic, domain.ClassificationConfidenceMedium
	}
	if amount.IsNegative() {
		return "other", domain.ClassificationSourceUnresolved, domain.ClassificationConfidenceLow
	}
	return "other", domain.ClassificationSourceUnresolved, domain.ClassificationConfidenceLow
}

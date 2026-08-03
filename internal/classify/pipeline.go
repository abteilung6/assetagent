package classify

import (
	"strings"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

// PatternRule is a DB-backed keyword → category mapping.
type PatternRule struct {
	Pattern    string
	CategoryID string // unused in matcher; slug resolved by caller
	Slug       string
	Priority   int
	Confidence string
}

// PatternMatch is the first pattern that hits booking text.
type PatternMatch struct {
	Pattern    string
	Slug       string
	Confidence string
	Priority   int
}

// MatchPattern finds the highest-priority (lowest number) pattern substring
// in counterparty + purpose text. Rules must be sorted by priority ASC.
func MatchPattern(counterparty, purpose string, rules []PatternRule) *PatternMatch {
	text := strings.ToUpper(strings.TrimSpace(counterparty + " " + purpose))
	if text == "" || len(rules) == 0 {
		return nil
	}
	for _, rule := range rules {
		needle := strings.ToUpper(strings.TrimSpace(rule.Pattern))
		if needle == "" {
			continue
		}
		if strings.Contains(text, needle) {
			conf := rule.Confidence
			if conf == "" {
				conf = domain.ClassificationConfidenceHigh
			}
			return &PatternMatch{
				Pattern:    rule.Pattern,
				Slug:       rule.Slug,
				Confidence: conf,
				Priority:   rule.Priority,
			}
		}
	}
	return nil
}

// ShouldAutoApply reports whether a pattern match is safe for the magic button.
func ShouldAutoApply(match *PatternMatch) bool {
	if match == nil || match.Slug == "" || match.Slug == "other" {
		return false
	}
	switch match.Confidence {
	case domain.ClassificationConfidenceHigh:
		return true
	case domain.ClassificationConfidenceMedium:
		return match.Slug != "other"
	default:
		return false
	}
}

// SuggestCategory picks a fallback category when no user or pattern rule matched.
func SuggestCategory(
	amount decimal.Decimal,
	merchantPattern string,
	isConfirmedTransfer bool,
) (slug, source, confidence string) {
	if isConfirmedTransfer {
		return "transfer", domain.ClassificationSourceHeuristic, domain.ClassificationConfidenceHigh
	}

	// Thin built-in fallbacks only — keyword maps live in classification_rules.
	switch merchantPattern {
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

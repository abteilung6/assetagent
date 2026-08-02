package classify

import (
	"strings"
	"unicode"
)

// MerchantLabel is a normalized merchant identity derived from booking text.
type MerchantLabel struct {
	DisplayName string
	Pattern     string
}

// NormalizeMerchantLabel builds a stable merchant pattern from counterparty/purpose.
// Returns false when both inputs are empty after trimming.
func NormalizeMerchantLabel(counterparty, purpose string) (MerchantLabel, bool) {
	raw := strings.TrimSpace(counterparty)
	if raw == "" {
		raw = strings.TrimSpace(purpose)
	}
	if raw == "" {
		return MerchantLabel{}, false
	}

	pattern := canonicalizePattern(normalizeKey(raw))
	if pattern == "" {
		return MerchantLabel{}, false
	}

	return MerchantLabel{
		DisplayName: displayNameForPattern(pattern, raw),
		Pattern:     pattern,
	}, true
}

func normalizeKey(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	prevSpace := false
	for _, r := range strings.ToUpper(value) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevSpace = false
		case unicode.IsSpace(r) || r == '-' || r == '/' || r == '.' || r == ',':
			if !prevSpace && b.Len() > 0 {
				b.WriteByte(' ')
				prevSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func canonicalizePattern(pattern string) string {
	switch {
	case strings.Contains(pattern, "AMAZON"):
		return "AMAZON"
	case strings.HasPrefix(pattern, "PAYPAL"):
		return "PAYPAL"
	case strings.Contains(pattern, "REWE"):
		return "REWE"
	case strings.Contains(pattern, "NETFLIX"):
		return "NETFLIX"
	default:
		return pattern
	}
}

func displayNameForPattern(pattern, original string) string {
	switch pattern {
	case "AMAZON":
		return "Amazon"
	case "PAYPAL":
		return "PayPal"
	case "REWE":
		return "REWE"
	case "NETFLIX":
		return "Netflix"
	default:
		// Prefer a cleaned title-ish form of the original counterparty/purpose.
		words := strings.Fields(strings.ToLower(normalizeKey(original)))
		for i, w := range words {
			if w == "" {
				continue
			}
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
		name := strings.Join(words, " ")
		if name == "" {
			return pattern
		}
		return name
	}
}

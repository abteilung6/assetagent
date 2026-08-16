package domain

import "strings"

const (
	LocaleDE = "de"
	LocaleEN = "en"
)

// SupportedLocales are the UI languages the product ships.
var SupportedLocales = []string{LocaleDE, LocaleEN}

// IsSupportedLocale reports whether locale is an exact product locale tag.
func IsSupportedLocale(locale string) bool {
	switch locale {
	case LocaleDE, LocaleEN:
		return true
	default:
		return false
	}
}

// NormalizeLocale maps a BCP-47-ish tag (Google locale, navigator.language, …)
// onto a supported product locale. Unknown or empty values become LocaleDE.
func NormalizeLocale(raw string) string {
	tag := strings.ToLower(strings.TrimSpace(raw))
	if tag == "" {
		return LocaleDE
	}
	primary, _, _ := strings.Cut(tag, "-")
	primary, _, _ = strings.Cut(primary, "_")
	switch primary {
	case LocaleEN:
		return LocaleEN
	case LocaleDE:
		return LocaleDE
	default:
		return LocaleDE
	}
}

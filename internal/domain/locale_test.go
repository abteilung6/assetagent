package domain_test

import (
	"testing"

	"github.com/abteilung6/assetagent/internal/domain"
)

func TestNormalizeLocale(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"", domain.LocaleDE},
		{"  ", domain.LocaleDE},
		{"de", domain.LocaleDE},
		{"DE", domain.LocaleDE},
		{"de-DE", domain.LocaleDE},
		{"de_AT", domain.LocaleDE},
		{"en", domain.LocaleEN},
		{"en-US", domain.LocaleEN},
		{"en_GB", domain.LocaleEN},
		{"fr-FR", domain.LocaleDE},
		{"zh-CN", domain.LocaleDE},
	}
	for _, tc := range cases {
		if got := domain.NormalizeLocale(tc.in); got != tc.want {
			t.Fatalf("NormalizeLocale(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsSupportedLocale(t *testing.T) {
	t.Parallel()

	if !domain.IsSupportedLocale(domain.LocaleDE) || !domain.IsSupportedLocale(domain.LocaleEN) {
		t.Fatal("expected de and en to be supported")
	}
	if domain.IsSupportedLocale("de-DE") || domain.IsSupportedLocale("fr") {
		t.Fatal("expected only exact product tags")
	}
}

package domain_test

import (
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

func TestFingerprint_stable(t *testing.T) {
	tx := domain.Transaction{
		OrderAccount:      "DE89370400440532013000",
		BookingDate:       time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
		BookingText:       "FOLGELASTSCHRIFT",
		Purpose:           "Payment to Example Shop",
		Counterparty:      "PayPal Europe S.a.r.l. et Cie S.C.A",
		EndToEndReference: "E2E-UBER-001",
		Amount:            decimal.RequireFromString("-23.97"),
		Currency:          "EUR",
	}

	first := domain.Fingerprint(tx)
	second := domain.Fingerprint(tx)
	if first != second {
		t.Fatalf("fingerprint not stable: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("fingerprint length = %d, want 64", len(first))
	}
}

func TestFingerprint_normalizesOrderAccountCase(t *testing.T) {
	base := domain.Transaction{
		OrderAccount:      "de89370400440532013000",
		BookingDate:       time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
		BookingText:       "FOLGELASTSCHRIFT",
		Purpose:           "Payment",
		Counterparty:      "PayPal",
		EndToEndReference: "E2E-001",
		Amount:            decimal.RequireFromString("-1.00"),
		Currency:          "eur",
	}

	upper := base
	upper.OrderAccount = "DE89370400440532013000"
	upper.Currency = "EUR"

	if domain.Fingerprint(base) != domain.Fingerprint(upper) {
		t.Fatal("expected case-normalized fingerprint to match")
	}
}

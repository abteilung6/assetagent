package classify_test

import (
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/classify"
)

func TestNormalizeMerchantLabel_amazonVariants(t *testing.T) {
	a, ok := classify.NormalizeMerchantLabel("AMAZON DIGITAL GERMANY GMBH", "Prime Video")
	if !ok {
		t.Fatal("expected ok")
	}
	b, ok := classify.NormalizeMerchantLabel("Amazon Payments Europe S.C.A.", "")
	if !ok {
		t.Fatal("expected ok")
	}
	if a.Pattern != "AMAZON" || b.Pattern != "AMAZON" {
		t.Fatalf("patterns = %q / %q, want AMAZON", a.Pattern, b.Pattern)
	}
	if a.DisplayName != "Amazon" {
		t.Fatalf("display = %q", a.DisplayName)
	}
}

func TestNormalizeMerchantLabel_paypal(t *testing.T) {
	label, ok := classify.NormalizeMerchantLabel("PayPal Europe S.a.r.l. et Cie S.C.A", "Payment to Example Shop")
	if !ok || label.Pattern != "PAYPAL" {
		t.Fatalf("got %+v ok=%v", label, ok)
	}
}

func TestNormalizeMerchantLabel_empty(t *testing.T) {
	if _, ok := classify.NormalizeMerchantLabel("  ", ""); ok {
		t.Fatal("expected false")
	}
}

func TestNormalizeMerchantLabel_fallsBackToPurpose(t *testing.T) {
	label, ok := classify.NormalizeMerchantLabel("", "Netflix monthly subscription")
	if !ok || label.Pattern != "NETFLIX" {
		t.Fatalf("got %+v ok=%v", label, ok)
	}
}

func TestNormalizeMerchantLabel_hausgeldPurposeVariants(t *testing.T) {
	a, ok := classify.NormalizeMerchantLabel(
		"",
		"50203.1700 Kathleen Moeller Hausgeld 06.2026",
	)
	if !ok || a.Pattern != "HAUSGELD" {
		t.Fatalf("got %+v ok=%v", a, ok)
	}
	b, ok := classify.NormalizeMerchantLabel(
		"",
		"50203.1700 Kathleen Moeller Hausgeld 07.2026",
	)
	if !ok || b.Pattern != "HAUSGELD" {
		t.Fatalf("got %+v ok=%v", b, ok)
	}
	if a.Pattern != b.Pattern {
		t.Fatalf("patterns diverged: %q vs %q", a.Pattern, b.Pattern)
	}
}

func TestNormalizeMerchantLabel_wegCounterparty(t *testing.T) {
	a, ok := classify.NormalizeMerchantLabel(
		"WEG Hermann-Hesse-Str. 13-15, Guellweg 2",
		"50203.1700 Kathleen Moeller Hausgeld 06.2026",
	)
	if !ok {
		t.Fatal("expected ok")
	}
	b, ok := classify.NormalizeMerchantLabel(
		"WEG Hermann-Hesse-Str. 13-15, Guellweg 2",
		"50203.1700 Kathleen Moeller Hausgeld 07.2026",
	)
	if !ok {
		t.Fatal("expected ok")
	}
	if a.Pattern != b.Pattern {
		t.Fatalf("patterns diverged: %q vs %q", a.Pattern, b.Pattern)
	}
	if !strings.HasPrefix(a.Pattern, "WEG ") {
		t.Fatalf("pattern = %q, want WEG prefix", a.Pattern)
	}
}

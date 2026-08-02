package classify_test

import (
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

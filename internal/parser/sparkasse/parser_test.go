package sparkasse_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/parser/sparkasse"
	"github.com/shopspring/decimal"
)

func TestParse_headersOnly(t *testing.T) {
	f, err := os.Open("../../../testdata/sparkasse/headers_only.csv")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	transactions, err := sparkasse.Parse(f)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(transactions) != 0 {
		t.Fatalf("len(transactions) = %d, want 0", len(transactions))
	}
}

func TestParse_minimalFixture(t *testing.T) {
	f, err := os.Open("../../../testdata/sparkasse/minimal.csv")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	transactions, err := sparkasse.Parse(f)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(transactions) != 6 {
		t.Fatalf("len(transactions) = %d, want 6", len(transactions))
	}

	assertDecimalEqual(t, transactions[0].Amount, "-23.97")
	assertDecimalEqual(t, transactions[2].Amount, "56000.00")
	if transactions[2].EndToEndReference != "SALARY-REF-001" {
		t.Fatalf("salary end_to_end_reference = %q", transactions[2].EndToEndReference)
	}
	if transactions[3].EndToEndReference != "" {
		t.Fatalf("cash deposit end_to_end_reference = %q, want empty", transactions[3].EndToEndReference)
	}
	if transactions[3].CounterpartyIBAN == nil || *transactions[3].CounterpartyIBAN != "6011880043" {
		t.Fatal("expected cash deposit counterparty IBAN")
	}
	if transactions[1].MandateReference != "ygIP7jsttecE5OCLr)ncS1Y)kw7EQ4" {
		t.Fatalf("mandate_reference = %q", transactions[1].MandateReference)
	}
	if domain.Fingerprint(transactions[0]) == domain.Fingerprint(transactions[1]) {
		t.Fatal("expected different fingerprints for different transactions")
	}
}

func TestParseGermanDate_pivotYear(t *testing.T) {
	tests := []struct {
		in   string
		want time.Time
	}{
		{"30.12.25", time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC)},
		{"01.01.00", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"01.01.70", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			tx, err := sparkasse.Parse(strings.NewReader(minimalHeader() + "\n" +
				`"DE89370400440532013000";"` + tt.in + `";"` + tt.in + `";"KARTENZAHLUNG";"test";"";"";"";"";"";"";"";"";"";"-1,00";"EUR";"Umsatz gebucht"`))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !tx[0].BookingDate.Equal(tt.want) {
				t.Fatalf("booking_date = %v, want %v", tx[0].BookingDate, tt.want)
			}
		})
	}
}

func TestParse_invalidRowReportsLineNumber(t *testing.T) {
	csv := minimalHeader() + "\n" +
		`"DE89370400440532013000";"not-a-date";"30.12.25";"KARTENZAHLUNG";"test";"";"";"";"";"";"";"";"";"";"-1,00";"EUR";"Umsatz gebucht"`

	_, err := sparkasse.Parse(strings.NewReader(csv))
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("error = %q, want line number", err.Error())
	}
}

func TestParseLenient_collectsInvalidRows(t *testing.T) {
	csv := minimalHeader() + "\n" +
		`"DE89370400440532013000";"30.12.25";"30.12.25";"KARTENZAHLUNG";"ok";"";"";"";"";"";"";"Cafe";"DE90100900002868569037";"BEVODEBBXXX";"-11,50";"EUR";"Umsatz gebucht"` + "\n" +
		`"DE89370400440532013000";"not-a-date";"30.12.25";"KARTENZAHLUNG";"bad";"";"";"";"";"";"";"";"";"";"-1,00";"EUR";"Umsatz gebucht"` + "\n" +
		`"DE89370400440532013000";"29.12.25";"29.12.25";"LOHN  GEHALT";"salary";"";"";"";"";"";"";"Employer";"DE12500105154832912731";"INGDDEFFXXX";"100,00";"EUR";"Umsatz gebucht"`

	result, err := sparkasse.ParseLenient(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseLenient() error = %v", err)
	}
	if len(result.Transactions) != 2 {
		t.Fatalf("len(transactions) = %d, want 2", len(result.Transactions))
	}
	if len(result.Invalid) != 1 {
		t.Fatalf("len(invalid) = %d, want 1", len(result.Invalid))
	}
	if result.Invalid[0].Line != 3 {
		t.Fatalf("invalid line = %d, want 3", result.Invalid[0].Line)
	}
	if result.Invalid[0].Field != "booking_date" {
		t.Fatalf("invalid field = %q, want booking_date", result.Invalid[0].Field)
	}
	if len(result.SourceLines) != 2 || result.SourceLines[0] != 2 || result.SourceLines[1] != 4 {
		t.Fatalf("source lines = %v, want [2 4]", result.SourceLines)
	}
}

func TestParseLenient_mixedInvalidFixture(t *testing.T) {
	f, err := os.Open("../../../testdata/sparkasse/mixed_invalid.csv")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	result, err := sparkasse.ParseLenient(f)
	if err != nil {
		t.Fatalf("ParseLenient() error = %v", err)
	}
	if len(result.Transactions) != 2 {
		t.Fatalf("len(transactions) = %d, want 2", len(result.Transactions))
	}
	if len(result.Invalid) != 1 {
		t.Fatalf("len(invalid) = %d, want 1", len(result.Invalid))
	}
	if result.Invalid[0].Line != 3 || result.Invalid[0].Field != "booking_date" {
		t.Fatalf("invalid = %+v", result.Invalid[0])
	}
}

func TestParse_nullableCounterpartyFields(t *testing.T) {
	csv := minimalHeader() + "\n" +
		`"DE89370400440532013000";"30.12.25";"30.12.25";"ENTGELTABSCHLUSS";"fee";"";"";"";"";"";"";"";"";"";"-4,95";"EUR";"Umsatz gebucht"`

	transactions, err := sparkasse.Parse(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if transactions[0].CounterpartyIBAN != nil || transactions[0].CounterpartyBIC != nil {
		t.Fatal("expected nil counterparty iban/bic")
	}
}

func minimalHeader() string {
	return `"Auftragskonto";"Buchungstag";"Valutadatum";"Buchungstext";"Verwendungszweck";"Glaeubiger ID";"Mandatsreferenz";"Kundenreferenz (End-to-End)";"Sammlerreferenz";"Lastschrift Ursprungsbetrag";"Auslagenersatz Ruecklastschrift";"Beguenstigter/Zahlungspflichtiger";"Kontonummer/IBAN";"BIC (SWIFT-Code)";"Betrag";"Waehrung";"Info"`
}

func assertDecimalEqual(t *testing.T, got decimal.Decimal, want string) {
	t.Helper()

	expected := decimal.RequireFromString(want)
	if !got.Equal(expected) {
		t.Fatalf("amount = %s, want %s", got.StringFixed(2), expected.StringFixed(2))
	}
}

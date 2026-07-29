package sparkasse

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

const expectedColumns = 17

// ParserVersion is stored on import previews/runs for reproducibility.
const ParserVersion = "sparkasse-1"

var expectedHeader = []string{
	"Auftragskonto",
	"Buchungstag",
	"Valutadatum",
	"Buchungstext",
	"Verwendungszweck",
	"Glaeubiger ID",
	"Mandatsreferenz",
	"Kundenreferenz (End-to-End)",
	"Sammlerreferenz",
	"Lastschrift Ursprungsbetrag",
	"Auslagenersatz Ruecklastschrift",
	"Beguenstigter/Zahlungspflichtiger",
	"Kontonummer/IBAN",
	"BIC (SWIFT-Code)",
	"Betrag",
	"Waehrung",
	"Info",
}

func Parse(r io.Reader) ([]domain.Transaction, error) {
	result, err := parseRecords(r, true)
	if err != nil {
		return nil, err
	}
	return result.Transactions, nil
}

// ParseLenient returns valid transactions and per-row errors without failing the whole file.
// Hard failures (empty file, bad header, unreadable CSV) still return an error.
func ParseLenient(r io.Reader) (ParseResult, error) {
	return parseRecords(r, false)
}

type ParseResult struct {
	Transactions []domain.Transaction
	SourceLines  []int
	Invalid      []InvalidRow
}

type InvalidRow struct {
	Line    int
	Field   string
	Message string
}

func parseRecords(r io.Reader, strict bool) (ParseResult, error) {
	reader := csv.NewReader(r)
	reader.Comma = ';'
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return ParseResult{}, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return ParseResult{}, fmt.Errorf("csv: empty file")
	}

	if err := validateHeader(records[0]); err != nil {
		return ParseResult{}, err
	}

	out := ParseResult{
		Transactions: make([]domain.Transaction, 0, len(records)-1),
		SourceLines:  make([]int, 0, len(records)-1),
	}
	for i, record := range records[1:] {
		line := i + 2
		tx, field, err := parseRecordDetailed(record)
		if err != nil {
			if strict {
				return ParseResult{}, fmt.Errorf("line %d: %w", line, err)
			}
			out.Invalid = append(out.Invalid, InvalidRow{
				Line:    line,
				Field:   field,
				Message: err.Error(),
			})
			continue
		}
		out.Transactions = append(out.Transactions, tx)
		out.SourceLines = append(out.SourceLines, line)
	}

	return out, nil
}

func validateHeader(header []string) error {
	if len(header) != expectedColumns {
		return fmt.Errorf("csv: expected %d columns, got %d", expectedColumns, len(header))
	}

	for i, want := range expectedHeader {
		if strings.TrimSpace(header[i]) != want {
			return fmt.Errorf("csv: unexpected header column %d: got %q, want %q", i+1, header[i], want)
		}
	}

	return nil
}

func parseRecord(record []string) (domain.Transaction, error) {
	tx, _, err := parseRecordDetailed(record)
	return tx, err
}

func parseRecordDetailed(record []string) (domain.Transaction, string, error) {
	if len(record) != expectedColumns {
		return domain.Transaction{}, "columns", fmt.Errorf("expected %d fields, got %d", expectedColumns, len(record))
	}

	bookingDate, err := parseGermanDate(record[1])
	if err != nil {
		return domain.Transaction{}, "booking_date", fmt.Errorf("booking_date: %w", err)
	}

	valueDate, err := parseGermanDate(record[2])
	if err != nil {
		return domain.Transaction{}, "value_date", fmt.Errorf("value_date: %w", err)
	}

	amount, err := parseAmount(record[14])
	if err != nil {
		return domain.Transaction{}, "amount", fmt.Errorf("amount: %w", err)
	}

	return domain.Transaction{
		OrderAccount:                   strings.TrimSpace(record[0]),
		BookingDate:                    bookingDate,
		ValueDate:                      valueDate,
		BookingText:                    strings.TrimSpace(record[3]),
		Purpose:                        strings.TrimSpace(record[4]),
		CreditorID:                     strings.TrimSpace(record[5]),
		MandateReference:               strings.TrimSpace(record[6]),
		EndToEndReference:              strings.TrimSpace(record[7]),
		CollectionReference:            strings.TrimSpace(record[8]),
		DirectDebitOriginalAmount:      strings.TrimSpace(record[9]),
		ChargebackExpenseReimbursement: strings.TrimSpace(record[10]),
		Counterparty:                   strings.TrimSpace(record[11]),
		CounterpartyIBAN:               optionalString(record[12]),
		CounterpartyBIC:                optionalString(record[13]),
		Amount:                         amount,
		Currency:                       strings.TrimSpace(record[15]),
		Info:                           strings.TrimSpace(record[16]),
	}, "", nil
}

func parseGermanDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date %q", raw)
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q", raw)
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q", raw)
	}

	yy, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q", raw)
	}

	year := 1900 + yy
	if yy <= 69 {
		year = 2000 + yy
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func parseAmount(raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	return decimal.NewFromString(raw)
}

func optionalString(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return &raw
}

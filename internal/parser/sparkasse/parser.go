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
	reader := csv.NewReader(r)
	reader.Comma = ';'
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("csv: empty file")
	}

	if err := validateHeader(records[0]); err != nil {
		return nil, err
	}

	transactions := make([]domain.Transaction, 0, len(records)-1)
	for i, record := range records[1:] {
		tx, err := parseRecord(record)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+2, err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
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
	if len(record) != expectedColumns {
		return domain.Transaction{}, fmt.Errorf("expected %d fields, got %d", expectedColumns, len(record))
	}

	bookingDate, err := parseGermanDate(record[1])
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("booking_date: %w", err)
	}

	valueDate, err := parseGermanDate(record[2])
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("value_date: %w", err)
	}

	amount, err := parseAmount(record[14])
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("amount: %w", err)
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
	}, nil
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

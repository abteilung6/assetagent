package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
)

type fakeTransactionRepo struct {
	inserted     int
	duplicates   int
	err          error
	batchCalls   int
	lastBatchLen int
}

func (f *fakeTransactionRepo) BatchInsert(ctx context.Context, txs []domain.Transaction) (int, int, error) {
	f.batchCalls++
	f.lastBatchLen = len(txs)
	if f.err != nil {
		return 0, 0, f.err
	}
	return f.inserted, f.duplicates, nil
}

func TestImport_ImportFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sparkasse", "minimal.csv")

	repo := &fakeTransactionRepo{inserted: 6, duplicates: 0}
	importer := service.NewImport(repo)

	result, err := importer.ImportFile(context.Background(), path)
	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}
	if result.Rows != 6 {
		t.Fatalf("Rows = %d, want 6", result.Rows)
	}
	if result.Inserted != 6 {
		t.Fatalf("Inserted = %d, want 6", result.Inserted)
	}
	if result.Duplicates != 0 {
		t.Fatalf("Duplicates = %d, want 0", result.Duplicates)
	}
	if repo.batchCalls != 1 {
		t.Fatalf("BatchInsert calls = %d, want 1", repo.batchCalls)
	}
}

func TestImport_ImportFile_missingFile(t *testing.T) {
	importer := service.NewImport(&fakeTransactionRepo{})

	_, err := importer.ImportFile(context.Background(), "missing.csv")
	if err == nil {
		t.Fatal("ImportFile() error = nil, want error")
	}
}

func TestImport_ImportFile_repoError(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sparkasse", "minimal.csv")

	repo := &fakeTransactionRepo{err: os.ErrInvalid}
	importer := service.NewImport(repo)

	_, err := importer.ImportFile(context.Background(), path)
	if err == nil {
		t.Fatal("ImportFile() error = nil, want error")
	}
}

func TestPreviewFile_minimal(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sparkasse", "minimal.csv")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	wantHash := sha256Hex(data)

	preview, err := service.PreviewFile(path)
	if err != nil {
		t.Fatalf("PreviewFile() error = %v", err)
	}
	if preview.FileHash != wantHash {
		t.Fatalf("FileHash = %q, want %q", preview.FileHash, wantHash)
	}
	if preview.RowTotal != 6 || preview.RowValid != 6 || preview.RowInvalid != 0 {
		t.Fatalf("counts = total=%d valid=%d invalid=%d, want 6/6/0", preview.RowTotal, preview.RowValid, preview.RowInvalid)
	}
	if preview.PeriodFrom == nil || preview.PeriodTo == nil {
		t.Fatal("expected period bounds")
	}
	if preview.PeriodFrom.Format("2006-01-02") != "2025-12-17" {
		t.Fatalf("PeriodFrom = %s, want 2025-12-17", preview.PeriodFrom.Format("2006-01-02"))
	}
	if preview.PeriodTo.Format("2006-01-02") != "2025-12-30" {
		t.Fatalf("PeriodTo = %s, want 2025-12-30", preview.PeriodTo.Format("2006-01-02"))
	}
	if preview.SuggestedAccount != "DE89…3000" {
		t.Fatalf("SuggestedAccount = %q, want DE89…3000", preview.SuggestedAccount)
	}
	if len(preview.SampleRows) == 0 {
		t.Fatal("expected sample rows")
	}
	if preview.ParserName != "sparkasse" || preview.ParserVersion == "" {
		t.Fatalf("parser = %s %s", preview.ParserName, preview.ParserVersion)
	}
}

func TestPreviewFile_doesNotInsert(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sparkasse", "minimal.csv")
	repo := &fakeTransactionRepo{inserted: 6}

	// Preview is a package function and must not touch the repository.
	if _, err := service.PreviewFile(path); err != nil {
		t.Fatalf("PreviewFile() error = %v", err)
	}
	if repo.batchCalls != 0 {
		t.Fatalf("BatchInsert calls = %d, want 0", repo.batchCalls)
	}
}

func TestPreviewBytes_invalidRowCollected(t *testing.T) {
	header := `"Auftragskonto";"Buchungstag";"Valutadatum";"Buchungstext";"Verwendungszweck";"Glaeubiger ID";"Mandatsreferenz";"Kundenreferenz (End-to-End)";"Sammlerreferenz";"Lastschrift Ursprungsbetrag";"Auslagenersatz Ruecklastschrift";"Beguenstigter/Zahlungspflichtiger";"Kontonummer/IBAN";"BIC (SWIFT-Code)";"Betrag";"Waehrung";"Info"`
	csv := header + "\n" +
		`"DE89370400440532013000";"30.12.25";"30.12.25";"KARTENZAHLUNG";"ok";"";"";"";"";"";"";"Cafe";"DE90100900002868569037";"BEVODEBBXXX";"-11,50";"EUR";"Umsatz gebucht"` + "\n" +
		`"DE89370400440532013000";"bad-date";"30.12.25";"KARTENZAHLUNG";"broken";"";"";"";"";"";"";"";"";"";"-1,00";"EUR";"Umsatz gebucht"`

	preview, err := service.PreviewBytes([]byte(csv), "mixed.csv")
	if err != nil {
		t.Fatalf("PreviewBytes() error = %v", err)
	}
	if preview.RowValid != 1 || preview.RowInvalid != 1 || preview.RowTotal != 2 {
		t.Fatalf("counts = %+v", preview)
	}
	if len(preview.InvalidRows) != 1 || preview.InvalidRows[0].Line != 3 {
		t.Fatalf("invalid rows = %+v", preview.InvalidRows)
	}
	if preview.InvalidRows[0].Field != "booking_date" {
		t.Fatalf("field = %q, want booking_date", preview.InvalidRows[0].Field)
	}
}

func TestPreviewBytes_emptyFile(t *testing.T) {
	_, err := service.PreviewBytes([]byte("   "), "empty.csv")
	if err == nil {
		t.Fatal("PreviewBytes() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %q, want empty", err.Error())
	}
}

func TestPreviewFile_headersOnly(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sparkasse", "headers_only.csv")
	preview, err := service.PreviewFile(path)
	if err != nil {
		t.Fatalf("PreviewFile() error = %v", err)
	}
	if preview.RowTotal != 0 || preview.RowValid != 0 {
		t.Fatalf("counts = total=%d valid=%d, want 0", preview.RowTotal, preview.RowValid)
	}
	if preview.PeriodFrom != nil || preview.PeriodTo != nil {
		t.Fatal("expected no period for headers-only file")
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

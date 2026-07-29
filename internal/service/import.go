package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/parser/sparkasse"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const (
	previewSampleLimit  = 5
	previewInvalidLimit = 20
	parserNameSparkasse = "sparkasse"
)

type TransactionRepository interface {
	BatchInsert(ctx context.Context, txs []domain.Transaction) (inserted, duplicates int, err error)
}

type Import struct {
	repo TransactionRepository
}

func NewImport(repo TransactionRepository) *Import {
	return &Import{repo: repo}
}

// PreviewFile parses and validates a CSV without writing to the database.
func PreviewFile(path string) (domain.ImportPreview, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.ImportPreview{}, fmt.Errorf("open file: %w", err)
	}
	return PreviewBytes(data, filepath.Base(path))
}

// PreviewBytes builds an import preview from raw CSV bytes. It never inserts transactions.
func PreviewBytes(data []byte, sourceFilename string) (domain.ImportPreview, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return domain.ImportPreview{}, fmt.Errorf("csv: empty file")
	}

	parsed, err := sparkasse.ParseLenient(csvReader(data))
	if err != nil {
		return domain.ImportPreview{}, fmt.Errorf("parse csv: %w", err)
	}

	preview := domain.ImportPreview{
		FileHash:         hashBytes(data),
		SourceFilename:   sourceFilename,
		ParserName:       parserNameSparkasse,
		ParserVersion:    sparkasse.ParserVersion,
		RowTotal:         len(parsed.Transactions) + len(parsed.Invalid),
		RowValid:         len(parsed.Transactions),
		RowInvalid:       len(parsed.Invalid),
		SuggestedAccount: suggestAccount(parsed.Transactions),
		SampleRows:       sampleRows(parsed.Transactions, parsed.SourceLines, previewSampleLimit),
		InvalidRows:      mapInvalidRows(parsed.Invalid, previewInvalidLimit),
		Warnings:         previewWarnings(parsed.Transactions),
	}

	if from, to, ok := periodBounds(parsed.Transactions); ok {
		preview.PeriodFrom = &from
		preview.PeriodTo = &to
	}

	return preview, nil
}

func (s *Import) ImportFile(ctx context.Context, path string) (domain.ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("open file: %w", err)
	}

	transactions, err := sparkasse.Parse(csvReader(data))
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("parse csv: %w", err)
	}

	inserted, duplicates, err := s.repo.BatchInsert(ctx, transactions)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("batch insert: %w", err)
	}

	return domain.ImportResult{
		Rows:       len(transactions),
		Inserted:   inserted,
		Duplicates: duplicates,
	}, nil
}

func csvReader(data []byte) io.Reader {
	if utf8.Valid(data) {
		return bytes.NewReader(data)
	}
	return transform.NewReader(bytes.NewReader(data), charmap.ISO8859_1.NewDecoder())
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func suggestAccount(txs []domain.Transaction) string {
	if len(txs) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, tx := range txs {
		if tx.OrderAccount == "" {
			continue
		}
		counts[tx.OrderAccount]++
	}
	if len(counts) == 0 {
		return ""
	}

	best := ""
	bestCount := -1
	for account, count := range counts {
		if count > bestCount || (count == bestCount && account < best) {
			best = account
			bestCount = count
		}
	}
	return maskAccount(best)
}

func maskAccount(account string) string {
	account = strings.TrimSpace(account)
	if account == "" {
		return ""
	}
	if len(account) <= 8 {
		return account
	}
	return account[:4] + "…" + account[len(account)-4:]
}

func periodBounds(txs []domain.Transaction) (time.Time, time.Time, bool) {
	if len(txs) == 0 {
		return time.Time{}, time.Time{}, false
	}
	from, to := txs[0].BookingDate, txs[0].BookingDate
	for _, tx := range txs[1:] {
		if tx.BookingDate.Before(from) {
			from = tx.BookingDate
		}
		if tx.BookingDate.After(to) {
			to = tx.BookingDate
		}
	}
	return from, to, true
}

func sampleRows(txs []domain.Transaction, sourceLines []int, limit int) []domain.ImportPreviewRow {
	if limit <= 0 || len(txs) == 0 {
		return nil
	}
	n := limit
	if n > len(txs) {
		n = len(txs)
	}
	out := make([]domain.ImportPreviewRow, 0, n)
	for i := 0; i < n; i++ {
		tx := txs[i]
		line := i + 2
		if i < len(sourceLines) {
			line = sourceLines[i]
		}
		out = append(out, domain.ImportPreviewRow{
			Line:         line,
			BookingDate:  tx.BookingDate,
			Counterparty: tx.Counterparty,
			Purpose:      truncate(tx.Purpose, 80),
			Amount:       tx.Amount.StringFixed(2),
			Currency:     tx.Currency,
		})
	}
	return out
}

func mapInvalidRows(rows []sparkasse.InvalidRow, limit int) []domain.ImportInvalidRow {
	if len(rows) == 0 {
		return nil
	}
	n := len(rows)
	if limit > 0 && n > limit {
		n = limit
	}
	out := make([]domain.ImportInvalidRow, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.ImportInvalidRow{
			Line:    rows[i].Line,
			Field:   rows[i].Field,
			Message: rows[i].Message,
		})
	}
	return out
}

func previewWarnings(txs []domain.Transaction) []string {
	accounts := make(map[string]struct{})
	for _, tx := range txs {
		if tx.OrderAccount == "" {
			continue
		}
		accounts[tx.OrderAccount] = struct{}{}
	}
	if len(accounts) <= 1 {
		return nil
	}

	names := make([]string, 0, len(accounts))
	for account := range accounts {
		names = append(names, maskAccount(account))
	}
	sort.Strings(names)
	return []string{fmt.Sprintf("file contains %d distinct order accounts: %s", len(names), strings.Join(names, ", "))}
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "…"
}

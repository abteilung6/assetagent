package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abteilung6/assetagent/internal/domain"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/parser/sparkasse"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const (
	previewSampleLimit  = 5
	previewInvalidLimit = 20
	parserNameSparkasse = "sparkasse"
)

type Import struct {
	pool *pgxpool.Pool
}

func NewImport(pool *pgxpool.Pool) *Import {
	return &Import{pool: pool}
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

func (s *Import) ImportFile(ctx context.Context, path string, opts domain.ImportOptions) (domain.ImportResult, error) {
	if s == nil || s.pool == nil {
		return domain.ImportResult{}, fmt.Errorf("import service is not configured")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("open file: %w", err)
	}

	transactions, err := sparkasse.Parse(csvReader(data))
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("parse csv: %w", err)
	}

	return s.commitTransactions(ctx, data, filepath.Base(path), transactions, opts)
}

func (s *Import) commitTransactions(
	ctx context.Context,
	data []byte,
	sourceFilename string,
	transactions []domain.Transaction,
	opts domain.ImportOptions,
) (domain.ImportResult, error) {
	orderAccount := primaryOrderAccount(transactions)
	displayName := strings.TrimSpace(opts.AccountName)
	if displayName == "" {
		if orderAccount != "" {
			displayName = maskAccount(orderAccount)
		} else {
			displayName = "Account"
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := sqldb.New(tx)

	account, err := ensureAccount(ctx, q, orderAccount, displayName)
	if err != nil {
		return domain.ImportResult{}, err
	}

	from, to, _ := periodBounds(transactions)
	warnings := previewWarnings(transactions)
	if warnings == nil {
		warnings = []string{}
	}
	warningsJSON, err := json.Marshal(warnings)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("encode warnings: %w", err)
	}

	now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	run, err := q.CreateImportRun(ctx, sqldb.CreateImportRunParams{
		AccountID:      account.ID,
		SourceFilename: sourceFilename,
		FileHash:       hashBytes(data),
		ParserName:     parserNameSparkasse,
		ParserVersion:  sparkasse.ParserVersion,
		Status:         domain.ImportRunStatusCommitted,
		PeriodFrom:     dateFromTime(from),
		PeriodTo:       dateFromTime(to),
		RowTotal:       int32(len(transactions)),
		RowValid:       int32(len(transactions)),
		RowInvalid:     0,
		RowInserted:    0,
		RowDuplicate:   0,
		Warnings:       warningsJSON,
		CommittedAt:    now,
	})
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("create import run: %w", err)
	}

	accountUUID := pgtype.UUID{Bytes: account.ID, Valid: true}
	runUUID := pgtype.UUID{Bytes: run.ID, Valid: true}

	inserted, duplicates := 0, 0
	for _, row := range transactions {
		params := insertIfNewParams(row, domain.Fingerprint(row), accountUUID, runUUID)
		_, err := q.InsertTransactionIfNew(ctx, params)
		if errors.Is(err, pgx.ErrNoRows) {
			duplicates++
			continue
		}
		if err != nil {
			return domain.ImportResult{}, fmt.Errorf("batch insert: %w", err)
		}
		inserted++
	}

	run, err = q.UpdateImportRunCounts(ctx, sqldb.UpdateImportRunCountsParams{
		ID:           run.ID,
		RowInserted:  int32(inserted),
		RowDuplicate: int32(duplicates),
	})
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("update import run: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ImportResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return domain.ImportResult{
		Rows:        len(transactions),
		Inserted:    inserted,
		Duplicates:  duplicates,
		ImportRunID: run.ID,
		AccountID:   account.ID,
		AccountName: account.DisplayName,
	}, nil
}

func (s *Import) ListRuns(ctx context.Context, limit int) ([]domain.ImportRunSummary, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("import service is not configured")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := sqldb.New(s.pool).ListImportRuns(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("list import runs: %w", err)
	}

	out := make([]domain.ImportRunSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, importRunToSummary(row))
	}
	return out, nil
}

func (s *Import) Rollback(ctx context.Context, runID uuid.UUID) (domain.ImportRollbackResult, error) {
	if s == nil || s.pool == nil {
		return domain.ImportRollbackResult{}, fmt.Errorf("import service is not configured")
	}
	if runID == uuid.Nil {
		return domain.ImportRollbackResult{}, fmt.Errorf("%w: empty id", ErrImportRunNotFound)
	}

	q := sqldb.New(s.pool)
	run, err := q.GetImportRun(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ImportRollbackResult{}, ErrImportRunNotFound
		}
		return domain.ImportRollbackResult{}, fmt.Errorf("get import run: %w", err)
	}

	switch run.Status {
	case domain.ImportRunStatusRolledBack:
		return domain.ImportRollbackResult{}, ErrImportRunAlreadyRolledBack
	case domain.ImportRunStatusCommitted:
		// ok
	default:
		return domain.ImportRollbackResult{}, fmt.Errorf("%w: status %q", ErrImportRunNotCommitted, run.Status)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ImportRollbackResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tq := sqldb.New(tx)
	deleted, err := tq.DeleteTransactionsByImportRun(ctx, pgtype.UUID{Bytes: runID, Valid: true})
	if err != nil {
		return domain.ImportRollbackResult{}, fmt.Errorf("delete transactions: %w", err)
	}

	if _, err := tq.MarkImportRunRolledBack(ctx, runID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ImportRollbackResult{}, ErrImportRunAlreadyRolledBack
		}
		return domain.ImportRollbackResult{}, fmt.Errorf("mark rolled back: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ImportRollbackResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return domain.ImportRollbackResult{
		ImportRunID:    runID,
		Deleted:        deleted,
		SourceFilename: run.SourceFilename,
	}, nil
}

func importRunToSummary(row sqldb.ImportRun) domain.ImportRunSummary {
	summary := domain.ImportRunSummary{
		ID:             row.ID,
		AccountID:      row.AccountID,
		SourceFilename: row.SourceFilename,
		Status:         row.Status,
		RowTotal:       int(row.RowTotal),
		RowValid:       int(row.RowValid),
		RowInserted:    int(row.RowInserted),
		RowDuplicate:   int(row.RowDuplicate),
	}
	if row.CreatedAt.Valid {
		summary.CreatedAt = row.CreatedAt.Time
	}
	if row.CommittedAt.Valid {
		t := row.CommittedAt.Time
		summary.CommittedAt = &t
	}
	if row.RolledBackAt.Valid {
		t := row.RolledBackAt.Time
		summary.RolledBackAt = &t
	}
	return summary
}

func ensureAccount(
	ctx context.Context,
	q *sqldb.Queries,
	orderAccount string,
	displayName string,
) (sqldb.Account, error) {
	if orderAccount != "" {
		account, err := q.GetAccountByOrderAccount(ctx, pgtype.Text{String: orderAccount, Valid: true})
		if err == nil {
			return account, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return sqldb.Account{}, fmt.Errorf("get account: %w", err)
		}
	}

	account, err := q.CreateAccount(ctx, sqldb.CreateAccountParams{
		DisplayName:      displayName,
		Bank:             parserNameSparkasse,
		Currency:         "EUR",
		OrderAccount:     textFromString(orderAccount),
		MaskedIdentifier: maskAccount(orderAccount),
	})
	if err != nil {
		return sqldb.Account{}, fmt.Errorf("create account: %w", err)
	}
	return account, nil
}

func insertIfNewParams(
	tx domain.Transaction,
	fingerprint string,
	accountID pgtype.UUID,
	importRunID pgtype.UUID,
) sqldb.InsertTransactionIfNewParams {
	return sqldb.InsertTransactionIfNewParams{
		OrderAccount:                   tx.OrderAccount,
		BookingDate:                    pgtype.Date{Time: tx.BookingDate, Valid: true},
		ValueDate:                      pgtype.Date{Time: tx.ValueDate, Valid: true},
		BookingText:                    tx.BookingText,
		Purpose:                        tx.Purpose,
		CreditorID:                     tx.CreditorID,
		MandateReference:               tx.MandateReference,
		EndToEndReference:              tx.EndToEndReference,
		CollectionReference:            tx.CollectionReference,
		DirectDebitOriginalAmount:      tx.DirectDebitOriginalAmount,
		ChargebackExpenseReimbursement: tx.ChargebackExpenseReimbursement,
		Counterparty:                   tx.Counterparty,
		CounterpartyIban:               textFromPtr(tx.CounterpartyIBAN),
		CounterpartyBic:                textFromPtr(tx.CounterpartyBIC),
		Amount:                         tx.Amount,
		Currency:                       tx.Currency,
		Info:                           tx.Info,
		Fingerprint:                    fingerprint,
		AccountID:                      accountID,
		ImportRunID:                    importRunID,
	}
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
	return maskAccount(primaryOrderAccount(txs))
}

func primaryOrderAccount(txs []domain.Transaction) string {
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
	return best
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

func dateFromTime(value time.Time) pgtype.Date {
	if value.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: value, Valid: true}
}

func textFromString(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textFromPtr(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}
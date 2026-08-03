package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/review"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrMoneyReviewNotFound = errors.New("money review not found")
	ErrMoneyReviewNotOpen  = errors.New("money review cannot be confirmed")
	ErrBaselineRequired    = errors.New("a financial baseline is required to create a review")
)

const (
	MoneyReviewStatusDraft             = "draft"
	MoneyReviewStatusNeedsConfirmation = "needs_confirmation"
	MoneyReviewStatusConfirmed         = "confirmed"
	MoneyReviewStatusSuperseded        = "superseded"
)

// MoneyReview is a persisted monthly review artifact.
type MoneyReview struct {
	ID            uuid.UUID
	BaselineID    uuid.UUID
	PeriodFrom    time.Time
	PeriodTo      time.Time
	Status        string
	Summary       string
	Findings      []review.Finding
	DataFreshness string
	ConfirmedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// MoneyReviewService generates and persists Money Reviews.
type MoneyReviewService struct {
	pool     *pgxpool.Pool
	baseline *BaselineService
	reports  *repository.Reports
	txs      *repository.Transaction
	now      func() time.Time
}

func NewMoneyReview(pool *pgxpool.Pool) *MoneyReviewService {
	return &MoneyReviewService{
		pool:     pool,
		baseline: NewBaseline(pool),
		reports:  repository.NewReports(pool),
		txs:      repository.NewTransaction(pool),
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// Create generates a review pinned to the current baseline (or baselineID if set).
func (s *MoneyReviewService) Create(ctx context.Context, baselineID *uuid.UUID) (MoneyReview, error) {
	var baseline ComputedBaseline
	var err error
	if baselineID != nil {
		row, getErr := sqldb.New(s.pool).GetFinancialBaseline(ctx, *baselineID)
		if getErr != nil {
			if errors.Is(getErr, pgx.ErrNoRows) {
				return MoneyReview{}, ErrBaselineNotFound
			}
			return MoneyReview{}, getErr
		}
		baseline, err = mapBaselineRow(row)
		if err != nil {
			return MoneyReview{}, err
		}
	} else {
		baseline, err = s.baseline.Current(ctx)
		if err != nil {
			if errors.Is(err, ErrBaselineNone) {
				return MoneyReview{}, ErrBaselineRequired
			}
			return MoneyReview{}, err
		}
	}

	series, err := NewRecurring(s.pool).List(ctx)
	if err != nil {
		return MoneyReview{}, fmt.Errorf("list recurring: %w", err)
	}

	cf, err := s.reports.GetCashflowV2Evidence(ctx, baseline.PeriodFrom, baseline.PeriodTo)
	if err != nil {
		return MoneyReview{}, fmt.Errorf("cashflow evidence: %w", err)
	}

	large, err := s.largeExpenses(ctx, baseline.PeriodFrom, baseline.PeriodTo, cf.Expenses)
	if err != nil {
		return MoneyReview{}, err
	}

	needsReview, err := s.needsReviewCount(ctx)
	if err != nil {
		return MoneyReview{}, err
	}

	summary, findings := review.Generate(review.Input{
		PeriodFrom:              baseline.PeriodFrom,
		PeriodTo:                baseline.PeriodTo,
		SustainableFreeCashflow: baseline.SustainableFreeCashflow,
		Series:                  series,
		LargeExpenses:           large,
		NeedsReviewCount:        needsReview,
		DataFreshness:           cf.DataFreshness,
	})

	artifact := MoneyReview{
		BaselineID:    baseline.ID,
		PeriodFrom:    baseline.PeriodFrom,
		PeriodTo:      baseline.PeriodTo,
		Status:        MoneyReviewStatusNeedsConfirmation,
		Summary:       summary,
		Findings:      findings,
		DataFreshness: cf.DataFreshness,
	}
	return s.insert(ctx, artifact)
}

func (s *MoneyReviewService) Get(ctx context.Context, id uuid.UUID) (MoneyReview, error) {
	row, err := sqldb.New(s.pool).GetMoneyReview(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MoneyReview{}, ErrMoneyReviewNotFound
		}
		return MoneyReview{}, err
	}
	return mapMoneyReviewRow(row)
}

func (s *MoneyReviewService) List(ctx context.Context, limit int) ([]MoneyReview, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := sqldb.New(s.pool).ListMoneyReviews(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]MoneyReview, 0, len(rows))
	for _, row := range rows {
		item, err := mapMoneyReviewRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *MoneyReviewService) Confirm(ctx context.Context, id uuid.UUID) (MoneyReview, error) {
	q := sqldb.New(s.pool)
	row, err := q.ConfirmMoneyReview(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := q.GetMoneyReview(ctx, id)
			if errors.Is(getErr, pgx.ErrNoRows) {
				return MoneyReview{}, ErrMoneyReviewNotFound
			}
			if getErr != nil {
				return MoneyReview{}, getErr
			}
			if existing.Status == MoneyReviewStatusConfirmed {
				return mapMoneyReviewRow(existing)
			}
			return MoneyReview{}, ErrMoneyReviewNotOpen
		}
		return MoneyReview{}, err
	}
	return mapMoneyReviewRow(row)
}

func (s *MoneyReviewService) insert(ctx context.Context, artifact MoneyReview) (MoneyReview, error) {
	findingsJSON, err := findingsToJSON(artifact.Findings)
	if err != nil {
		return MoneyReview{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MoneyReview{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := sqldb.New(tx)

	if err := qtx.SupersedeOpenMoneyReviews(ctx); err != nil {
		return MoneyReview{}, fmt.Errorf("supersede: %w", err)
	}

	row, err := qtx.InsertMoneyReview(ctx, sqldb.InsertMoneyReviewParams{
		BaselineID:    artifact.BaselineID,
		PeriodFrom:    pgtype.Date{Time: dateOnlyUTC(artifact.PeriodFrom), Valid: true},
		PeriodTo:      pgtype.Date{Time: dateOnlyUTC(artifact.PeriodTo), Valid: true},
		Status:        artifact.Status,
		Summary:       artifact.Summary,
		Findings:      findingsJSON,
		DataFreshness: artifact.DataFreshness,
	})
	if err != nil {
		return MoneyReview{}, fmt.Errorf("insert review: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return MoneyReview{}, err
	}
	return mapMoneyReviewRow(row)
}

func (s *MoneyReviewService) largeExpenses(
	ctx context.Context,
	from, to time.Time,
	periodExpenses decimal.Decimal,
) ([]review.LargeExpense, error) {
	threshold := review.LargeExpenseThreshold(periodExpenses)
	listed, err := s.txs.List(ctx, domain.ListParams{
		Limit:    20,
		FromDate: &from,
		ToDate:   &to,
		Sort:     domain.SortAmount,
		SortAsc:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	out := make([]review.LargeExpense, 0)
	for _, tx := range listed.Transactions {
		if tx.OneOff {
			continue
		}
		if !tx.Amount.IsNegative() {
			continue
		}
		abs := tx.Amount.Abs()
		if abs.LessThan(threshold) {
			continue
		}
		label := tx.Counterparty
		if label == "" {
			label = tx.Purpose
		}
		if label == "" {
			label = tx.BookingText
		}
		if label == "" {
			label = "Transaction"
		}
		out = append(out, review.LargeExpense{
			TransactionID: tx.ID.String(),
			Label:         label,
			Amount:        abs,
		})
	}
	return out, nil
}

func (s *MoneyReviewService) needsReviewCount(ctx context.Context) (int, error) {
	transfers, err := NewTransfers(s.pool).ListCandidates(ctx)
	if err != nil {
		return 0, fmt.Errorf("transfer candidates: %w", err)
	}
	queue, err := NewClassify(s.pool).ListQueue(ctx)
	if err != nil {
		return 0, fmt.Errorf("classification queue: %w", err)
	}
	// Avoid Scan side-effect spam: count uncertain from list after lightweight query if possible.
	uncertain, err := sqldb.New(s.pool).ListUncertainRecurringSeries(ctx)
	if err != nil {
		return 0, fmt.Errorf("uncertain recurring: %w", err)
	}
	return len(transfers) + len(queue) + len(uncertain), nil
}

func findingsToJSON(findings []review.Finding) ([]byte, error) {
	type wire struct {
		Type               string   `json:"type"`
		Title              string   `json:"title"`
		Amount             *string  `json:"amount,omitempty"`
		PeriodFrom         string   `json:"period_from"`
		PeriodTo           string   `json:"period_to"`
		Confidence         string   `json:"confidence"`
		EvidenceIDs        []string `json:"evidence_ids"`
		SuggestedActionKey string   `json:"suggested_action_key,omitempty"`
	}
	out := make([]wire, len(findings))
	for i, f := range findings {
		ids := f.EvidenceIDs
		if ids == nil {
			ids = []string{}
		}
		item := wire{
			Type:               f.Type,
			Title:              f.Title,
			PeriodFrom:         f.PeriodFrom.Format("2006-01-02"),
			PeriodTo:           f.PeriodTo.Format("2006-01-02"),
			Confidence:         f.Confidence,
			EvidenceIDs:        ids,
			SuggestedActionKey: f.SuggestedActionKey,
		}
		if f.Amount != nil {
			s := f.Amount.StringFixed(2)
			item.Amount = &s
		}
		out[i] = item
	}
	return json.Marshal(out)
}

func mapMoneyReviewRow(row sqldb.MoneyReview) (MoneyReview, error) {
	findings, err := findingsFromJSON(row.Findings)
	if err != nil {
		return MoneyReview{}, err
	}
	out := MoneyReview{
		ID:            row.ID,
		BaselineID:    row.BaselineID,
		PeriodFrom:    row.PeriodFrom.Time,
		PeriodTo:      row.PeriodTo.Time,
		Status:        row.Status,
		Summary:       row.Summary,
		Findings:      findings,
		DataFreshness: row.DataFreshness,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
	if row.ConfirmedAt.Valid {
		t := row.ConfirmedAt.Time
		out.ConfirmedAt = &t
	}
	return out, nil
}

func findingsFromJSON(raw []byte) ([]review.Finding, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	type wire struct {
		Type               string   `json:"type"`
		Title              string   `json:"title"`
		Amount             *string  `json:"amount"`
		PeriodFrom         string   `json:"period_from"`
		PeriodTo           string   `json:"period_to"`
		Confidence         string   `json:"confidence"`
		EvidenceIDs        []string `json:"evidence_ids"`
		SuggestedActionKey string   `json:"suggested_action_key"`
	}
	var items []wire
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	out := make([]review.Finding, len(items))
	for i, item := range items {
		f := review.Finding{
			Type:               item.Type,
			Title:              item.Title,
			Confidence:         item.Confidence,
			EvidenceIDs:        item.EvidenceIDs,
			SuggestedActionKey: item.SuggestedActionKey,
		}
		if item.PeriodFrom != "" {
			if t, err := time.Parse("2006-01-02", item.PeriodFrom); err == nil {
				f.PeriodFrom = t
			}
		}
		if item.PeriodTo != "" {
			if t, err := time.Parse("2006-01-02", item.PeriodTo); err == nil {
				f.PeriodTo = t
			}
		}
		if item.Amount != nil {
			d, err := decimal.NewFromString(*item.Amount)
			if err != nil {
				return nil, err
			}
			f.Amount = &d
		}
		out[i] = f
	}
	return out, nil
}

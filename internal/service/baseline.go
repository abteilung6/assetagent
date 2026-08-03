package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/finance"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrBaselineNotFound      = errors.New("baseline not found")
	ErrBaselineNotDraft      = errors.New("baseline is not in draft status")
	ErrBaselineNotAdjustable = errors.New("baseline cannot be adjusted")
	ErrInvalidBaselineMetric = errors.New("invalid baseline metric key")
	ErrInvalidBaselineAmount = errors.New("invalid baseline amount")
	ErrEmptyAdjustReason     = errors.New("adjustment reason is required")
	ErrInvalidBaselinePeriod = errors.New("invalid baseline period")
	ErrBaselineNone          = errors.New("no baseline available")
)

const (
	BaselineStatusDraft      = "draft"
	BaselineStatusConfirmed  = "confirmed"
	BaselineStatusSuperseded = "superseded"
)

// BaselineService computes and persists FinancialBaseline snapshots.
type BaselineService struct {
	pool    *pgxpool.Pool
	reports *repository.Reports
	now     func() time.Time
}

func NewBaseline(pool *pgxpool.Pool) *BaselineService {
	return &BaselineService{
		pool:    pool,
		reports: repository.NewReports(pool),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// ComputedBaseline is the domain-facing baseline artifact.
type ComputedBaseline struct {
	ID                      uuid.UUID
	PeriodFrom              time.Time
	PeriodTo                time.Time
	AlgorithmVersion        string
	Status                  string
	RegularMonthlyIncome    decimal.Decimal
	MonthlyFixedCosts       decimal.Decimal
	MonthlyIrregularCosts   decimal.Decimal
	AvgVariableSpend        decimal.Decimal
	SustainableFreeCashflow decimal.Decimal
	Confidence              string
	Assumptions             []string
	Metrics                 []finance.MetricEvidence
	ConfirmedAt             *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Recompute derives a baseline without persisting (CLI preview).
func (s *BaselineService) Recompute(ctx context.Context, from, to *time.Time) (ComputedBaseline, error) {
	return s.compute(ctx, from, to)
}

// RecomputeAndSave computes a new draft baseline and supersedes prior open rows.
func (s *BaselineService) RecomputeAndSave(ctx context.Context, from, to *time.Time) (ComputedBaseline, error) {
	computed, err := s.compute(ctx, from, to)
	if err != nil {
		return ComputedBaseline{}, err
	}
	return s.insertDraft(ctx, computed)
}

// Current returns the preferred open baseline (confirmed over draft, newest).
func (s *BaselineService) Current(ctx context.Context) (ComputedBaseline, error) {
	row, err := sqldb.New(s.pool).GetCurrentFinancialBaseline(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ComputedBaseline{}, ErrBaselineNone
		}
		return ComputedBaseline{}, err
	}
	return mapBaselineRow(row)
}

// Confirm marks a draft baseline as confirmed (idempotent if already confirmed).
func (s *BaselineService) Confirm(ctx context.Context, id uuid.UUID) (ComputedBaseline, error) {
	q := sqldb.New(s.pool)
	row, err := q.ConfirmFinancialBaseline(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := q.GetFinancialBaseline(ctx, id)
			if errors.Is(getErr, pgx.ErrNoRows) {
				return ComputedBaseline{}, ErrBaselineNotFound
			}
			if getErr != nil {
				return ComputedBaseline{}, getErr
			}
			if existing.Status == BaselineStatusConfirmed {
				return mapBaselineRow(existing)
			}
			return ComputedBaseline{}, ErrBaselineNotDraft
		}
		return ComputedBaseline{}, err
	}
	return mapBaselineRow(row)
}

// Adjust creates a new draft baseline with one corrected metric and records the change.
func (s *BaselineService) Adjust(
	ctx context.Context,
	id uuid.UUID,
	metricKey string,
	newValue decimal.Decimal,
	reason string,
) (ComputedBaseline, error) {
	metricKey = strings.TrimSpace(metricKey)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ComputedBaseline{}, ErrEmptyAdjustReason
	}
	if !validMetricKey(metricKey) {
		return ComputedBaseline{}, ErrInvalidBaselineMetric
	}

	q := sqldb.New(s.pool)
	existing, err := q.GetFinancialBaseline(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ComputedBaseline{}, ErrBaselineNotFound
		}
		return ComputedBaseline{}, err
	}
	if existing.Status != BaselineStatusDraft && existing.Status != BaselineStatusConfirmed {
		return ComputedBaseline{}, ErrBaselineNotAdjustable
	}

	current, err := mapBaselineRow(existing)
	if err != nil {
		return ComputedBaseline{}, err
	}
	previous := metricValue(current, metricKey)
	adjusted, err := applyMetricAdjustment(current, metricKey, newValue)
	if err != nil {
		return ComputedBaseline{}, err
	}
	adjusted.Status = BaselineStatusDraft
	adjusted.ConfirmedAt = nil
	adjusted.Assumptions = append(append([]string{}, adjusted.Assumptions...),
		fmt.Sprintf("adjusted=%s", metricKey),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ComputedBaseline{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := sqldb.New(tx)

	if err := qtx.SupersedeOpenFinancialBaselines(ctx); err != nil {
		return ComputedBaseline{}, fmt.Errorf("supersede: %w", err)
	}

	saved, err := insertBaselineRow(ctx, qtx, adjusted)
	if err != nil {
		return ComputedBaseline{}, err
	}

	if _, err := qtx.InsertBaselineAdjustment(ctx, sqldb.InsertBaselineAdjustmentParams{
		BaselineID:    saved.ID,
		MetricKey:     metricKey,
		PreviousValue: previous,
		NewValue:      newValue,
		Reason:        reason,
	}); err != nil {
		return ComputedBaseline{}, fmt.Errorf("insert adjustment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ComputedBaseline{}, err
	}
	return saved, nil
}

func (s *BaselineService) insertDraft(ctx context.Context, computed ComputedBaseline) (ComputedBaseline, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ComputedBaseline{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := sqldb.New(tx)

	if err := qtx.SupersedeOpenFinancialBaselines(ctx); err != nil {
		return ComputedBaseline{}, fmt.Errorf("supersede: %w", err)
	}
	computed.Status = BaselineStatusDraft
	computed.ConfirmedAt = nil
	saved, err := insertBaselineRow(ctx, qtx, computed)
	if err != nil {
		return ComputedBaseline{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ComputedBaseline{}, err
	}
	return saved, nil
}

func insertBaselineRow(ctx context.Context, q *sqldb.Queries, b ComputedBaseline) (ComputedBaseline, error) {
	assumptions, err := json.Marshal(b.Assumptions)
	if err != nil {
		return ComputedBaseline{}, err
	}
	evidence, err := metricsToJSON(b.Metrics)
	if err != nil {
		return ComputedBaseline{}, err
	}
	row, err := q.InsertFinancialBaseline(ctx, sqldb.InsertFinancialBaselineParams{
		PeriodFrom:              pgtype.Date{Time: dateOnlyUTC(b.PeriodFrom), Valid: true},
		PeriodTo:                pgtype.Date{Time: dateOnlyUTC(b.PeriodTo), Valid: true},
		AlgorithmVersion:        b.AlgorithmVersion,
		Status:                  b.Status,
		RegularMonthlyIncome:    b.RegularMonthlyIncome,
		MonthlyFixedCosts:       b.MonthlyFixedCosts,
		MonthlyIrregularCosts:   b.MonthlyIrregularCosts,
		AvgVariableSpend:        b.AvgVariableSpend,
		SustainableFreeCashflow: b.SustainableFreeCashflow,
		Confidence:              b.Confidence,
		Assumptions:             assumptions,
		Evidence:                evidence,
	})
	if err != nil {
		return ComputedBaseline{}, fmt.Errorf("insert baseline: %w", err)
	}
	return mapBaselineRow(row)
}

func (s *BaselineService) compute(ctx context.Context, from, to *time.Time) (ComputedBaseline, error) {
	assumptions := []string{}
	var periodFrom, periodTo time.Time

	if from != nil && to != nil {
		periodFrom = dateOnlyUTC(*from)
		periodTo = dateOnlyUTC(*to)
		if periodTo.Before(periodFrom) {
			return ComputedBaseline{}, ErrInvalidBaselinePeriod
		}
		assumptions = append(assumptions, "period=explicit")
	} else if (from == nil) != (to == nil) {
		return ComputedBaseline{}, ErrInvalidBaselinePeriod
	} else {
		latest, err := s.latestBooking(ctx)
		if err != nil {
			return ComputedBaseline{}, err
		}
		var assumption string
		periodFrom, periodTo, assumption = finance.DefaultPeriod(latest, s.now())
		assumptions = append(assumptions, assumption)
	}

	series, err := NewRecurring(s.pool).List(ctx)
	if err != nil {
		return ComputedBaseline{}, fmt.Errorf("list recurring: %w", err)
	}

	cf, err := s.reports.GetCashflowV2(ctx, periodFrom, periodTo)
	if err != nil {
		return ComputedBaseline{}, fmt.Errorf("cashflow v2: %w", err)
	}

	raw := finance.Compute(finance.Input{
		PeriodFrom:      periodFrom,
		PeriodTo:        periodTo,
		Series:          series,
		CashflowExpense: cf.Expenses,
		Assumptions:     assumptions,
	})

	now := s.now()
	return ComputedBaseline{
		PeriodFrom:              raw.PeriodFrom,
		PeriodTo:                raw.PeriodTo,
		AlgorithmVersion:        raw.AlgorithmVersion,
		Status:                  BaselineStatusDraft,
		RegularMonthlyIncome:    raw.RegularMonthlyIncome,
		MonthlyFixedCosts:       raw.MonthlyFixedCosts,
		MonthlyIrregularCosts:   raw.MonthlyIrregularCosts,
		AvgVariableSpend:        raw.AvgVariableSpend,
		SustainableFreeCashflow: raw.SustainableFreeCashflow,
		Confidence:              raw.Confidence,
		Assumptions:             raw.Assumptions,
		Metrics:                 raw.Metrics,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}

func (s *BaselineService) latestBooking(ctx context.Context) (time.Time, error) {
	latest, err := sqldb.New(s.pool).GetLatestBookingDate(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("latest booking: %w", err)
	}
	if !latest.Valid {
		return time.Time{}, nil
	}
	return latest.Time, nil
}

func dateOnlyUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func mapBaselineRow(row sqldb.FinancialBaseline) (ComputedBaseline, error) {
	assumptions, err := unmarshalStringSlice(row.Assumptions)
	if err != nil {
		return ComputedBaseline{}, fmt.Errorf("assumptions: %w", err)
	}
	metrics, err := unmarshalMetrics(row.Evidence)
	if err != nil {
		return ComputedBaseline{}, fmt.Errorf("evidence: %w", err)
	}
	out := ComputedBaseline{
		ID:                      row.ID,
		PeriodFrom:              row.PeriodFrom.Time,
		PeriodTo:                row.PeriodTo.Time,
		AlgorithmVersion:        row.AlgorithmVersion,
		Status:                  row.Status,
		RegularMonthlyIncome:    row.RegularMonthlyIncome,
		MonthlyFixedCosts:       row.MonthlyFixedCosts,
		MonthlyIrregularCosts:   row.MonthlyIrregularCosts,
		AvgVariableSpend:        row.AvgVariableSpend,
		SustainableFreeCashflow: row.SustainableFreeCashflow,
		Confidence:              row.Confidence,
		Assumptions:             assumptions,
		Metrics:                 metrics,
		CreatedAt:               row.CreatedAt.Time,
		UpdatedAt:               row.UpdatedAt.Time,
	}
	if row.ConfirmedAt.Valid {
		t := row.ConfirmedAt.Time
		out.ConfirmedAt = &t
	}
	return out, nil
}

func unmarshalStringSlice(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func unmarshalMetrics(raw []byte) ([]finance.MetricEvidence, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	type wireMetric struct {
		Key         string          `json:"key"`
		Value       json.RawMessage `json:"value"`
		Calculation string          `json:"calculation"`
		Confidence  string          `json:"confidence"`
		EvidenceIDs []string        `json:"evidence_ids"`
		Assumptions []string        `json:"assumptions"`
	}
	var wire []wireMetric
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, err
	}
	out := make([]finance.MetricEvidence, len(wire))
	for i, m := range wire {
		val, err := parseDecimalJSON(m.Value)
		if err != nil {
			return nil, fmt.Errorf("metric %s value: %w", m.Key, err)
		}
		out[i] = finance.MetricEvidence{
			Key:         m.Key,
			Value:       val,
			Calculation: m.Calculation,
			Confidence:  m.Confidence,
			EvidenceIDs: m.EvidenceIDs,
			Assumptions: m.Assumptions,
		}
	}
	return out, nil
}

func parseDecimalJSON(raw json.RawMessage) (decimal.Decimal, error) {
	if len(raw) == 0 {
		return decimal.Zero, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return decimal.NewFromString(asString)
	}
	var asNumber json.Number
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return decimal.NewFromString(asNumber.String())
	}
	return decimal.Zero, fmt.Errorf("unsupported decimal json %s", string(raw))
}

func metricsToJSON(metrics []finance.MetricEvidence) ([]byte, error) {
	type wireMetric struct {
		Key         string   `json:"key"`
		Value       string   `json:"value"`
		Calculation string   `json:"calculation"`
		Confidence  string   `json:"confidence"`
		EvidenceIDs []string `json:"evidence_ids"`
		Assumptions []string `json:"assumptions,omitempty"`
	}
	wire := make([]wireMetric, len(metrics))
	for i, m := range metrics {
		ids := m.EvidenceIDs
		if ids == nil {
			ids = []string{}
		}
		wire[i] = wireMetric{
			Key:         m.Key,
			Value:       m.Value.StringFixed(2),
			Calculation: m.Calculation,
			Confidence:  m.Confidence,
			EvidenceIDs: ids,
			Assumptions: m.Assumptions,
		}
	}
	return json.Marshal(wire)
}

func validMetricKey(key string) bool {
	switch key {
	case finance.MetricRegularMonthlyIncome,
		finance.MetricMonthlyFixedCosts,
		finance.MetricMonthlyIrregularCosts,
		finance.MetricAvgVariableSpend,
		finance.MetricSustainableFreeCash:
		return true
	default:
		return false
	}
}

func metricValue(b ComputedBaseline, key string) decimal.Decimal {
	switch key {
	case finance.MetricRegularMonthlyIncome:
		return b.RegularMonthlyIncome
	case finance.MetricMonthlyFixedCosts:
		return b.MonthlyFixedCosts
	case finance.MetricMonthlyIrregularCosts:
		return b.MonthlyIrregularCosts
	case finance.MetricAvgVariableSpend:
		return b.AvgVariableSpend
	case finance.MetricSustainableFreeCash:
		return b.SustainableFreeCashflow
	default:
		return decimal.Zero
	}
}

func applyMetricAdjustment(b ComputedBaseline, key string, newValue decimal.Decimal) (ComputedBaseline, error) {
	switch key {
	case finance.MetricRegularMonthlyIncome:
		b.RegularMonthlyIncome = newValue
	case finance.MetricMonthlyFixedCosts:
		b.MonthlyFixedCosts = newValue
	case finance.MetricMonthlyIrregularCosts:
		b.MonthlyIrregularCosts = newValue
	case finance.MetricAvgVariableSpend:
		b.AvgVariableSpend = newValue
	case finance.MetricSustainableFreeCash:
		b.SustainableFreeCashflow = newValue
		syncMetricValue(&b, key, newValue)
		return b, nil
	default:
		return ComputedBaseline{}, ErrInvalidBaselineMetric
	}

	b.SustainableFreeCashflow = b.RegularMonthlyIncome.
		Sub(b.MonthlyFixedCosts).
		Sub(b.MonthlyIrregularCosts).
		Sub(b.AvgVariableSpend).
		Round(2)
	syncMetricValue(&b, key, newValue)
	syncMetricValue(&b, finance.MetricSustainableFreeCash, b.SustainableFreeCashflow)
	return b, nil
}

func syncMetricValue(b *ComputedBaseline, key string, value decimal.Decimal) {
	for i := range b.Metrics {
		if b.Metrics[i].Key == key {
			b.Metrics[i].Value = value
			return
		}
	}
}

// MonthlyCashflowPoint is one calendar month for baseline charts.
type MonthlyCashflowPoint struct {
	MonthStart time.Time
	Income     decimal.Decimal
	Expenses   decimal.Decimal
	Net        decimal.Decimal
}

// MonthlyCashflow returns transfer-aware totals for the last n calendar months
// that contain bookings (ending at the latest booking month).
func (s *BaselineService) MonthlyCashflow(ctx context.Context, months int) ([]MonthlyCashflowPoint, error) {
	if months < 2 {
		months = 6
	}
	if months > 12 {
		months = 12
	}
	latest, err := sqldb.New(s.pool).GetLatestBookingDate(ctx)
	if err != nil {
		return nil, err
	}
	if !latest.Valid {
		return []MonthlyCashflowPoint{}, nil
	}
	end := dateOnlyUTC(latest.Time)
	// Inclusive window: first day of (endMonth - (months-1)) through end of endMonth.
	endMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	startMonth := endMonth.AddDate(0, -(months - 1), 0)
	lastDay := endMonth.AddDate(0, 1, -1)
	if end.Before(lastDay) {
		lastDay = end
	}

	rows, err := s.reports.ListMonthlyCashflowV2(ctx, startMonth, lastDay)
	if err != nil {
		return nil, err
	}
	out := make([]MonthlyCashflowPoint, len(rows))
	for i, row := range rows {
		out[i] = MonthlyCashflowPoint{
			MonthStart: dateOnlyUTC(row.MonthStart),
			Income:     row.Income,
			Expenses:   row.Expenses,
			Net:        row.Net,
		}
	}
	return out, nil
}

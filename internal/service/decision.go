package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/decisions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrDecisionNotFound      = errors.New("decision not found")
	ErrActionNotFound        = errors.New("action not found")
	ErrInvalidDecision       = errors.New("invalid decision")
	ErrInvalidActionStatus   = errors.New("invalid action status")
	ErrActionStatusForbidden = errors.New("action status transition not allowed")
	ErrScenarioNotFound      = errors.New("scenario not found")
)

// Decision is a persisted choice tied to a review and/or scenario.
type Decision struct {
	ID          uuid.UUID
	ReviewID    *uuid.UUID
	ScenarioID  *uuid.UUID
	Title       string
	Assumptions map[string]any
	TargetValue *decimal.Decimal
	DecidedAt   time.Time
	CreatedAt   time.Time
	Action      Action
}

// Action is the measurable follow-up for a decision.
type Action struct {
	ID                   uuid.UUID
	DecisionID           uuid.UUID
	Title                string
	ExpectedAnnualEffect decimal.Decimal
	DueOn                time.Time
	Status               string
	OutcomeNote          string
	VerifiedAt           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CreateDecisionRequest creates a decision with exactly one action.
type CreateDecisionRequest struct {
	ReviewID    *uuid.UUID
	ScenarioID  *uuid.UUID
	Title       string
	Assumptions map[string]any
	TargetValue *decimal.Decimal
	Action      CreateActionRequest
}

// CreateActionRequest is the action payload on create.
type CreateActionRequest struct {
	Title                string
	ExpectedAnnualEffect decimal.Decimal
	DueOn                time.Time
}

// UpdateActionStatusRequest updates an action lifecycle status.
type UpdateActionStatusRequest struct {
	Status      string
	OutcomeNote string
}

// DecisionService persists decisions and actions.
type DecisionService struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewDecision(pool *pgxpool.Pool) *DecisionService {
	return &DecisionService{
		pool: pool,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *DecisionService) Create(ctx context.Context, req CreateDecisionRequest) (Decision, error) {
	if req.Title == "" {
		return Decision{}, fmt.Errorf("%w: title is required", ErrInvalidDecision)
	}
	if req.Action.Title == "" {
		return Decision{}, fmt.Errorf("%w: action title is required", ErrInvalidDecision)
	}
	if req.ReviewID == nil && req.ScenarioID == nil {
		return Decision{}, fmt.Errorf("%w: review_id or scenario_id is required", ErrInvalidDecision)
	}
	if req.Action.DueOn.IsZero() {
		return Decision{}, fmt.Errorf("%w: action due_on is required", ErrInvalidDecision)
	}

	q := sqldb.New(s.pool)
	if req.ReviewID != nil {
		if _, err := q.GetMoneyReview(ctx, *req.ReviewID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Decision{}, ErrMoneyReviewNotFound
			}
			return Decision{}, err
		}
	}
	if req.ScenarioID != nil {
		if _, err := q.GetScenario(ctx, *req.ScenarioID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Decision{}, ErrScenarioNotFound
			}
			return Decision{}, err
		}
	}

	assumptions := req.Assumptions
	if assumptions == nil {
		assumptions = map[string]any{}
	}
	assumptionsJSON, err := json.Marshal(assumptions)
	if err != nil {
		return Decision{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Decision{}, err
	}
	defer tx.Rollback(ctx)
	qtx := q.WithTx(tx)

	drow, err := qtx.InsertDecision(ctx, sqldb.InsertDecisionParams{
		ReviewID:    optionalUUID(req.ReviewID),
		ScenarioID:  optionalUUID(req.ScenarioID),
		Title:       req.Title,
		Assumptions: assumptionsJSON,
		TargetValue: optionalNumeric(req.TargetValue),
	})
	if err != nil {
		return Decision{}, fmt.Errorf("insert decision: %w", err)
	}

	arow, err := qtx.InsertAction(ctx, sqldb.InsertActionParams{
		DecisionID:           drow.ID,
		Title:                req.Action.Title,
		ExpectedAnnualEffect: req.Action.ExpectedAnnualEffect,
		DueOn:                pgtype.Date{Time: dateOnlyUTC(req.Action.DueOn), Valid: true},
		Status:               decisions.StatusPlanned,
		OutcomeNote:          "",
	})
	if err != nil {
		return Decision{}, fmt.Errorf("insert action: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Decision{}, err
	}

	return mapDecision(drow, arow)
}

func (s *DecisionService) List(ctx context.Context, limit int) ([]Decision, error) {
	if limit <= 0 {
		limit = 50
	}
	q := sqldb.New(s.pool)
	rows, err := q.ListDecisions(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]Decision, 0, len(rows))
	for _, row := range rows {
		actions, err := q.ListActionsForDecision(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if len(actions) == 0 {
			continue
		}
		d, err := mapDecision(row, actions[0])
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *DecisionService) ListActions(ctx context.Context, status *string, limit int) ([]Action, error) {
	if limit <= 0 {
		limit = 50
	}
	var statusArg pgtype.Text
	if status != nil {
		if err := decisions.ValidateStatus(*status); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidActionStatus, err)
		}
		statusArg = pgtype.Text{String: *status, Valid: true}
	}
	rows, err := sqldb.New(s.pool).ListActions(ctx, sqldb.ListActionsParams{
		Status:   statusArg,
		RowLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Action, len(rows))
	for i, row := range rows {
		a, err := mapAction(row)
		if err != nil {
			return nil, err
		}
		out[i] = a
	}
	return out, nil
}

func (s *DecisionService) UpdateActionStatus(ctx context.Context, id uuid.UUID, req UpdateActionStatusRequest) (Action, error) {
	if err := decisions.ValidateStatus(req.Status); err != nil {
		return Action{}, fmt.Errorf("%w: %v", ErrInvalidActionStatus, err)
	}
	q := sqldb.New(s.pool)
	row, err := q.GetAction(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Action{}, ErrActionNotFound
		}
		return Action{}, err
	}
	if !decisions.CanTransition(row.Status, req.Status) {
		return Action{}, ErrActionStatusForbidden
	}

	var verified pgtype.Timestamptz
	if req.Status == decisions.StatusDone || req.Status == decisions.StatusSkipped {
		verified = pgtype.Timestamptz{Time: s.now(), Valid: true}
	} else if row.VerifiedAt.Valid {
		verified = row.VerifiedAt
	}

	note := req.OutcomeNote
	if note == "" {
		note = row.OutcomeNote
	}

	updated, err := q.UpdateActionStatus(ctx, sqldb.UpdateActionStatusParams{
		ID:          id,
		Status:      req.Status,
		OutcomeNote: note,
		VerifiedAt:  verified,
	})
	if err != nil {
		return Action{}, err
	}
	return mapAction(updated)
}

func mapDecision(d sqldb.Decision, a sqldb.Action) (Decision, error) {
	assumptions := map[string]any{}
	if len(d.Assumptions) > 0 {
		if err := json.Unmarshal(d.Assumptions, &assumptions); err != nil {
			return Decision{}, err
		}
	}
	action, err := mapAction(a)
	if err != nil {
		return Decision{}, err
	}
	out := Decision{
		ID:          d.ID,
		ReviewID:    uuidPtr(d.ReviewID),
		ScenarioID:  uuidPtr(d.ScenarioID),
		Title:       d.Title,
		Assumptions: assumptions,
		DecidedAt:   d.DecidedAt.Time,
		CreatedAt:   d.CreatedAt.Time,
		Action:      action,
	}
	if d.TargetValue.Valid {
		v, err := decimalFromNumeric(d.TargetValue)
		if err != nil {
			return Decision{}, err
		}
		out.TargetValue = &v
	}
	return out, nil
}

func mapAction(a sqldb.Action) (Action, error) {
	out := Action{
		ID:                   a.ID,
		DecisionID:           a.DecisionID,
		Title:                a.Title,
		ExpectedAnnualEffect: a.ExpectedAnnualEffect,
		Status:               a.Status,
		OutcomeNote:          a.OutcomeNote,
		CreatedAt:            a.CreatedAt.Time,
		UpdatedAt:            a.UpdatedAt.Time,
	}
	if a.DueOn.Valid {
		out.DueOn = a.DueOn.Time
	}
	if a.VerifiedAt.Valid {
		t := a.VerifiedAt.Time
		out.VerifiedAt = &t
	}
	return out, nil
}

func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func uuidPtr(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	v := uuid.UUID(id.Bytes)
	return &v
}

func optionalNumeric(v *decimal.Decimal) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(v.StringFixed(2))
	return n
}

func decimalFromNumeric(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, nil
	}
	f, err := n.Float64Value()
	if err != nil {
		return decimal.Zero, err
	}
	if !f.Valid {
		return decimal.Zero, nil
	}
	return decimal.NewFromFloat(f.Float64).Round(2), nil
}

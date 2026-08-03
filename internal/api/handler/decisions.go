package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
)

type DecisionService interface {
	Create(ctx context.Context, req service.CreateDecisionRequest) (service.Decision, error)
	List(ctx context.Context, limit int) ([]service.Decision, error)
	ListActions(ctx context.Context, status *string, limit int) ([]service.Action, error)
	UpdateActionStatus(ctx context.Context, id uuid.UUID, req service.UpdateActionStatusRequest) (service.Action, error)
}

func (h *Handler) GetDecisions(w http.ResponseWriter, r *http.Request, params gen.GetDecisionsParams) {
	if h.decision == nil {
		writeInternalError(w, "decision service is not configured")
		return
	}
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	items, err := h.decision.List(r.Context(), limit)
	if err != nil {
		writeInternalError(w, "failed to list decisions")
		return
	}
	data := make([]gen.Decision, len(items))
	for i, item := range items {
		data[i] = toAPIDecision(item)
	}
	writeJSON(w, http.StatusOK, gen.DecisionListResponse{Data: data})
}

func (h *Handler) PostDecisions(w http.ResponseWriter, r *http.Request) {
	if h.decision == nil {
		writeInternalError(w, "decision service is not configured")
		return
	}
	var body gen.DecisionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	effect, err := decimal.NewFromString(body.Action.ExpectedAnnualEffect)
	if err != nil {
		writeValidationError(w, "invalid expected_annual_effect")
		return
	}
	req := service.CreateDecisionRequest{
		Title: body.Title,
		Action: service.CreateActionRequest{
			Title:                body.Action.Title,
			ExpectedAnnualEffect: effect,
			DueOn:                body.Action.DueOn.Time,
		},
	}
	if body.ReviewId != nil {
		id := uuid.UUID(*body.ReviewId)
		req.ReviewID = &id
	}
	if body.ScenarioId != nil {
		id := uuid.UUID(*body.ScenarioId)
		req.ScenarioID = &id
	}
	if body.Assumptions != nil {
		req.Assumptions = *body.Assumptions
	}
	if body.TargetValue != nil && *body.TargetValue != "" {
		v, err := decimal.NewFromString(*body.TargetValue)
		if err != nil {
			writeValidationError(w, "invalid target_value")
			return
		}
		req.TargetValue = &v
	}

	item, err := h.decision.Create(r.Context(), req)
	if err != nil {
		writeDecisionCreateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIDecision(item))
}

func (h *Handler) GetActions(w http.ResponseWriter, r *http.Request, params gen.GetActionsParams) {
	if h.decision == nil {
		writeInternalError(w, "decision service is not configured")
		return
	}
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	var status *string
	if params.Status != nil {
		s := string(*params.Status)
		status = &s
	}
	items, err := h.decision.ListActions(r.Context(), status, limit)
	if err != nil {
		if errors.Is(err, service.ErrInvalidActionStatus) {
			writeValidationError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to list actions")
		return
	}
	data := make([]gen.Action, len(items))
	for i, item := range items {
		data[i] = toAPIAction(item)
	}
	writeJSON(w, http.StatusOK, gen.ActionListResponse{Data: data})
}

func (h *Handler) PostActionStatus(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.decision == nil {
		writeInternalError(w, "decision service is not configured")
		return
	}
	var body gen.ActionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	req := service.UpdateActionStatusRequest{Status: string(body.Status)}
	if body.OutcomeNote != nil {
		req.OutcomeNote = *body.OutcomeNote
	}
	item, err := h.decision.UpdateActionStatus(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrActionNotFound):
			writeNotFoundError(w, err.Error())
		case errors.Is(err, service.ErrInvalidActionStatus), errors.Is(err, service.ErrActionStatusForbidden):
			writeValidationError(w, err.Error())
		default:
			writeInternalError(w, "failed to update action")
		}
		return
	}
	writeJSON(w, http.StatusOK, toAPIAction(item))
}

func writeDecisionCreateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidDecision):
		writeValidationError(w, err.Error())
	case errors.Is(err, service.ErrMoneyReviewNotFound), errors.Is(err, service.ErrScenarioNotFound):
		writeNotFoundError(w, err.Error())
	default:
		writeInternalError(w, "failed to create decision")
	}
}

func toAPIDecision(d service.Decision) gen.Decision {
	out := gen.Decision{
		Id:          d.ID,
		Title:       d.Title,
		Assumptions: d.Assumptions,
		DecidedAt:   d.DecidedAt,
		CreatedAt:   d.CreatedAt,
		Action:      toAPIAction(d.Action),
	}
	if d.ReviewID != nil {
		id := openapi_types.UUID(*d.ReviewID)
		out.ReviewId = &id
	}
	if d.ScenarioID != nil {
		id := openapi_types.UUID(*d.ScenarioID)
		out.ScenarioId = &id
	}
	if d.TargetValue != nil {
		s := d.TargetValue.StringFixed(2)
		out.TargetValue = &s
	}
	return out
}

func toAPIAction(a service.Action) gen.Action {
	out := gen.Action{
		Id:                   a.ID,
		DecisionId:           a.DecisionID,
		Title:                a.Title,
		ExpectedAnnualEffect: a.ExpectedAnnualEffect.StringFixed(2),
		DueOn:                openapi_types.Date{Time: a.DueOn},
		Status:               gen.ActionStatus(a.Status),
		OutcomeNote:          a.OutcomeNote,
		CreatedAt:            a.CreatedAt,
		UpdatedAt:            a.UpdatedAt,
	}
	if a.VerifiedAt != nil {
		out.VerifiedAt = a.VerifiedAt
	}
	return out
}

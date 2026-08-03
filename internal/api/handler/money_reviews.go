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
)

type MoneyReviewService interface {
	Create(ctx context.Context, baselineID *uuid.UUID) (service.MoneyReview, error)
	Get(ctx context.Context, id uuid.UUID) (service.MoneyReview, error)
	List(ctx context.Context, limit int) ([]service.MoneyReview, error)
	Confirm(ctx context.Context, id uuid.UUID) (service.MoneyReview, error)
}

func (h *Handler) GetMoneyReviews(w http.ResponseWriter, r *http.Request, params gen.GetMoneyReviewsParams) {
	if h.moneyReview == nil {
		writeInternalError(w, "money review service is not configured")
		return
	}
	limit := 50
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}
	items, err := h.moneyReview.List(r.Context(), limit)
	if err != nil {
		writeInternalError(w, "failed to list money reviews")
		return
	}
	data := make([]gen.MoneyReview, len(items))
	for i, item := range items {
		data[i] = toAPIMoneyReview(item)
	}
	writeJSON(w, http.StatusOK, gen.MoneyReviewListResponse{Data: data})
}

func (h *Handler) PostMoneyReviews(w http.ResponseWriter, r *http.Request) {
	if h.moneyReview == nil {
		writeInternalError(w, "money review service is not configured")
		return
	}

	var body gen.MoneyReviewCreateRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
	}

	var baselineID *uuid.UUID
	if body.BaselineId != nil {
		id := uuid.UUID(*body.BaselineId)
		baselineID = &id
	}

	item, err := h.moneyReview.Create(r.Context(), baselineID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBaselineRequired):
			writeValidationError(w, err.Error())
		case errors.Is(err, service.ErrBaselineNotFound), errors.Is(err, service.ErrBaselineNone):
			writeNotFoundError(w, err.Error())
		default:
			writeInternalError(w, "failed to create money review")
		}
		return
	}
	writeJSON(w, http.StatusOK, toAPIMoneyReview(item))
}

func (h *Handler) GetMoneyReview(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.moneyReview == nil {
		writeInternalError(w, "money review service is not configured")
		return
	}
	item, err := h.moneyReview.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrMoneyReviewNotFound) {
			writeNotFoundError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to load money review")
		return
	}
	writeJSON(w, http.StatusOK, toAPIMoneyReview(item))
}

func (h *Handler) PostMoneyReviewConfirm(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.moneyReview == nil {
		writeInternalError(w, "money review service is not configured")
		return
	}
	item, err := h.moneyReview.Confirm(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMoneyReviewNotFound):
			writeNotFoundError(w, err.Error())
		case errors.Is(err, service.ErrMoneyReviewNotOpen):
			writeConflictError(w, err.Error())
		default:
			writeInternalError(w, "failed to confirm money review")
		}
		return
	}
	writeJSON(w, http.StatusOK, toAPIMoneyReview(item))
}

func toAPIMoneyReview(m service.MoneyReview) gen.MoneyReview {
	findings := make([]gen.MoneyReviewFinding, len(m.Findings))
	for i, f := range m.Findings {
		ids := f.EvidenceIDs
		if ids == nil {
			ids = []string{}
		}
		finding := gen.MoneyReviewFinding{
			Type:        gen.MoneyReviewFindingType(f.Type),
			Title:       f.Title,
			Confidence:  gen.MoneyReviewFindingConfidence(f.Confidence),
			EvidenceIds: ids,
			PeriodFrom:  openapi_types.Date{Time: dateOnly(f.PeriodFrom)},
			PeriodTo:    openapi_types.Date{Time: dateOnly(f.PeriodTo)},
		}
		if f.Amount != nil {
			s := f.Amount.StringFixed(2)
			finding.Amount = &s
		}
		if f.SuggestedActionKey != "" {
			key := f.SuggestedActionKey
			finding.SuggestedActionKey = &key
		}
		findings[i] = finding
	}
	out := gen.MoneyReview{
		Id:            m.ID,
		BaselineId:    m.BaselineID,
		PeriodFrom:    openapi_types.Date{Time: dateOnly(m.PeriodFrom)},
		PeriodTo:      openapi_types.Date{Time: dateOnly(m.PeriodTo)},
		Status:        gen.MoneyReviewStatus(m.Status),
		Summary:       m.Summary,
		Findings:      findings,
		DataFreshness: m.DataFreshness,
		CreatedAt:     m.CreatedAt.UTC(),
	}
	if m.ConfirmedAt != nil {
		t := m.ConfirmedAt.UTC()
		out.ConfirmedAt = &t
	}
	return out
}

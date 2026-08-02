package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type RecurringService interface {
	ListUncertain(ctx context.Context) ([]domain.RecurringSeries, error)
	Confirm(ctx context.Context, id uuid.UUID) (domain.RecurringSeries, error)
	Reject(ctx context.Context, id uuid.UUID) (domain.RecurringSeries, error)
}

func (h *Handler) GetUncertainRecurring(w http.ResponseWriter, r *http.Request) {
	if h.recurring == nil {
		writeInternalError(w, "recurring service is not configured")
		return
	}
	items, err := h.recurring.ListUncertain(r.Context())
	if err != nil {
		writeInternalError(w, "failed to list uncertain recurring series")
		return
	}
	data := make([]gen.RecurringSeries, len(items))
	for i, item := range items {
		data[i] = toAPIRecurringSeries(item)
	}
	writeJSON(w, http.StatusOK, gen.RecurringSeriesListResponse{Data: data})
}

func (h *Handler) PostRecurringConfirm(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.recurring == nil {
		writeInternalError(w, "recurring service is not configured")
		return
	}
	series, err := h.recurring.Confirm(r.Context(), id)
	if err != nil {
		writeRecurringDecideError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIRecurringSeries(series))
}

func (h *Handler) PostRecurringReject(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.recurring == nil {
		writeInternalError(w, "recurring service is not configured")
		return
	}
	series, err := h.recurring.Reject(r.Context(), id)
	if err != nil {
		writeRecurringDecideError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIRecurringSeries(series))
}

func writeRecurringDecideError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrRecurringSeriesNotFound):
		writeNotFoundError(w, err.Error())
	case errors.Is(err, service.ErrRecurringSeriesNotUncertain):
		writeConflictError(w, err.Error())
	default:
		writeInternalError(w, "failed to update recurring series")
	}
}

func toAPIRecurringSeries(s domain.RecurringSeries) gen.RecurringSeries {
	out := gen.RecurringSeries{
		Id:            s.ID,
		DisplayName:   s.DisplayName,
		Interval:      gen.RecurringSeriesInterval(s.Interval),
		Kind:          gen.RecurringSeriesKind(s.Kind),
		Status:        gen.RecurringSeriesStatus(s.Status),
		AmountTypical: s.AmountTypical.StringFixed(2),
		AmountLast:    s.AmountLast.StringFixed(2),
		AmountChanged: s.AmountChanged,
		Uncertainty:   gen.RecurringSeriesUncertainty(s.Uncertainty),
		MemberCount:   s.MemberCount,
		CreatedAt:     s.CreatedAt.UTC(),
	}
	if s.NextExpected != nil {
		d := openapi_types.Date{Time: dateOnly(*s.NextExpected)}
		out.NextExpected = &d
	}
	return out
}

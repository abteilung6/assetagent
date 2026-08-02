package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type TransferService interface {
	ListCandidates(ctx context.Context) ([]domain.TransferCandidate, error)
	Confirm(ctx context.Context, id uuid.UUID) (domain.TransferPair, error)
	Reject(ctx context.Context, id uuid.UUID) (domain.TransferPair, error)
}

func (h *Handler) GetTransferCandidates(w http.ResponseWriter, r *http.Request) {
	if h.transfers == nil {
		writeInternalError(w, "transfer service is not configured")
		return
	}

	candidates, err := h.transfers.ListCandidates(r.Context())
	if err != nil {
		writeInternalError(w, "failed to list transfer candidates")
		return
	}

	data := make([]gen.TransferCandidate, len(candidates))
	for i, c := range candidates {
		data[i] = toAPITransferCandidate(c)
	}
	writeJSON(w, http.StatusOK, gen.TransferCandidateListResponse{Data: data})
}

func (h *Handler) PostTransferConfirm(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.transfers == nil {
		writeInternalError(w, "transfer service is not configured")
		return
	}

	pair, err := h.transfers.Confirm(r.Context(), id)
	if err != nil {
		writeTransferDecideError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITransferPair(pair))
}

func (h *Handler) PostTransferReject(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.transfers == nil {
		writeInternalError(w, "transfer service is not configured")
		return
	}

	pair, err := h.transfers.Reject(r.Context(), id)
	if err != nil {
		writeTransferDecideError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITransferPair(pair))
}

func writeTransferDecideError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTransferPairNotFound):
		writeNotFoundError(w, err.Error())
	case errors.Is(err, service.ErrTransferPairNotSuggested):
		writeConflictError(w, err.Error())
	default:
		writeInternalError(w, "failed to update transfer pair")
	}
}

func toAPITransferCandidate(c domain.TransferCandidate) gen.TransferCandidate {
	return gen.TransferCandidate{
		Id:         c.ID,
		Status:     gen.TransferCandidateStatus(c.Status),
		Confidence: gen.TransferCandidateConfidence(c.Confidence),
		Amount:     c.Amount.StringFixed(2),
		Out:        toAPITransferLeg(c.Out),
		In:         toAPITransferLeg(c.In),
		CreatedAt:  c.CreatedAt.UTC(),
	}
}

func toAPITransferLeg(leg domain.TransferLegView) gen.TransferLeg {
	return gen.TransferLeg{
		TransactionId: leg.TransactionID,
		AccountName:   leg.AccountName,
		BookingDate:   openapi_types.Date{Time: dateOnly(leg.BookingDate)},
		Amount:        leg.Amount.StringFixed(2),
		BookingText:   leg.BookingText,
		Purpose:       leg.Purpose,
		Counterparty:  leg.Counterparty,
	}
}

func toAPITransferPair(pair domain.TransferPair) gen.TransferPair {
	out := gen.TransferPair{
		Id:         pair.ID,
		TxOutId:    pair.TxOutID,
		TxInId:     pair.TxInID,
		Status:     gen.TransferPairStatus(pair.Status),
		Confidence: gen.TransferPairConfidence(pair.Confidence),
		CreatedAt:  pair.CreatedAt.UTC(),
	}
	if pair.ConfirmedAt != nil {
		t := pair.ConfirmedAt.UTC()
		out.ConfirmedAt = &t
	}
	return out
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

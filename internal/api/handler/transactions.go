package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type ListService interface {
	ListTransactions(ctx context.Context, params domain.ListParams) (domain.ListResult, error)
}

type Handler struct {
	list ListService
}

func New(list ListService) *Handler {
	return &Handler{list: list}
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, gen.HealthResponse{Status: "ok"})
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request, params gen.GetTransactionsParams) {
	listParams := domain.ListParams{}
	if params.Limit != nil {
		listParams.Limit = *params.Limit
	}
	if params.Offset != nil {
		listParams.Offset = *params.Offset
	}
	if params.From != nil {
		t := params.From.Time
		listParams.FromDate = &t
	}
	if params.To != nil {
		t := params.To.Time
		listParams.ToDate = &t
	}

	result, err := h.list.ListTransactions(r.Context(), listParams)
	if err != nil {
		var validationErr service.ValidationError
		if errors.As(err, &validationErr) {
			writeJSON(w, http.StatusBadRequest, gen.Error{
				Error:   "validation_failed",
				Message: validationErr.Message,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, gen.Error{
			Error:   "internal_error",
			Message: "failed to list transactions",
		})
		return
	}

	limit := domain.DefaultListLimit
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}
	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}

	data := make([]gen.Transaction, len(result.Transactions))
	for i, tx := range result.Transactions {
		data[i] = toAPITransaction(tx)
	}

	writeJSON(w, http.StatusOK, gen.TransactionListResponse{
		Data: data,
		Pagination: gen.Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  result.Total,
		},
	})
}

func toAPITransaction(tx domain.Transaction) gen.Transaction {
	return gen.Transaction{
		Id:                             openapi_types.UUID(tx.ID),
		OrderAccount:                   tx.OrderAccount,
		BookingDate:                    openapi_types.Date{Time: tx.BookingDate},
		ValueDate:                      openapi_types.Date{Time: tx.ValueDate},
		BookingText:                    tx.BookingText,
		Purpose:                        tx.Purpose,
		CreditorId:                     tx.CreditorID,
		MandateReference:               tx.MandateReference,
		EndToEndReference:              tx.EndToEndReference,
		CollectionReference:            tx.CollectionReference,
		DirectDebitOriginalAmount:      tx.DirectDebitOriginalAmount,
		ChargebackExpenseReimbursement: tx.ChargebackExpenseReimbursement,
		Counterparty:                   tx.Counterparty,
		CounterpartyIban:               tx.CounterpartyIBAN,
		CounterpartyBic:                tx.CounterpartyBIC,
		Amount:                         tx.Amount.StringFixed(2),
		Currency:                       tx.Currency,
		Info:                           tx.Info,
	}
}

func APIErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *gen.InvalidParamFormatError
	if errors.As(err, &invalid) {
		writeJSON(w, http.StatusBadRequest, gen.Error{
			Error:   "validation_failed",
			Message: invalid.Error(),
		})
		return
	}

	writeJSON(w, http.StatusBadRequest, gen.Error{
		Error:   "bad_request",
		Message: err.Error(),
	})
}

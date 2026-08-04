package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
)

type ListService interface {
	ListTransactions(ctx context.Context, params domain.ListParams) (domain.ListResult, error)
	SetTransactionOneOff(ctx context.Context, id uuid.UUID, oneOff bool) (domain.Transaction, error)
}

type ImportService interface {
	ImportBytes(ctx context.Context, data []byte, filename string, opts domain.ImportOptions) (domain.ImportResult, error)
	ListRuns(ctx context.Context, limit int) ([]domain.ImportRunSummary, error)
	GetRun(ctx context.Context, runID uuid.UUID) (domain.ImportRunSummary, error)
	Rollback(ctx context.Context, runID uuid.UUID) (domain.ImportRollbackResult, error)
}

type Handler struct {
	list        ListService
	chat        ChatService
	llmRegistry *llm.Registry
	importer    ImportService
	transfers   TransferService
	classify    ClassifyService
	categories  CategoryService
	recurring   RecurringService
	baseline    BaselineService
	moneyReview MoneyReviewService
	forecast    ForecastService
	decision    DecisionService
	sessions    *service.SessionService
}

func New(
	list ListService,
	chat ChatService,
	registry *llm.Registry,
	importer ImportService,
	transfers TransferService,
	classify ClassifyService,
	categories CategoryService,
	recurring RecurringService,
	baseline BaselineService,
	moneyReview MoneyReviewService,
	forecast ForecastService,
	decision DecisionService,
	sessions *service.SessionService,
) *Handler {
	return &Handler{
		list:        list,
		chat:        chat,
		llmRegistry: registry,
		importer:    importer,
		transfers:   transfers,
		classify:    classify,
		categories:  categories,
		recurring:   recurring,
		baseline:    baseline,
		moneyReview: moneyReview,
		forecast:    forecast,
		decision:    decision,
		sessions:    sessions,
	}
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, gen.HealthResponse{Status: "ok"})
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request, params gen.GetTransactionsParams) {
	listParams, err := toListParams(params)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result, err := h.list.ListTransactions(r.Context(), listParams)
	if err != nil {
		writeServiceError(w, err)
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

func toListParams(params gen.GetTransactionsParams) (domain.ListParams, error) {
	listParams := domain.ListParams{}
	if params.Limit != nil {
		if *params.Limit == 0 {
			return domain.ListParams{}, service.ErrInvalidLimit
		}
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
	if params.Account != nil {
		listParams.Account = params.Account
	}
	if params.Counterparty != nil {
		listParams.Counterparty = params.Counterparty
	}
	if params.Q != nil {
		listParams.Search = params.Q
	}
	if params.MinAmount != nil {
		amount, err := decimal.NewFromString(*params.MinAmount)
		if err != nil {
			return domain.ListParams{}, service.ErrInvalidMinAmount
		}
		listParams.MinAmount = &amount
	}
	if params.MaxAmount != nil {
		amount, err := decimal.NewFromString(*params.MaxAmount)
		if err != nil {
			return domain.ListParams{}, service.ErrInvalidMaxAmount
		}
		listParams.MaxAmount = &amount
	}
	if params.Sort != nil {
		listParams.Sort = domain.SortField(*params.Sort)
	}
	listParams.SortAsc = false
	if params.Order != nil {
		listParams.SortAsc = *params.Order == gen.Asc
	}

	return listParams, nil
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
		OneOff:                         tx.OneOff,
		Recurring:                      tx.Recurring,
	}
}

func (h *Handler) PostTransactionOneOff(
	w http.ResponseWriter,
	r *http.Request,
	transactionId openapi_types.UUID,
) {
	var body gen.TransactionOneOffRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}

	tx, err := h.list.SetTransactionOneOff(r.Context(), uuid.UUID(transactionId), body.OneOff)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			writeNotFoundError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to update transaction")
		return
	}

	writeJSON(w, http.StatusOK, toAPITransaction(tx))
}

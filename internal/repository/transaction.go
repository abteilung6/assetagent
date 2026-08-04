package repository

import (
	"context"
	"errors"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	queries sqldb.Querier
}

func NewTransaction(pool *pgxpool.Pool) *Transaction {
	return &Transaction{queries: sqldb.New(pool)}
}

func (r *Transaction) Insert(ctx context.Context, tx domain.Transaction) (uuid.UUID, error) {
	return r.queries.InsertTransaction(ctx, buildParams(tx, domain.Fingerprint(tx)))
}

func (r *Transaction) BatchInsert(ctx context.Context, txs []domain.Transaction) (inserted, duplicates int, err error) {
	for _, tx := range txs {
		params := buildIfNewParams(tx, domain.Fingerprint(tx))
		_, err := r.queries.InsertTransactionIfNew(ctx, params)
		if errors.Is(err, pgx.ErrNoRows) {
			duplicates++
			continue
		}
		if err != nil {
			return inserted, duplicates, err
		}
		inserted++
	}

	return inserted, duplicates, nil
}

func (r *Transaction) Count(ctx context.Context) (int64, error) {
	return r.queries.CountTransactions(ctx)
}

func (r *Transaction) List(ctx context.Context, params domain.ListParams) (domain.ListResult, error) {
	filter := buildFilterParams(params)

	total, err := r.queries.CountTransactionsFiltered(ctx, filter)
	if err != nil {
		return domain.ListResult{}, err
	}

	var sortField any
	if params.Sort != "" {
		sortField = string(params.Sort)
	}

	rows, err := r.queries.ListTransactions(ctx, sqldb.ListTransactionsParams{
		FromDate:     filter.FromDate,
		ToDate:       filter.ToDate,
		Account:      filter.Account,
		Counterparty: filter.Counterparty,
		MinAmount:    filter.MinAmount,
		MaxAmount:    filter.MaxAmount,
		Search:       filter.Search,
		SortField:    sortField,
		SortAsc:      params.SortAsc,
		Limit:        int32(params.Limit),
		Offset:       int32(params.Offset),
	})
	if err != nil {
		return domain.ListResult{}, err
	}

	transactions := make([]domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = listRowToDomain(row)
	}

	return domain.ListResult{
		Transactions: transactions,
		Total:        total,
	}, nil
}

func (r *Transaction) SetOneOff(ctx context.Context, id uuid.UUID, oneOff bool) (domain.Transaction, error) {
	row, err := r.queries.SetTransactionOneOff(ctx, sqldb.SetTransactionOneOffParams{
		ID:     id,
		OneOff: oneOff,
	})
	if err != nil {
		return domain.Transaction{}, err
	}
	return setOneOffRowToDomain(row), nil
}

func buildFilterParams(params domain.ListParams) sqldb.CountTransactionsFilteredParams {
	return sqldb.CountTransactionsFilteredParams{
		FromDate:     dateFromPtr(params.FromDate),
		ToDate:       dateFromPtr(params.ToDate),
		Account:      textFromPtr(params.Account),
		Counterparty: textFromPtr(params.Counterparty),
		MinAmount:    numericFromPtr(params.MinAmount),
		MaxAmount:    numericFromPtr(params.MaxAmount),
		Search:       textFromPtr(params.Search),
	}
}

func listRowToDomain(row sqldb.ListTransactionsRow) domain.Transaction {
	return domain.Transaction{
		ID:                             row.ID,
		OrderAccount:                   row.OrderAccount,
		BookingDate:                    row.BookingDate.Time,
		ValueDate:                      row.ValueDate.Time,
		BookingText:                    row.BookingText,
		Purpose:                        row.Purpose,
		CreditorID:                     row.CreditorID,
		MandateReference:               row.MandateReference,
		EndToEndReference:              row.EndToEndReference,
		CollectionReference:            row.CollectionReference,
		DirectDebitOriginalAmount:      row.DirectDebitOriginalAmount,
		ChargebackExpenseReimbursement: row.ChargebackExpenseReimbursement,
		Counterparty:                   row.Counterparty,
		CounterpartyIBAN:               ptrFromText(row.CounterpartyIban),
		CounterpartyBIC:                ptrFromText(row.CounterpartyBic),
		Amount:                         row.Amount,
		Currency:                       row.Currency,
		Info:                           row.Info,
		OneOff:                         row.OneOff,
		Recurring:                      row.Recurring,
	}
}

func setOneOffRowToDomain(row sqldb.SetTransactionOneOffRow) domain.Transaction {
	return domain.Transaction{
		ID:                             row.ID,
		OrderAccount:                   row.OrderAccount,
		BookingDate:                    row.BookingDate.Time,
		ValueDate:                      row.ValueDate.Time,
		BookingText:                    row.BookingText,
		Purpose:                        row.Purpose,
		CreditorID:                     row.CreditorID,
		MandateReference:               row.MandateReference,
		EndToEndReference:              row.EndToEndReference,
		CollectionReference:            row.CollectionReference,
		DirectDebitOriginalAmount:      row.DirectDebitOriginalAmount,
		ChargebackExpenseReimbursement: row.ChargebackExpenseReimbursement,
		Counterparty:                   row.Counterparty,
		CounterpartyIBAN:               ptrFromText(row.CounterpartyIban),
		CounterpartyBIC:                ptrFromText(row.CounterpartyBic),
		Amount:                         row.Amount,
		Currency:                       row.Currency,
		Info:                           row.Info,
		OneOff:                         row.OneOff,
		Recurring:                      row.Recurring,
	}
}

func dateFromPtr(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *value, Valid: true}
}

func ptrFromText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	s := value.String
	return &s
}

func numericFromPtr(value *decimal.Decimal) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(value.String())
	return n
}

func buildParams(tx domain.Transaction, fingerprint string) sqldb.InsertTransactionParams {
	return sqldb.InsertTransactionParams{
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
		AccountID:                      pgtype.UUID{},
		ImportRunID:                    pgtype.UUID{},
	}
}

func buildIfNewParams(tx domain.Transaction, fingerprint string) sqldb.InsertTransactionIfNewParams {
	return sqldb.InsertTransactionIfNewParams(buildParams(tx, fingerprint))
}

func textFromPtr(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

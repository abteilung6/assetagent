package repository

import (
	"context"

	"github.com/abteilung6/assetagent/internal/domain"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Transaction struct {
	queries sqldb.Querier
}

func NewTransaction(pool *pgxpool.Pool) *Transaction {
	return &Transaction{queries: sqldb.New(pool)}
}

func (r *Transaction) Insert(ctx context.Context, tx domain.Transaction) (uuid.UUID, error) {
	return r.queries.InsertTransaction(ctx, sqldb.InsertTransactionParams{
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
	})
}

func (r *Transaction) Count(ctx context.Context) (int64, error) {
	return r.queries.CountTransactions(ctx)
}

func textFromPtr(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

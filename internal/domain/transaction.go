package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID                             uuid.UUID
	OrderAccount                   string
	BookingDate                    time.Time
	ValueDate                      time.Time
	BookingText                    string
	Purpose                        string
	CreditorID                     string
	MandateReference               string
	EndToEndReference              string
	CollectionReference            string
	DirectDebitOriginalAmount      string
	ChargebackExpenseReimbursement string
	Counterparty                   string
	CounterpartyIBAN               *string
	CounterpartyBIC                *string
	Amount                         decimal.Decimal
	Currency                       string
	Info                           string
}

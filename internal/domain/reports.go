package domain

import "github.com/shopspring/decimal"

type CashflowReport struct {
	Income   decimal.Decimal
	Expenses decimal.Decimal
	Net      decimal.Decimal
}

type CounterpartySpend struct {
	Counterparty     string
	TotalSpent       decimal.Decimal
	TransactionCount int64
}

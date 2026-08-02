package domain

import "github.com/shopspring/decimal"

type CashflowReport struct {
	Income   decimal.Decimal
	Expenses decimal.Decimal
	Net      decimal.Decimal
}

// CashflowReportV2 is household cashflow with confirmed internal transfers excluded.
type CashflowReportV2 struct {
	Income            decimal.Decimal
	Expenses          decimal.Decimal
	Net               decimal.Decimal
	TransfersExcluded bool
}

type CounterpartySpend struct {
	Counterparty     string
	TotalSpent       decimal.Decimal
	TransactionCount int64
}

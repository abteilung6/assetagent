package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

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

// CashflowV2Evidence is transfer-aware cashflow plus the Phase B evidence contract.
type CashflowV2Evidence struct {
	Income            decimal.Decimal
	Expenses          decimal.Decimal
	Net               decimal.Decimal
	Currency          string
	PeriodFrom        time.Time
	PeriodTo          time.Time
	AccountsIncluded  []string
	TransfersExcluded bool
	Calculation       string
	Confidence        string
	DataFreshness     string
	Assumptions       []string
	EvidenceIDs       []string
}

type CounterpartySpend struct {
	Counterparty     string
	TotalSpent       decimal.Decimal
	TransactionCount int64
}

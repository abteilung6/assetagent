package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	RecurringIntervalMonthly   = "monthly"
	RecurringIntervalQuarterly = "quarterly"
	RecurringIntervalYearly    = "yearly"

	RecurringKindFixed           = "fixed"
	RecurringKindVariableRegular = "variable_regular"
	RecurringKindIncome          = "income"

	RecurringStatusActive     = "active"
	RecurringStatusUncertain  = "uncertain"
	RecurringStatusEnded      = "ended"

	RecurringUncertaintyLow    = "low"
	RecurringUncertaintyMedium = "medium"
	RecurringUncertaintyHigh   = "high"
)

// RecurringScanTx is a minimal transaction view for series detection.
type RecurringScanTx struct {
	ID           uuid.UUID
	AccountID    uuid.UUID
	BookingDate  time.Time
	Amount       decimal.Decimal
	Counterparty string
	Purpose      string
	BookingText  string
}

// RecurringSeries is a detected or stored recurring payment series.
type RecurringSeries struct {
	ID            uuid.UUID
	Fingerprint   string
	DisplayName   string
	Interval      string
	Kind          string
	Status        string
	AmountTypical decimal.Decimal
	AmountLast    decimal.Decimal
	AmountChanged bool
	NextExpected  *time.Time
	Uncertainty   string
	MemberCount   int
	MemberIDs     []uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RecurringSeriesMember is one booking that belongs to a detected series.
type RecurringSeriesMember struct {
	TransactionID uuid.UUID
	BookingDate   time.Time
	Amount        decimal.Decimal
	Counterparty  string
	Purpose       string
}

type RecurringScanResult struct {
	TransactionsConsidered int
	SkippedExisting        int
	Suggested              int
	Series                 []RecurringSeries
}

package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 200
)

type SortField string

const (
	SortBookingDate  SortField = "booking_date"
	SortAmount       SortField = "amount"
	SortCounterparty SortField = "counterparty"
)

type ListParams struct {
	Limit        int
	Offset       int
	FromDate     *time.Time
	ToDate       *time.Time
	Account      *string
	Counterparty *string
	MinAmount    *decimal.Decimal
	MaxAmount    *decimal.Decimal
	Search       *string
	Sort         SortField
	SortAsc      bool
}

type ListResult struct {
	Transactions []Transaction
	Total        int64
}

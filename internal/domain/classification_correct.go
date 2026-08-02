package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ClassifyCorrectOptions struct {
	CategorySlug    string
	ApplyToMerchant bool
}

type ClassifyCorrectResult struct {
	TransactionID uuid.UUID
	CategorySlug  string
	RuleCreated   bool
	MerchantID    *uuid.UUID
}

// ClassificationQueueItem is a high-impact booking awaiting category review.
type ClassificationQueueItem struct {
	TransactionID uuid.UUID
	BookingDate   time.Time
	Amount        decimal.Decimal
	Counterparty  string
	Purpose       string
	BookingText   string
	CategorySlug  string
	CategoryName  string
	Source        string
	Confidence    string
	MerchantID    *uuid.UUID
	MerchantName  string
}

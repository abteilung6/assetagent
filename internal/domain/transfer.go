package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	TransferStatusSuggested = "suggested"
	TransferStatusConfirmed = "confirmed"
	TransferStatusRejected  = "rejected"

	TransferConfidenceExact    = "exact"
	TransferConfidenceProbable = "probable"
)

// TransferScanTx is a minimal transaction view for pairing.
type TransferScanTx struct {
	ID               uuid.UUID
	AccountID        uuid.UUID
	BookingDate      time.Time
	Amount           decimal.Decimal
	Purpose          string
	BookingText      string
	Counterparty     string
	CounterpartyIBAN string
}

type TransferPair struct {
	ID          uuid.UUID
	TxOutID     uuid.UUID
	TxInID      uuid.UUID
	Status      string
	Confidence  string
	Rationale   map[string]any
	ConfirmedAt *time.Time
	CreatedAt   time.Time
}

type TransferScanResult struct {
	CandidatesConsidered int
	Suggested            int
	SkippedExisting      int
	Pairs                []TransferPair
}

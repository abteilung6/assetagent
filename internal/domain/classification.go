package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ClassificationSourceUserRule   = "user_rule"
	ClassificationSourceExactRule  = "exact_rule"
	ClassificationSourceMerchant   = "merchant"
	ClassificationSourceHeuristic  = "heuristic"
	ClassificationSourceUnresolved = "unresolved"

	ClassificationConfidenceHigh   = "high"
	ClassificationConfidenceMedium = "medium"
	ClassificationConfidenceLow    = "low"

	ClassifyAlgorithmVersion = "classify-v1"
)

type Classification struct {
	TransactionID     uuid.UUID
	CategoryID        uuid.UUID
	MerchantID        *uuid.UUID
	Source            string
	Confidence        string
	AlgorithmVersion  string
	UpdatedAt         time.Time
}

type ClassifyRunResult struct {
	Transactions int
	Upserted     int
	SkippedUser  int
	BySource     map[string]int64
	ByCategory   map[string]int64
}

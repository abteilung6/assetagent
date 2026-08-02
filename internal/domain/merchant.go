package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	MerchantMatchExact      = "exact"
	MerchantMatchNormalized = "normalized"
)

type Merchant struct {
	ID                uuid.UUID
	DisplayName       string
	DefaultCategoryID *uuid.UUID
	AliasCount        int64
	CreatedAt         time.Time
}

type MerchantRebuildResult struct {
	LabelsConsidered int
	MerchantsCreated int
	AliasesCreated   int
	AliasesExisting  int
	SkippedEmpty     int
}

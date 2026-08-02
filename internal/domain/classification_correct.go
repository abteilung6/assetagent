package domain

import "github.com/google/uuid"

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

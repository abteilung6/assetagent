package domain

import "github.com/google/uuid"

const (
	CategoryKindIncome   = "income"
	CategoryKindExpense  = "expense"
	CategoryKindTransfer = "transfer"
	CategoryKindSaving   = "saving"
	CategoryKindOther    = "other"
)

// SystemCategorySlugs is the stable household taxonomy seeded by migrations.
var SystemCategorySlugs = []string{
	"income",
	"housing",
	"insurance",
	"mobility",
	"groceries",
	"dining",
	"shopping",
	"subscriptions",
	"leisure",
	"travel",
	"health",
	"personal_care",
	"education",
	"children",
	"pets",
	"taxes_fees",
	"cash_atm",
	"saving_investing",
	"transfer",
	"other",
}

type Category struct {
	ID          uuid.UUID
	Slug        string
	DisplayName string
	Kind        string
	ParentID    *uuid.UUID
	IsSystem    bool
}

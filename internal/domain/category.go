package domain

import "github.com/google/uuid"

const (
	CategoryKindIncome   = "income"
	CategoryKindExpense  = "expense"
	CategoryKindTransfer = "transfer"
	CategoryKindSaving   = "saving"
	CategoryKindOther    = "other"
)

// SystemCategorySlugs is the stable Phase B taxonomy seeded by migration.
var SystemCategorySlugs = []string{
	"income",
	"housing",
	"insurance",
	"mobility",
	"groceries",
	"leisure",
	"health",
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

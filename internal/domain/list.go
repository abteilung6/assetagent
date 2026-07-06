package domain

import "time"

const (
	DefaultListLimit = 50
	MaxListLimit     = 200
)

type ListParams struct {
	Limit    int
	Offset   int
	FromDate *time.Time
	ToDate   *time.Time
}

type ListResult struct {
	Transactions []Transaction
	Total        int64
}

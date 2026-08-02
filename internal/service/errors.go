package service

import "errors"

var (
	ErrInvalidLimit              = errors.New("limit must be between 1 and 200")
	ErrInvalidOffset             = errors.New("offset must be non-negative")
	ErrInvalidSort               = errors.New("sort must be one of: booking_date, amount, counterparty")
	ErrInvalidAmountRange        = errors.New("min_amount must be less than or equal to max_amount")
	ErrInvalidMinAmount          = errors.New("invalid min_amount")
	ErrInvalidMaxAmount          = errors.New("invalid max_amount")
	ErrImportRunNotFound          = errors.New("import run not found")
	ErrImportRunNotCommitted      = errors.New("import run is not committed")
	ErrImportRunAlreadyRolledBack = errors.New("import run already rolled back")
	ErrPreviewHashMismatch        = errors.New("preview_hash does not match uploaded file")
	ErrAccountNotFound            = errors.New("account not found")
)

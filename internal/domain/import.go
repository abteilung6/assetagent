package domain

import "time"

type ImportResult struct {
	Rows       int
	Inserted   int
	Duplicates int
	Errors     int
}

type ImportPreview struct {
	FileHash           string
	SourceFilename     string
	ParserName         string
	ParserVersion      string
	PeriodFrom         *time.Time
	PeriodTo           *time.Time
	SuggestedAccount   string
	RowTotal           int
	RowValid           int
	RowInvalid         int
	SampleRows         []ImportPreviewRow
	InvalidRows        []ImportInvalidRow
	Warnings           []string
}

type ImportPreviewRow struct {
	Line         int
	BookingDate  time.Time
	Counterparty string
	Purpose      string
	Amount       string
	Currency     string
}

type ImportInvalidRow struct {
	Line    int
	Field   string
	Message string
}

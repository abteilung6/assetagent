package domain

type ImportResult struct {
	Rows       int
	Inserted   int
	Duplicates int
	Errors     int
}

package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/parser/sparkasse"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type TransactionRepository interface {
	BatchInsert(ctx context.Context, txs []domain.Transaction) (inserted, duplicates int, err error)
}

type Import struct {
	repo TransactionRepository
}

func NewImport(repo TransactionRepository) *Import {
	return &Import{repo: repo}
}

func (s *Import) ImportFile(ctx context.Context, path string) (domain.ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("open file: %w", err)
	}

	reader := csvReader(data)
	transactions, err := sparkasse.Parse(reader)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("parse csv: %w", err)
	}

	inserted, duplicates, err := s.repo.BatchInsert(ctx, transactions)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("batch insert: %w", err)
	}

	return domain.ImportResult{
		Rows:       len(transactions),
		Inserted:   inserted,
		Duplicates: duplicates,
	}, nil
}

func csvReader(data []byte) io.Reader {
	if utf8.Valid(data) {
		return bytes.NewReader(data)
	}
	return transform.NewReader(bytes.NewReader(data), charmap.ISO8859_1.NewDecoder())
}

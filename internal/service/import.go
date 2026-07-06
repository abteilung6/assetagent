package service

import (
	"context"
	"fmt"
	"os"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/parser/sparkasse"
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
	file, err := os.Open(path)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	transactions, err := sparkasse.Parse(file)
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

package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
)

type fakeTransactionRepo struct {
	inserted   int
	duplicates int
	err        error
}

func (f *fakeTransactionRepo) BatchInsert(ctx context.Context, txs []domain.Transaction) (int, int, error) {
	if f.err != nil {
		return 0, 0, f.err
	}
	return f.inserted, f.duplicates, nil
}

func TestImport_ImportFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sparkasse", "minimal.csv")

	repo := &fakeTransactionRepo{inserted: 6, duplicates: 0}
	importer := service.NewImport(repo)

	result, err := importer.ImportFile(context.Background(), path)
	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}
	if result.Rows != 6 {
		t.Fatalf("Rows = %d, want 6", result.Rows)
	}
	if result.Inserted != 6 {
		t.Fatalf("Inserted = %d, want 6", result.Inserted)
	}
	if result.Duplicates != 0 {
		t.Fatalf("Duplicates = %d, want 0", result.Duplicates)
	}
}

func TestImport_ImportFile_missingFile(t *testing.T) {
	importer := service.NewImport(&fakeTransactionRepo{})

	_, err := importer.ImportFile(context.Background(), "missing.csv")
	if err == nil {
		t.Fatal("ImportFile() error = nil, want error")
	}
}

func TestImport_ImportFile_repoError(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sparkasse", "minimal.csv")

	repo := &fakeTransactionRepo{err: os.ErrInvalid}
	importer := service.NewImport(repo)

	_, err := importer.ImportFile(context.Background(), path)
	if err == nil {
		t.Fatal("ImportFile() error = nil, want error")
	}
}

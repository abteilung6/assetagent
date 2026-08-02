package service_test

import (
	"context"
	"testing"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestIntegration_ClassifyCorrectRule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	q := sqldb.New(pool)
	acc, err := q.CreateAccount(ctx, sqldb.CreateAccountParams{
		DisplayName: "Checking", Bank: "sparkasse", Currency: "EUR",
		OrderAccount: pgtype.Text{String: "DE-R-1", Valid: true}, MaskedIdentifier: "x",
	})
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	day := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	amazonID := insertClassifyTx(t, ctx, q, acc.ID, "DE-R-1", "-9.99", "AMAZON DIGITAL GERMANY GMBH", "Prime", "fp-rule-amz-1", day)

	svc := service.NewClassify(pool)
	if _, err := svc.Run(ctx); err != nil {
		t.Fatalf("initial run: %v", err)
	}

	corrected, err := svc.Correct(ctx, amazonID, domain.ClassifyCorrectOptions{
		CategorySlug:    "groceries",
		ApplyToMerchant: true,
	})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	if !corrected.RuleCreated || corrected.MerchantID == nil {
		t.Fatalf("correct result = %+v", corrected)
	}

	class, err := q.GetTransactionClassification(ctx, amazonID)
	if err != nil {
		t.Fatalf("get class: %v", err)
	}
	if class.Source != domain.ClassificationSourceUserRule {
		t.Fatalf("source = %q", class.Source)
	}

	secondID := insertClassifyTx(t, ctx, q, acc.ID, "DE-R-1", "-4.50", "Amazon Payments Europe S.C.A.", "Order", "fp-rule-amz-2", day)

	rerun, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if rerun.Upserted < 1 {
		t.Fatalf("expected new tx classified, rerun=%+v", rerun)
	}

	secondClass, err := q.GetTransactionClassification(ctx, secondID)
	if err != nil {
		t.Fatalf("get second class: %v", err)
	}
	if secondClass.Source != domain.ClassificationSourceUserRule {
		t.Fatalf("second source = %q, want user_rule", secondClass.Source)
	}
	groceries, err := q.GetCategoryBySlug(ctx, "groceries")
	if err != nil {
		t.Fatalf("groceries: %v", err)
	}
	if secondClass.CategoryID != groceries.ID {
		t.Fatalf("second category = %s, want groceries", secondClass.CategoryID)
	}

	// Original correction must remain user_rule (not overwritten).
	class, err = q.GetTransactionClassification(ctx, amazonID)
	if err != nil {
		t.Fatalf("get class after rerun: %v", err)
	}
	if class.Source != domain.ClassificationSourceUserRule {
		t.Fatalf("source after rerun = %q", class.Source)
	}
}

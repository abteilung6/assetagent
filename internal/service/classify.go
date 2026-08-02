package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/classify"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Classify struct {
	pool *pgxpool.Pool
}

func NewClassify(pool *pgxpool.Pool) *Classify {
	return &Classify{pool: pool}
}

func (s *Classify) Run(ctx context.Context) (domain.ClassifyRunResult, error) {
	q := sqldb.New(s.pool)

	// Ensure merchant aliases exist for current txs.
	if _, err := NewMerchants(s.pool).Rebuild(ctx); err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("merchant rebuild: %w", err)
	}

	categories, err := q.ListCategories(ctx)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list categories: %w", err)
	}
	bySlug := make(map[string]uuid.UUID, len(categories))
	for _, c := range categories {
		bySlug[c.Slug] = c.ID
	}

	transferRows, err := q.ListConfirmedTransferTransactionIDs(ctx)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list transfers: %w", err)
	}
	transfers := make(map[uuid.UUID]struct{}, len(transferRows))
	for _, id := range transferRows {
		transfers[id] = struct{}{}
	}

	txs, err := q.ListTransactionsForClassify(ctx)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list txs: %w", err)
	}

	result := domain.ClassifyRunResult{
		Transactions: len(txs),
		BySource:     map[string]int64{},
		ByCategory:   map[string]int64{},
	}

	for _, tx := range txs {
		_, isTransfer := transfers[tx.ID]
		pattern := ""
		var merchantID pgtype.UUID

		if label, ok := classify.NormalizeMerchantLabel(tx.Counterparty, tx.Purpose); ok {
			pattern = label.Pattern
			alias, err := q.GetMerchantAlias(ctx, sqldb.GetMerchantAliasParams{
				MatchType: domain.MerchantMatchNormalized,
				Pattern:   label.Pattern,
			})
			if err == nil {
				merchantID = pgtype.UUID{Bytes: alias.MerchantID, Valid: true}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return domain.ClassifyRunResult{}, fmt.Errorf("get alias: %w", err)
			}
		}

		slug, source, confidence := classify.SuggestCategory(tx.Amount, pattern, isTransfer)
		categoryID, ok := bySlug[slug]
		if !ok {
			return domain.ClassifyRunResult{}, fmt.Errorf("missing category slug %q", slug)
		}

		row, err := q.UpsertTransactionClassification(ctx, sqldb.UpsertTransactionClassificationParams{
			TransactionID:    tx.ID,
			CategoryID:       categoryID,
			MerchantID:       merchantID,
			Source:           source,
			Confidence:       confidence,
			AlgorithmVersion: domain.ClassifyAlgorithmVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				result.SkippedUser++
				continue
			}
			return domain.ClassifyRunResult{}, fmt.Errorf("upsert classification: %w", err)
		}
		result.Upserted++
		result.BySource[row.Source]++
		result.ByCategory[slug]++
	}

	return result, nil
}

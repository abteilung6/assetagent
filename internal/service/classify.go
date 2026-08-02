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

	if _, err := NewMerchants(s.pool).Rebuild(ctx); err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("merchant rebuild: %w", err)
	}

	categories, err := q.ListCategories(ctx)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list categories: %w", err)
	}
	bySlug := make(map[string]uuid.UUID, len(categories))
	slugByID := make(map[uuid.UUID]string, len(categories))
	for _, c := range categories {
		bySlug[c.Slug] = c.ID
		slugByID[c.ID] = c.Slug
	}

	rules, err := q.ListClassificationRules(ctx)
	if err != nil {
		return domain.ClassifyRunResult{}, fmt.Errorf("list rules: %w", err)
	}
	ruleByMerchant := make(map[uuid.UUID]sqldb.ClassificationRule, len(rules))
	for _, rule := range rules {
		if rule.MerchantID.Valid {
			ruleByMerchant[uuid.UUID(rule.MerchantID.Bytes)] = rule
		}
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

		var (
			categoryID uuid.UUID
			source     string
			confidence string
			slug       string
		)

		if merchantID.Valid {
			if rule, ok := ruleByMerchant[uuid.UUID(merchantID.Bytes)]; ok {
				categoryID = rule.CategoryID
				source = domain.ClassificationSourceUserRule
				confidence = domain.ClassificationConfidenceHigh
				slug = slugByID[rule.CategoryID]
			}
		}

		if source == "" {
			slug, source, confidence = classify.SuggestCategory(tx.Amount, pattern, isTransfer)
			var ok bool
			categoryID, ok = bySlug[slug]
			if !ok {
				return domain.ClassifyRunResult{}, fmt.Errorf("missing category slug %q", slug)
			}
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
		if slug == "" {
			slug = slugByID[row.CategoryID]
		}
		result.ByCategory[slug]++
	}

	return result, nil
}

func (s *Classify) Correct(
	ctx context.Context,
	txID uuid.UUID,
	opts domain.ClassifyCorrectOptions,
) (domain.ClassifyCorrectResult, error) {
	q := sqldb.New(s.pool)

	slug := opts.CategorySlug
	category, err := q.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ClassifyCorrectResult{}, fmt.Errorf("unknown category %q", slug)
		}
		return domain.ClassifyCorrectResult{}, err
	}

	tx, err := q.GetTransactionForClassify(ctx, txID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ClassifyCorrectResult{}, fmt.Errorf("transaction not found")
		}
		return domain.ClassifyCorrectResult{}, err
	}

	var merchantID pgtype.UUID
	if label, ok := classify.NormalizeMerchantLabel(tx.Counterparty, tx.Purpose); ok {
		if _, err := NewMerchants(s.pool).Rebuild(ctx); err != nil {
			return domain.ClassifyCorrectResult{}, err
		}
		alias, err := q.GetMerchantAlias(ctx, sqldb.GetMerchantAliasParams{
			MatchType: domain.MerchantMatchNormalized,
			Pattern:   label.Pattern,
		})
		if err == nil {
			merchantID = pgtype.UUID{Bytes: alias.MerchantID, Valid: true}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return domain.ClassifyCorrectResult{}, err
		}
	}

	if _, err := q.ForceUpsertTransactionClassification(ctx, sqldb.ForceUpsertTransactionClassificationParams{
		TransactionID:    txID,
		CategoryID:       category.ID,
		MerchantID:       merchantID,
		Source:           domain.ClassificationSourceUserRule,
		Confidence:       domain.ClassificationConfidenceHigh,
		AlgorithmVersion: domain.ClassifyAlgorithmVersion,
	}); err != nil {
		return domain.ClassifyCorrectResult{}, err
	}

	result := domain.ClassifyCorrectResult{
		TransactionID: txID,
		CategorySlug:  slug,
	}

	if opts.ApplyToMerchant && merchantID.Valid {
		mid := uuid.UUID(merchantID.Bytes)
		result.MerchantID = &mid
		existing, err := q.GetClassificationRuleByMerchant(ctx, merchantID)
		switch {
		case err == nil:
			if _, err := q.UpdateClassificationRuleCategory(ctx, sqldb.UpdateClassificationRuleCategoryParams{
				ID:                        existing.ID,
				CategoryID:                category.ID,
				CreatedFromTransactionID:  pgtype.UUID{Bytes: txID, Valid: true},
				Priority:                  10,
			}); err != nil {
				return domain.ClassifyCorrectResult{}, err
			}
			result.RuleCreated = true
		case errors.Is(err, pgx.ErrNoRows):
			if _, err := q.CreateClassificationRule(ctx, sqldb.CreateClassificationRuleParams{
				Priority:                 10,
				MerchantID:               merchantID,
				CategoryID:               category.ID,
				CreatedFromTransactionID: pgtype.UUID{Bytes: txID, Valid: true},
			}); err != nil {
				return domain.ClassifyCorrectResult{}, err
			}
			result.RuleCreated = true
		default:
			return domain.ClassifyCorrectResult{}, err
		}

		if _, err := q.UpdateMerchantDefaultCategory(ctx, sqldb.UpdateMerchantDefaultCategoryParams{
			ID:                mid,
			DefaultCategoryID: pgtype.UUID{Bytes: category.ID, Valid: true},
		}); err != nil {
			return domain.ClassifyCorrectResult{}, err
		}
	}

	return result, nil
}

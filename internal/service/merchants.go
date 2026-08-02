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

type Merchants struct {
	pool *pgxpool.Pool
}

func NewMerchants(pool *pgxpool.Pool) *Merchants {
	return &Merchants{pool: pool}
}

func (s *Merchants) Rebuild(ctx context.Context) (domain.MerchantRebuildResult, error) {
	q := sqldb.New(s.pool)
	rows, err := q.ListMerchantSourceLabels(ctx)
	if err != nil {
		return domain.MerchantRebuildResult{}, fmt.Errorf("list labels: %w", err)
	}

	result := domain.MerchantRebuildResult{LabelsConsidered: len(rows)}
	seenPatterns := make(map[string]struct{})

	for _, row := range rows {
		label, ok := classify.NormalizeMerchantLabel(row.Counterparty, row.Purpose)
		if !ok {
			result.SkippedEmpty++
			continue
		}
		if _, seen := seenPatterns[label.Pattern]; seen {
			continue
		}
		seenPatterns[label.Pattern] = struct{}{}

		existing, err := q.GetMerchantAlias(ctx, sqldb.GetMerchantAliasParams{
			MatchType: domain.MerchantMatchNormalized,
			Pattern:   label.Pattern,
		})
		if err == nil {
			result.AliasesExisting++
			_ = existing
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.MerchantRebuildResult{}, fmt.Errorf("get alias: %w", err)
		}

		merchant, err := q.CreateMerchant(ctx, sqldb.CreateMerchantParams{
			DisplayName:       label.DisplayName,
			DefaultCategoryID: pgtype.UUID{},
		})
		if err != nil {
			return domain.MerchantRebuildResult{}, fmt.Errorf("create merchant: %w", err)
		}
		result.MerchantsCreated++

		if _, err := q.CreateMerchantAlias(ctx, sqldb.CreateMerchantAliasParams{
			MerchantID: merchant.ID,
			MatchType:  domain.MerchantMatchNormalized,
			Pattern:    label.Pattern,
		}); err != nil {
			return domain.MerchantRebuildResult{}, fmt.Errorf("create alias: %w", err)
		}
		result.AliasesCreated++
	}

	return result, nil
}

func (s *Merchants) List(ctx context.Context) ([]domain.Merchant, error) {
	rows, err := sqldb.New(s.pool).ListMerchants(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Merchant, len(rows))
	for i, row := range rows {
		m := domain.Merchant{
			ID:          row.ID,
			DisplayName: row.DisplayName,
			AliasCount:  row.AliasCount,
			CreatedAt:   row.CreatedAt.Time,
		}
		if row.DefaultCategoryID.Valid {
			id := uuid.UUID(row.DefaultCategoryID.Bytes)
			m.DefaultCategoryID = &id
		}
		out[i] = m
	}
	return out, nil
}

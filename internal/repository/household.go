package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/authctx"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResolveHouseholdID returns the household from context, or falls back to the
// local seed household so unauthenticated/dev callers keep working.
func ResolveHouseholdID(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	if id, ok := authctx.HouseholdID(ctx); ok {
		return id, nil
	}
	return ResolveSeedHouseholdID(ctx, pool)
}

// ResolveSeedHouseholdID finds the local seed household without requiring auth context.
func ResolveSeedHouseholdID(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	auth := NewAuth(pool)
	if h, err := auth.GetUnclaimedSeedHousehold(ctx); err == nil {
		return h.ID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	if h, err := auth.GetHouseholdByName(ctx, domain.SeedHouseholdName); err == nil {
		return h.ID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	h, err := auth.GetFirstHousehold(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("no household found: run migrations to create seed household")
		}
		return uuid.Nil, err
	}
	return h.ID, nil
}

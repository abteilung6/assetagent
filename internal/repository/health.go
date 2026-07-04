package repository

import (
	"context"
	"fmt"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Health struct {
	queries sqldb.Querier
}

func NewHealth(pool *pgxpool.Pool) *Health {
	return &Health{queries: sqldb.New(pool)}
}

func (h *Health) Ping(ctx context.Context) error {
	ok, err := h.queries.Ping(ctx)
	if err != nil {
		return err
	}
	if ok != 1 {
		return fmt.Errorf("unexpected ping result: %d", ok)
	}
	return nil
}

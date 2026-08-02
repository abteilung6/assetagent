package repository

import (
	"context"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Categories struct {
	queries sqldb.Querier
}

func NewCategories(pool *pgxpool.Pool) *Categories {
	return &Categories{queries: sqldb.New(pool)}
}

func (c *Categories) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := c.queries.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Category, len(rows))
	for i, row := range rows {
		out[i] = mapCategory(row)
	}
	return out, nil
}

func (c *Categories) GetBySlug(ctx context.Context, slug string) (domain.Category, error) {
	row, err := c.queries.GetCategoryBySlug(ctx, slug)
	if err != nil {
		return domain.Category{}, err
	}
	return mapCategory(row), nil
}

func mapCategory(row sqldb.Category) domain.Category {
	var parentID *uuid.UUID
	if row.ParentID.Valid {
		id := uuid.UUID(row.ParentID.Bytes)
		parentID = &id
	}
	return domain.Category{
		ID:          row.ID,
		Slug:        row.Slug,
		DisplayName: row.DisplayName,
		Kind:        row.Kind,
		ParentID:    parentID,
		IsSystem:    row.IsSystem,
	}
}

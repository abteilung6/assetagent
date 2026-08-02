package repository_test

import (
	"context"
	"testing"

	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestCategories_ListSystemSeed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("assetagent"),
		postgres.WithUsername("assetagent"),
		postgres.WithPassword("assetagent"),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	waitForDB(ctx, t, connStr)

	if err := db.RunMigrations(connStr, "up"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	pool, err := db.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)

	repo := repository.NewCategories(pool)
	categories, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(categories) != len(domain.SystemCategorySlugs) {
		t.Fatalf("len = %d, want %d", len(categories), len(domain.SystemCategorySlugs))
	}

	bySlug := make(map[string]domain.Category, len(categories))
	for _, cat := range categories {
		if !cat.IsSystem {
			t.Fatalf("category %q is_system = false, want true", cat.Slug)
		}
		bySlug[cat.Slug] = cat
	}

	for _, slug := range domain.SystemCategorySlugs {
		cat, ok := bySlug[slug]
		if !ok {
			t.Fatalf("missing system category %q", slug)
		}
		if cat.DisplayName == "" {
			t.Fatalf("category %q has empty display name", slug)
		}
		if cat.Kind == "" {
			t.Fatalf("category %q has empty kind", slug)
		}
	}

	transfer, err := repo.GetBySlug(ctx, "transfer")
	if err != nil {
		t.Fatalf("get transfer: %v", err)
	}
	if transfer.Kind != domain.CategoryKindTransfer {
		t.Fatalf("transfer kind = %q, want %q", transfer.Kind, domain.CategoryKindTransfer)
	}
}

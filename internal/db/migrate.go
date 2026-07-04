package db

import (
	"database/sql"
	"fmt"

	"github.com/abteilung6/assetagent/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func RunMigrations(databaseURL, direction string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	switch direction {
	case "up":
		if err := goose.Up(db, "."); err != nil {
			return fmt.Errorf("migrate up: %w", err)
		}
	case "down":
		if err := goose.Down(db, "."); err != nil {
			return fmt.Errorf("migrate down: %w", err)
		}
	case "status":
		if err := goose.Status(db, "."); err != nil {
			return fmt.Errorf("migrate status: %w", err)
		}
	default:
		return fmt.Errorf("unknown migration direction %q", direction)
	}

	return nil
}

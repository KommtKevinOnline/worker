package database

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

func RunMigrations(embedMigrations embed.FS, db *sql.DB) {
	goose.SetBaseFS(embedMigrations)

	err := goose.SetDialect("postgres")

	if err != nil {
		panic(err)
	}

	err = goose.Up(db, "migrations")

	if err != nil {
		panic(err)
	}
}

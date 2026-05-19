package ent

import (
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" //revive:disable:blank-imports pgx driver registration

	gen "github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/postgres"
)

// NewClient creates a new Ent client connected to a PostgreSQL database.
func NewClient(dsn string) (*gen.Client, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	postgres.ApplyDefaults(db)
	drv := entsql.OpenDB(dialect.Postgres, db)
	return gen.NewClient(gen.Driver(drv)), nil
}

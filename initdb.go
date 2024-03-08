package violet

import (
	"database/sql"
	"embed"
	"errors"
	"github.com/1f349/violet/database"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed database/migrations/*.sql
var migrations embed.FS

func InitDB(p string) (*database.Queries, error) {
	migDrv, err := iofs.New(migrations, "database/migrations")
	if err != nil {
		return nil, err
	}
	dbOpen, err := sql.Open("sqlite3", p)
	if err != nil {
		return nil, err
	}
	dbDrv, err := sqlite3.WithInstance(dbOpen, &sqlite3.Config{})
	if err != nil {
		return nil, err
	}
	mig, err := migrate.NewWithInstance("iofs", migDrv, "sqlite3", dbDrv)
	if err != nil {
		return nil, err
	}
	err = mig.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, err
	}
	return database.New(dbOpen), nil
}

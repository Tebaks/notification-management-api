package postgres

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	migrations "github.com/kenanabbak/notification-management-api/migrations"
)

func InvokeMigrations(db *sqlx.DB, log *zap.Logger) error {
	return RunMigrations(db, log)
}

func RunMigrations(db *sqlx.DB, log *zap.Logger) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	driver, err := migratepg.WithInstance(db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	version, _, _ := m.Version()
	log.Info("migrations applied", zap.Uint("version", version))
	return nil
}

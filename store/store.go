package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"linkshrink/store/sqlc"
	"linkshrink/utils"

	"github.com/golang-migrate/migrate/v4"
	pgx_migrate "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/httpfs"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultQueryTimeout = time.Minute
	DefaultLimit        = 20
	MaxLimit            = 100
	DefaultOffset       = 0
)

func DefaultContextTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), DefaultQueryTimeout)
}

// Store represents a store that is backed by a Postgres database.
type Store struct {
	cfg     *Config
	db      *pgxpool.Pool
	queries *sqlc.Queries
	logger  *slog.Logger
	clock   utils.Clock
}

func calculateLimitOffset(limit, offset int32) (int32, int32, error) {
	if limit > MaxLimit {
		return 0, 0, fmt.Errorf("limit exceeds the maximum allowed value of %d",
			MaxLimit)
	}

	if limit < 0 || offset < 0 {
		return 0, 0, fmt.Errorf("limit and offset must be non-negative")
	}

	if limit == 0 {
		limit = DefaultLimit
	}

	return limit, offset, nil
}

// runMigrations runs the migrations on the database.
//
// NOTE: this function uses its own db connection.
func runMigrations(logger *slog.Logger, cfg *Config) error {
	db, err := sql.Open("pgx", cfg.DSN(false))
	if err != nil {
		return err
	}

	// Close the db connection after we are done.
	defer func() {
		if err := db.Close(); err != nil && err != sql.ErrConnDone {

			logger.Error(
				"Unable to close db after running migrations",
				"error", err,
			)
		}
	}()

	// With the migrate instance open, we'll create a new migration source
	// using the embedded file system stored in sqlSchemas. The library
	// we're using can't handle a raw file system interface, so we wrap it
	// in this intermediate layer.
	migrateFileServer, err := httpfs.New(
		http.FS(sqlSchemas), "sqlc/migrations",
	)
	if err != nil {
		return err
	}

	driver, err := pgx_migrate.WithInstance(
		db, &pgx_migrate.Config{},
	)
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Finally, we'll run the migration with our driver above based on the
	// open DB, and also the migration source stored in the file system
	// above.
	m, err := migrate.NewWithInstance(
		"migrations", migrateFileServer, cfg.DBName, driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w",
			err)
	}

	start := time.Now().UTC()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, _, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return err
	}

	logger.Info(
		"DB migrations applied",
		"version", version,
		"time", time.Since(start),
	)

	return nil
}

// NewStore creates a new store that is backed by a Postgres database backend.
func NewStore(logger *slog.Logger, cfg *Config, clock utils.Clock) (*Store,
	error) {

	logger.Info(
		"Creating new store",
		"dsn", cfg.DSN(true),
	)

	config, err := pgxpool.ParseConfig(cfg.DSN(false))
	if err != nil {
		return nil, err
	}

	// If we use the "pool_max_conns" parameter that pgxpool allegedly
	// supports, that string is sent to the backend which results in an
	// error (because Postgres doesn't know that parameter). So this is a
	// workaround to set the max open connections on the config manually.
	if cfg.MaxOpenConnections > 0 {
		config.MaxConns = cfg.MaxOpenConnections
	}

	db, err := pgxpool.NewWithConfig(context.TODO(), config)
	if err != nil {
		return nil, err
	}

	if !cfg.SkipMigrations {
		if err := runMigrations(logger, cfg); err != nil {
			return nil, fmt.Errorf("unable to run migrations: %v",
				err)
		}
	}

	queries := sqlc.New(db)

	return &Store{
		cfg:     cfg,
		db:      db,
		queries: queries,
		logger:  logger,
		clock:   clock,
	}, nil
}

// ExecTx is a wrapper for txBody to abstract the creation and commit of a db
// transaction. The db transaction is embedded in a `*postgres.Queries` that
// txBody needs to use when executing each one of the queries that need to be
// applied atomically.
func (s *Store) ExecTx(ctx context.Context,
	txBody func(*sqlc.Queries) error) error {

	// Create the db transaction.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op.
	defer func() {
		err := tx.Rollback(ctx)
		switch {
		// If this is an unexpected error, log it.
		case err != nil && err != pgx.ErrTxClosed:
			s.logger.Error(
				"Unable to rollback db tx",
				"error", err,
				"type", fmt.Sprintf("%T", err),
			)
		}
	}()

	if err := txBody(s.queries.WithTx(tx)); err != nil {
		return err
	}

	// Commit transaction.
	return tx.Commit(ctx)
}

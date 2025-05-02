package store

import (
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	dsnTemplate = "postgres://%v:%v@%v:%d/%v?sslmode=%v"
)

// DefaultConfig returns all default values for the Config struct.
func DefaultConfig() *Config {
	return &Config{
		SkipMigrations:     false,
		Host:               "db", // Default for docker-compose
		Port:               5432,
		User:               "user",
		Password:           "password",
		DBName:             "linkshrink",
		MaxOpenConnections: 25,
		RequireSSL:         false,
		MigrationsPath:     "store/sqlc/migrations",
	}
}

// Config holds the postgres database configuration.
type Config struct {
	SkipMigrations     bool   `long:"skip_migrations" description:"Skip applying migrations on startup."`
	Host               string `long:"host" description:"Database server hostname."`
	Port               int    `long:"port" description:"Database server port."`
	User               string `long:"user" description:"Database user."`
	Password           string `long:"password" description:"Database user's password."`
	DBName             string `long:"dbname" description:"Database name to use."`
	MaxOpenConnections int32  `long:"max_connections" description:"Max open connections to keep alive to the database server."`
	RequireSSL         bool   `long:"require_ssl" description:"Whether to require using SSL (mode: require) when connecting to the server."`
	MigrationsPath     string `long:"migrations_path" description:"Path to the migrations folder"`
}

// DSN returns the dns to connect to the database.
func (s *Config) DSN(hidePassword bool) string {
	var sslMode = "disable"
	if s.RequireSSL {
		sslMode = "require"
	}

	password := s.Password
	if hidePassword {
		// Placeholder used for logging the DSN safely.
		password = "****"
	}

	return fmt.Sprintf(dsnTemplate, s.User, password, s.Host, s.Port,
		s.DBName, sslMode)
}

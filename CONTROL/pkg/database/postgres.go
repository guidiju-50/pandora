// Package database provides database connectivity.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/guidiju-50/pandora/CONTROL/internal/config"
	"go.uber.org/zap"
)

// NewPostgresDB creates a new PostgreSQL database connection.
func NewPostgresDB(cfg config.DatabaseConfig, logger *zap.Logger) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	logger.Info("connecting to PostgreSQL",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	logger.Info("connected to PostgreSQL successfully")
	return db, nil
}

// Migrate runs database migrations.
func Migrate(db *sqlx.DB, migrationsPath string, logger *zap.Logger) error {
	logger.Info("running database migrations")
	// Migration logic would go here
	// In production, use a migration tool like golang-migrate
	return nil
}

// HealthCheck checks database health.
func HealthCheck(db *sqlx.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

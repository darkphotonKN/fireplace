package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/logger"
	"github.com/jmoiron/sqlx"

	// Importing for side effects - Dont Remove
	// This IS being used!
	_ "github.com/lib/pq"
)

const (
	maxOpenConnections = 25
	maxIdleConnections = 5
)

/**
* Sets up the Database connection and provides its access as a singleton to
* the entire application.
**/
func InitDB() *sqlx.DB {

	// construct the db connection string
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	logger.Debug("Database connection string constructed", "dsn", dsn)

	// pass the db connection string to connect to our database
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	logger.Info("Connected to database", "maxOpen", maxOpenConnections, "maxIdle", maxIdleConnections)

	db.SetMaxOpenConns(maxOpenConnections) // Max 25 concurrent connections
	db.SetMaxIdleConns(maxIdleConnections) // Keep 5 connections alive when idle
	db.SetConnMaxLifetime(5 * time.Minute) // Recycle connections every 5 minutes
	db.SetConnMaxIdleTime(1 * time.Minute) // Close idle connections after 1 minute

	// Migrations are NOT run here (ADR-0010 §4). They run as <db>_owner via
	// ./cmd/migrate, a discrete step that completes before this process starts.
	// This connection is <db>_app, which holds no DDL and cannot migrate.

	// set global instance for the database
	return db
}

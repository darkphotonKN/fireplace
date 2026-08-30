package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDB() *sqlx.DB {
	var (
		host     = getEnvAsString("DB_HOST", "localhost")
		port     = getEnvAsInt("DB_PORT", 5303)
		user     = getEnvAsString("DB_USER", "user")
		password = getEnvAsString("DB_PASSWORD", "password")
		dbname   = getEnvAsString("DB_NAME", "fireplace_plan_service_db")
	)

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		panic(fmt.Errorf("failed to ping database: %w", err))
	}

	// Migrations are NOT run here (ADR-0010 §4) — see ./cmd/migrate.
	// This connection is <db>_app and holds no DDL.

	slog.Info("Connected to database", slog.String("dbname", dbname))
	return db
}


func getEnvAsString(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			return iv
		}
	}
	return defaultValue
}

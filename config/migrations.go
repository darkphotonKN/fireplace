package config

import (
	"log"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// Fixed IDs for test entities
var (
	testUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testPlanID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

/**
* Installs and uses uuid extension for *postgres*.
**/
func SetupUUIDExtension(db *sqlx.DB) {
	query := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create uuid-ossp extension: %v", err)
	}
	log.Println("UUID extension setup completed.")
}

/**
* Runs all migrations using golang-migrate.
**/
func RunMigrations(db *sqlx.DB) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations ran successfully.")

	// Seed development test data
	if err := SeedDBData(db); err != nil {
		log.Printf("Warning: Failed to seed test data: %v", err)
	}
}

/**
* Seeds test data for development purposes.
**/
func SeedDBData(db *sqlx.DB) error {
	// Seed test user
	if err := seedTestUser(db); err != nil {
		return err
	}

	// Seed test plan
	if err := seedTestPlan(db); err != nil {
		return err
	}

	return nil
}

/**
* Seeds a test user for development purposes.
**/
func seedTestUser(db *sqlx.DB) error {
	// Check if test user already exists
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM users WHERE email = $1", "darkphoton20@gmail.com")
	if err != nil {
		return err
	}

	// If test user already exists, return
	if count > 0 {
		log.Println("Test user already exists, skipping seed.")
		return nil
	}

	// Generate password hash
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		BaseDBDateModel: models.BaseDBDateModel{
			ID: testUserID,
		},
		Name:     "Kranti",
		Email:    "darkphoton20@gmail.com",
		Password: string(hash),
	}

	// Insert test user
	_, err = db.NamedExec(`
		INSERT INTO users (id, name, email, password, created_at, updated_at)
		VALUES (:id, :name, :email, :password, NOW(), NOW())
	`, user)

	if err != nil {
		return err
	}

	log.Println("Test user seeded successfully.")
	return nil
}

/**
* Seeds a test plan for development purposes.
**/
func seedTestPlan(db *sqlx.DB) error {
	// Check if test plan already exists
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM plans WHERE id = $1", testPlanID)
	if err != nil {
		return err
	}

	// If test plan already exists, return
	if count > 0 {
		log.Println("Test plan already exists, skipping seed.")
		return nil
	}

	// Check if the users table has the test user
	var userCount int
	err = db.Get(&userCount, "SELECT COUNT(*) FROM users WHERE id = $1", testUserID)
	if err != nil {
		return err
	}

	if userCount == 0 {
		log.Println("Test user does not exist, cannot create plan.")
		return nil
	}

	// Create test plan
	plan := models.Plan{
		BaseDBDateModel: models.BaseDBDateModel{
			ID: testPlanID,
		},
		UserID:      testUserID,
		Name:        "Flow Project",
		Focus:       "Making a nextjs project frontend and go gin backend app about productivity.",
		Description: "A project for testing the Flow application",
		PlanType:    "development",
	}

	// Insert test plan
	_, err = db.NamedExec(`
		INSERT INTO plans (id, user_id, name, focus, description, plan_type, created_at, updated_at)
		VALUES (:id, :user_id, :name, :focus, :description, :plan_type, NOW(), NOW())
	`, plan)

	if err != nil {
		return err
	}

	log.Println("Test plan seeded successfully.")
	return nil
}

# CLAUDE.md - Go Project Template

This template provides comprehensive guidance for Claude Code to generate new Go projects following established best practices.

## Project Generation Instructions

When asked to create a new Go project using this template, follow these patterns exactly.

## Architecture Principles

### 1. Domain-Driven Design
- Organize code by business domain, not by technical layer
- Each domain should be self-contained with its own model, repository, service, and handler

### 2. SOLID Principles
- **Single Responsibility**: Each struct/function should have one reason to change
- **Open/Closed**: Use interfaces for extension points
- **Liskov Substitution**: Interfaces should be substitutable
- **Interface Segregation**: Define interfaces where they are consumed, not where implemented
- **Dependency Inversion**: Depend on abstractions (interfaces), not concrete implementations

### 3. Dependency Injection Pattern
- Pass dependencies through constructors
- Use interfaces for all dependencies
- Define interfaces in the consuming package

## Project Structure

```
project-name/
├── cmd/
│   └── main.go                 # Application entry point
├── config/
│   ├── database.go             # Database connection and setup
│   └── routes.go               # HTTP route configuration
├── internal/
│   ├── [domain]/               # Each domain follows this structure
│   │   ├── model.go           # Domain models and validation
│   │   ├── repository.go      # Database interactions
│   │   ├── service.go         # Business logic
│   │   └── handler.go         # HTTP handlers
│   └── middleware/
│       └── auth.go            # Authentication middleware
├── pkg/
│   └── utils/                 # Shared utilities
├── migrations/                # SQL migration files
├── .env.sample               # Environment variable template
├── docker-compose.yml        # Local development database
├── Makefile                  # Build and development commands
├── go.mod
└── go.sum
```

## Core Components Template

### 1. Main Entry Point (cmd/main.go)
```go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "[project]/config"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
    
    // Initialize database
    db, err := config.InitDB()
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()
    
    // Run migrations
    if err := config.RunMigrations(db); err != nil {
        log.Fatal("Failed to run migrations:", err)
    }
    
    // Setup and start server
    router := config.SetupRoutes(db)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
    if err := router.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
```

### 2. Database Configuration (config/database.go)
```go
package config

import (
    "fmt"
    "os"
    
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

func InitDB() (*sqlx.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    
    db, err := sqlx.Connect("postgres", dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    
    return db, nil
}

func RunMigrations(db *sqlx.DB) error {
    driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
    if err != nil {
        return err
    }
    
    m, err := migrate.NewWithDatabaseInstance(
        "file://migrations",
        "postgres", driver)
    if err != nil {
        return err
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    
    return nil
}
```

### 3. Route Configuration (config/routes.go)
```go
package config

import (
    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    
    // Import domain packages
    "[project]/internal/[domain]"
    "[project]/internal/middleware"
)

func SetupRoutes(db *sqlx.DB) *gin.Engine {
    router := gin.Default()
    
    // Initialize repositories
    exampleRepo := example.NewRepository(db)
    
    // Initialize services with dependency injection
    exampleService := example.NewService(exampleRepo)
    
    // Initialize handlers
    exampleHandler := example.NewHandler(exampleService)
    
    // API routes
    api := router.Group("/api")
    {
        // Public routes
        api.POST("/signup", exampleHandler.SignUp)
        api.POST("/signin", exampleHandler.SignIn)
        
        // Protected routes
        protected := api.Group("/")
        protected.Use(middleware.AuthMiddleware())
        {
            protected.GET("/examples", exampleHandler.List)
            protected.POST("/examples", exampleHandler.Create)
            protected.GET("/examples/:id", exampleHandler.Get)
            protected.PUT("/examples/:id", exampleHandler.Update)
            protected.DELETE("/examples/:id", exampleHandler.Delete)
        }
    }
    
    return router
}
```

### 4. Domain Package Template (internal/[domain]/)

#### Model (model.go)
```go
package [domain]

import (
    "time"
    "github.com/go-playground/validator/v10"
)

type [Domain] struct {
    ID        int       `db:"id" json:"id"`
    Name      string    `db:"name" json:"name" validate:"required,min=1,max=255"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
    UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (d *[Domain]) Validate() error {
    validate := validator.New()
    return validate.Struct(d)
}
```

#### Repository Interface and Implementation (repository.go)
```go
package [domain]

import (
    "context"
    "github.com/jmoiron/sqlx"
)

// Repository interface - defined where consumed
type Repository interface {
    Create(ctx context.Context, item *[Domain]) error
    GetByID(ctx context.Context, id int) (*[Domain], error)
    List(ctx context.Context) ([][Domain], error)
    Update(ctx context.Context, item *[Domain]) error
    Delete(ctx context.Context, id int) error
}

type repository struct {
    db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
    return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, item *[Domain]) error {
    query := `
        INSERT INTO [table] (name, created_at, updated_at)
        VALUES ($1, NOW(), NOW())
        RETURNING id, created_at, updated_at
    `
    return r.db.GetContext(ctx, item, query, item.Name)
}

func (r *repository) GetByID(ctx context.Context, id int) (*[Domain], error) {
    var item [Domain]
    query := `SELECT * FROM [table] WHERE id = $1`
    err := r.db.GetContext(ctx, &item, query, id)
    return &item, err
}

func (r *repository) List(ctx context.Context) ([][Domain], error) {
    var items [][Domain]
    query := `SELECT * FROM [table] ORDER BY created_at DESC`
    err := r.db.SelectContext(ctx, &items, query)
    return items, err
}

func (r *repository) Update(ctx context.Context, item *[Domain]) error {
    query := `
        UPDATE [table] 
        SET name = $1, updated_at = NOW()
        WHERE id = $2
        RETURNING updated_at
    `
    return r.db.GetContext(ctx, item, query, item.Name, item.ID)
}

func (r *repository) Delete(ctx context.Context, id int) error {
    query := `DELETE FROM [table] WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}
```

#### Service (service.go)
```go
package [domain]

import (
    "context"
    "errors"
)

// Service interface - can be defined here or in handler
type Service interface {
    Create(ctx context.Context, item *[Domain]) error
    GetByID(ctx context.Context, id int) (*[Domain], error)
    List(ctx context.Context) ([][Domain], error)
    Update(ctx context.Context, id int, item *[Domain]) error
    Delete(ctx context.Context, id int) error
}

type service struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, item *[Domain]) error {
    if err := item.Validate(); err != nil {
        return err
    }
    return s.repo.Create(ctx, item)
}

func (s *service) GetByID(ctx context.Context, id int) (*[Domain], error) {
    if id <= 0 {
        return nil, errors.New("invalid ID")
    }
    return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context) ([][Domain], error) {
    return s.repo.List(ctx)
}

func (s *service) Update(ctx context.Context, id int, item *[Domain]) error {
    if err := item.Validate(); err != nil {
        return err
    }
    item.ID = id
    return s.repo.Update(ctx, item)
}

func (s *service) Delete(ctx context.Context, id int) error {
    if id <= 0 {
        return errors.New("invalid ID")
    }
    return s.repo.Delete(ctx, id)
}
```

#### Handler (handler.go)
```go
package [domain]

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
)

type Handler struct {
    service Service
}

func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
    var item [Domain]
    if err := c.ShouldBindJSON(&item); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    if err := h.service.Create(c.Request.Context(), &item); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, item)
}

func (h *Handler) Get(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
        return
    }
    
    item, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }
    
    c.JSON(http.StatusOK, item)
}

func (h *Handler) List(c *gin.Context) {
    items, err := h.service.List(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, items)
}

func (h *Handler) Update(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
        return
    }
    
    var item [Domain]
    if err := c.ShouldBindJSON(&item); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    if err := h.service.Update(c.Request.Context(), id, &item); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, item)
}

func (h *Handler) Delete(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
        return
    }
    
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusNoContent, nil)
}
```

## Supporting Files

### Environment Variables (.env.sample)
```
DB_USER=postgres
DB_PASSWORD=password
DB_HOST=localhost
DB_PORT=5432
DB_NAME=project_db
JWT_SECRET=your-secret-key
PORT=8080
```

### Docker Compose (docker-compose.yml)
```yaml
version: '3.8'

services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: project_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

### Makefile
```makefile
.PHONY: build run dev test migrate-up migrate-down migrate-create

# Build the application
build:
	go build -o bin/app cmd/main.go

# Run the application
run:
	go run cmd/main.go

# Run with hot reload (requires air)
dev:
	air

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Database migrations
migrate-up:
	migrate -path migrations -database "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" down

migrate-create:
	migrate create -ext sql -dir migrations -seq $(NAME)

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f
```

### Example Migration (migrations/000001_init.up.sql)
```sql
CREATE TABLE IF NOT EXISTS examples (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_examples_created_at ON examples(created_at);
```

## Key Patterns to Follow

### 1. Interface Definition
- Define interfaces where they are consumed, not where implemented
- Keep interfaces small and focused (Interface Segregation Principle)

### 2. Dependency Injection
- All dependencies passed through constructors
- Use interfaces for all injected dependencies
- No global state or singletons

### 3. Error Handling
- Return errors from functions, don't panic
- Wrap errors with context when propagating
- Handle errors at the appropriate level

### 4. Context Usage
- Pass context.Context as first parameter
- Use for cancellation and request-scoped values
- Don't store in structs

### 5. Database Patterns
- Use sqlx for database operations
- Prepared statements for all queries
- Transactions where needed for consistency

## Dependencies to Install

```go
// go.mod dependencies
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/jmoiron/sqlx v1.3.5
    github.com/lib/pq v1.10.9
    github.com/joho/godotenv v1.5.1
    github.com/golang-migrate/migrate/v4 v4.16.2
    github.com/go-playground/validator/v10 v10.15.5
    github.com/golang-jwt/jwt/v5 v5.0.0
)
```

## Generation Instructions

When asked to generate a new project:

1. Create the directory structure exactly as shown
2. Replace `[project]` with the actual project name
3. Replace `[domain]` with actual domain names (e.g., users, products, orders)
4. Replace `[Domain]` with capitalized domain name
5. Replace `[table]` with the database table name
6. Create at least one complete example domain to demonstrate the pattern
7. Include all supporting files (Makefile, docker-compose.yml, .env.sample)
8. Create an initial migration file for the example domain
9. Use the exact import paths and package names as shown

## Important Notes

- Never add implementation details from the original project
- Keep examples simple and focused on structure
- Always follow the interface definition pattern (define where consumed)
- Maintain consistency in naming and structure across all domains
- Use context.Context for all database operations
- Include proper error handling in all functions
# Go MCP Project Template

This template is derived from analyzing a production Go application and provides a comprehensive structure for Model Context Protocol (MCP) projects.

## Directory Structure Template

```
project-name/
├── cmd/                          # Application entry points
│   └── main.go                   # Main application entry
├── config/                       # Configuration and initialization
│   ├── db.go                     # Database setup and connection pooling
│   ├── migrations.go             # Migration runner
│   └── routes.go                 # Route setup and dependency injection
├── internal/                     # Private application code
│   ├── [domain]/                 # Each business domain follows this pattern:
│   │   ├── handler.go            # HTTP handlers
│   │   ├── model.go              # Request/response DTOs and validation
│   │   ├── repository.go         # Data access layer
│   │   └── service.go            # Business logic layer
│   ├── auth/                     # Authentication utilities
│   │   └── jwt.go                # JWT token handling
│   ├── constants/                # Application constants
│   │   └── api.go                # API-related constants
│   ├── interfaces/               # Shared interfaces
│   ├── jobs/                     # Background job system
│   │   └── manager.go            # Job lifecycle management
│   ├── models/                   # Shared data models
│   │   └── entities.go           # Common entity definitions
│   └── utils/                    # Utility packages
│       ├── dbutils/              # Database utilities
│       └── errorutils/           # Error handling utilities
├── migrations/                   # Database migration files
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── scripts/                      # Development and deployment scripts
├── docker/                       # Docker configuration files
│   └── Dockerfile
├── bin/                          # Compiled binaries (gitignored)
├── tmp/                          # Temporary files (gitignored)
├── .env.example                  # Example environment variables
├── .gitignore
├── docker-compose.yml            # Local development setup
├── Makefile                      # Build and development commands
├── go.mod                        # Go module file
├── go.sum                        # Go checksum file
└── README.md                     # Project documentation
```

## Package Organization Patterns

### Domain Package Structure

Each business domain should follow this consistent 4-layer pattern:

1. **handler.go** - HTTP request handling
2. **model.go** - Data structures and validation
3. **repository.go** - Database operations
4. **service.go** - Business logic and interfaces

### Naming Conventions

- **Packages**: lowercase, single words (e.g., `user`, `product`, `order`)
- **Files**: snake_case for multi-word names (e.g., `user_profile.go`)
- **Interfaces**: Start with capital letter (e.g., `UserService`)
- **Implementations**: lowercase (e.g., `userService`)
- **Constants**: PascalCase (e.g., `MaxRetryAttempts`)

## Standard File Templates

### 1. Main Entry Point (cmd/main.go)

```go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "github.com/yourusername/projectname/config"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
    
    // Initialize database
    db, err := config.InitDB()
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer db.Close()
    
    // Run migrations
    if err := config.RunMigrations(db); err != nil {
        log.Fatal("Failed to run migrations:", err)
    }
    
    // Setup routes
    router := config.SetupRoutes(db)
    
    // Start server
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

### 2. Database Configuration (config/db.go)

```go
package config

import (
    "fmt"
    "os"
    "time"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

func InitDB() (*sqlx.DB, error) {
    // Build connection string from environment
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )
    
    db, err := sqlx.Connect("postgres", connStr)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(1 * time.Minute)
    
    return db, nil
}
```

### 3. Route Configuration (config/routes.go)

```go
package config

import (
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    
    "github.com/yourusername/projectname/internal/user"
    // Import other domain packages
)

func SetupRoutes(db *sqlx.DB) *gin.Engine {
    router := gin.Default()
    
    // CORS configuration
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
    }))
    
    // API routes
    api := router.Group("/api")
    
    // User routes
    userRepo := user.NewRepository(db)
    userService := user.NewService(userRepo)
    userHandler := user.NewHandler(userService)
    
    userRoutes := api.Group("/users")
    {
        userRoutes.POST("/signup", userHandler.SignUp)
        userRoutes.POST("/signin", userHandler.SignIn)
        userRoutes.GET("/:id", userHandler.GetUser)
    }
    
    // Add other domain routes here
    
    return router
}
```

### 4. Domain Model Template (internal/[domain]/model.go)

```go
package domain

import (
    "time"
    
    "github.com/google/uuid"
)

// Entity represents the main domain entity
type Entity struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateEntityRequest represents the request to create an entity
type CreateEntityRequest struct {
    Name string `json:"name" binding:"required"`
}

// UpdateEntityRequest represents the request to update an entity
type UpdateEntityRequest struct {
    Name string `json:"name"`
}

// Validate performs validation on the create request
func (r *CreateEntityRequest) Validate() error {
    if r.Name == "" {
        return ErrNameRequired
    }
    return nil
}
```

### 5. Domain Repository Template (internal/[domain]/repository.go)

```go
package domain

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
)

// Repository interface defines the data access methods
type Repository interface {
    Create(ctx context.Context, entity *Entity) error
    GetByID(ctx context.Context, id uuid.UUID) (*Entity, error)
    Update(ctx context.Context, entity *Entity) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, limit, offset int) ([]*Entity, error)
}

// repository implements the Repository interface
type repository struct {
    db *sqlx.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *sqlx.DB) Repository {
    return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, entity *Entity) error {
    query := `
        INSERT INTO entities (id, name, created_at, updated_at)
        VALUES ($1, $2, $3, $4)
    `
    
    _, err := r.db.ExecContext(ctx, query,
        entity.ID,
        entity.Name,
        entity.CreatedAt,
        entity.UpdatedAt,
    )
    
    if err != nil {
        return fmt.Errorf("failed to create entity: %w", err)
    }
    
    return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Entity, error) {
    query := `
        SELECT id, name, created_at, updated_at
        FROM entities
        WHERE id = $1
    `
    
    var entity Entity
    err := r.db.GetContext(ctx, &entity, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to get entity: %w", err)
    }
    
    return &entity, nil
}

// Implement other methods...
```

### 6. Domain Service Template (internal/[domain]/service.go)

```go
package domain

import (
    "context"
    "time"
    
    "github.com/google/uuid"
)

// Service interface defines the business logic methods
type Service interface {
    CreateEntity(ctx context.Context, req *CreateEntityRequest) (*Entity, error)
    GetEntity(ctx context.Context, id uuid.UUID) (*Entity, error)
    UpdateEntity(ctx context.Context, id uuid.UUID, req *UpdateEntityRequest) (*Entity, error)
    DeleteEntity(ctx context.Context, id uuid.UUID) error
    ListEntities(ctx context.Context, page, pageSize int) ([]*Entity, error)
}

// service implements the Service interface
type service struct {
    repo Repository
}

// NewService creates a new service instance
func NewService(repo Repository) Service {
    return &service{repo: repo}
}

func (s *service) CreateEntity(ctx context.Context, req *CreateEntityRequest) (*Entity, error) {
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    entity := &Entity{
        ID:        uuid.New(),
        Name:      req.Name,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    if err := s.repo.Create(ctx, entity); err != nil {
        return nil, err
    }
    
    return entity, nil
}

func (s *service) GetEntity(ctx context.Context, id uuid.UUID) (*Entity, error) {
    return s.repo.GetByID(ctx, id)
}

// Implement other methods...
```

### 7. Domain Handler Template (internal/[domain]/handler.go)

```go
package domain

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// Handler handles HTTP requests for the domain
type Handler struct {
    service Service
}

// NewHandler creates a new handler instance
func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

// Create handles entity creation
func (h *Handler) Create(c *gin.Context) {
    var req CreateEntityRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    entity, err := h.service.CreateEntity(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, entity)
}

// Get handles fetching an entity by ID
func (h *Handler) Get(c *gin.Context) {
    idStr := c.Param("id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    
    entity, err := h.service.GetEntity(c.Request.Context(), id)
    if err != nil {
        if err == ErrNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, entity)
}

// Implement other handlers...
```

## Dependencies (go.mod)

```go
module github.com/yourusername/projectname

go 1.23

require (
    github.com/gin-contrib/cors v1.5.0
    github.com/gin-gonic/gin v1.10.0
    github.com/go-co-op/gocron/v2 v2.2.9
    github.com/golang-jwt/jwt/v5 v5.2.1
    github.com/golang-migrate/migrate/v4 v4.17.1
    github.com/google/uuid v1.6.0
    github.com/jmoiron/sqlx v1.4.0
    github.com/joho/godotenv v1.5.1
    github.com/lib/pq v1.10.9
    golang.org/x/crypto v0.24.0
)
```

## Makefile Template

```makefile
# Build variables
BINARY_NAME=app
BUILD_DIR=./bin
GO_FILES=$(shell find . -name '*.go' -type f)

# Database variables
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: all build clean run dev test migrate-up migrate-down migrate-status

all: clean build

build:
	@echo "Building..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/main.go

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

run: build
	@echo "Running..."
	@$(BUILD_DIR)/$(BINARY_NAME)

dev:
	@echo "Running in development mode..."
	@air

test:
	@echo "Running tests..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

# Database migrations
migrate-up:
	@echo "Running migrations..."
	@migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	@echo "Reverting migrations..."
	@migrate -path ./migrations -database "$(DB_URL)" down 1

migrate-status:
	@echo "Migration status..."
	@migrate -path ./migrations -database "$(DB_URL)" version

migrate-create:
	@echo "Creating migration $(NAME)..."
	@migrate create -ext sql -dir ./migrations -seq $(NAME)

# Docker commands
docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose logs -f
```

## Docker Compose Template

```yaml
version: '3.8'

services:
  db:
    image: postgres:16-alpine
    container_name: ${PROJECT_NAME}_db
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    ports:
      - "${DB_PORT}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  app:
    build:
      context: .
      dockerfile: docker/Dockerfile
    container_name: ${PROJECT_NAME}_app
    environment:
      - DB_HOST=db
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - JWT_SECRET=${JWT_SECRET}
      - PORT=${PORT}
    ports:
      - "${PORT}:${PORT}"
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - ./:/app
    command: air

volumes:
  postgres_data:
```

## Environment Variables (.env.example)

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=projectname_db

# Application Configuration
PORT=8080
JWT_SECRET=your-secret-key-here
ENV=development

# External Services (if needed)
OPENAI_API_KEY=your-api-key-here

# Project Name (for Docker)
PROJECT_NAME=projectname
```

## Testing Structure

```
internal/
├── [domain]/
│   ├── handler_test.go
│   ├── service_test.go
│   ├── repository_test.go
│   └── testdata/
│       └── fixtures.go
└── testutils/
    ├── db.go          # Test database setup
    └── mocks.go       # Common mocks
```

### Test Template (service_test.go)

```go
package domain_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/yourusername/projectname/internal/domain"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, entity *domain.Entity) error {
    args := m.Called(ctx, entity)
    return args.Error(0)
}

// Implement other mock methods...

func TestService_CreateEntity(t *testing.T) {
    tests := []struct {
        name    string
        req     *domain.CreateEntityRequest
        wantErr bool
    }{
        {
            name: "valid entity",
            req: &domain.CreateEntityRequest{
                Name: "Test Entity",
            },
            wantErr: false,
        },
        {
            name: "empty name",
            req: &domain.CreateEntityRequest{
                Name: "",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := new(MockRepository)
            service := domain.NewService(mockRepo)
            
            if !tt.wantErr {
                mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
            }
            
            _, err := service.CreateEntity(context.Background(), tt.req)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                mockRepo.AssertExpectations(t)
            }
        })
    }
}
```

## Configuration Management Patterns

### 1. Environment-Based Configuration

```go
package config

import (
    "os"
    "strconv"
)

type Config struct {
    Database DatabaseConfig
    Server   ServerConfig
    JWT      JWTConfig
}

type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    Name     string
}

type ServerConfig struct {
    Port string
    Env  string
}

type JWTConfig struct {
    Secret string
}

func Load() (*Config, error) {
    dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
    if err != nil {
        dbPort = 5432 // default
    }
    
    return &Config{
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     dbPort,
            User:     getEnv("DB_USER", "postgres"),
            Password: os.Getenv("DB_PASSWORD"),
            Name:     getEnv("DB_NAME", "app_db"),
        },
        Server: ServerConfig{
            Port: getEnv("PORT", "8080"),
            Env:  getEnv("ENV", "development"),
        },
        JWT: JWTConfig{
            Secret: os.Getenv("JWT_SECRET"),
        },
    }, nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### 2. Migration Management

```go
package config

import (
    "embed"
    "fmt"
    
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"
    "github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(db *sqlx.DB) error {
    driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("failed to create migration driver: %w", err)
    }
    
    source, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return fmt.Errorf("failed to create migration source: %w", err)
    }
    
    m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
    if err != nil {
        return fmt.Errorf("failed to create migration instance: %w", err)
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run migrations: %w", err)
    }
    
    return nil
}
```

## Project Commands

### Initial Setup
```bash
# Create project structure
mkdir -p cmd config internal/{user,auth,constants,interfaces,jobs,models,utils/{dbutils,errorutils}} migrations scripts docker bin tmp

# Initialize Go module
go mod init github.com/yourusername/projectname

# Install dependencies
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors
go get github.com/jmoiron/sqlx
go get github.com/lib/pq
go get github.com/golang-jwt/jwt/v5
go get github.com/golang-migrate/migrate/v4
go get github.com/google/uuid
go get github.com/joho/godotenv
go get golang.org/x/crypto

# Create initial migration
make migrate-create NAME=init

# Start development
make docker-up
make migrate-up
make dev
```

This template provides a production-ready structure for Go MCP projects with clean architecture, proper separation of concerns, and modern Go best practices.
#!/bin/bash

# Go MCP Project Setup Script
# Usage: ./setup-go-mcp-project.sh <project-name> <github-username>

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ "$#" -ne 2 ]; then
    echo -e "${RED}Error: Invalid number of arguments${NC}"
    echo "Usage: $0 <project-name> <github-username>"
    echo "Example: $0 my-mcp-server myusername"
    exit 1
fi

PROJECT_NAME=$1
GITHUB_USERNAME=$2
MODULE_NAME="github.com/${GITHUB_USERNAME}/${PROJECT_NAME}"

echo -e "${GREEN}🚀 Setting up Go MCP project: ${PROJECT_NAME}${NC}"

# Create project directory
mkdir -p "${PROJECT_NAME}"
cd "${PROJECT_NAME}"

# Create directory structure
echo -e "${YELLOW}📁 Creating directory structure...${NC}"
mkdir -p cmd \
         config \
         internal/{auth,constants,interfaces,jobs,models,utils/{dbutils,errorutils}} \
         internal/user \
         internal/example \
         migrations \
         scripts \
         docker \
         bin \
         tmp

# Create .gitignore
echo -e "${YELLOW}📝 Creating .gitignore...${NC}"
cat > .gitignore << 'EOF'
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment files
.env
.env.local
.env.*.local

# IDE files
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Temporary files
tmp/
temp/

# Air live reload
.air.toml

# Database
*.db
*.sqlite

# Logs
*.log
EOF

# Create .env.example
echo -e "${YELLOW}🔐 Creating .env.example...${NC}"
cat > .env.example << 'EOF'
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=${PROJECT_NAME}_db

# Application Configuration
PORT=8080
JWT_SECRET=your-secret-key-here
ENV=development

# External Services (if needed)
# OPENAI_API_KEY=your-api-key-here

# Project Name (for Docker)
PROJECT_NAME=${PROJECT_NAME}
EOF

# Initialize Go module
echo -e "${YELLOW}📦 Initializing Go module...${NC}"
go mod init "${MODULE_NAME}"

# Create main.go
echo -e "${YELLOW}💻 Creating main.go...${NC}"
cat > cmd/main.go << EOF
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "${MODULE_NAME}/config"
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
EOF

# Create config files
echo -e "${YELLOW}⚙️  Creating config files...${NC}"

# config/db.go
cat > config/db.go << 'EOF'
package config

import (
    "fmt"
    "os"
    "time"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

func InitDB() (*sqlx.DB, error) {
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
EOF

# config/migrations.go
cat > config/migrations.go << 'EOF'
package config

import (
    "fmt"
    
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/jmoiron/sqlx"
)

func RunMigrations(db *sqlx.DB) error {
    driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("failed to create migration driver: %w", err)
    }
    
    m, err := migrate.NewWithDatabaseInstance(
        "file://migrations",
        "postgres", driver)
    if err != nil {
        return fmt.Errorf("failed to create migration instance: %w", err)
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run migrations: %w", err)
    }
    
    return nil
}
EOF

# config/routes.go
cat > config/routes.go << EOF
package config

import (
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    
    "${MODULE_NAME}/internal/user"
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
    }
    
    return router
}
EOF

# Create constants
echo -e "${YELLOW}📋 Creating constants...${NC}"
cat > internal/constants/errors.go << 'EOF'
package constants

import "errors"

var (
    ErrNotFound         = errors.New("resource not found")
    ErrAlreadyExists    = errors.New("resource already exists")
    ErrInvalidInput     = errors.New("invalid input")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrInternalServer   = errors.New("internal server error")
)
EOF

# Create initial migration
echo -e "${YELLOW}🗄️  Creating initial migration...${NC}"
cat > migrations/000001_init.up.sql << 'EOF'
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EOF

cat > migrations/000001_init.down.sql << 'EOF'
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP EXTENSION IF EXISTS "uuid-ossp";
EOF

# Create Makefile
echo -e "${YELLOW}🔧 Creating Makefile...${NC}"
cat > Makefile << 'EOF'
# Build variables
BINARY_NAME=app
BUILD_DIR=./bin
GO_FILES=$(shell find . -name '*.go' -type f)

# Database variables
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

-include .env

.PHONY: all build clean run dev test migrate-up migrate-down

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

migrate-up:
	@echo "Running migrations..."
	@migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	@echo "Reverting migrations..."
	@migrate -path ./migrations -database "$(DB_URL)" down 1

migrate-create:
	@echo "Creating migration $(NAME)..."
	@migrate create -ext sql -dir ./migrations -seq $(NAME)
EOF

# Create docker-compose.yml
echo -e "${YELLOW}🐳 Creating docker-compose.yml...${NC}"
cat > docker-compose.yml << EOF
version: '3.8'

services:
  db:
    image: postgres:16-alpine
    container_name: \${PROJECT_NAME}_db
    environment:
      POSTGRES_USER: \${DB_USER}
      POSTGRES_PASSWORD: \${DB_PASSWORD}
      POSTGRES_DB: \${DB_NAME}
    ports:
      - "\${DB_PORT}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
EOF

# Create basic user domain files
echo -e "${YELLOW}👤 Creating user domain files...${NC}"

# User model
cat > internal/user/model.go << EOF
package user

import (
    "time"
    
    "github.com/google/uuid"
)

type User struct {
    ID           uuid.UUID \`json:"id" db:"id"\`
    Email        string    \`json:"email" db:"email"\`
    PasswordHash string    \`json:"-" db:"password_hash"\`
    Name         string    \`json:"name" db:"name"\`
    CreatedAt    time.Time \`json:"created_at" db:"created_at"\`
    UpdatedAt    time.Time \`json:"updated_at" db:"updated_at"\`
}

type SignUpRequest struct {
    Email    string \`json:"email" binding:"required,email"\`
    Password string \`json:"password" binding:"required,min=8"\`
    Name     string \`json:"name" binding:"required"\`
}

type SignInRequest struct {
    Email    string \`json:"email" binding:"required,email"\`
    Password string \`json:"password" binding:"required"\`
}

type AuthResponse struct {
    User         *User  \`json:"user"\`
    AccessToken  string \`json:"access_token"\`
    RefreshToken string \`json:"refresh_token"\`
}
EOF

# User repository
cat > internal/user/repository.go << EOF
package user

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "${MODULE_NAME}/internal/constants"
)

type Repository interface {
    Create(ctx context.Context, user *User) error
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type repository struct {
    db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
    return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *User) error {
    query := \`
        INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
        VALUES (\$1, \$2, \$3, \$4, \$5, \$6)
    \`
    
    _, err := r.db.ExecContext(ctx, query,
        user.ID,
        user.Email,
        user.PasswordHash,
        user.Name,
        user.CreatedAt,
        user.UpdatedAt,
    )
    
    return err
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    query := "SELECT * FROM users WHERE email = \$1"
    
    err := r.db.GetContext(ctx, &user, query, email)
    if err == sql.ErrNoRows {
        return nil, constants.ErrNotFound
    }
    
    return &user, err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
    var user User
    query := "SELECT * FROM users WHERE id = \$1"
    
    err := r.db.GetContext(ctx, &user, query, id)
    if err == sql.ErrNoRows {
        return nil, constants.ErrNotFound
    }
    
    return &user, err
}
EOF

# User service
cat > internal/user/service.go << EOF
package user

import (
    "context"
    "errors"
    "time"
    
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "${MODULE_NAME}/internal/constants"
)

type Service interface {
    SignUp(ctx context.Context, req *SignUpRequest) (*User, error)
    SignIn(ctx context.Context, req *SignInRequest) (*User, error)
}

type service struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &service{repo: repo}
}

func (s *service) SignUp(ctx context.Context, req *SignUpRequest) (*User, error) {
    // Check if user exists
    _, err := s.repo.GetByEmail(ctx, req.Email)
    if err == nil {
        return nil, constants.ErrAlreadyExists
    }
    if err != constants.ErrNotFound {
        return nil, err
    }
    
    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    
    // Create user
    user := &User{
        ID:           uuid.New(),
        Email:        req.Email,
        PasswordHash: string(hashedPassword),
        Name:         req.Name,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    if err := s.repo.Create(ctx, user); err != nil {
        return nil, err
    }
    
    return user, nil
}

func (s *service) SignIn(ctx context.Context, req *SignInRequest) (*User, error) {
    user, err := s.repo.GetByEmail(ctx, req.Email)
    if err != nil {
        if err == constants.ErrNotFound {
            return nil, constants.ErrUnauthorized
        }
        return nil, err
    }
    
    // Verify password
    err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
    if err != nil {
        return nil, constants.ErrUnauthorized
    }
    
    return user, nil
}
EOF

# User handler
cat > internal/user/handler.go << EOF
package user

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "${MODULE_NAME}/internal/constants"
)

type Handler struct {
    service Service
}

func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) SignUp(c *gin.Context) {
    var req SignUpRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    user, err := h.service.SignUp(c.Request.Context(), &req)
    if err != nil {
        if err == constants.ErrAlreadyExists {
            c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (h *Handler) SignIn(c *gin.Context) {
    var req SignInRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    user, err := h.service.SignIn(c.Request.Context(), &req)
    if err != nil {
        if err == constants.ErrUnauthorized {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign in"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"user": user})
}
EOF

# Create README
echo -e "${YELLOW}📚 Creating README.md...${NC}"
cat > README.md << EOF
# ${PROJECT_NAME}

A Go-based MCP (Model Context Protocol) server built with clean architecture principles.

## Getting Started

### Prerequisites

- Go 1.23+
- PostgreSQL
- Docker and Docker Compose (optional, for local development)
- golang-migrate CLI tool

### Installation

1. Clone the repository:
\`\`\`bash
git clone https://github.com/${GITHUB_USERNAME}/${PROJECT_NAME}.git
cd ${PROJECT_NAME}
\`\`\`

2. Copy the environment file:
\`\`\`bash
cp .env.example .env
\`\`\`

3. Start the database:
\`\`\`bash
docker-compose up -d
\`\`\`

4. Install dependencies:
\`\`\`bash
go mod download
\`\`\`

5. Run migrations:
\`\`\`bash
make migrate-up
\`\`\`

6. Start the server:
\`\`\`bash
make run
\`\`\`

### Development

For hot-reload during development:
\`\`\`bash
go install github.com/cosmtrek/air@latest
make dev
\`\`\`

### Testing

Run tests:
\`\`\`bash
make test
\`\`\`

### API Endpoints

- \`POST /api/users/signup\` - User registration
- \`POST /api/users/signin\` - User login

## Project Structure

\`\`\`
├── cmd/                    # Application entry points
├── config/                 # Configuration
├── internal/               # Private application code
│   ├── user/              # User domain
│   ├── auth/              # Authentication
│   └── ...                # Other domains
├── migrations/            # Database migrations
└── docker-compose.yml     # Local development setup
\`\`\`

## License

MIT
EOF

# Install dependencies
echo -e "${YELLOW}📦 Installing dependencies...${NC}"
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors
go get github.com/jmoiron/sqlx
go get github.com/lib/pq
go get github.com/golang-jwt/jwt/v5
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/google/uuid
go get github.com/joho/godotenv
go get golang.org/x/crypto

# Install development tools
echo -e "${YELLOW}🛠️  Installing development tools...${NC}"
go install github.com/cosmtrek/air@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create .air.toml for hot reload
cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/main.go"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
EOF

echo -e "${GREEN}✅ Project setup complete!${NC}"
echo -e "${GREEN}📂 Project created at: $(pwd)${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. cd ${PROJECT_NAME}"
echo "2. Update .env with your configuration"
echo "3. docker-compose up -d  # Start database"
echo "4. make migrate-up      # Run migrations"
echo "5. make dev            # Start development server"
echo ""
echo -e "${GREEN}Happy coding! 🚀${NC}"
EOF

chmod +x setup-go-mcp-project.sh
</invoke>
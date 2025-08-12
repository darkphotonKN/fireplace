# Go MCP Code Templates

Reusable code templates for common patterns in Go MCP projects.

## Error Handling Templates

### Custom Error Types (internal/constants/errors.go)

```go
package constants

import "errors"

// Domain errors
var (
    ErrNotFound          = errors.New("resource not found")
    ErrAlreadyExists     = errors.New("resource already exists")
    ErrInvalidInput      = errors.New("invalid input")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrForbidden         = errors.New("forbidden")
    ErrInternalServer    = errors.New("internal server error")
    ErrBadRequest        = errors.New("bad request")
    ErrConflict          = errors.New("resource conflict")
    ErrValidationFailed  = errors.New("validation failed")
)

// Database errors
var (
    ErrDuplicateKey      = errors.New("duplicate key violation")
    ErrForeignKey        = errors.New("foreign key violation")
    ErrNullConstraint    = errors.New("null constraint violation")
    ErrCheckConstraint   = errors.New("check constraint violation")
)
```

### Error Utils (internal/utils/errorutils/errors.go)

```go
package errorutils

import (
    "database/sql"
    "errors"
    "strings"
    
    "github.com/lib/pq"
    "github.com/yourusername/projectname/internal/constants"
)

// AnalyzeError converts database errors to domain errors
func AnalyzeError(err error) error {
    if err == nil {
        return nil
    }
    
    // Check for no rows error
    if errors.Is(err, sql.ErrNoRows) {
        return constants.ErrNotFound
    }
    
    // Check for PostgreSQL errors
    var pgErr *pq.Error
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505": // unique_violation
            return constants.ErrDuplicateKey
        case "23503": // foreign_key_violation
            return constants.ErrForeignKey
        case "23502": // not_null_violation
            return constants.ErrNullConstraint
        case "23514": // check_violation
            return constants.ErrCheckConstraint
        }
    }
    
    return err
}

// IsDuplicateKeyError checks if error is a duplicate key error
func IsDuplicateKeyError(err error) bool {
    var pgErr *pq.Error
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23505"
    }
    return false
}

// ExtractConstraintName extracts the constraint name from a PostgreSQL error
func ExtractConstraintName(err error) string {
    var pgErr *pq.Error
    if errors.As(err, &pgErr) {
        return pgErr.Constraint
    }
    return ""
}
```

## Authentication Templates

### JWT Utilities (internal/auth/jwt.go)

```go
package auth

import (
    "errors"
    "fmt"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

var (
    ErrInvalidToken = errors.New("invalid token")
    ErrExpiredToken = errors.New("token expired")
)

type TokenClaims struct {
    UserID uuid.UUID `json:"user_id"`
    Email  string    `json:"email"`
    jwt.RegisteredClaims
}

type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
}

type JWTManager struct {
    secretKey           string
    accessTokenDuration time.Duration
    refreshTokenDuration time.Duration
}

func NewJWTManager(secretKey string) *JWTManager {
    return &JWTManager{
        secretKey:            secretKey,
        accessTokenDuration:  15 * time.Minute,
        refreshTokenDuration: 7 * 24 * time.Hour,
    }
}

func (j *JWTManager) GenerateTokenPair(userID uuid.UUID, email string) (*TokenPair, error) {
    accessToken, err := j.generateToken(userID, email, j.accessTokenDuration)
    if err != nil {
        return nil, fmt.Errorf("failed to generate access token: %w", err)
    }
    
    refreshToken, err := j.generateToken(userID, email, j.refreshTokenDuration)
    if err != nil {
        return nil, fmt.Errorf("failed to generate refresh token: %w", err)
    }
    
    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}

func (j *JWTManager) generateToken(userID uuid.UUID, email string, duration time.Duration) (string, error) {
    claims := TokenClaims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ID:        uuid.New().String(),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.secretKey))
}

func (j *JWTManager) ValidateToken(tokenString string) (*TokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(j.secretKey), nil
    })
    
    if err != nil {
        return nil, ErrInvalidToken
    }
    
    claims, ok := token.Claims.(*TokenClaims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }
    
    if time.Now().After(claims.ExpiresAt.Time) {
        return nil, ErrExpiredToken
    }
    
    return claims, nil
}
```

### Auth Middleware (internal/auth/middleware.go)

```go
package auth

import (
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
            c.Abort()
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "bearer token required"})
            c.Abort()
            return
        }
        
        claims, err := jwtManager.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
            c.Abort()
            return
        }
        
        // Set user info in context
        c.Set("userID", claims.UserID)
        c.Set("email", claims.Email)
        
        c.Next()
    }
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
    userID, exists := c.Get("userID")
    if !exists {
        return uuid.Nil, ErrUnauthorized
    }
    
    id, ok := userID.(uuid.UUID)
    if !ok {
        return uuid.Nil, ErrUnauthorized
    }
    
    return id, nil
}
```

## Database Templates

### Transaction Helper (internal/utils/dbutils/transaction.go)

```go
package dbutils

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/jmoiron/sqlx"
)

// TransactionFunc represents a function to be executed within a transaction
type TransactionFunc func(*sqlx.Tx) error

// WithTransaction executes a function within a database transaction
func WithTransaction(ctx context.Context, db *sqlx.DB, fn TransactionFunc) error {
    tx, err := db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p) // re-throw panic after rollback
        }
    }()
    
    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("failed to rollback: %v (original error: %w)", rbErr, err)
        }
        return err
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
```

### Pagination Helper (internal/utils/dbutils/pagination.go)

```go
package dbutils

import (
    "fmt"
    "math"
)

type PaginationParams struct {
    Page     int `json:"page" form:"page"`
    PageSize int `json:"page_size" form:"page_size"`
}

type PaginatedResponse struct {
    Data       interface{} `json:"data"`
    Page       int         `json:"page"`
    PageSize   int         `json:"page_size"`
    TotalItems int64       `json:"total_items"`
    TotalPages int         `json:"total_pages"`
}

func (p *PaginationParams) Validate() error {
    if p.Page < 1 {
        p.Page = 1
    }
    if p.PageSize < 1 {
        p.PageSize = 10
    }
    if p.PageSize > 100 {
        p.PageSize = 100
    }
    return nil
}

func (p *PaginationParams) Offset() int {
    return (p.Page - 1) * p.PageSize
}

func (p *PaginationParams) Limit() int {
    return p.PageSize
}

func NewPaginatedResponse(data interface{}, page, pageSize int, totalItems int64) *PaginatedResponse {
    totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
    
    return &PaginatedResponse{
        Data:       data,
        Page:       page,
        PageSize:   pageSize,
        TotalItems: totalItems,
        TotalPages: totalPages,
    }
}

// BuildPaginationQuery adds LIMIT and OFFSET to a query
func BuildPaginationQuery(baseQuery string, params *PaginationParams) string {
    return fmt.Sprintf("%s LIMIT %d OFFSET %d", baseQuery, params.Limit(), params.Offset())
}
```

## Background Jobs Templates

### Job Manager (internal/jobs/manager.go)

```go
package jobs

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
    
    "github.com/go-co-op/gocron/v2"
)

type Job interface {
    Name() string
    Schedule() string // cron expression
    Execute(ctx context.Context) error
}

type Manager struct {
    scheduler gocron.Scheduler
    jobs      []Job
    mu        sync.RWMutex
    ctx       context.Context
    cancel    context.CancelFunc
}

func NewManager() (*Manager, error) {
    scheduler, err := gocron.NewScheduler()
    if err != nil {
        return nil, fmt.Errorf("failed to create scheduler: %w", err)
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    
    return &Manager{
        scheduler: scheduler,
        jobs:      make([]Job, 0),
        ctx:       ctx,
        cancel:    cancel,
    }, nil
}

func (m *Manager) RegisterJob(job Job) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    _, err := m.scheduler.NewJob(
        gocron.CronJob(job.Schedule(), false),
        gocron.NewTask(func() {
            if err := job.Execute(m.ctx); err != nil {
                log.Printf("Job %s failed: %v", job.Name(), err)
            }
        }),
        gocron.WithName(job.Name()),
    )
    
    if err != nil {
        return fmt.Errorf("failed to register job %s: %w", job.Name(), err)
    }
    
    m.jobs = append(m.jobs, job)
    return nil
}

func (m *Manager) Start() {
    log.Println("Starting job manager...")
    m.scheduler.Start()
}

func (m *Manager) Stop() {
    log.Println("Stopping job manager...")
    m.cancel()
    
    if err := m.scheduler.Shutdown(); err != nil {
        log.Printf("Error shutting down scheduler: %v", err)
    }
}

// Example job implementation
type CleanupJob struct {
    db *sqlx.DB
}

func NewCleanupJob(db *sqlx.DB) *CleanupJob {
    return &CleanupJob{db: db}
}

func (j *CleanupJob) Name() string {
    return "cleanup_expired_tokens"
}

func (j *CleanupJob) Schedule() string {
    return "0 2 * * *" // Run at 2 AM every day
}

func (j *CleanupJob) Execute(ctx context.Context) error {
    query := `DELETE FROM tokens WHERE expires_at < NOW()`
    
    result, err := j.db.ExecContext(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to delete expired tokens: %w", err)
    }
    
    rows, _ := result.RowsAffected()
    log.Printf("Deleted %d expired tokens", rows)
    
    return nil
}
```

## Validation Templates

### Request Validation (internal/utils/validation/validator.go)

```go
package validation

import (
    "fmt"
    "regexp"
    "strings"
)

var (
    emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
)

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
    var msgs []string
    for _, err := range v {
        msgs = append(msgs, fmt.Sprintf("%s: %s", err.Field, err.Message))
    }
    return strings.Join(msgs, "; ")
}

type Validator struct {
    errors ValidationErrors
}

func New() *Validator {
    return &Validator{
        errors: make(ValidationErrors, 0),
    }
}

func (v *Validator) Required(field, value string) *Validator {
    if strings.TrimSpace(value) == "" {
        v.errors = append(v.errors, ValidationError{
            Field:   field,
            Message: "is required",
        })
    }
    return v
}

func (v *Validator) MinLength(field, value string, min int) *Validator {
    if len(value) < min {
        v.errors = append(v.errors, ValidationError{
            Field:   field,
            Message: fmt.Sprintf("must be at least %d characters", min),
        })
    }
    return v
}

func (v *Validator) MaxLength(field, value string, max int) *Validator {
    if len(value) > max {
        v.errors = append(v.errors, ValidationError{
            Field:   field,
            Message: fmt.Sprintf("must be at most %d characters", max),
        })
    }
    return v
}

func (v *Validator) Email(field, value string) *Validator {
    if !emailRegex.MatchString(value) {
        v.errors = append(v.errors, ValidationError{
            Field:   field,
            Message: "must be a valid email address",
        })
    }
    return v
}

func (v *Validator) Phone(field, value string) *Validator {
    if !phoneRegex.MatchString(value) {
        v.errors = append(v.errors, ValidationError{
            Field:   field,
            Message: "must be a valid phone number",
        })
    }
    return v
}

func (v *Validator) Valid() bool {
    return len(v.errors) == 0
}

func (v *Validator) Errors() ValidationErrors {
    return v.errors
}

// Usage example
func ValidateUserRequest(req *CreateUserRequest) error {
    v := New()
    
    v.Required("email", req.Email).Email("email", req.Email)
    v.Required("password", req.Password).MinLength("password", req.Password, 8)
    v.Required("name", req.Name).MaxLength("name", req.Name, 100)
    
    if !v.Valid() {
        return v.Errors()
    }
    
    return nil
}
```

## HTTP Response Templates

### Standard Response Format (internal/utils/response/response.go)

```go
package response

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
)

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type ErrorInfo struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

type Meta struct {
    Page       int   `json:"page,omitempty"`
    PageSize   int   `json:"page_size,omitempty"`
    TotalItems int64 `json:"total_items,omitempty"`
    TotalPages int   `json:"total_pages,omitempty"`
}

// Success responses
func OK(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Success: true,
        Data:    data,
    })
}

func Created(c *gin.Context, data interface{}) {
    c.JSON(http.StatusCreated, Response{
        Success: true,
        Data:    data,
    })
}

func NoContent(c *gin.Context) {
    c.Status(http.StatusNoContent)
}

// Error responses
func BadRequest(c *gin.Context, message string, details interface{}) {
    c.JSON(http.StatusBadRequest, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    "BAD_REQUEST",
            Message: message,
            Details: details,
        },
    })
}

func Unauthorized(c *gin.Context, message string) {
    c.JSON(http.StatusUnauthorized, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    "UNAUTHORIZED",
            Message: message,
        },
    })
}

func Forbidden(c *gin.Context, message string) {
    c.JSON(http.StatusForbidden, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    "FORBIDDEN",
            Message: message,
        },
    })
}

func NotFound(c *gin.Context, message string) {
    c.JSON(http.StatusNotFound, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    "NOT_FOUND",
            Message: message,
        },
    })
}

func InternalServerError(c *gin.Context, message string) {
    c.JSON(http.StatusInternalServerError, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    "INTERNAL_SERVER_ERROR",
            Message: message,
        },
    })
}

// Paginated response
func Paginated(c *gin.Context, data interface{}, page, pageSize int, totalItems int64, totalPages int) {
    c.JSON(http.StatusOK, Response{
        Success: true,
        Data:    data,
        Meta: &Meta{
            Page:       page,
            PageSize:   pageSize,
            TotalItems: totalItems,
            TotalPages: totalPages,
        },
    })
}
```

## Database Migration Templates

### Initial Schema (migrations/000001_init.up.sql)

```sql
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
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Example domain table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_user_id ON products(user_id);
CREATE INDEX idx_products_name ON products(name);
CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Sessions/tokens table for auth
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

### Down Migration (migrations/000001_init.down.sql)

```sql
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP EXTENSION IF EXISTS "uuid-ossp";
```

## Dockerfile Template

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
```

## GitHub Actions CI/CD Template

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

env:
  GO_VERSION: '1.23'

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: testuser
          POSTGRES_PASSWORD: testpass
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run tests
      env:
        DB_HOST: localhost
        DB_PORT: 5432
        DB_USER: testuser
        DB_PASSWORD: testpass
        DB_NAME: testdb
        JWT_SECRET: test-secret
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -func=coverage.out
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}
    
    - name: golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [test, lint]
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}
    
    - name: Build
      run: go build -v ./cmd/main.go
    
    - name: Build Docker image
      run: docker build -t ${{ github.repository }}:${{ github.sha }} .
```

These templates provide a comprehensive starting point for building production-ready Go MCP projects with best practices, proper error handling, authentication, database management, and deployment configurations.
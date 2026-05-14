# Fireplace Backend Architecture

## Overview
The Fireplace backend is a Go-based REST API server built with the Gin web framework. It provides productivity and task management features with AI-powered insights.

## Tech Stack
- **Language**: Go
- **Web Framework**: Gin (HTTP router and middleware)
- **Database**: PostgreSQL with sqlx
- **Authentication**: JWT tokens (in development)
- **AI Integration**: Custom AI generators for insights
- **CORS**: Enabled for frontend integration

## Project Structure

```
fireplace/
├── cmd/
│   └── main.go                 # Application entry point
├── config/
│   ├── db.go                   # Database configuration
│   ├── migrations.go           # Database migrations
│   └── routes.go              # Route setup and middleware
├── internal/
│   ├── ai/                    # AI generation services
│   │   ├── checklist_generator.go
│   │   ├── focus_generator.go
│   │   ├── generator.go
│   │   └── search_terms_generator.go
│   ├── auth/                  # Authentication
│   │   └── jwt.go
│   ├── checklistitems/        # Checklist domain
│   │   ├── handler.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── concepts/              # Concept definitions
│   │   └── concepts.go
│   ├── constants/             # Application constants
│   │   ├── ai.go
│   │   ├── api.go
│   │   ├── checklistitems.go
│   │   ├── discovery.go
│   │   └── plans.go
│   ├── discovery/             # Resource discovery
│   │   └── discovery.go       # YouTube video finder
│   ├── insights/              # AI insights domain
│   │   ├── handler.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── interfaces/            # Shared interfaces
│   │   └── ai.go
│   ├── jobs/                  # Background jobs
│   │   ├── daily_reset_job.go
│   │   ├── manager.go
│   │   └── scheduled_items_job.go
│   ├── models/                # Shared data models
│   │   └── entities.go
│   ├── plans/                 # Plans domain
│   │   ├── handler.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── types/                 # Type definitions
│   │   └── general.go
│   ├── user/                  # User domain
│   │   ├── handler.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── useranalytics/         # User analytics
│   │   ├── handler.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   └── utils/                 # Utility functions
│       ├── dbutils/
│       │   └── helpers.go
│       └── errorutils/
│           ├── analyzer.go
│           └── helpers.go
```

## Architecture Patterns

### 1. Layered Architecture
Each domain follows a consistent three-layer pattern:

- **Handler Layer**: HTTP request/response handling, validation
- **Service Layer**: Business logic, orchestration
- **Repository Layer**: Database operations

Example flow:
```
HTTP Request → Handler → Service → Repository → Database
             ← Handler ← Service ← Repository ←
```

### 2. Domain Organization
The codebase is organized by business domains:
- **Users**: User management and authentication
- **Plans**: Project/learning plan management
- **Checklist Items**: Task and todo management
- **Insights**: AI-powered suggestions and insights
- **Discovery**: External resource discovery (YouTube)
- **User Analytics**: Usage tracking and analytics

### 3. Dependency Injection
Services are initialized with their dependencies in `config/routes.go`:
```go
userRepo := user.NewRepository(db)
userService := user.NewService(userRepo)
userHandler := user.NewHandler(userService)
```

## Core Components

### Database Layer
- Uses PostgreSQL with sqlx for type-safe SQL
- Base models provide common fields (ID, timestamps)
- UUID-based primary keys
- Supports migrations

### Authentication & Authorization
- JWT-based authentication (implementation in progress)
- Currently uses static user ID for development
- Middleware for protected routes

### Background Jobs
The job manager runs two critical background tasks:

1. **Daily Reset Job**: Resets daily tasks at midnight
2. **Scheduled Items Job**: Processes scheduled checklist items

Jobs are managed by a central JobManager that handles:
- Concurrent job execution
- Graceful shutdown
- Error handling

### AI Integration
Multiple AI generators provide intelligent features:

1. **Checklist Generator**: Creates task suggestions
2. **Focus Generator**: Generates focus recommendations
3. **Search Terms Generator**: Creates search queries for resource discovery

### Resource Discovery
YouTube video finder that:
- Crawls YouTube search results
- Extracts video IDs from HTML
- Returns relevant learning resources

## Data Models

### Base Models
```go
type BaseDBDateModel struct {
    ID        uuid.UUID
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Core Entities

**User**
- Email, Name, Password
- Base authentication entity

**Plan**
- Name, Focus, Description
- Type (development/learning)
- Daily reset flag
- Belongs to User

**ChecklistItem**
- Description, Done status
- Scope (daily/longterm)
- Schedule time (optional)
- Archive status
- Belongs to Plan

## API Design

### RESTful Conventions
- Standard HTTP methods (GET, POST, PATCH, DELETE)
- Resource-based URLs
- JSON request/response format
- Consistent error handling

### Response Format
```go
gin.H{
    "statusCode": http.StatusCode,
    "message": "Operation message",
    "result": data
}
```

### Route Groups
- `/api/users` - User management
- `/api/plans` - Plan operations
- `/api/plans/:id/checklists` - Checklist items
- `/api/insights` - AI-powered insights
- `/api/analytics` - User analytics

## Configuration

### Environment Variables
- Database connection string
- JWT secret (future)
- API keys for external services
- Port configuration

### CORS Configuration
Currently configured for development:
- Origin: `http://localhost:3010`
- Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
- Credentials: Enabled

## Development Workflow

### Running the Server
```bash
go run cmd/main.go
```
Server runs on port 6060 by default.

### Database Migrations
Handled in `config/migrations.go`

### Testing
- Unit tests for services
- Integration tests for handlers
- Mock repositories for testing

## Security Considerations

1. **Password Hashing**: Bcrypt for secure password storage
2. **Input Validation**: Request validation using Gin's binding
3. **SQL Injection Prevention**: Parameterized queries via sqlx
4. **CORS**: Restricted to specific origins
5. **Authentication**: JWT tokens (in development)

## Performance Optimizations

1. **Connection Pooling**: Database connection pool management
2. **Concurrent Processing**: Goroutines for background jobs
3. **Efficient Queries**: Indexed database columns
4. **Response Caching**: (Planned)

## Future Enhancements

1. **WebSocket Support**: Real-time updates
2. **Rate Limiting**: API throttling
3. **Caching Layer**: Redis integration
4. **Metrics & Monitoring**: Prometheus/Grafana
5. **API Versioning**: Version management strategy
6. **Full JWT Implementation**: Complete auth flow
7. **User Analytics**: Complete implementation
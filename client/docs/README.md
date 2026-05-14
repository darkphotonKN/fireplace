# Flow Client Documentation

This documentation provides comprehensive information about the Flow Client frontend application and its integration with the Fireplace Go backend.

## Quick Start

1. **Backend Setup**: Ensure the Fireplace Go server is running on port 6060
2. **Frontend Setup**: Run `npm run dev` to start the React client on port 3010
3. **Environment**: Copy `.env.example` to `.env.local` and configure API endpoints

## Documentation Structure

### [API Reference](./api-reference.md)
Complete documentation of all backend API endpoints including:
- Request/response formats
- Authentication requirements
- Error handling
- Data models

### [Backend Architecture](./backend-architecture.md)
Overview of the Go server architecture including:
- Project structure and organization
- Design patterns and conventions
- Core components and services
- Database models

### [Integration Guide](./integration-guide.md)
Frontend integration patterns and best practices:
- Environment setup
- API service layer
- Error handling
- State management
- Testing strategies

## Key Features

### Core Functionality
- **Plan Management**: Create and manage development/learning plans
- **Task System**: Daily and long-term task management with scheduling
- **AI Insights**: Intelligent task suggestions and learning recommendations
- **Progress Tracking**: Monitor completion rates and productivity metrics

### Technical Highlights
- **Next.js 15**: App Router with React 19
- **TypeScript**: Full type safety
- **Tailwind CSS**: Responsive design system
- **Real-time Updates**: Optimistic UI updates
- **Error Handling**: Comprehensive error management

## Development Workflow

### Prerequisites
- Node.js 18+
- Go 1.21+
- PostgreSQL database

### Getting Started
```bash
# Start backend server
cd ~/Documents/Code/Go/fireplace
go run cmd/main.go

# Start frontend development server
cd flow-client
npm install
npm run dev
```

### Testing
```bash
# Run linting
npm run lint

# Build for production
npm run build

# Start production server
npm start
```

## Project Status

### Current State
- ✅ Core CRUD operations for plans and tasks
- ✅ AI-powered task suggestions
- ✅ YouTube video recommendations
- ✅ Task scheduling and archiving
- ✅ Responsive UI with dark mode

### In Development
- 🔄 JWT authentication implementation
- 🔄 User analytics dashboard
- 🔄 Real-time notifications

### Future Enhancements
- 📋 WebSocket integration for live updates
- 📋 PWA capabilities with offline support
- 📋 Mobile application
- 📋 Team collaboration features

## API Endpoints Summary

| Category | Endpoint | Description |
|----------|----------|-------------|
| Users | `POST /api/users/signup` | Create account |
| Users | `POST /api/users/signin` | Login user |
| Plans | `GET /api/plans` | List all plans |
| Plans | `POST /api/plans` | Create new plan |
| Tasks | `GET /api/plans/:id/checklists` | Get tasks for plan |
| Tasks | `POST /api/plans/:id/checklists` | Create new task |
| Insights | `GET /api/insights/checklist-suggestion` | AI task suggestions |
| Videos | `GET /api/insights/suggest-videos` | Learning video recommendations |

## Support

For detailed implementation examples and troubleshooting:
- See [Integration Guide](./integration-guide.md) for common patterns
- Check [API Reference](./api-reference.md) for endpoint specifications
- Review [Backend Architecture](./backend-architecture.md) for system understanding

## Contributing

When making changes:
1. Follow existing code conventions
2. Update documentation as needed
3. Test both frontend and backend integration
4. Ensure TypeScript types match Go models
5. Run linting and build checks before committing
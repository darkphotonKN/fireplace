# SPECIFICATIONS.md

## Project Overview

**Name**: Flow Client
**Type**: Progressive Web Application (PWA)
**Framework**: Next.js 15.3.1 with App Router
**Language**: TypeScript 5.x
**UI Library**: React 19
**Styling**: Tailwind CSS with custom components
**State Management**: React hooks and Context API

## Technical Architecture

### Framework Configuration
- **Next.js 15**: Latest App Router architecture
- **Turbopack**: Development bundler for faster builds
- **TypeScript**: Strict type checking enabled
- **React 19**: Latest React features and optimizations

### Directory Structure
```
src/
├── app/                 # App Router pages and layouts
│   ├── layout.tsx       # Root layout with global providers
│   ├── page.tsx         # Landing/home page
│   ├── plans/           # Plan-related pages
│   ├── myplans/         # User's plan list
│   ├── create-plan/     # Plan creation flow
│   └── learning/        # Learning-specific sections
├── components/          # Reusable components
│   ├── ui/              # Base UI components (shadcn/ui inspired)
│   ├── notes/           # Note-taking components
│   └── [feature].tsx    # Feature-specific components
├── services/            # API integration layer
├── context/             # React Context providers
├── types/               # TypeScript type definitions
├── lib/                 # Utility functions
└── config/              # Configuration files
```

## Component Architecture

### Core Components

#### Layout Components
1. **LayoutWrapper** (`/components/LayoutWrapper.tsx`)
   - Global state providers
   - Authentication context
   - Theme management
   - Error boundaries

2. **LayoutContent** (`/components/LayoutContent.tsx`)
   - Main layout structure
   - Navigation integration
   - Responsive design container

3. **Header** (`/components/Header.tsx`)
   - Navigation bar
   - User profile dropdown
   - Quick actions menu

4. **Sidebar** (`/components/Sidebar.tsx`)
   - Plan navigation
   - Category filters
   - Quick links

#### Feature Components

1. **Todo** (`/components/Todo.tsx`)
   - Main checklist management interface
   - Daily/Long-term task tabs
   - Task CRUD operations
   - AI suggestions integration
   - Archive management

2. **UserProfile** (`/components/UserProfile.tsx`)
   - User information display
   - Settings management
   - Analytics dashboard

3. **Notes Components** (`/components/notes/`)
   - **NotesContainer**: Main notes management
   - **NoteCard**: Individual note display
   - **TaskNoteRelations**: Task-note linking

### UI Component Library

Built on shadcn/ui principles with custom implementations:

```typescript
// Base components in /components/ui/
- Button        // Variants: default, destructive, outline, secondary, ghost, link
- Input         // Text input with validation states
- Textarea      // Multi-line text input
- Card          // Container with header, content, footer
- Checkbox      // Controlled checkbox with label
- Badge         // Status/category indicators
- Progress      // Visual progress indicator
- Tabs          // Tab navigation component
- ScrollArea    // Custom scrollbar container
- Toast         // Notification system
```

## State Management

### Local State
- Component-level state using `useState`
- Complex state logic with `useReducer`
- Form state management with controlled components

### Global State
1. **Authentication Context**
   - User session management
   - Token refresh logic
   - Protected route handling

2. **Notes Context** (`/context/NotesContext.tsx`)
   - Centralized notes state
   - CRUD operations
   - Real-time synchronization

3. **Theme Context** (planned)
   - Dark/light mode toggle
   - User preferences persistence

### Data Fetching
- Server-side rendering with Next.js App Router
- Client-side fetching with native fetch API
- Optimistic updates for better UX
- Error handling with fallback UI

## API Integration

### Service Layer (`/services/`)

#### Core API Service (`api.ts`)
```typescript
// Base configuration
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6060'

// Main service methods
- fetchPlan(id: string): Promise<PlanDetailResponse>
- fetchChecklist(planId: string, scope: ScopeType): Promise<ChecklistResponse>
- createChecklistItem(description: string, planId: string, scope: ScopeType): Promise<ChecklistItem>
- updateChecklistItem(id: string, updates: UpdateRequest, planId: string): Promise<UpdateResponse>
- deleteChecklistItem(id: string, planId: string): Promise<DeleteResponse>
- scheduleChecklistItem(id: string, planId: string, time: Date): Promise<UpdateResponse>
- archiveChecklistItem(id: string, planId: string): Promise<UpdateResponse>
- getChecklistSuggestion(planId: string, scope: ScopeType): Promise<SuggestionResponse>
- getDailyInsights(planId: string): Promise<InsightsResponse>
- toggleDailyReset(planId: string): Promise<ApiResponse>
```

#### Notes Service (`notesService.ts`)
```typescript
// Note-specific operations
- fetchNotes(planId: string): Promise<Note[]>
- createNote(note: CreateNoteRequest): Promise<Note>
- updateNote(id: string, updates: UpdateNoteRequest): Promise<Note>
- deleteNote(id: string): Promise<void>
- linkNoteToTask(noteId: string, taskId: string): Promise<void>
```

### Type Definitions

```typescript
// Core types from /types/
interface ChecklistItem {
  id: string
  description: string
  done: boolean
  scheduledTime?: string
  scope?: 'daily' | 'longterm'
  archived: boolean
}

interface Plan {
  id: string
  name: string
  focus: string
  description: string
  planType: string
  dailyReset: boolean
}

interface Note {
  id: string
  title: string
  content: string
  planId: string
  taskIds?: string[]
  createdAt: string
  updatedAt: string
}
```

## Routing Structure

### App Router Pages

```
/                           # Landing page with focus selection
/myplans                    # User's plan list
/create-plan                # New plan creation
/plans/[planId]             # Plan detail with checklist
/plan/[planId]              # Alternative plan route (legacy)
/learning/genai             # GenAI learning section
/learning/microservices     # Microservices learning section
/daily-summary              # Daily task summary view
```

### Route Protection
- Client-side authentication checks
- Redirect to login for protected routes
- Loading states during authentication

## UI/UX Specifications

### Design System

#### Color Palette
```css
Primary: Gradient backgrounds with blur effects
Secondary: White/transparent overlays
Text: High contrast for readability
Accent: Interactive element highlights
```

#### Typography
- **Font Family**: Merriweather (serif)
- **Weights**: 300 (light), 400 (regular), 700 (bold), 900 (black)
- **Responsive sizing**: Tailwind's responsive utilities

#### Layout Principles
1. **Responsive Design**
   - Mobile-first approach
   - Breakpoints: sm (640px), md (768px), lg (1024px), xl (1280px)
   - Flexible grids and containers

2. **Glass Morphism**
   - Backdrop blur effects
   - Transparent backgrounds
   - Subtle shadows and borders

3. **Animation**
   - Smooth transitions (300ms default)
   - Hover states for interactive elements
   - Loading skeletons for data fetching

### Component Patterns

#### Form Patterns
```typescript
// Controlled input example
<input
  value={state.value}
  onChange={(e) => setState({ value: e.target.value })}
  className="w-full px-4 py-3 bg-transparent border-b"
/>
```

#### List Patterns
- Virtualized scrolling for long lists
- Drag-and-drop for reordering (planned)
- Bulk actions toolbar

#### Modal Patterns
- Portal-based rendering
- Focus trap implementation
- Escape key handling

## Performance Optimization

### Build-time Optimization
1. **Next.js Optimization**
   - Automatic code splitting
   - Image optimization with next/image
   - Font optimization with next/font
   - Static generation where possible

2. **Bundle Size**
   - Tree shaking enabled
   - Dynamic imports for large components
   - Lazy loading for routes

### Runtime Optimization
1. **React Optimization**
   - Memoization with React.memo
   - useMemo for expensive computations
   - useCallback for stable function references
   - Suspense boundaries for async components

2. **Data Fetching**
   - Request deduplication
   - Optimistic updates
   - Stale-while-revalidate strategy
   - Pagination for large datasets

### Caching Strategy
- Browser caching for static assets
- API response caching
- LocalStorage for user preferences
- SessionStorage for temporary data

## Testing Strategy

### Unit Testing (Planned)
```javascript
// Test framework: Jest + React Testing Library
- Component rendering tests
- User interaction tests
- API service tests
- Utility function tests
```

### Integration Testing (Planned)
```javascript
// E2E framework: Playwright
- User flow testing
- API integration tests
- Cross-browser testing
- Mobile responsiveness tests
```

### Test Coverage Goals
- Component coverage: 70%
- Service layer: 80%
- Utility functions: 90%
- Critical paths: 100%

## Build & Deployment

### Development
```bash
npm run dev          # Start dev server on port 3010 with Turbopack
npm run build        # Production build
npm run start        # Start production server
npm run lint         # Run ESLint
```

### Environment Configuration
```env
NEXT_PUBLIC_API_URL=http://localhost:6060
NEXT_PUBLIC_APP_ENV=development
NEXT_PUBLIC_VERSION=0.1.0
```

### Production Build
1. **Build Process**
   - TypeScript compilation
   - CSS optimization with PostCSS
   - Bundle optimization
   - Static file generation

2. **Output**
   - `.next/` directory with optimized bundles
   - Static assets in `public/`
   - Server and client bundles separated

### Deployment Targets
- **Vercel**: Optimized for Next.js (recommended)
- **Docker**: Containerized deployment
- **Static Export**: For CDN hosting (limited features)

## Browser Support

### Supported Browsers
- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions
- Mobile: iOS Safari, Chrome Android

### Progressive Enhancement
- Core functionality without JavaScript
- Enhanced features with JS enabled
- Fallbacks for unsupported features

## Accessibility (A11y)

### Standards
- WCAG 2.1 Level AA compliance
- Semantic HTML structure
- ARIA attributes where needed

### Implementation
- Keyboard navigation support
- Screen reader compatibility
- Focus management
- Color contrast ratios
- Alt text for images

## Security Considerations

### Client-side Security
1. **XSS Prevention**
   - React's automatic escaping
   - Content Security Policy headers
   - Input sanitization

2. **Authentication**
   - JWT token storage (httpOnly cookies preferred)
   - Token refresh mechanism
   - Session timeout handling

3. **Data Protection**
   - HTTPS enforcement
   - Sensitive data masking
   - Secure form submissions

### API Security
- CORS configuration
- Request validation
- Rate limiting awareness
- Error message sanitization

## Monitoring & Analytics (Planned)

### Performance Monitoring
- Core Web Vitals tracking
- API response time monitoring
- Error tracking with Sentry
- User session recording

### Analytics
- Page view tracking
- User interaction events
- Conversion funnel analysis
- Feature usage metrics

## Future Enhancements

### Planned Features
1. **Real-time Collaboration**
   - WebSocket integration
   - Live task updates
   - Shared planning sessions

2. **Progressive Web App**
   - Service worker implementation
   - Offline functionality
   - Push notifications
   - App installation prompt

3. **Advanced UI Features**
   - Dark mode theme
   - Customizable dashboards
   - Drag-and-drop task management
   - Rich text editor for notes

4. **AI Integration Enhancement**
   - Voice command support
   - Natural language task creation
   - Smart scheduling
   - Personalized insights

5. **Mobile Application**
   - React Native implementation
   - Native features integration
   - Synchronized state across platforms

6. **Gamification**
   - Achievement system
   - Progress streaks
   - Leaderboards
   - Rewards and badges

7. **Integration Ecosystem**
   - Calendar integration (Google, Outlook)
   - Project management tools (Jira, Trello)
   - Note-taking apps (Notion, Obsidian)
   - Time tracking tools

8. **Advanced Analytics**
   - Productivity metrics dashboard
   - Time analysis
   - Goal tracking
   - Custom reports

## Development Workflow

### Code Style
- ESLint configuration with Next.js rules
- Prettier for code formatting
- TypeScript strict mode
- Conventional commits

### Git Workflow
- Feature branch strategy
- PR reviews required
- Automated testing on PR
- Semantic versioning

### CI/CD Pipeline (Planned)
1. **Continuous Integration**
   - Automated testing
   - Linting and formatting checks
   - Build verification
   - Bundle size monitoring

2. **Continuous Deployment**
   - Staging environment deployment
   - Production deployment with approval
   - Rollback capability
   - Performance regression testing
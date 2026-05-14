# Frontend Integration Guide

## Environment Setup

### Prerequisites
1. **Go Backend**: The Fireplace server must be running on port 6060
2. **Database**: PostgreSQL database configured and accessible
3. **Node.js**: Version 18+ for the React frontend

### Environment Variables

Create `.env.local` in the flow-client root:
```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:6060
NEXT_PUBLIC_API_URL=http://localhost:6060
```

Configuration is centralized in `src/config/environment.ts`:
```typescript
export const config = {
  apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:6060',
  apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6060'
}
```

### CORS Configuration
The Go server is configured to accept requests from:
- `http://localhost:3010` (React dev server)

If you change the frontend port, update the CORS settings in `config/routes.go`.

## API Service Layer

### Central API Service
All API calls are centralized in `src/services/api.ts`:

```typescript
// Base API configuration
const API_BASE_URL = config.apiBaseUrl;

// Standard fetch wrapper with error handling
async function apiCall<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const response = await fetch(`${API_BASE_URL}/api${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  });

  return response.json();
}
```

### Response Types
All API responses follow this interface:
```typescript
interface ApiResponse<T> {
  statusCode: number;
  message: string;
  result: T;
}
```

### Data Models
Core TypeScript interfaces matching Go models:

```typescript
interface Plan {
  id: string;
  userId: string;
  name: string;
  focus: string;
  description: string;
  planType: 'development' | 'learning';
  dailyReset: boolean;
  created_at: string;
  updated_at: string;
}

interface ChecklistItem {
  id: string;
  description: string;
  done: boolean;
  sequence: string;
  scheduledTime?: string;
  scope: 'daily' | 'longterm';
  archived: boolean;
  planId: string;
  created_at: string;
  updated_at: string;
}
```

## Common Integration Patterns

### 1. Plan Management
```typescript
// Get all plans
const plans = await getPlans();

// Create a new plan
const newPlan = await createPlan({
  name: 'My Development Plan',
  focus: 'React and TypeScript',
  description: 'Learning modern web development',
  planType: 'development'
});

// Update plan
await updatePlan(planId, {
  name: 'Updated Plan Name'
});

// Delete plan
await deletePlan(planId);
```

### 2. Checklist Operations
```typescript
// Get tasks for a plan (with scope filter)
const dailyTasks = await getChecklistItems(planId, 'daily');
const longtermTasks = await getChecklistItems(planId, 'longterm');

// Create new task
const task = await createChecklistItem(planId, {
  description: 'Complete React component',
  scope: 'daily'
});

// Update task status
await updateChecklistItem(planId, taskId, {
  done: true
});

// Schedule a task
await scheduleChecklistItem(planId, taskId, {
  scheduledTime: '2024-01-01T10:00:00Z'
});

// Archive completed task
await archiveChecklistItem(planId, taskId);
```

### 3. AI Insights Integration
```typescript
// Get AI task suggestion
const suggestion = await getChecklistSuggestion(planId);

// Get daily insights from long-term goals
const dailyInsights = await getDailyInsights(planId);

// Get suggested learning videos
const videos = await getSuggestedVideos(planId);
```

## Error Handling

### API Error Response Format
```typescript
interface ApiError {
  statusCode: number;
  message: string;
  error?: string;
}
```

### Error Handling Pattern
```typescript
try {
  const result = await apiCall('/plans', options);
  if (result.statusCode !== 200) {
    throw new Error(result.message);
  }
  return result.result;
} catch (error) {
  console.error('API Error:', error);
  throw error;
}
```

### Common Error Scenarios
1. **Network Issues**: Server not running on port 6060
2. **CORS Errors**: Frontend running on different port
3. **Invalid UUIDs**: Malformed plan or task IDs
4. **Validation Errors**: Missing required fields
5. **Server Errors**: Database connection issues

## State Management Patterns

### Component-Level State
For simple operations, use local state:
```typescript
const [tasks, setTasks] = useState<ChecklistItem[]>([]);
const [loading, setLoading] = useState(false);
const [error, setError] = useState<string | null>(null);

useEffect(() => {
  const fetchTasks = async () => {
    setLoading(true);
    try {
      const result = await getChecklistItems(planId, 'daily');
      setTasks(result);
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  fetchTasks();
}, [planId]);
```

### Real-Time Updates
For immediate UI updates after API operations:
```typescript
// Optimistic updates
const handleToggleTask = async (taskId: string) => {
  // Optimistically update UI
  setTasks(prev => prev.map(task =>
    task.id === taskId
      ? { ...task, done: !task.done }
      : task
  ));

  try {
    await updateChecklistItem(planId, taskId, { done: !task.done });
  } catch (error) {
    // Revert on error
    setTasks(prev => prev.map(task =>
      task.id === taskId
        ? { ...task, done: !task.done }
        : task
    ));
    setError('Failed to update task');
  }
};
```

## Development Workflow

### 1. Start Backend Server
```bash
cd ~/Documents/Code/Go/fireplace
go run cmd/main.go
```
Server should be running on `http://localhost:6060`

### 2. Start Frontend Development Server
```bash
cd flow-client
npm run dev
```
Frontend should be running on `http://localhost:3010`

### 3. Verify Integration
Check that API calls work by opening browser DevTools Network tab and observing requests to `localhost:6060`.

## Testing Integration

### Manual Testing Checklist
- [ ] Plans CRUD operations work
- [ ] Task creation, updating, deletion work
- [ ] Task scheduling functions properly
- [ ] Archive/unarchive tasks work
- [ ] AI insights load correctly
- [ ] Video suggestions appear
- [ ] Error handling displays user-friendly messages

### Debug Tips

1. **Network Tab**: Monitor API requests/responses
2. **Console Logs**: Check for JavaScript errors
3. **Server Logs**: Go server prints request details
4. **CORS Issues**: Verify frontend port matches CORS config
5. **UUID Format**: Ensure UUIDs are properly formatted

### Common Issues

**Problem**: "Failed to fetch" error
**Solution**: Ensure Go server is running on port 6060

**Problem**: CORS policy error
**Solution**: Check frontend port matches CORS AllowOrigins

**Problem**: "Invalid uuid format" error
**Solution**: Verify plan/task IDs are valid UUIDs

**Problem**: Empty responses
**Solution**: Check database has test data or create via API

## Performance Considerations

### 1. API Call Optimization
- Batch multiple operations when possible
- Use query parameters for filtering
- Implement client-side caching for static data

### 2. Loading States
- Show loading indicators for async operations
- Implement skeleton screens for better UX
- Cache frequently accessed data

### 3. Error Recovery
- Retry failed requests with exponential backoff
- Provide fallback UI for error states
- Store critical data locally during network issues

## Security Considerations

### 1. API Communication
- Use HTTPS in production
- Validate all input data
- Sanitize user input before display

### 2. Authentication (Future)
- Store JWT tokens securely
- Implement token refresh logic
- Handle authentication errors gracefully

### 3. Data Validation
- Validate data on both client and server
- Use TypeScript for compile-time type checking
- Implement proper error boundaries

## Future Enhancements

1. **WebSocket Integration**: Real-time task updates
2. **Offline Support**: PWA capabilities with data sync
3. **Authentication Flow**: Complete JWT implementation
4. **Push Notifications**: Task reminders and insights
5. **Data Caching**: Implement proper caching strategy
6. **API Versioning**: Handle API version changes
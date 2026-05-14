# Fireplace API Reference

## Base Configuration
- **Base URL**: `http://localhost:6060/api`
- **Authentication**: JWT tokens (currently using static user ID: `11111111-1111-1111-1111-111111111111`)
- **CORS**: Configured for `http://localhost:3010`

## Response Format
All API responses follow this structure:
```json
{
  "statusCode": 200,
  "message": "Success message",
  "result": {} // Response data
}
```

## User Endpoints

### POST /api/users/signup
Create a new user account.

**Request Body:**
```json
{
  "email": "string",
  "name": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "statusCode": 201,
  "message": "Successfully created user."
}
```

### POST /api/users/signin
Authenticate user and receive JWT token.

**Request Body:**
```json
{
  "email": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully logged in.",
  "result": {
    "user": {},
    "token": "string"
  }
}
```

### GET /api/users/:id
Get user by ID.

**Parameters:**
- `id` (path): User UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully retrieved user.",
  "result": {
    "id": "uuid",
    "email": "string",
    "name": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
}
```

### GET /api/users
Get all users.

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully retrieved users.",
  "result": []
}
```

## Plans Endpoints

### GET /api/plans
Get all plans for the authenticated user.

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully retrieved all plans",
  "result": [
    {
      "id": "uuid",
      "userId": "uuid",
      "name": "string",
      "focus": "string",
      "description": "string",
      "planType": "string",
      "dailyReset": false,
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ]
}
```

### GET /api/plans/:id
Get a specific plan by ID.

**Parameters:**
- `id` (path): Plan UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully retrieved plan.",
  "result": {
    "id": "uuid",
    "userId": "uuid",
    "name": "string",
    "focus": "string",
    "description": "string",
    "planType": "string",
    "dailyReset": false,
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
}
```

### POST /api/plans
Create a new plan.

**Request Body:**
```json
{
  "name": "string",
  "focus": "string",
  "description": "string",
  "planType": "string"
}
```

**Response:**
```json
{
  "statusCode": 201,
  "message": "Successfully created plan",
  "result": {
    "id": "uuid",
    "userId": "uuid",
    "name": "string",
    "focus": "string",
    "description": "string",
    "planType": "string",
    "dailyReset": false,
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
}
```

### PATCH /api/plans/:id
Update an existing plan.

**Parameters:**
- `id` (path): Plan UUID

**Request Body:**
```json
{
  "name": "string",
  "focus": "string",
  "description": "string",
  "dailyReset": true
}
```
*Note: All fields are optional*

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully updated plan"
}
```

### PATCH /api/plans/:id/toggle-daily-reset
Toggle the daily reset setting for a plan.

**Parameters:**
- `id` (path): Plan UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully toggled daily reset"
}
```

### DELETE /api/plans/:id
Delete a plan.

**Parameters:**
- `id` (path): Plan UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully deleted plan"
}
```

## Checklist Endpoints

### GET /api/plans/:id/checklists
Get all checklist items for a plan.

**Parameters:**
- `id` (path): Plan UUID
- `scope` (query, optional): Filter by scope ("daily" or "longterm")

**Response:**
```json
{
  "statusCode": 200,
  "message": "successfully retrieved all checklist items.",
  "result": [
    {
      "id": "uuid",
      "description": "string",
      "done": false,
      "sequence": "string",
      "scheduledTime": "timestamp",
      "scope": "daily|longterm",
      "archived": false,
      "planId": "uuid",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ]
}
```

### GET /api/plans/:id/checklists/archived
Get all archived checklist items for a plan.

**Parameters:**
- `id` (path): Plan UUID
- `scope` (query, optional): Filter by scope

**Response:**
```json
{
  "statusCode": 200,
  "message": "successfully retrieved archived checklist items.",
  "result": []
}
```

### GET /api/plans/:id/checklists/upcoming
Get upcoming scheduled tasks for a plan.

**Parameters:**
- `id` (path): Plan UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully retrieved upcoming tasks.",
  "result": []
}
```

### GET /api/plans/:id/checklists/:checklist_id
Get a specific checklist item.

**Parameters:**
- `id` (path): Plan UUID
- `checklist_id` (path): Checklist item UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully retrieved checklist item.",
  "result": {}
}
```

### POST /api/plans/:id/checklists
Create a new checklist item.

**Parameters:**
- `id` (path): Plan UUID

**Request Body:**
```json
{
  "description": "string",
  "scope": "daily|longterm"
}
```

**Response:**
```json
{
  "statusCode": 201,
  "message": "successfully created checklist item.",
  "result": {
    "id": "uuid",
    "description": "string",
    "done": false,
    "sequence": "string",
    "scope": "daily",
    "archived": false,
    "planId": "uuid",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
}
```

### PATCH /api/plans/:id/checklists/:checklist_id
Update a checklist item.

**Parameters:**
- `id` (path): Plan UUID
- `checklist_id` (path): Checklist item UUID

**Request Body:**
```json
{
  "description": "string",
  "done": true,
  "sequence": true,
  "scope": "string",
  "archived": true
}
```
*Note: All fields are optional*

**Response:**
```json
{
  "statusCode": 200,
  "message": "successfully update checklist item.",
  "result": "success"
}
```

### DELETE /api/plans/:id/checklists/:checklist_id
Delete a checklist item.

**Parameters:**
- `id` (path): Plan UUID
- `checklist_id` (path): Checklist item UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "successfully deleted checklist item.",
  "result": "success"
}
```

### PATCH /api/plans/:id/checklists/:checklist_id/schedule
Set schedule for a checklist item.

**Parameters:**
- `id` (path): Plan UUID
- `checklist_id` (path): Checklist item UUID

**Request Body:**
```json
{
  "scheduledTime": "2024-01-01T10:00:00Z"
}
```

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully set schedule on checklist item.",
  "result": "success"
}
```

### PATCH /api/plans/:id/checklists/:checklist_id/archive
Archive a checklist item.

**Parameters:**
- `id` (path): Plan UUID
- `checklist_id` (path): Checklist item UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully archived checklist item.",
  "result": "success"
}
```

## Insights Endpoints

### GET /api/insights/checklist-suggestion
Get AI-generated checklist suggestions for a plan.

**Query Parameters:**
- `plan_id`: Plan UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "successfully generated completion",
  "result": "string"
}
```

### GET /api/insights/checklist-suggestion-daily
Get AI-generated daily task suggestions based on long-term goals.

**Query Parameters:**
- `plan_id`: Plan UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "successfully generated completion",
  "result": ["string", "string", "string"]
}
```

### GET /api/insights/suggest-videos
Get suggested learning videos for a plan.

**Query Parameters:**
- `plan_id`: Plan UUID

**Response:**
```json
{
  "statusCode": 200,
  "message": "Successfully generated suggested video links.",
  "result": [
    {
      "title": "string",
      "url": "string",
      "source": "string",
      "type": "video",
      "description": "string"
    }
  ]
}
```

## User Analytics Endpoints

### GET /api/analytics/user/:userId
Get analytics for a user.
*Note: Currently not implemented*

**Parameters:**
- `userId` (path): User UUID

**Response:**
```json
{
  "error": "not implemented"
}
```

## Error Responses

All error responses follow this format:
```json
{
  "statusCode": 400,
  "message": "Error description",
  "error": "Detailed error message"
}
```

Common HTTP status codes:
- `200`: Success
- `201`: Created
- `400`: Bad Request
- `401`: Unauthorized
- `404`: Not Found
- `500`: Internal Server Error
- `501`: Not Implemented

## Notes

1. **Authentication**: Currently using static user ID (`11111111-1111-1111-1111-111111111111`) for development. JWT implementation is in progress.

2. **Plan Types**: Supported values are "development" and "learning".

3. **Checklist Scopes**: Valid values are "daily" and "longterm".

4. **Daily Reset Job**: Runs automatically to reset daily tasks marked with `dailyReset: true`.

5. **Scheduled Items Job**: Processes scheduled checklist items based on their `scheduledTime`.

6. **AI Integrations**: The insights endpoints use AI models to generate task suggestions and find relevant learning resources.
# Go Project Generation Prompts

## Simple Prompt for New Projects

Use this prompt when you want Claude to generate a new Go project following the template:

### Basic Project Generation
```
Generate a new Go project called [PROJECT_NAME] following the CLAUDE_PROJECT_TEMPLATE.md guidelines. 

Create a simple [DOMAIN_NAME] domain as an example with basic CRUD operations.

Use PostgreSQL with sqlx, Gin for HTTP routing, and implement proper dependency injection with interfaces defined where consumed.
```

### Example Usage
```
Generate a new Go project called "taskmaster" following the CLAUDE_PROJECT_TEMPLATE.md guidelines. 

Create a simple "tasks" domain as an example with basic CRUD operations.

Use PostgreSQL with sqlx, Gin for HTTP routing, and implement proper dependency injection with interfaces defined where consumed.
```

## Advanced Prompt with Multiple Domains

```
Generate a new Go project called [PROJECT_NAME] following the CLAUDE_PROJECT_TEMPLATE.md guidelines.

Include these domains:
- [DOMAIN_1]: [brief description]
- [DOMAIN_2]: [brief description]
- [DOMAIN_3]: [brief description]

Implement:
- JWT authentication in middleware
- Database migrations for all domains
- Docker compose for local development
- Comprehensive Makefile
- Environment configuration

Follow SOLID principles with dependency injection and interfaces defined where consumed.
```

### Example Advanced Usage
```
Generate a new Go project called "inventory-manager" following the CLAUDE_PROJECT_TEMPLATE.md guidelines.

Include these domains:
- products: Product catalog with categories
- warehouses: Warehouse locations and capacity
- stock: Stock levels and movements

Implement:
- JWT authentication in middleware
- Database migrations for all domains
- Docker compose for local development
- Comprehensive Makefile
- Environment configuration

Follow SOLID principles with dependency injection and interfaces defined where consumed.
```

## Minimal Prompt

For the absolute minimum, you can simply say:

```
Create a new Go project called [PROJECT_NAME] following CLAUDE_PROJECT_TEMPLATE.md
```

## Adding to Existing Projects

If you want to add a new domain to an existing project:

```
Add a new [DOMAIN_NAME] domain to this project following the patterns in CLAUDE_PROJECT_TEMPLATE.md.

The domain should handle [brief description of functionality].

Include: model, repository, service, handler, and migration files.
```

## Tips for Using These Prompts

1. **Always reference the template**: Make sure to mention "CLAUDE_PROJECT_TEMPLATE.md" in your prompt
2. **Specify domain names**: Be clear about what domains/entities you want
3. **Keep it simple**: The template has all the details, your prompt just needs to specify what to build
4. **Add specifics only when needed**: Only add extra requirements if they differ from the template

## What Claude Will Generate

When you use these prompts with the template, Claude will:

1. Create the complete folder structure
2. Generate all boilerplate files (main.go, config files, etc.)
3. Implement example domain(s) with full CRUD operations
4. Set up database connections with sqlx
5. Configure Gin router with proper middleware
6. Create Makefile with common commands
7. Generate docker-compose.yml for local PostgreSQL
8. Create .env.sample with required variables
9. Add initial database migrations
10. Follow SOLID principles and dependency injection patterns

All code will follow the patterns from your existing Fireplace project without including any of its specific business logic.
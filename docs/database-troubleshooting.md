# Database Migration Troubleshooting Guide

## Quick Fix Commands

### When Database is in Dirty State
```bash
# Check current migration version
make migrate-status

# If you see "X (dirty)", fix it:
make migrate-fix version=X

# Alternative: Force clean reset (nuclear option)
make db-force-clean
```

## Common Issues and Solutions

### 1. Dirty Database State

**Symptoms:**
- Running `make migrate-status` shows "X (dirty)"
- Migrations won't run up or down
- Database schema is in inconsistent state

**Causes:**
- Migration failed halfway through execution
- Database connection lost during migration
- Syntax error in migration SQL
- Constraint violations during migration

**Solutions:**

#### Option A: Fix and Continue (Recommended)
```bash
# 1. Check current version
make migrate-status
# Output: 11 (dirty)

# 2. Fix the dirty state
make migrate-fix version=11

# 3. Review what went wrong in the migration file
cat migrations/000011_*.up.sql

# 4. Fix any issues manually if needed, then continue
make migrate-up
```

#### Option B: Force Clean Reset (Data Loss!)
```bash
# WARNING: This will delete all data!
make db-force-clean
```

#### Option C: Manual Recovery
```bash
# 1. Connect to database
psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME

# 2. Check schema_migrations table
SELECT * FROM schema_migrations;

# 3. Manually fix issues, then update version
UPDATE schema_migrations SET dirty = false WHERE version = 11;

# 4. Exit psql and continue
make migrate-up
```

### 2. Migration Version Mismatch

**Symptoms:**
- Error: "no migration found for version X"
- Migrations are out of sync

**Solution:**
```bash
# Force to a specific version
make migrate-fix version=10

# Then migrate to latest
make migrate-up
```

### 3. Cannot Drop Database (Active Connections)

**Symptoms:**
- Error: "database is being accessed by other users"

**Solution:**
```bash
# Kill all connections and recreate
make db-drop-create
```

## Prevention Tips

1. **Always backup before major migrations:**
   ```bash
   pg_dump -U $DB_USER -h $DB_HOST -p $DB_PORT $DB_NAME > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **Test migrations locally first:**
   ```bash
   # Use a test database
   make migrate-up
   make migrate-down
   make migrate-up  # Ensure it's idempotent
   ```

3. **Review migration files before running:**
   ```bash
   ls -la migrations/
   cat migrations/XXXXXX_*.up.sql
   ```

4. **Keep migrations small and focused:**
   - One table/feature per migration
   - Easier to debug and rollback

## Emergency Recovery Procedures

### Complete Database Reset (Nuclear Option)
```bash
# WARNING: Destroys all data!
# 1. Drop and recreate database
make db-drop-create

# 2. Run all migrations fresh
make migrate-up

# 3. Reseed data if needed
# (Add your seed commands here)
```

### Restore from Backup
```bash
# 1. Drop current database
make db-drop

# 2. Create fresh database
make db-create

# 3. Restore from backup
psql -U $DB_USER -h $DB_HOST -p $DB_PORT $DB_NAME < backup_YYYYMMDD_HHMMSS.sql
```

## Useful Diagnostic Commands

```bash
# Check migration status
make migrate-status

# View migration history
psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "SELECT * FROM schema_migrations;"

# List all tables
psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "\dt"

# Check specific table structure
psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "\d table_name"
```

## Makefile Commands Reference

| Command | Description | Data Loss Risk |
|---------|-------------|----------------|
| `make migrate-up` | Run pending migrations | No |
| `make migrate-down` | Rollback last migration | Possible |
| `make migrate-status` | Check current version | No |
| `make migrate-fix version=X` | Fix dirty state at version X | No |
| `make migrate-reset` | Down then up all migrations | Yes |
| `make db-force-clean` | Fix dirty + full reset | Yes |
| `make db-drop-create` | Drop and recreate database | Yes |

## When to Use Each Recovery Method

1. **Use `migrate-fix`** when:
   - You know what caused the issue
   - You want to preserve data
   - The migration can be safely retried

2. **Use `db-force-clean`** when:
   - Development environment
   - Test data can be lost
   - Quick reset needed

3. **Use `db-drop-create`** when:
   - Nothing else works
   - Starting fresh is acceptable
   - Setting up new environment

## Support

If issues persist:
1. Check migration SQL files for syntax errors
2. Verify database connection settings in `.env`
3. Check PostgreSQL logs for detailed errors
4. Ensure PostgreSQL version compatibility
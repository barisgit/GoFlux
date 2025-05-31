# Database Management

This GoFlux advanced template includes a complete PostgreSQL database setup with migrations, seeding, and type-safe code generation using SQLC and Goose. The database layer uses the modern `pgx/v5` driver for optimal performance and PostgreSQL feature support.

## Quick Start

```bash
# Or run the script directly
go run scripts/db.go setup
```

This will:

1. Start PostgreSQL database using Docker Compose
2. Run migrations to create tables
3. Generate type-safe Go code from SQL queries
4. Seed the database with sample data

## Prerequisites

- Docker and Docker Compose
- Go 1.21+
- PostgreSQL client tools (psql) - for seeding

The script will automatically install required tools:

- [Goose](https://github.com/pressly/goose) - Database migrations
- [SQLC](https://sqlc.dev/) - Type-safe Go code generation

**Note:** The database management script uses the same `pgx/v5` driver as the main application for consistency and optimal PostgreSQL compatibility.

## Database Commands

### Using Go Script

```bash
# Development database
go run scripts/db.go setup
go run scripts/db.go start
go run scripts/db.go stop
go run scripts/db.go migrate
go run scripts/db.go seed
go run scripts/db.go generate
go run scripts/db.go status

# Test database
go run scripts/db.go setup --test
go run scripts/db.go start --test
go run scripts/db.go migrate --test

# Reset database (deletes all data)
go run scripts/db.go setup --reset
go run scripts/db.go setup --test --reset
```

## Database Configuration

### Default Configuration

**Development Database:**

- Host: localhost
- Port: 5432
- Database: goflux_dev
- Username: goflux_user
- Password: goflux_pass

**Test Database:**

- Host: localhost
- Port: 5433
- Database: goflux_test
- Username: goflux_user
- Password: goflux_pass

### Environment Variables

You can override the default configuration using environment variables. The database script will automatically load variables from a `.env` file in the project root, or you can set them directly in your shell:

```bash
# Development database
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=goflux_dev
export DB_USER=goflux_user
export DB_PASSWORD=goflux_pass
export DB_SSLMODE=disable

# Test database
export DB_TEST_HOST=localhost
export DB_TEST_PORT=5433
export DB_TEST_NAME=goflux_test
export DB_TEST_USER=goflux_user
export DB_TEST_PASSWORD=goflux_pass
export DB_TEST_SSLMODE=disable
```

## File Structure

```text
├── docker-compose.yml          # PostgreSQL containers
├── sqlc.yaml                   # SQLC configuration
├── sql/
│   ├── migrations/             # Database migrations (Goose)
│   │   └── 001_initial_schema.sql
│   ├── queries/                # SQL queries for SQLC
│   │   ├── users.sql
│   │   ├── posts.sql
│   │   └── comments.sql
│   └── seed.sql               # Sample data
├── internal/db/
│   ├── db.go                  # Database interface
│   └── sqlc/                  # Generated code (auto-created)
└── scripts/
    └── db.go                  # Database management script
```

## Migrations

Migrations are managed using [Goose](https://github.com/pressly/goose) and stored in `sql/migrations/`.

### Creating a New Migration

```bash
# Install goose if not already installed
go install github.com/pressly/goose/v3/cmd/goose@latest

# Create a new migration
goose -dir sql/migrations create add_user_roles sql
```

This creates a new migration file with up and down sections:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    role VARCHAR(50) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_roles;
-- +goose StatementEnd
```

### Running Migrations

```bash
# Run all pending migrations
go run scripts/db.go migrate

# Or for test database
go run scripts/db.go migrate --test
```

## SQLC Code Generation

[SQLC](https://sqlc.dev/) generates type-safe Go code from SQL queries.

### Configuration

The `sqlc.yaml` file configures code generation:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/db/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_db_tags: true
```

### Writing Queries

Add SQL queries in `sql/queries/` with special comments:

```sql
-- name: GetUserByID :one
SELECT id, name, email, age, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (name, email, age)
VALUES ($1, $2, $3)
RETURNING id, name, email, age, created_at, updated_at;
```

### Generating Code

```bash
# Generate Go code from SQL
go run scripts/db.go generate
```

This creates type-safe Go functions in `internal/db/sqlc/`:

```go
func (q *Queries) GetUserByID(ctx context.Context, id int32) (User, error)
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
```

## Sample Data

The `sql/seed.sql` file contains sample data for development. It includes:

- Sample users
- Blog posts
- Comments
- Categories and tags
- User profiles

### Loading Sample Data

```bash
# Load sample data
go run scripts/db.go seed

# Or for test database
go run scripts/db.go seed --test
```

## Database Schema

The initial schema includes:

### Core Tables

- **users** - User accounts
- **posts** - Blog posts/articles
- **comments** - Post comments
- **categories** - Content categories
- **tags** - Content tags

### Junction Tables

- **post_categories** - Many-to-many posts ↔ categories
- **post_tags** - Many-to-many posts ↔ tags

### Extended Tables

- **user_profiles** - Extended user information

### Features

- Automatic `created_at` and `updated_at` timestamps
- Foreign key constraints with cascade deletes
- Indexes for performance
- Data validation constraints

## Troubleshooting

### Database Connection Issues

1. **Check if Docker is running:**

   ```bash
   docker --version
   docker-compose --version
   ```

2. **Check container status:**

   ```bash
   go run scripts/db.go status
   docker-compose ps
   ```

3. **View container logs:**

   ```bash
   docker-compose logs postgres
   docker-compose logs postgres_test
   ```

### Port Conflicts

If ports 5432 or 5433 are already in use:

1. **Stop conflicting services:**

   ```bash
   # Stop local PostgreSQL
   sudo systemctl stop postgresql
   
   # Or kill processes using the ports
   sudo lsof -ti:5432 | xargs kill -9
   sudo lsof -ti:5433 | xargs kill -9
   ```

2. **Or change ports in `docker-compose.yml`**

### Migration Issues

1. **Reset database if migrations fail:**

   ```bash
   go run scripts/db.go setup --reset
   ```

2. **Check migration status:**

   ```bash
   goose -dir sql/migrations postgres "postgres://goflux_user:goflux_pass@localhost:5432/goflux_dev?sslmode=disable" status
   ```

### SQLC Generation Issues

1. **Ensure migrations are up to date:**

   ```bash
   go run scripts/db.go migrate
   ```

2. **Check SQLC configuration:**

   ```bash
   sqlc vet
   ```

3. **Regenerate code:**

   ```bash
   rm -rf internal/db/sqlc/
   go run scripts/db.go generate
   ```

## Production Considerations

1. **Environment Variables:** Use environment variables for database configuration in production
2. **SSL Mode:** Enable SSL for production databases
3. **Connection Pooling:** Consider using connection pooling for high-traffic applications
4. **Backup Strategy:** Implement regular database backups
5. **Migration Strategy:** Use blue-green deployments for zero-downtime migrations

## Integration with GoFlux

The database layer integrates seamlessly with the GoFlux application:

1. **Type Safety:** SQLC generates Go structs that match your database schema
2. **API Integration:** Use generated functions in your API handlers
3. **Frontend Types:** Database types can be used for frontend type generation
4. **Testing:** Separate test database for isolated testing

This setup provides a robust, type-safe database layer that scales with your application needs.

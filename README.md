# Account Management Service

A production-ready Go authentication microservice providing JWT-based user account management with PostgreSQL persistence and comprehensive API documentation.

## Features

- **User Registration & Authentication** - Secure account creation with email validation and strong password requirements
- **JWT Access Tokens** - Short-lived JWT tokens (15 minutes) for secure API access
- **Refresh Token Management** - Long-lived refresh tokens (24 hours) with automatic rotation
- **Session Management** - Secure logout with token revocation
- **Password Security** - bcrypt hashing with complexity requirements (uppercase, lowercase, digit, special character)
- **Comprehensive Testing** - Unit tests with >90% coverage using testify
- **API Documentation** - API docs with OpenAPI spec and Redoc
- **Docker Support** - Complete containerization with PostgreSQL and Caddy reverse proxy
- **Observability** - Structured logging and container log monitoring via Dozzle

## Project Structure

```
account-management/
├── cmd/
│   └── account-management/          
│       └── main.go                 # Launches webserver with loaded config
├── internal/
│   ├── config/                     # Configuration management
│   │   └── config.go               # Loads config from either .env or environment
│   ├── database/                   # Database layer code
│   │   ├── database.go             # PostgreSQL connection & health checks
│   │   ├── accounts.go             # Account CRUD operations
│   │   ├── tokens.go               # Refresh token operations  
│   │   └── *_test.go
│   ├── service/
│   │   └── auth/                   
│   │       ├── passwords.go        # Password validation & hashing
│   │       ├── jwts.go             # JWT token generation & validation
│   │       └── *_test.go
│   └── webserver/                  
│       ├── webserver.go            # Webserver and router setup
│       ├── middleware.go           # Custom HTTP middleware
│       ├── accounts/               
│       │   ├── handlers.go         # Account HTTP handlers
│       │   └── handlers_test.go   
│       └── httputils/
│           ├── respones.go
│           └── errors.go
├── docs/                           # API documentation
│   ├── docs.go                     # Embeds docs and provides a file serving handler
│   └── api/
│       ├── index.html
│       └── api.yml
├── migrations/                    
│   ├── 000001_init_schema.up.sql  
│   └── 000001_init_schema.down.sql
├── hurls/                         # A simple test flow using hurl
├── docker-compose.yml
├── Dockerfile
├── Makefile
```

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate) (for database migrations)

### Local Development

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd account-management
   ```

2. **Start dependencies**
   ```bash
   docker compose up postgres -d
   ```

3. **Run database migrations**
   ```bash
   make migrate-up
   ```

4. **Set environment variables**
    The minimum is a .env file with:
    ```
    PSQL_URL=postgres://postgres:password@localhost:5432/account_management?sslmode=disable (if debugging or running directly)
    PSQL_URL=postgres://postgres:password@postgres:5432/account_management?sslmode=disable (if running with Caddy)
    JWT_SECRET_KEY=some-super-secret-really-long-key-for-making-jwts-please-dont-commit-something-real.12345
    ```

5. **Run the application**
   ```bash
   make build
   ./bin/account-management
   ```

   Or run directly:
   ```bash
   go run ./cmd/account-management
   ```

### Docker Deployment

Run the complete stack with PostgreSQL, application, and Caddy reverse proxy:

```bash
docker compose up -d
```

This starts:
- **PostgreSQL** on port 5432
    - IMPORTANT: if you want to run this through Caddy, the PSQL_URL host needs to be postgres not localhost
- **Account Management API** (internal)
- **Caddy** reverse proxy on ports 80/443
- **Dozzle** log viewer on port 9999

## 🔧 Development

### Available Make Commands

```bash
make help          # Show all available commands
make build         # Build the application binary
make test          # Run all tests
make test-coverage # Generate HTML coverage report
make lint          # Run golangci-lint
make fmt           # Format code
make vet           # Run go vet
make check         # Run all quality checks (fmt, vet, lint, test)

# Database migrations
make migrate-up    # Apply pending migrations
make migrate-down  # Rollback last migration
make migrate-create name=<name>  # Create new migration files
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage
open coverage.html

# Run tests for specific package
go test ./internal/service/auth -v
```

### API Documentation

Interactive API documentation is available at:
- **Local Development**: http://localhost:8080/docs/
- **Docker**: http://localhost/api/docs/

The documentation is auto-generated from the OpenAPI 3.0 specification in `docs/api/api.yml`.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/accounts/register` | Create new user account |
| POST | `/v1/accounts/login` | Authenticate and get tokens |
| POST | `/v1/accounts/refresh` | Refresh access token |
| POST | `/v1/accounts/logout` | Revoke refresh token |

### Authentication Flow

1. **Register** - Create account with email/password
2. **Login** - Get access token (15min) + refresh token (24hr)  
3. **Use Access Token** - Include `Authorization: Bearer <token>` in requests
4. **Refresh** - Use refresh token to get new access token when expired
5. **Logout** - Revoke refresh token to end session

## Security Features

- **Password Requirements**: Minimum 8 characters with uppercase, lowercase, digit, and special character
- **bcrypt Hashing**: Industry-standard password hashing with salt
- **JWT Security**: Short-lived access tokens with secure signing
- **Token Rotation**: Refresh tokens are rotated on each use
- **Session Revocation**: Logout immediately invalidates refresh tokens
- **Input Validation**: Comprehensive email and password validation
- **SQL Injection Protection**: Parameterized queries throughout

## Environment Configuration

```bash
# Required
POSTGRES_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable

JWT_SECRET_KEY=your-super-secret-jwt-key-here

# Optional but set with these defaults via config.go
ACCESS_TOKEN_TTL_MINUTES=15
REFRESH_TOKEN_TTL_MINUTES=1440

HTTP_ADDRESS=:8080
```

## Monitoring & Observability

- **Structured Logging**: JSON logs with correlation IDs
- **Health Checks**: Database connectivity monitoring at `/health`
- **Container Logs**: Dozzle web interface at http://localhost:9999 (but maybe Loki later?)
- **Test Coverage**: HTML reports generated via `make test-coverage`
- **Metrics**: Coming soon! (Probably prometheus)

## Author

**Austin Wofford**
- Email: wofford.austin@live.com
- GitHub: [@austinwofford](https://github.com/austinwofford)

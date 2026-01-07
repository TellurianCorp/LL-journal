# LL-journal - Journal Service for LifeLogger
> **Document Type:** README > **Owner/Process:** LL-journal Engineering > **Version:** v0.1.0 > **Status:** DRAFT > **Last Updated:** 2025-12-26 > **Approver:** LL-journal Tech Lead

**LifeLogger LL-journal**
**API**: https://api.lifelogger.life
**Company**: Tellurian Corp (https://www.telluriancorp.com)
**Created**: December 2025

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Building](#building)
- [Running](#running)
- [Configuration](#configuration)
- [Database Migrations](#database-migrations)
- [Testing](#testing)
- [API Endpoints](#api-endpoints)
- [Development](#development)
- [Support and Troubleshooting](#support-and-troubleshooting)
- [Change Log / Release Notes](#change-log--release-notes)

## Overview

LL-journal is the **journal and diary management** service for the LifeLogger ecosystem. It allows users to create, edit, and version personal journals written in Markdown format. The system uses Git for version control and S3 for storage of Markdown files.

**Important**: LL-journal is a REST API service. All authentication is handled by LL-proxy (the API gateway), which validates tokens before forwarding requests. LL-journal receives authenticated requests with user information in headers (e.g., `X-User-Sub`).

## Architecture

- **Language**: Go
- **Framework**: Chi router
- **Database**: PostgreSQL (metadata and references)
- **Storage**: S3/MinIO (Markdown files)
- **Version Control**: Git (one repository per user)
- **Port**: 9002 (configurable)
- **Authentication**: Handled by LL-proxy (API Gateway) - LL-journal receives authenticated requests only

## Features

- ✅ REST API for journal management
- ✅ Markdown-based journal entries
- ✅ Git version control per user
- ✅ S3 storage for Markdown files
- ✅ PostgreSQL for metadata
- ✅ Entry versioning and history
- ✅ Word count tracking
- ✅ Health check endpoint
- ✅ Automatic Git commits on edits

## Building

### Prerequisites

- Go 1.24 or later
- PostgreSQL (for metadata)
- S3-compatible storage (MinIO or AWS S3)
- Git (for version control)

### Build

```bash
# From LL-journal directory
go build ./cmd/ll-journal

# Or using Makefile
make build

# Or using CMake from root
cmake --build . --target ll-journal
```

## Running

### Using Makefile (Recommended)

From the LL-journal directory:

```bash
# Build and run
make run
```

### Direct Execution

```bash
cd LL-journal
go run ./cmd/ll-journal
```

### Environment Variables

- `LL_JOURNAL_HOST`: Server host (default: `0.0.0.0`)
- `LL_JOURNAL_PORT`: Server port (default: `9002`)
- `LL_JOURNAL_DATABASE_URL`: PostgreSQL connection string (required)
- `LL_JOURNAL_S3_ENDPOINT`: S3/MinIO endpoint (required)
- `LL_JOURNAL_S3_BUCKET`: S3 bucket name (default: `lifelogger-journals`)
- `LL_JOURNAL_S3_ACCESS_KEY`: S3 access key (required)
- `LL_JOURNAL_S3_SECRET_KEY`: S3 secret key (required)
- `LL_JOURNAL_GIT_ROOT`: Git repositories root directory (default: `/var/lib/ll-journal/git`)
- `LL_JOURNAL_LOG_LEVEL`: Log level (default: `info`)

**Note**: LL-journal requires database, S3, and Git configuration to function properly. Authentication is handled by LL-proxy, which validates tokens before forwarding requests. LL-journal receives authenticated requests with user information in headers (e.g., `X-User-Sub`).

## Configuration

Configuration is loaded with priority:
1. `.env` file (development only)
2. Environment variables
3. JSON config file (if exists)

See `internal/config/config.go` for all configuration options.

## Database Migrations

Migrations are located in the `migrations/` directory:

- `001_journal_schema.sql` - Initial schema (journals, journal_entries, journal_versions)

### Running Migrations

```bash
# From root repository
psql $LL_JOURNAL_DATABASE_URL < LL-journal/migrations/001_journal_schema.sql
```

## Testing

### Unit Tests

```bash
cd LL-journal
go test ./...
```

### BDD Tests

Cucumber feature files are in `features/` directory. Integration tests are located in the root repository `tests/` directory (Rust-based Cucumber tests).

## API Endpoints

### Health Check

```
GET /health
```

Returns service status and version.

### Journal Management

```
POST   /api/journals              # Create journal
GET    /api/journals              # List journals
GET    /api/journals/{id}         # Get journal
PUT    /api/journals/{id}         # Update journal
DELETE /api/journals/{id}         # Delete journal
```

### Entry Management

```
POST   /api/journals/{journalId}/entries           # Create entry
GET    /api/journals/{journalId}/entries           # List entries
GET    /api/journals/{journalId}/entries/{date}   # Get entry
PUT    /api/journals/{journalId}/entries/{date}    # Update entry
DELETE /api/journals/{journalId}/entries/{date}    # Delete entry
```

### Version Management

```
GET /api/journals/{journalId}/entries/{date}/versions        # List versions
GET /api/journals/{journalId}/entries/{date}/versions/{commit} # Get specific version
```

### Request/Response Examples

#### Create Journal

```bash
POST /api/journals
Content-Type: application/json
X-User-Sub: user-123

{
  "title": "My Daily Journal",
  "description": "A journal for daily thoughts"
}
```

#### Create Entry

```bash
POST /api/journals/{journalId}/entries
Content-Type: application/json
X-User-Sub: user-123

{
  "entry_date": "2025-12-26",
  "content": "# Today's Entry\n\nThis is my journal entry for today."
}
```

#### Get Entry

```bash
GET /api/journals/{journalId}/entries/2025-12-26
X-User-Sub: user-123
```

Response:
```json
{
  "entry": {
    "id": "entry-uuid",
    "journal_id": "journal-uuid",
    "entry_date": "2025-12-26",
    "word_count": 10,
    "created_at": "2025-12-26T10:00:00Z",
    "updated_at": "2025-12-26T10:00:00Z"
  },
  "content": "# Today's Entry\n\nThis is my journal entry for today."
}
```

## Development

### Project Structure

```
LL-journal/
├── cmd/
│   └── ll-journal/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── handlers/            # HTTP handlers
│   ├── journal/             # Business logic
│   ├── store/               # Database store layer
│   ├── s3/                  # S3 client
│   └── git/                 # Git operations
├── migrations/              # SQL migration files
├── features/                 # Cucumber BDD tests
├── go.mod                   # Go module definition
└── go.sum                   # Go module checksums
```

### Adding New Features

1. Update database schema if needed (create migration)
2. Update store layer (`internal/store/`)
3. Update business logic (`internal/journal/`)
4. Update handlers (`internal/handlers/`)
5. Add tests
6. Update documentation

## Support and Troubleshooting

### Common Issues

**Service won't start:**
- Check if port 9002 is available: `lsof -i :9002`
- Verify Go version: `go version` (requires 1.24+)
- Check logs for configuration errors

**Database connection issues:**
- Verify `LL_JOURNAL_DATABASE_URL` is set correctly
- Check PostgreSQL is running: `pg_isready`
- Ensure migrations have been run

**S3 connection issues:**
- Verify S3 credentials are correct
- Check S3 endpoint is accessible
- Ensure bucket exists and is accessible

**Git operations failing:**
- Verify `LL_JOURNAL_GIT_ROOT` directory exists and is writable
- Check filesystem permissions
- Ensure Git is installed on the system

**API requests failing:**
- Verify LL-proxy is running and routing correctly
- Check authentication headers are being passed
- Review service logs for errors

### Getting Help

- Check logs for detailed error messages
- Review documentation in root repository `docs/`
- Check GitHub issues in main repository

## Change Log / Release Notes

### v0.1.0 (2025-12-26)
- Initial implementation
- Journal CRUD operations
- Entry CRUD operations
- Git version control integration
- S3 storage integration
- PostgreSQL metadata storage
- Health check endpoint
- REST API with authentication via LL-proxy

---

**Last Updated**: December 2025
**Maintained By**: LifeLogger Engineering Team

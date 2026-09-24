# Gebeta API — Food Delivery Backend MVP

A production-oriented Go backend for a food delivery platform, built on top of Supabase (Auth, Postgres, Storage) with Firebase Cloud Messaging for push notifications.

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.26 |
| HTTP router | [chi](https://github.com/go-chi/chi) |
| Database | PostgreSQL via [Supabase](https://supabase.com) |
| DB driver | [pgx/v5](https://github.com/jackc/pgx) |
| Query generation | [sqlc](https://sqlc.dev) |
| Migrations | [goose](https://github.com/pressly/goose) |
| Auth | Supabase Auth (JWT, verified via JWKS) |
| Storage | Supabase Storage |
| Push notifications | Firebase Cloud Messaging |
| Config | [cleanenv](https://github.com/ilyakaznacheev/cleanenv) |
| Validation | [go-playground/validator](https://github.com/go-playground/validator) |
| Logging | `log/slog`, colorized in development via [tint](https://github.com/lmittmann/tint) |
| Hot reload (dev) | [air](https://github.com/air-verse/air) |

## Architecture

The project follows a **package-by-feature** layout rather than a layered MVC split — each domain (`users`, and future ones like `orders`) owns its full stack: handler → service → repository.

```
cmd/
└── api/
    ├── main.go       # composition root: config, logger, db pool, then mount + run
    └── api.go         # router assembly, graceful shutdown, DB pool setup

internal/
├── config/            # env-based config loading (cleanenv), fail-fast validation
├── middleware/         # auth (JWKS-based JWT verification), etc.
├── users/              # handler.go, service.go, repository.go, types.go, routes.go
├── db/
│   ├── migrations/      # goose-managed schema migrations
│   ├── queries/          # sqlc source queries (.sql)
│   └── sqlc/              # sqlc-generated Go code (committed)
└── platform/            # thin wrappers around Supabase/Firebase clients (WIP)

common/
├── apperrors/    # sentinel errors + HTTP status mapping
├── httpx/         # JSON response envelope, request decode/validate helpers
└── constants/      # shared, non-domain constants
```

**Feature packages depend on `common/`, never the other way around, and never on each other directly.**

## Authentication

Clients (e.g. a Flutter app) authenticate directly against **Supabase Auth**, never through this backend. The resulting JWT is sent as a Bearer token on every request to this API.

The API verifies tokens **locally**, using Supabase's public JWKS endpoint (`SUPABASE_JWKS_URL`) — no network call to Supabase per request. Public keys are fetched once and cached, with automatic refresh on key rotation (`kid` mismatch).

A Postgres trigger (`handle_new_auth_user`) keeps `public.users` in sync with `auth.users` automatically on signup — the API never creates users itself.

## Database

- All primary keys are **UUIDs** (`gen_random_uuid()`), never sequential integers — prevents enumeration of records.
- `public.users.id` is a foreign key to Supabase's `auth.users.id`.
- Spatial data (customer addresses, rider locations) is stored as PostGIS `geography(Point, 4326)` and exposed to the Go layer as plain `lat`/`lng` floats via `ST_X`/`ST_Y` in queries — no PostGIS types leak into application code.

### Connections

Supabase offers three connection modes; this project uses two of them, for different purposes:

| Use | Connection | Port |
|---|---|---|
| Application runtime (`DATABASE_URL`) | Transaction pooler (Supavisor) | 6543 |
| Migrations (`GOOSE_DBSTRING`) | Session pooler | 5432 |

The transaction pooler requires pgx's simple query protocol (`QueryExecModeSimpleProtocol`), since it doesn't support server-side prepared statements.

## Getting Started

### Prerequisites
- Go 1.26+
- A Supabase project (Postgres + Auth + Storage enabled)
- A Firebase project with a service account key (for push notifications)

### Setup

```bash
# Clone and install dependencies
go mod download

# Copy the example env file and fill in your credentials
cp .env.example .env

# Install CLI tools
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/air-verse/air@latest

# Run migrations
goose -dir internal/db/migrations up

# Generate sqlc code (after any query/schema change)
sqlc generate
```

### Running locally

```bash
air
```

Hot-reloads on file changes. No Docker required for local development — this project connects to Supabase's hosted Postgres directly, so there's no local database to containerize.

### Running in production

```bash
docker build -t gebeta-api .
docker run -p 8080:8080 --env-file .env gebeta-api
```

## Environment Variables

See `.env.example` for the full list. Key groups:
- `SERVER_*` — HTTP server host/port/timeouts
- `DATABASE_URL` — Supabase transaction pooler connection string
- `SUPABASE_*` — project URL, API keys, JWKS URL for auth verification
- `FIREBASE_*` — project ID and service account credentials path
- `CORS_ALLOWED_ORIGINS` — comma-separated list; must not be `*` in production
- `GOOSE_*` — migration-only connection string and settings

Configuration is validated at startup — the app refuses to boot with missing secrets, an unreachable database, or an insecure production configuration (e.g. wildcard CORS).

## API Overview

All routes are prefixed with `/api/v1` and (except `/health`) require a valid Supabase-issued Bearer token.

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness check, including DB connectivity |
| GET | `/users/me` | Current user's account, profile, role, and addresses |
| PUT | `/users/me/profile` | Update profile fields |
| GET | `/users/me/addresses` | List saved addresses |
| POST | `/users/me/addresses` | Add a new address (max 10 per user) |
| PUT | `/users/me/addresses/{id}` | Update an address (partial updates supported) |
| DELETE | `/users/me/addresses/{id}` | Remove an address |
| GET | `/users/me/device-tokens` | List registered FCM device tokens |
| POST | `/users/me/device-tokens` | Register/refresh a device's FCM token |
| DELETE | `/users/me/device-tokens` | Remove a device token (e.g. on logout) |

### Business rules worth knowing
- A user always has exactly one default address; the API rejects requests that would leave zero defaults.
- Users are capped at 10 saved addresses and 5 registered device tokens (oldest evicted beyond the limit).

## Response Format

Every response follows a consistent envelope:

```json
{
  "success": true,
  "data": { }
}
```

```json
{
  "success": false,
  "error": { "message": "..." }
}
```

Internal errors are logged server-side with full detail; only safe, generic messages are ever returned to clients for 5xx errors.

## Status

🚧 Active MVP development. Implemented so far: configuration, graceful shutdown, JWT auth middleware, and the full `users` domain (profile, addresses, device tokens). Orders, restaurants, riders, and notifications are planned next.

## License

Proprietary — all rights reserved
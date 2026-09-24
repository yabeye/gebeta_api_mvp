# Gebeta API MVP

Food delivery backend API.

## Stack

Go · Chi · Supabase · Firebase FCM · sqlc · Goose

* **Supabase** — Auth, PostgreSQL, Storage
* **Firebase FCM** — Push notifications

## Setup

Install tools:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go get github.com/google/uuid
```

Run migrations:

```bash
goose up
```

Generate SQLC code:

```bash
sqlc generate
```

Run the API:

```bash
go run .
```

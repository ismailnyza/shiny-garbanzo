# QR Restaurant Backend

Go/Gin backend for QR-based dine-in restaurant ordering. Owners manage restaurants, staff, tables, QR codes, themes, menus, sessions, and orders. Customers scan table QR codes, start or reuse an active table session, view the menu, place orders, and receive order updates.

## Requirements

- Go `1.26.2`
- PostgreSQL `15`
- Optional: Docker and Docker Compose for local Postgres and container builds

## Quick Start

1. Copy `.env.example` to `.env` and fill in the values you need.
2. Start Postgres:

```bash
docker compose up -d postgres
```

3. Run the server:

```bash
make run
```

The app listens on `:8080` by default.

## Common Commands

```bash
make build
make test
make test-race
make test-integration
make govulncheck
docker build -t qr-restaurant:test .
```

The sandboxed Codex environment may need a writable Go cache:

```bash
GOCACHE=/tmp/go-build go test ./...
```

## Testing

Unit tests:

```bash
go test ./...
```

Race tests:

```bash
go test ./... -race -count=1
```

Integration tests use a disposable Postgres URL and the `integration` build tag:

```bash
QR_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/qr_restaurant?sslmode=disable' \
  go test -tags=integration ./internal/apptest -count=1
```

The harness creates a unique schema per test run, applies migrations to that schema, and drops it during cleanup.

## Environment

The repository includes [.env.example](.env.example) with the standard local settings:

- `APP_ENV`
- `PORT`
- `FRONTEND_BASE_URL`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE`
- `JWT_SECRET`, `JWT_EXPIRY_HOURS`
- `SESSION_TTL_HOURS`
- `BCRYPT_COST`
- `RATE_LIMIT_PUBLIC`, `RATE_LIMIT_ADMIN`
- Optional Cloudflare R2 settings
- `MIGRATE_PATH`

## API and Operations

- `GET /api/v1/health` checks the app and database.
- `GET /metrics` exposes request, order, WebSocket, panic, rate-limit, and DB pool metrics.
- `make seed` loads `scripts/seed.sql`.
- `make lint` runs `golangci-lint`.

See [docs/RUNBOOK.md](docs/RUNBOOK.md) for deployment, rollback, backup/restore, migration recovery, and secret rotation.
See [docs/PLAN.md](docs/PLAN.md) for the broader product and schema notes.

## Product Notes

- Idempotency keys are scoped to a dining session. Repeating the same `idempotency_key` for the same session returns the original order response instead of creating another order.
- QR URLs use the restaurant slug and table code: `/r/{slug}/t/{table_code}?token={qr_token}`. The QR token starts or reuses only the current active table session.
- Customer sessions expire after `SESSION_TTL_HOURS` hours, default `8`. Expired session tokens cannot order or connect to WebSockets; scanning the same QR creates a fresh active session.
- Menu `options_config` supports MVP modifiers. The backend validates modifier selections during order creation, applies flat `price_delta` values with quantity multiplication, and stores immutable modifier snapshots on order items.
- All API responses include a `request_id` when request middleware is active. Clients may pass `X-Request-ID`; otherwise the server generates one and echoes it in the response header.

## Modifier Contract

`options_config` uses this shape:

```json
{
  "modifier_groups": [
    {
      "id": "size",
      "name": "Choose 1 size",
      "description": "Optional helper text",
      "display_order": 0,
      "required": true,
      "min_selected": 1,
      "max_selected": 1,
      "selection_strategy": "SINGLE_SELECT",
      "allow_multiple_quantities": false,
      "active": true,
      "options": [
        {
          "id": "large",
          "name": "Large",
          "description": "",
          "display_order": 0,
          "price_delta": 200,
          "active": true,
          "maximum_quantity_per_option": 1
        }
      ]
    }
  ]
}
```

`selected_options` uses this shape:

```json
{
  "modifier_selections": [
    {
      "group_id": "size",
      "options": [
        { "option_id": "large", "quantity": 1 }
      ]
    }
  ]
}
```

Validation errors return the standard API error envelope with `code: "VALIDATION_ERROR"` and a short `message`. The backend owns final total calculation; clients should render modifier UI from `options_config` but should treat server totals as authoritative.

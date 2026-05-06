# Production Runbook

Practical pilot runbook for a single-instance deployment.

## Production Environment

Set these before launch:

- `APP_ENV=production`
- `FRONTEND_BASE_URL=https://<frontend-host>`
- `JWT_SECRET=<strong random secret>`
- `SESSION_TTL_HOURS=8`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE`
- R2 variables if image uploads are enabled
- `MIGRATE_PATH=file://migrations`

Use `GIN_MODE=release` in the runtime environment. Do not reuse staging or development secrets.

## Deployment

1. Build the image: `docker build -t qr-restaurant:<version> .`
2. Backup production Postgres before migrations.
3. Run migrations against the target database.
4. Start the new container with the production env file.
5. Verify `/api/v1/health` returns `{"status":"ok","db":"ok"}`.
6. Verify `/metrics` returns HTTP and DB metrics.
7. Place a test order in staging before promoting the same image tag.

## Rollback

1. Keep the previous image tag available.
2. If the new container fails health checks, stop it and restart the previous tag with the same env.
3. If migrations changed schema, use the matching down migrations only after confirming the old code cannot run against the new schema.
4. Restore from backup if data was corrupted or a migration partially applied.

## Backup

Run before every deploy and at least daily during the pilot:

```bash
pg_dump "$DATABASE_URL" --format=custom --file=backup-$(date +%Y%m%d-%H%M%S).dump
```

Store backups outside the application host.

## Restore

1. Stop the application.
2. Create a fresh database or empty the broken one.
3. Restore: `pg_restore --clean --if-exists --dbname "$DATABASE_URL" backup.dump`
4. Run migrations if restoring an older backup.
5. Start the application and verify health, login, QR scan, and order placement.

## Migration Failure

1. Stop the deploy.
2. Capture the migration error and database version from the `schema_migrations` table.
3. If no data changed, apply the down migration and redeploy the previous image.
4. If data changed, restore the pre-deploy backup.
5. Fix the migration in a new version; do not edit an already-applied production migration.

## Staging Verification

Run before pilot launch:

```bash
go test ./...
go test ./... -race -count=1
QR_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/qr_restaurant?sslmode=disable' go test -tags=integration ./internal/apptest -count=1
docker build -t qr-restaurant:test .
govulncheck ./...
```

Manual smoke test:

1. Register owner.
2. Create restaurant, table, category, menu item, and modifier.
3. Scan QR and place order.
4. Update order status and confirm WebSocket update.
5. Close session and confirm old token cannot order.
6. Confirm audit rows exist for admin mutations.

## Secret Rotation

Rotate `JWT_SECRET` during a low-traffic window. Existing staff sessions are invalidated. Restart the app with the new secret, then verify owner login.

Rotate DB and R2 credentials by updating the provider secret first, then the application env, then restarting the app.

## Failed Deploy Recovery

If the app starts but behaves incorrectly:

1. Check `/api/v1/health`.
2. Check recent structured logs by `request_id`.
3. Check `/metrics` for elevated `panic_total`, `rate_limit_rejections_total`, 5xx status counts, DB pool saturation, or WebSocket spikes.
4. Roll back to the previous image if user ordering is affected.

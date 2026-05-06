# Engineering Decisions

## Idempotency

Order idempotency is scoped to `(session_id, idempotency_key)`.

Repeated requests with the same key in the same dining session return the existing order. This prevents duplicate order side effects on client retries. The service does not compare request bodies yet; the key is treated as the client's statement that the retry is the same logical request.

## QR URLs and Sessions

QR URLs include restaurant slug, table code, and QR token. The QR token identifies the table, not a customer session. Scanning creates or reuses one active session for the table. Closing the session prevents old session tokens from being used for new customers.

## Menu Modifiers

`options_config` is exposed on menu item create/update/admin/public responses and stored in `menu_items.options_config`.

For MVP, the schema supports modifier groups, options, required/min/max rules, single-select and multi-select behavior, option active flags, quantity limits, and flat `price_delta` values. Order creation validates selections against the current menu config, calculates totals server-side, and stores immutable snapshots in `order_items.selected_options`.

The modifier system intentionally remains JSONB-backed instead of introducing separate modifier tables. That keeps the MVP smaller and preserves historical order snapshots. If restaurants need global modifier libraries, reusable groups, inventory-aware options, or analytics by option, move modifiers into normalized tables later.

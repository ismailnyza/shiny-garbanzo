CREATE TABLE orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES sessions(id),
    restaurant_id   UUID NOT NULL REFERENCES restaurants(id),
    idempotency_key TEXT,
    request_hash    TEXT,
    status          TEXT NOT NULL DEFAULT 'PENDING'
                    CHECK (status IN ('PENDING','ACCEPTED','PREPARING','READY','COMPLETED','CANCELLED')),
    total_cents     INT NOT NULL DEFAULT 0,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_orders_session ON orders(session_id);
CREATE INDEX idx_orders_restaurant_status ON orders(restaurant_id, status);
CREATE UNIQUE INDEX idx_orders_session_idempotency_key
    ON orders(session_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE TABLE order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id        UUID REFERENCES menu_items(id) ON DELETE SET NULL,
    name_snapshot       TEXT NOT NULL,
    price_cents_snapshot INT NOT NULL,
    quantity            INT NOT NULL CHECK (quantity > 0),
    selected_options    JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_order_items_order ON order_items(order_id);

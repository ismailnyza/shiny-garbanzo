# QR Restaurant — Updated Backend MVP Planning Doc

## Changes from Original Spec:
- **Restored WebSockets:** We reverted back to the WebSocket architecture for real-time order updates.
- **Added Modifiers/Add-ons:** Added a `JSONB` `options` column to `order_items` and `menu_items` to handle sizes, extras, and complex configurations cleanly without schema bloat.
- **Added Payment Status:** Added `payment_status` to the `sessions` table (`PENDING`, `PARTIAL`, `PAID`) to track if a table has settled their physical bill.
- **Idempotency:** Added an `idempotency_key` constraint to `orders` to prevent double-charging on network retries.

---

## Database Schema Updates

*We drop the `themes` table and migrate its concept directly into `restaurants` as a structured JSONB object.*

```sql
-- 000002_restaurants.up.sql
CREATE TABLE restaurants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT,
    address     TEXT,
    phone       TEXT,
    settings    JSONB NOT NULL DEFAULT '{
        "theme": {
            "primary_color": "#FF6B35",
            "secondary_color": "#F7C59F",
            "logo_url": null,
            "banner_url": null,
            "font_style": "inter",
            "dark_mode": false
        },
        "features": {
            "hero_video_url": null,
            "max_gallery_images": 5,
            "require_staff_acceptance": true
        }
    }'::jsonb,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
```

### 2. Modifiers on Menu Items & Order Items
```sql
-- 000008_menu_items.up.sql
CREATE TABLE menu_items (
    -- ... existing fields ...
    price_cents     INT NOT NULL CHECK (price_cents >= 0),
    options_config  JSONB, -- E.g. {"sizes": [{"name":"Large", "price_diff": 200}], "extras": [...]}
    is_available    BOOLEAN NOT NULL DEFAULT TRUE,
    -- ...
);

-- 000009_orders.up.sql
CREATE TABLE orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES sessions(id),
    restaurant_id   UUID NOT NULL REFERENCES restaurants(id),
    idempotency_key TEXT UNIQUE, -- Prevents double-clicks from charging twice
    status          TEXT NOT NULL DEFAULT 'PENDING',
    total_cents     INT NOT NULL DEFAULT 0,
    -- ...
);

CREATE TABLE order_items (
    -- ... existing fields ...
    quantity            INT NOT NULL CHECK (quantity > 0),
    selected_options    JSONB, -- The specific choices the customer made, snapshotted
    -- ...
);
```

### 3. Payment Tracking on Sessions
```sql
-- 000006_sessions.up.sql
CREATE TABLE sessions (
    -- ... existing fields ...
    status          TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','CLOSED')),
    payment_status  TEXT NOT NULL DEFAULT 'PENDING' CHECK (payment_status IN ('PENDING', 'PARTIAL', 'PAID')),
    -- ...
);
```

---

## GORM Models Updates

```go
// internal/shared/models/models.go
import "gorm.io/datatypes" // Required for JSONB

type Restaurant struct {
    // ...
    Settings    datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
    // ...
}

type MenuItem struct {
    // ...
    OptionsConfig datatypes.JSON `gorm:"type:jsonb"`
    // ...
}

type Order struct {
    // ...
    IdempotencyKey *string `gorm:"uniqueIndex"`
    // ...
}

type OrderItem struct {
    // ...
    SelectedOptions datatypes.JSON `gorm:"type:jsonb"`
    // ...
}

type Session struct {
    // ...
    PaymentStatus string `gorm:"default:'PENDING'"`
    // ...
}
```

---

## API Adjustments

### `PATCH /api/v1/restaurants/:restaurantId`
Now handles the full `settings` object. You can pass the entire JSON to update theme and feature flags in one go.

### `POST /api/v1/public/sessions/:sessionToken/orders`
**Request:**
```json
{
  "idempotency_key": "uuid-from-client",
  "items": [
    { 
      "menu_item_id": "uuid", 
      "quantity": 1,
      "selected_options": { "size": "Large", "add_ons": ["Extra Cheese"] }
    }
  ],
  "notes": "Extra spicy please"
}
```

### `PATCH /api/v1/restaurants/:restaurantId/sessions/:sessionId/payment`
> OWNER + STAFF. Update payment tracking.

**Request:**
```json
{ "payment_status": "PAID" }
```

---

## Admin Flow / Onboarding Improvements

1.  **Registration Simplification:** When an `OWNER` registers (`POST /auth/register`), the backend can optionally auto-create their first dummy `Restaurant` and a default `MenuCategory` behind the scenes. This gives them an immediate "Aha!" moment when they log in to the dashboard, rather than an empty screen.
2.  **Soft Deletes Logic:** As noted in the QA report, the backend service layer (e.g., `menu.Service`) will manually handle cascading soft-deletes (deleting a category explicitly calls GORM to soft-delete its child items) to avoid orphaned items appearing in the public feed.

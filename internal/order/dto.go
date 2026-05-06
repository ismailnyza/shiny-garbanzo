package order

import (
	"github.com/google/uuid"
)

type PlaceOrderRequest struct {
	IdempotencyKey *string            `json:"idempotency_key" binding:"omitempty,uuid"`
	Items          []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
	Notes          *string            `json:"notes" binding:"omitempty,max=500"`
}

type OrderItemRequest struct {
	MenuItemID      uuid.UUID              `json:"menu_item_id" binding:"required"`
	Quantity        int                    `json:"quantity"      binding:"required,min=1"`
	SelectedOptions map[string]interface{} `json:"selected_options" binding:"omitempty"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=PENDING ACCEPTED PREPARING READY COMPLETED CANCELLED"`
}

type OrderResponse struct {
	ID             uuid.UUID          `json:"id"`
	SessionID      uuid.UUID          `json:"session_id"`
	IdempotencyKey *string            `json:"idempotency_key,omitempty"`
	Status         string             `json:"status"`
	TotalCents     int                `json:"total_cents"`
	Notes          *string            `json:"notes"`
	CreatedAt      string             `json:"created_at"`
	Items          []OrderItemResp    `json:"items"`
}

type OrderItemResp struct {
	NameSnapshot       string                 `json:"name_snapshot"`
	PriceCentsSnapshot int                    `json:"price_cents_snapshot"`
	Quantity           int                    `json:"quantity"`
	SelectedOptions    map[string]interface{} `json:"selected_options,omitempty"`
}

type AdminOrderListResponse struct {
	ID         uuid.UUID          `json:"id"`
	SessionID  uuid.UUID          `json:"session_id"`
	TableCode  string             `json:"table_code"`
	Status     string             `json:"status"`
	TotalCents int                `json:"total_cents"`
	Notes      *string            `json:"notes"`
	CreatedAt  string             `json:"created_at"`
	Items      []OrderItemResp    `json:"items"`
}

type StatusUpdateResponse struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	UpdatedAt string    `json:"updated_at"`
}

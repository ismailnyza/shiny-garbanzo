package session

import (
	"github.com/google/uuid"
)

type ScanResponse struct {
	SessionToken uuid.UUID          `json:"session_token"`
	Restaurant   RestaurantRef      `json:"restaurant"`
	TableCode    string             `json:"table_code"`
	Theme        ThemeRef           `json:"theme"`
}

type RestaurantRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ThemeRef struct {
	PrimaryColor   string  `json:"primary_color"`
	SecondaryColor string  `json:"secondary_color"`
	LogoURL        *string `json:"logo_url"`
	BannerURL      *string `json:"banner_url"`
	FontStyle      string  `json:"font_style"`
	DarkMode       bool    `json:"dark_mode"`
}

type PublicSessionResponse struct {
	SessionToken uuid.UUID `json:"session_token"`
	Status       string    `json:"status"`
	TableCode    string    `json:"table_code"`
	StartedAt    string    `json:"started_at"`
}

type AdminSessionListResponse struct {
	ID                uuid.UUID  `json:"id"`
	TableID           uuid.UUID  `json:"table_id"`
	TableCode         string     `json:"table_code"`
	SessionToken      uuid.UUID  `json:"session_token"`
	Status            string     `json:"status"`
	StartedAt         string     `json:"started_at"`
	ClosedAt          *string    `json:"closed_at"`
	OrderCount        int        `json:"order_count"`
	PendingOrderCount int        `json:"pending_order_count"`
}

type AdminSessionDetailResponse struct {
	ID           uuid.UUID       `json:"id"`
	TableCode    string          `json:"table_code"`
	SessionToken uuid.UUID       `json:"session_token"`
	Status       string          `json:"status"`
	StartedAt    string          `json:"started_at"`
	ClosedAt     *string         `json:"closed_at"`
	Orders       []OrderSummary  `json:"orders"`
}

type OrderSummary struct {
	ID         uuid.UUID          `json:"id"`
	Status     string             `json:"status"`
	TotalCents int                `json:"total_cents"`
	CreatedAt  string             `json:"created_at"`
	Items      []OrderItemSummary `json:"items"`
}

type OrderItemSummary struct {
	NameSnapshot      string `json:"name_snapshot"`
	PriceCentsSnapshot int    `json:"price_cents_snapshot"`
	Quantity          int    `json:"quantity"`
}

type CloseSessionRequest struct {
	Force *bool `json:"force"`
}

type CloseSessionResponse struct {
	ID       uuid.UUID `json:"id"`
	Status   string    `json:"status"`
	ClosedAt string    `json:"closed_at"`
}

type UpdatePaymentRequest struct {
	PaymentStatus string `json:"payment_status" binding:"required,oneof=PENDING PARTIAL PAID"`
}

type UpdatePaymentResponse struct {
	ID            uuid.UUID `json:"id"`
	PaymentStatus string    `json:"payment_status"`
}

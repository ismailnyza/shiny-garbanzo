package table

import (
	"github.com/google/uuid"
)

type CreateRequest struct {
	TableCode string `json:"table_code" binding:"required,min=1,max=50"`
}

type UpdateRequest struct {
	TableCode *string `json:"table_code" binding:"omitempty,min=1,max=50"`
	IsActive  *bool   `json:"is_active"`
}

type Response struct {
	ID              uuid.UUID  `json:"id"`
	RestaurantID    uuid.UUID  `json:"restaurant_id"`
	TableCode       string     `json:"table_code"`
	QRToken         string     `json:"qr_token"`
	QRURL           string     `json:"qr_url"`
	IsActive        bool       `json:"is_active"`
	ActiveSessionID *uuid.UUID `json:"active_session_id,omitempty"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
}

type QRRegenResponse struct {
	QRToken string `json:"qr_token"`
	QRURL   string `json:"qr_url"`
}

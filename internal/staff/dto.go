package staff

import (
	"github.com/google/uuid"
)

type CreateRequest struct {
	Email    string `json:"email"    binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type Response struct {
	StaffID      uuid.UUID `json:"staff_id"`
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	CreatedAt    string    `json:"created_at"`
}

type StaffMemberResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	CreatedAt string    `json:"created_at"`
}

package theme

import (
	"github.com/google/uuid"
)

type UpsertRequest struct {
	PrimaryColor   *string `json:"primary_color"   binding:"omitempty,min=4,max=7"`
	SecondaryColor *string `json:"secondary_color" binding:"omitempty,min=4,max=7"`
	LogoURL        *string `json:"logo_url"        binding:"omitempty"`
	BannerURL      *string `json:"banner_url"      binding:"omitempty"`
	FontStyle      *string `json:"font_style"      binding:"omitempty,min=1,max=50"`
	DarkMode       *bool   `json:"dark_mode"`
}

type Response struct {
	ID             uuid.UUID `json:"id"`
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	PrimaryColor   string    `json:"primary_color"`
	SecondaryColor string    `json:"secondary_color"`
	LogoURL        *string   `json:"logo_url"`
	BannerURL      *string   `json:"banner_url"`
	FontStyle      string    `json:"font_style"`
	DarkMode       bool      `json:"dark_mode"`
}

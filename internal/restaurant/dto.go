package restaurant

import "github.com/google/uuid"

type CreateRequest struct {
	Name        string  `json:"name"        binding:"required,min=1,max=255"`
	Slug        string  `json:"slug"        binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Address     *string `json:"address"     binding:"omitempty,max=500"`
	Phone       *string `json:"phone"       binding:"omitempty,max=50"`
}

type UpdateRequest struct {
	Name        *string `json:"name"        binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Address     *string `json:"address"     binding:"omitempty,max=500"`
	Phone       *string `json:"phone"       binding:"omitempty,max=50"`
	IsActive    *bool   `json:"is_active"`
}

type Response struct {
	ID          uuid.UUID `json:"id"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description"`
	Address     *string   `json:"address"`
	Phone       *string   `json:"phone"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	Theme       *ThemeRef `json:"theme,omitempty"`
}

type ThemeRef struct {
	PrimaryColor   string  `json:"primary_color"`
	SecondaryColor string  `json:"secondary_color"`
	LogoURL        *string `json:"logo_url"`
	BannerURL      *string `json:"banner_url"`
	FontStyle      string  `json:"font_style"`
	DarkMode       bool    `json:"dark_mode"`
}

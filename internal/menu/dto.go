package menu

import (
	"github.com/google/uuid"
)

// --- Category DTOs ---

type CreateCategoryRequest struct {
	Name      string `json:"name"       binding:"required,min=1,max=255"`
	SortOrder int    `json:"sort_order"  binding:"omitempty,min=0"`
	IsVisible *bool  `json:"is_visible"`
}

type UpdateCategoryRequest struct {
	Name      *string `json:"name"       binding:"omitempty,min=1,max=255"`
	SortOrder *int    `json:"sort_order"  binding:"omitempty,min=0"`
	IsVisible *bool   `json:"is_visible"`
}

type CategoryResponse struct {
	ID           uuid.UUID `json:"id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Name         string    `json:"name"`
	SortOrder    int       `json:"sort_order"`
	IsVisible    bool      `json:"is_visible"`
	ItemCount    int       `json:"item_count,omitempty"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}

type ReorderRequest struct {
	Order []ReorderItem `json:"order" binding:"required"`
}

type ReorderItem struct {
	ID        uuid.UUID `json:"id"        binding:"required"`
	SortOrder int       `json:"sort_order" binding:"required,min=0"`
}

type ReorderResponse struct {
	Updated int `json:"updated"`
}

// --- Item DTOs ---

type CreateItemRequest struct {
	Name          string                 `json:"name"        binding:"required,min=1,max=255"`
	Description   *string                `json:"description" binding:"omitempty,max=1000"`
	ImageURL      *string                `json:"image_url"   binding:"omitempty"`
	PriceCents    int                    `json:"price_cents" binding:"required,min=0"`
	OptionsConfig map[string]interface{} `json:"options_config" binding:"omitempty"`
	IsAvailable   *bool                  `json:"is_available"`
	SortOrder     int                    `json:"sort_order"  binding:"omitempty,min=0"`
}

type UpdateItemRequest struct {
	Name          *string                `json:"name"        binding:"omitempty,min=1,max=255"`
	Description   *string                `json:"description" binding:"omitempty"`
	ImageURL      *string                `json:"image_url"   binding:"omitempty"`
	PriceCents    *int                   `json:"price_cents" binding:"omitempty,min=0"`
	OptionsConfig map[string]interface{} `json:"options_config" binding:"omitempty"`
	IsAvailable   *bool                  `json:"is_available"`
	SortOrder     *int                   `json:"sort_order"  binding:"omitempty,min=0"`
}

type ItemResponse struct {
	ID            uuid.UUID              `json:"id"`
	CategoryID    uuid.UUID              `json:"category_id"`
	RestaurantID  uuid.UUID              `json:"restaurant_id"`
	Name          string                 `json:"name"`
	Description   *string                `json:"description"`
	ImageURL      *string                `json:"image_url"`
	PriceCents    int                    `json:"price_cents"`
	OptionsConfig map[string]interface{} `json:"options_config,omitempty"`
	IsAvailable   bool                   `json:"is_available"`
	SortOrder     int                    `json:"sort_order"`
	CategoryName  string                 `json:"category_name,omitempty"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

// --- Public Menu DTOs ---

type PublicMenuResponse struct {
	RestaurantName string                `json:"restaurant_name"`
	Categories     []PublicCategoryGroup `json:"categories"`
}

type PublicCategoryGroup struct {
	ID        uuid.UUID        `json:"id"`
	Name      string           `json:"name"`
	SortOrder int              `json:"sort_order"`
	Items     []PublicMenuItem `json:"items"`
}

type PublicMenuItem struct {
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	Description   *string                `json:"description"`
	ImageURL      *string                `json:"image_url"`
	PriceCents    int                    `json:"price_cents"`
	OptionsConfig map[string]interface{} `json:"options_config,omitempty"`
	IsAvailable   bool                   `json:"is_available"`
	SortOrder     int                    `json:"sort_order"`
}

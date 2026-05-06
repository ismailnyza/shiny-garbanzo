package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User represents a system user (OWNER or STAFF).
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string         `gorm:"uniqueIndex;not null"`
	PasswordHash string         `gorm:"not null"`
	Role         string         `gorm:"not null;check:role IN ('OWNER','STAFF')"`
	CreatedAt    time.Time      `gorm:"not null;default:now()"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// Restaurant represents a restaurant owned by an OWNER user.
type Restaurant struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID     uuid.UUID      `gorm:"type:uuid;not null"`
	Name        string         `gorm:"not null"`
	Slug        string         `gorm:"uniqueIndex;not null"`
	Description *string        `gorm:"type:text"`
	Address     *string        `gorm:"type:text"`
	Phone       *string        `gorm:"type:text"`
	IsActive    bool           `gorm:"not null;default:true"`
	CreatedAt   time.Time      `gorm:"not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations
	Owner User              `gorm:"foreignKey:OwnerID"`
	Theme *Theme            `gorm:"foreignKey:RestaurantID"`
	Staff []RestaurantStaff `gorm:"foreignKey:RestaurantID"`
}

// RestaurantStaff links a STAFF user to a restaurant.
type RestaurantStaff struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_restaurant_staff_unique"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_restaurant_staff_unique"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`

	User       User       `gorm:"foreignKey:UserID"`
	Restaurant Restaurant `gorm:"foreignKey:RestaurantID"`
}

// Theme stores visual customisation for a restaurant.
type Theme struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID   uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	PrimaryColor   string    `gorm:"not null;default:'#FF6B35'"`
	SecondaryColor string    `gorm:"not null;default:'#F7C59F'"`
	LogoURL        *string   `gorm:"type:text"`
	BannerURL      *string   `gorm:"type:text"`
	FontStyle      string    `gorm:"not null;default:'inter'"`
	DarkMode       bool      `gorm:"not null;default:false"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`
}

// Table represents a physical table in a restaurant.
type Table struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID `gorm:"type:uuid;not null"`
	TableCode    string    `gorm:"not null"`
	QRToken      string    `gorm:"uniqueIndex;not null"`
	IsActive     bool      `gorm:"not null;default:true"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
	UpdatedAt    time.Time `gorm:"not null;default:now()"`
}

// Session represents a customer dining session at a table.
type Session struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TableID       uuid.UUID `gorm:"type:uuid;not null"`
	RestaurantID  uuid.UUID `gorm:"type:uuid;not null"`
	SessionToken  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null;default:gen_random_uuid()"`
	Status        string    `gorm:"not null;default:'ACTIVE';check:status IN ('ACTIVE','CLOSED')"`
	PaymentStatus string    `gorm:"not null;default:'PENDING';check:payment_status IN ('PENDING','PARTIAL','PAID')"`
	StartedAt     time.Time `gorm:"not null;default:now()"`
	ClosedAt      *time.Time
	ExpiresAt     *time.Time
	LastSeenAt    time.Time `gorm:"not null;default:now()"`

	Table  Table   `gorm:"foreignKey:TableID"`
	Orders []Order `gorm:"foreignKey:SessionID"`
}

// MenuCategory groups menu items.
type MenuCategory struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RestaurantID uuid.UUID      `gorm:"type:uuid;not null"`
	Name         string         `gorm:"not null"`
	SortOrder    int            `gorm:"not null;default:0"`
	IsVisible    bool           `gorm:"not null;default:true"`
	CreatedAt    time.Time      `gorm:"not null;default:now()"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	Items []MenuItem `gorm:"foreignKey:CategoryID"`
}

// MenuItem is an individual item in a menu category.
type MenuItem struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CategoryID    uuid.UUID      `gorm:"type:uuid;not null"`
	RestaurantID  uuid.UUID      `gorm:"type:uuid;not null"`
	Name          string         `gorm:"not null"`
	Description   *string        `gorm:"type:text"`
	ImageURL      *string        `gorm:"type:text"`
	PriceCents    int            `gorm:"not null;check:price_cents >= 0"`
	OptionsConfig datatypes.JSON `gorm:"type:jsonb"`
	IsAvailable   bool           `gorm:"not null;default:true"`
	SortOrder     int            `gorm:"not null;default:0"`
	CreatedAt     time.Time      `gorm:"not null;default:now()"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// Order represents a customer order placed during a session.
type Order struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID      uuid.UUID `gorm:"type:uuid;not null"`
	RestaurantID   uuid.UUID `gorm:"type:uuid;not null"`
	IdempotencyKey *string
	RequestHash    string
	Status         string    `gorm:"not null;default:'PENDING';check:status IN ('PENDING','ACCEPTED','PREPARING','READY','COMPLETED','CANCELLED')"`
	TotalCents     int       `gorm:"not null;default:0"`
	Notes          *string   `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`

	Items []OrderItem `gorm:"foreignKey:OrderID"`
}

// OrderItem is a snapshot of a menu item at time of ordering.
type OrderItem struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID            uuid.UUID      `gorm:"type:uuid;not null"`
	MenuItemID         *uuid.UUID     `gorm:"type:uuid"`
	NameSnapshot       string         `gorm:"not null"`
	PriceCentsSnapshot int            `gorm:"not null"`
	Quantity           int            `gorm:"not null;check:quantity > 0"`
	SelectedOptions    datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt          time.Time      `gorm:"not null;default:now()"`
}

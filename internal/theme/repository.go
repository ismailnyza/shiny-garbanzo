package theme

import (
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	FindByRestaurant(restaurantID uuid.UUID) (*models.Theme, error)
	Upsert(theme *models.Theme) error
	Create(theme *models.Theme) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindByRestaurant(restaurantID uuid.UUID) (*models.Theme, error) {
	var theme models.Theme
	err := r.db.Where("restaurant_id = ?", restaurantID).First(&theme).Error
	if err != nil {
		return nil, err
	}
	return &theme, nil
}

func (r *repository) Upsert(theme *models.Theme) error {
	return r.db.Save(theme).Error
}

func (r *repository) Create(theme *models.Theme) error {
	return r.db.Create(theme).Error
}

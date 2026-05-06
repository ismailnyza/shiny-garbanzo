package staff

import (
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	Create(staff *models.RestaurantStaff) error
	ListByRestaurant(restaurantID uuid.UUID) ([]models.RestaurantStaff, error)
	FindByID(id uuid.UUID) (*models.RestaurantStaff, error)
	Delete(id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(staff *models.RestaurantStaff) error {
	return r.db.Create(staff).Error
}

func (r *repository) ListByRestaurant(restaurantID uuid.UUID) ([]models.RestaurantStaff, error) {
	var staff []models.RestaurantStaff
	err := r.db.Preload("User").Where("restaurant_id = ?", restaurantID).Find(&staff).Error
	if err != nil {
		return nil, err
	}
	return staff, nil
}

func (r *repository) FindByID(id uuid.UUID) (*models.RestaurantStaff, error) {
	var staff models.RestaurantStaff
	err := r.db.Preload("User").First(&staff, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &staff, nil
}

func (r *repository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.RestaurantStaff{}, "id = ?", id).Error
}

package restaurant

import (
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	Create(restaurant *models.Restaurant) error
	FindByID(id uuid.UUID) (*models.Restaurant, error)
	FindBySlug(slug string) (*models.Restaurant, error)
	ListByOwner(ownerID uuid.UUID, page, perPage, offset int, isActive *bool) ([]models.Restaurant, int64, error)
	Update(restaurant *models.Restaurant) error
	SoftDelete(id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(restaurant *models.Restaurant) error {
	return r.db.Create(restaurant).Error
}

func (r *repository) FindByID(id uuid.UUID) (*models.Restaurant, error) {
	var restaurant models.Restaurant
	err := r.db.Preload("Theme").First(&restaurant, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

func (r *repository) FindBySlug(slug string) (*models.Restaurant, error) {
	var restaurant models.Restaurant
	err := r.db.First(&restaurant, "slug = ?", slug).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

func (r *repository) ListByOwner(ownerID uuid.UUID, page, perPage, offset int, isActive *bool) ([]models.Restaurant, int64, error) {
	var restaurants []models.Restaurant
	var total int64

	query := r.db.Model(&models.Restaurant{}).Where("owner_id = ?", ownerID)
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&restaurants).Error; err != nil {
		return nil, 0, err
	}

	return restaurants, total, nil
}

func (r *repository) Update(restaurant *models.Restaurant) error {
	return r.db.Save(restaurant).Error
}

func (r *repository) SoftDelete(id uuid.UUID) error {
	return r.db.Delete(&models.Restaurant{}, "id = ?", id).Error
}

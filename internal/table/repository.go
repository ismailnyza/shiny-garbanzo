package table

import (
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	Create(table *models.Table) error
	FindByID(id uuid.UUID) (*models.Table, error)
	FindByQRToken(token string) (*models.Table, error)
	FindByRestaurant(restaurantID uuid.UUID, page, perPage, offset int, isActive *bool) ([]models.Table, int64, error)
	Update(table *models.Table) error
	Delete(id uuid.UUID) error
	FindActiveSessionID(tableID uuid.UUID) (*uuid.UUID, error)
	ExistsByRestaurantAndCode(restaurantID uuid.UUID, code string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(table *models.Table) error {
	return r.db.Create(table).Error
}

func (r *repository) FindByID(id uuid.UUID) (*models.Table, error) {
	var table models.Table
	err := r.db.First(&table, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &table, nil
}

func (r *repository) FindByQRToken(token string) (*models.Table, error) {
	var table models.Table
	err := r.db.First(&table, "qr_token = ?", token).Error
	if err != nil {
		return nil, err
	}
	return &table, nil
}

func (r *repository) FindByRestaurant(restaurantID uuid.UUID, page, perPage, offset int, isActive *bool) ([]models.Table, int64, error) {
	var tables []models.Table
	var total int64

	query := r.db.Model(&models.Table{}).Where("restaurant_id = ?", restaurantID)
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(perPage).Order("table_code ASC").Find(&tables).Error; err != nil {
		return nil, 0, err
	}

	return tables, total, nil
}

func (r *repository) Update(table *models.Table) error {
	return r.db.Save(table).Error
}

func (r *repository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Table{}, "id = ?", id).Error
}

func (r *repository) FindActiveSessionID(tableID uuid.UUID) (*uuid.UUID, error) {
	var session models.Session
	err := r.db.Where("table_id = ? AND status = 'ACTIVE'", tableID).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &session.ID, nil
}

func (r *repository) ExistsByRestaurantAndCode(restaurantID uuid.UUID, code string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Table{}).
		Where("restaurant_id = ? AND table_code = ?", restaurantID, code).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

package order

import (
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	CreateOrder(tx *gorm.DB, order *models.Order) error
	FindByID(id uuid.UUID) (*models.Order, error)
	FindBySessionAndIdempotencyKey(sessionID uuid.UUID, key string) (*models.Order, error)
	FindBySession(sessionID uuid.UUID) ([]models.Order, error)
	FindByRestaurant(restaurantID uuid.UUID, status, sessionIDStr, sort string, page, perPage, offset int) ([]models.Order, int64, error)
	UpdateStatus(id uuid.UUID, status string) error
	CountBySession(sessionID uuid.UUID) (total int, pending int, err error)
	CountOrdersBySession(sessionID uuid.UUID) (total int, pending int, err error)
	FindOrdersBySession(sessionID uuid.UUID) ([]models.Order, error)
	GetDB() *gorm.DB
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetDB() *gorm.DB {
	return r.db
}

func (r *repository) CreateOrder(tx *gorm.DB, order *models.Order) error {
	return tx.Create(order).Error
}

func (r *repository) FindByID(id uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Items").First(&order, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *repository) FindBySessionAndIdempotencyKey(sessionID uuid.UUID, key string) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Items").
		Where("session_id = ? AND idempotency_key = ?", sessionID, key).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *repository) FindBySession(sessionID uuid.UUID) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.Preload("Items").
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *repository) FindByRestaurant(restaurantID uuid.UUID, status, sessionIDStr, sort string, page, perPage, offset int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	query := r.db.Model(&models.Order{}).Where("restaurant_id = ?", restaurantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if sessionIDStr != "" {
		query = query.Where("session_id = ?", sessionIDStr)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderClause := "created_at DESC"
	if sort != "" {
		orderClause = sort
	}

	if err := query.Preload("Items").Offset(offset).Limit(perPage).Order(orderClause).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *repository) UpdateStatus(id uuid.UUID, status string) error {
	return r.db.Model(&models.Order{}).Where("id = ?", id).Update("status", status).Error
}

func (r *repository) CountOrdersBySession(sessionID uuid.UUID) (int, int, error) {
	return r.CountBySession(sessionID)
}

func (r *repository) CountBySession(sessionID uuid.UUID) (int, int, error) {
	var total int64
	if err := r.db.Model(&models.Order{}).Where("session_id = ?", sessionID).Count(&total).Error; err != nil {
		return 0, 0, err
	}

	var pending int64
	if err := r.db.Model(&models.Order{}).
		Where("session_id = ? AND status IN ('PENDING','ACCEPTED','PREPARING')", sessionID).
		Count(&pending).Error; err != nil {
		return int(total), 0, err
	}

	return int(total), int(pending), nil
}

func (r *repository) FindOrdersBySession(sessionID uuid.UUID) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.Preload("Items").
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

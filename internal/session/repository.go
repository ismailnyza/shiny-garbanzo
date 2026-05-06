package session

import (
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	Create(session *models.Session) error
	FindByID(id uuid.UUID) (*models.Session, error)
	FindBySessionToken(token uuid.UUID) (*models.Session, error)
	FindActiveByTableID(tableID uuid.UUID) (*models.Session, error)
	ListByRestaurant(restaurantID uuid.UUID, status string, page, perPage, offset int) ([]models.Session, int64, error)
	Update(session *models.Session) error
	TouchLastSeen(sessionID uuid.UUID) error
	CountOrdersBySession(sessionID uuid.UUID) (total int, pending int, err error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(session *models.Session) error {
	return r.db.Create(session).Error
}

func (r *repository) FindByID(id uuid.UUID) (*models.Session, error) {
	var session models.Session
	err := r.db.Preload("Table").First(&session, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repository) FindBySessionToken(token uuid.UUID) (*models.Session, error) {
	var session models.Session
	err := r.db.Preload("Table").Preload("Orders", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Preload("Orders.Items").First(&session, "session_token = ?", token).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repository) FindActiveByTableID(tableID uuid.UUID) (*models.Session, error) {
	var session models.Session
	err := r.db.Where("table_id = ? AND status = 'ACTIVE'", tableID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repository) ListByRestaurant(restaurantID uuid.UUID, status string, page, perPage, offset int) ([]models.Session, int64, error) {
	var sessions []models.Session
	var total int64

	query := r.db.Model(&models.Session{}).Where("restaurant_id = ?", restaurantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("Table").Offset(offset).Limit(perPage).Order("started_at DESC").Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

func (r *repository) Update(session *models.Session) error {
	return r.db.Save(session).Error
}

func (r *repository) TouchLastSeen(sessionID uuid.UUID) error {
	return r.db.Model(&models.Session{}).Where("id = ?", sessionID).Update("last_seen_at", gorm.Expr("NOW()")).Error
}

func (r *repository) CountOrdersBySession(sessionID uuid.UUID) (int, int, error) {
	var total int64
	if err := r.db.Model(&models.Order{}).Where("session_id = ?", sessionID).Count(&total).Error; err != nil {
		return 0, 0, err
	}

	var pending int64
	if err := r.db.Model(&models.Order{}).
		Where("session_id = ? AND status IN ('PENDING','ACCEPTED')", sessionID).
		Count(&pending).Error; err != nil {
		return int(total), 0, err
	}

	return int(total), int(pending), nil
}

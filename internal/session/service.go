package session

import (
	"errors"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type TableRepository interface {
	FindByQRToken(token string) (*models.Table, error)
	FindByID(id uuid.UUID) (*models.Table, error)
}

type RestaurantRepository interface {
	FindByID(id uuid.UUID) (*models.Restaurant, error)
}

// OrderRepository interface for cross-module access
type OrderRepository interface {
	CountOrdersBySession(sessionID uuid.UUID) (int, int, error)
	FindOrdersBySession(sessionID uuid.UUID) ([]models.Order, error)
}

type Service interface {
	// Public
	Scan(db *gorm.DB, qrToken string) (*ScanResponse, error)
	GetPublicSession(sessionToken uuid.UUID) (*PublicSessionResponse, error)

	// Admin
	ListByRestaurant(restaurantID uuid.UUID, status string, page, perPage, offset int) ([]AdminSessionListResponse, int64, error)
	GetByID(restaurantID, sessionID uuid.UUID) (*AdminSessionDetailResponse, error)
	Close(restaurantID, sessionID uuid.UUID, force bool) (*CloseSessionResponse, error)
	UpdatePayment(restaurantID, sessionID uuid.UUID, paymentStatus string) (*UpdatePaymentResponse, error)
}

type service struct {
	repo       Repository
	tableRepo  TableRepository
	restRepo   RestaurantRepository
	orderRepo  OrderRepository
	sessionTTL time.Duration
}

func NewService(repo Repository, tableRepo TableRepository, restRepo RestaurantRepository, orderRepo OrderRepository, ttlHours ...int) Service {
	ttl := 8 * time.Hour
	if len(ttlHours) > 0 && ttlHours[0] > 0 {
		ttl = time.Duration(ttlHours[0]) * time.Hour
	}
	return &service{
		repo:       repo,
		tableRepo:  tableRepo,
		restRepo:   restRepo,
		orderRepo:  orderRepo,
		sessionTTL: ttl,
	}
}

// Scan handles the public QR code scan flow.
func (s *service) Scan(db *gorm.DB, qrToken string) (*ScanResponse, error) {
	table, err := s.tableRepo.FindByQRToken(qrToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Invalid QR token")
		}
		return nil, apperrors.NewInternal(err)
	}

	if !table.IsActive {
		return nil, apperrors.NewUnprocessable("This table is no longer active")
	}

	// Check for existing active session
	activeSession, err := s.repo.FindActiveByTableID(table.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewInternal(err)
	}

	if activeSession != nil {
		if s.isExpired(activeSession) {
			activeSession.Status = "CLOSED"
			now := time.Now()
			activeSession.ClosedAt = &now
			_ = s.repo.Update(activeSession)
		} else {
			_ = s.repo.TouchLastSeen(activeSession.ID)
			return s.buildScanResponse(activeSession, table)
		}
	}

	// Create new session in a transaction to handle the unique index
	var session *models.Session
	err = db.Transaction(func(tx *gorm.DB) error {
		// Lock the table row
		var t models.Table
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&t, "id = ?", table.ID).Error; err != nil {
			return err
		}

		// Double-check no active session exists
		var count int64
		tx.Model(&models.Session{}).Where("table_id = ? AND status = 'ACTIVE'", table.ID).Count(&count)
		if count > 0 {
			return tx.Where("table_id = ? AND status = 'ACTIVE'", table.ID).First(&session).Error
		}

		expiresAt := time.Now().Add(s.sessionTTL)
		session = &models.Session{
			TableID:      table.ID,
			RestaurantID: table.RestaurantID,
			Status:       "ACTIVE",
			ExpiresAt:    &expiresAt,
			LastSeenAt:   time.Now(),
		}
		return tx.Create(session).Error
	})
	if err != nil {
		activeSession, findErr := s.repo.FindActiveByTableID(table.ID)
		if findErr == nil {
			if s.isExpired(activeSession) {
				return nil, apperrors.NewForbidden("Session has expired")
			}
			_ = s.repo.TouchLastSeen(activeSession.ID)
			return s.buildScanResponse(activeSession, table)
		}
		return nil, err
	}

	return s.buildScanResponse(session, table)
}

func (s *service) buildScanResponse(session *models.Session, table *models.Table) (*ScanResponse, error) {
	restaurant, err := s.restRepo.FindByID(table.RestaurantID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	if session.Status == "ACTIVE" && s.isExpired(session) {
		return nil, apperrors.NewForbidden("Session has expired")
	}
	_ = s.repo.TouchLastSeen(session.ID)

	theme := &ThemeRef{
		PrimaryColor:   "#FF6B35",
		SecondaryColor: "#F7C59F",
		FontStyle:      "inter",
		DarkMode:       false,
	}
	if restaurant.Theme != nil {
		theme = &ThemeRef{
			PrimaryColor:   restaurant.Theme.PrimaryColor,
			SecondaryColor: restaurant.Theme.SecondaryColor,
			LogoURL:        restaurant.Theme.LogoURL,
			BannerURL:      restaurant.Theme.BannerURL,
			FontStyle:      restaurant.Theme.FontStyle,
			DarkMode:       restaurant.Theme.DarkMode,
		}
	}

	return &ScanResponse{
		SessionToken: session.SessionToken,
		Restaurant: RestaurantRef{
			Name: restaurant.Name,
			Slug: restaurant.Slug,
		},
		TableCode: table.TableCode,
		Theme:     *theme,
	}, nil
}

func (s *service) isExpired(session *models.Session) bool {
	return session != nil && session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt)
}

// GetPublicSession returns session status for customer polling.
func (s *service) GetPublicSession(sessionToken uuid.UUID) (*PublicSessionResponse, error) {
	session, err := s.repo.FindBySessionToken(sessionToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Session not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	tableCode := ""
	if session.Table.ID != uuid.Nil {
		tableCode = session.Table.TableCode
	}

	if session.Status == "ACTIVE" && s.isExpired(session) {
		return nil, apperrors.NewForbidden("Session has expired")
	}
	_ = s.repo.TouchLastSeen(session.ID)

	return &PublicSessionResponse{
		SessionToken: session.SessionToken,
		Status:       session.Status,
		TableCode:    tableCode,
		StartedAt:    session.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// Admin: List sessions for a restaurant.
func (s *service) ListByRestaurant(restaurantID uuid.UUID, status string, page, perPage, offset int) ([]AdminSessionListResponse, int64, error) {
	sessions, total, err := s.repo.ListByRestaurant(restaurantID, status, page, perPage, offset)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err)
	}

	responses := make([]AdminSessionListResponse, len(sessions))
	for i, sess := range sessions {
		tableCode := ""
		if sess.Table.ID != uuid.Nil {
			tableCode = sess.Table.TableCode
		}

		totalOrders, pendingOrders, _ := s.repo.CountOrdersBySession(sess.ID)

		var closedAt *string
		if sess.ClosedAt != nil {
			t := sess.ClosedAt.Format("2006-01-02T15:04:05Z07:00")
			closedAt = &t
		}

		responses[i] = AdminSessionListResponse{
			ID:                sess.ID,
			TableID:           sess.TableID,
			TableCode:         tableCode,
			SessionToken:      sess.SessionToken,
			Status:            sess.Status,
			StartedAt:         sess.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			ClosedAt:          closedAt,
			OrderCount:        totalOrders,
			PendingOrderCount: pendingOrders,
		}
	}

	return responses, total, nil
}

// Admin: Get full session detail with orders.
func (s *service) GetByID(restaurantID, sessionID uuid.UUID) (*AdminSessionDetailResponse, error) {
	session, err := s.repo.FindByID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Session not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if session.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Session not found")
	}

	tableCode := ""
	if session.Table.ID != uuid.Nil {
		tableCode = session.Table.TableCode
	}

	orders, err := s.orderRepo.FindOrdersBySession(sessionID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	orderSummaries := make([]OrderSummary, len(orders))
	for i, o := range orders {
		items := make([]OrderItemSummary, len(o.Items))
		for j, item := range o.Items {
			items[j] = OrderItemSummary{
				NameSnapshot:       item.NameSnapshot,
				PriceCentsSnapshot: item.PriceCentsSnapshot,
				Quantity:           item.Quantity,
			}
		}
		orderSummaries[i] = OrderSummary{
			ID:         o.ID,
			Status:     o.Status,
			TotalCents: o.TotalCents,
			CreatedAt:  o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Items:      items,
		}
	}

	var closedAt *string
	if session.ClosedAt != nil {
		t := session.ClosedAt.Format("2006-01-02T15:04:05Z07:00")
		closedAt = &t
	}

	return &AdminSessionDetailResponse{
		ID:           session.ID,
		TableCode:    tableCode,
		SessionToken: session.SessionToken,
		Status:       session.Status,
		StartedAt:    session.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		ClosedAt:     closedAt,
		Orders:       orderSummaries,
	}, nil
}

// Admin: Close a session.
func (s *service) Close(restaurantID, sessionID uuid.UUID, force bool) (*CloseSessionResponse, error) {
	session, err := s.repo.FindByID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Session not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if session.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Session not found")
	}

	if session.Status == "CLOSED" {
		return nil, apperrors.NewConflict("Session is already closed")
	}

	if !force {
		_, pendingOrders, err := s.repo.CountOrdersBySession(sessionID)
		if err != nil {
			return nil, apperrors.NewInternal(err)
		}
		if pendingOrders > 0 {
			return nil, apperrors.NewConflict("Session has pending or accepted orders. Use force=true to close anyway")
		}
	}

	now := time.Now()
	session.Status = "CLOSED"
	session.ClosedAt = &now

	if err := s.repo.Update(session); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return &CloseSessionResponse{
		ID:       session.ID,
		Status:   session.Status,
		ClosedAt: now.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// Admin: Update payment status of a session.
func (s *service) UpdatePayment(restaurantID, sessionID uuid.UUID, paymentStatus string) (*UpdatePaymentResponse, error) {
	session, err := s.repo.FindByID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Session not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if session.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Session not found")
	}

	session.PaymentStatus = paymentStatus

	if err := s.repo.Update(session); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return &UpdatePaymentResponse{
		ID:            session.ID,
		PaymentStatus: session.PaymentStatus,
	}, nil
}

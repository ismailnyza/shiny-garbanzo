package order

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/metrics"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"github.com/ismael/qr-restaurant/internal/shared/modifiers"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SessionRepository provides session lookups for order placement.
type SessionRepository interface {
	FindBySessionToken(token uuid.UUID) (*models.Session, error)
	FindByID(id uuid.UUID) (*models.Session, error)
	TouchLastSeen(sessionID uuid.UUID) error
}

// MenuRepository provides menu item lookups for order placement.
type MenuRepository interface {
	FindItemByID(id uuid.UUID) (*models.MenuItem, error)
}

// Valid status transitions
var validTransitions = map[string][]string{
	"PENDING":   {"ACCEPTED", "CANCELLED"},
	"ACCEPTED":  {"PREPARING", "CANCELLED"},
	"PREPARING": {"READY"},
	"READY":     {"COMPLETED"},
	"COMPLETED": {},
	"CANCELLED": {},
}

// OrderStatusBroadcaster is called when an order status changes (for WebSocket updates)
type OrderStatusBroadcaster interface {
	BroadcastOrderUpdate(sessionToken string, order *OrderResponse)
}

type Service interface {
	PlaceOrder(sessionToken uuid.UUID, req *PlaceOrderRequest) (*OrderResponse, error)
	ListCustomerOrders(sessionToken uuid.UUID) ([]OrderResponse, error)
	ListAdminOrders(restaurantID uuid.UUID, status, sessionIDStr, sort string, page, perPage, offset int) ([]AdminOrderListResponse, int64, error)
	GetAdminOrder(restaurantID, orderID uuid.UUID) (*OrderResponse, error)
	UpdateStatus(restaurantID, orderID uuid.UUID, status string) (*StatusUpdateResponse, error)
}

type service struct {
	repo        Repository
	sessionRepo SessionRepository
	menuRepo    MenuRepository
	broadcaster OrderStatusBroadcaster
}

func NewService(repo Repository, sessionRepo SessionRepository, menuRepo MenuRepository, broadcaster OrderStatusBroadcaster) Service {
	return &service{
		repo:        repo,
		sessionRepo: sessionRepo,
		menuRepo:    menuRepo,
		broadcaster: broadcaster,
	}
}

func (s *service) PlaceOrder(sessionToken uuid.UUID, req *PlaceOrderRequest) (*OrderResponse, error) {
	// Validate session exists and is ACTIVE
	session, err := s.sessionRepo.FindBySessionToken(sessionToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Session not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	if session.Status != "ACTIVE" {
		metrics.Default.IncOrderFailure()
		return nil, apperrors.NewForbidden("Session is not active")
	}
	if isSessionExpired(session) {
		metrics.Default.IncOrderFailure()
		return nil, apperrors.NewForbidden("Session has expired")
	}

	requestHash := requestFingerprint(req)
	if req.IdempotencyKey != nil {
		existing, err := s.repo.FindBySessionAndIdempotencyKey(session.ID, *req.IdempotencyKey)
		if err == nil {
			if existing.RequestHash != "" && existing.RequestHash != requestHash {
				return nil, apperrors.NewConflict("Idempotency key was already used for a different order request")
			}
			return orderToResponse(existing), nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewInternal(err)
		}
	}

	// Fetch all menu items and validate
	type itemInfo struct {
		menuItem        *models.MenuItem
		quantity        int
		selectedOptions datatypes.JSON
		modifierDelta   int
	}
	var items []itemInfo

	for _, reqItem := range req.Items {
		menuItem, err := s.menuRepo.FindItemByID(reqItem.MenuItemID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.NewValidation(fmt.Sprintf("Menu item not found: %s", reqItem.MenuItemID), nil)
			}
			return nil, apperrors.NewInternal(err)
		}

		if menuItem.RestaurantID != session.RestaurantID {
			return nil, apperrors.NewValidation(fmt.Sprintf("Menu item %s does not belong to this restaurant", reqItem.MenuItemID), nil)
		}

		if !menuItem.IsAvailable {
			return nil, apperrors.NewUnprocessable(fmt.Sprintf("Menu item '%s' is not available", menuItem.Name))
		}

		selectedOptions, modifierDelta, err := modifiers.BuildSnapshot(menuItem.OptionsConfig, reqItem.SelectedOptions, reqItem.Quantity)
		if err != nil {
			return nil, err
		}

		items = append(items, itemInfo{
			menuItem:        menuItem,
			quantity:        reqItem.Quantity,
			selectedOptions: selectedOptions,
			modifierDelta:   modifierDelta,
		})
	}

	// Calculate total
	totalCents := 0
	for _, item := range items {
		totalCents += item.menuItem.PriceCents * item.quantity
		totalCents += item.modifierDelta
	}

	// Create order and order items in transaction
	db := s.repo.GetDB()
	var order *models.Order

	err = db.Transaction(func(tx *gorm.DB) error {
		// Lock session row
		var sess models.Session
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&sess, "id = ?", session.ID).Error; err != nil {
			return err
		}
		if sess.Status != "ACTIVE" {
			return apperrors.NewForbidden("Session is not active")
		}
		if isSessionExpired(&sess) {
			return apperrors.NewForbidden("Session has expired")
		}

		order = &models.Order{
			SessionID:      session.ID,
			RestaurantID:   session.RestaurantID,
			IdempotencyKey: req.IdempotencyKey,
			RequestHash:    requestHash,
			Status:         "PENDING",
			TotalCents:     totalCents,
			Notes:          req.Notes,
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		for _, item := range items {
			orderItem := &models.OrderItem{
				OrderID:            order.ID,
				MenuItemID:         &item.menuItem.ID,
				NameSnapshot:       item.menuItem.Name,
				PriceCentsSnapshot: item.menuItem.PriceCents,
				Quantity:           item.quantity,
				SelectedOptions:    item.selectedOptions,
			}
			if err := tx.Create(orderItem).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		if req.IdempotencyKey != nil {
			existing, findErr := s.repo.FindBySessionAndIdempotencyKey(session.ID, *req.IdempotencyKey)
			if findErr == nil {
				if existing.RequestHash != "" && existing.RequestHash != requestHash {
					return nil, apperrors.NewConflict("Idempotency key was already used for a different order request")
				}
				return orderToResponse(existing), nil
			}
		}
		metrics.Default.IncOrderFailure()
		return nil, err
	}
	_ = s.sessionRepo.TouchLastSeen(session.ID)

	// Reload order with items
	loadedOrder, err := s.repo.FindByID(order.ID)
	if err != nil {
		metrics.Default.IncOrderFailure()
		return nil, apperrors.NewInternal(err)
	}

	resp := orderToResponse(loadedOrder)
	metrics.Default.IncOrderSuccess()
	return resp, nil
}

func isSessionExpired(session *models.Session) bool {
	return session != nil && session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt)
}

func requestFingerprint(req *PlaceOrderRequest) string {
	b, _ := json.Marshal(req)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *service) ListCustomerOrders(sessionToken uuid.UUID) ([]OrderResponse, error) {
	session, err := s.sessionRepo.FindBySessionToken(sessionToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Session not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if session.Status != "ACTIVE" {
		return nil, apperrors.NewForbidden("Session is not active")
	}
	if isSessionExpired(session) {
		return nil, apperrors.NewForbidden("Session has expired")
	}
	_ = s.sessionRepo.TouchLastSeen(session.ID)

	orders, err := s.repo.FindBySession(session.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	responses := make([]OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = *orderToResponse(&o)
	}

	return responses, nil
}

func (s *service) ListAdminOrders(restaurantID uuid.UUID, status, sessionIDStr, sort string, page, perPage, offset int) ([]AdminOrderListResponse, int64, error) {
	sortClause, err := normalizeOrderSort(sort)
	if err != nil {
		return nil, 0, err
	}

	orders, total, err := s.repo.FindByRestaurant(restaurantID, status, sessionIDStr, sortClause, page, perPage, offset)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err)
	}

	responses := make([]AdminOrderListResponse, len(orders))
	for i, o := range orders {
		items := make([]OrderItemResp, len(o.Items))
		for j, item := range o.Items {
			var opts map[string]interface{}
			if len(item.SelectedOptions) > 0 {
				_ = json.Unmarshal(item.SelectedOptions, &opts)
			}
			items[j] = OrderItemResp{
				NameSnapshot:       item.NameSnapshot,
				PriceCentsSnapshot: item.PriceCentsSnapshot,
				Quantity:           item.Quantity,
				SelectedOptions:    opts,
			}
		}
		responses[i] = AdminOrderListResponse{
			ID:         o.ID,
			SessionID:  o.SessionID,
			TableCode:  "",
			Status:     o.Status,
			TotalCents: o.TotalCents,
			Notes:      o.Notes,
			CreatedAt:  o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Items:      items,
		}
	}

	return responses, total, nil
}

func (s *service) GetAdminOrder(restaurantID, orderID uuid.UUID) (*OrderResponse, error) {
	order, err := s.repo.FindByID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Order not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if order.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Order not found")
	}

	return orderToResponse(order), nil
}

func (s *service) UpdateStatus(restaurantID, orderID uuid.UUID, status string) (*StatusUpdateResponse, error) {
	order, err := s.repo.FindByID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Order not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if order.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Order not found")
	}

	// Validate transition
	allowed, ok := validTransitions[order.Status]
	if !ok {
		return nil, apperrors.NewConflict(fmt.Sprintf("Invalid current status: %s", order.Status))
	}

	isValid := false
	for _, s := range allowed {
		if s == status {
			isValid = true
			break
		}
	}

	if !isValid {
		return nil, apperrors.NewConflict(
			fmt.Sprintf("Cannot transition from '%s' to '%s'", order.Status, status),
		)
	}

	if err := s.repo.UpdateStatus(orderID, status); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	// Broadcast WebSocket update
	if s.broadcaster != nil {
		// Find session to get session token
		session, err := s.sessionRepo.FindByID(order.SessionID)
		if err == nil {
			updatedOrder, _ := s.repo.FindByID(orderID)
			orderResp := orderToResponse(updatedOrder)
			s.broadcaster.BroadcastOrderUpdate(session.SessionToken.String(), orderResp)
		}
	}

	return &StatusUpdateResponse{
		ID:        order.ID,
		Status:    status,
		UpdatedAt: "", // We don't re-fetch, just return
	}, nil
}

func normalizeOrderSort(sort string) (string, error) {
	switch sort {
	case "", "created_at_desc":
		return "created_at DESC", nil
	case "created_at_asc":
		return "created_at ASC", nil
	case "status_asc":
		return "status ASC, created_at DESC", nil
	case "status_desc":
		return "status DESC, created_at DESC", nil
	default:
		return "", apperrors.NewValidation("Invalid sort parameter", "Allowed values: created_at_desc, created_at_asc, status_asc, status_desc")
	}
}

func orderToResponse(o *models.Order) *OrderResponse {
	items := make([]OrderItemResp, len(o.Items))
	for i, item := range o.Items {
		var opts map[string]interface{}
		if len(item.SelectedOptions) > 0 {
			_ = json.Unmarshal(item.SelectedOptions, &opts)
		}
		items[i] = OrderItemResp{
			NameSnapshot:       item.NameSnapshot,
			PriceCentsSnapshot: item.PriceCentsSnapshot,
			Quantity:           item.Quantity,
			SelectedOptions:    opts,
		}
	}

	return &OrderResponse{
		ID:             o.ID,
		SessionID:      o.SessionID,
		IdempotencyKey: o.IdempotencyKey,
		Status:         o.Status,
		TotalCents:     o.TotalCents,
		Notes:          o.Notes,
		CreatedAt:      o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Items:          items,
	}
}

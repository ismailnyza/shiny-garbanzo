package order

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type fakeOrderRepo struct {
	order        *models.Order
	updateCalled bool
	sortSeen     string
}

func (r *fakeOrderRepo) CreateOrder(tx *gorm.DB, order *models.Order) error { return nil }
func (r *fakeOrderRepo) FindByID(id uuid.UUID) (*models.Order, error) {
	if r.order == nil || r.order.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return r.order, nil
}
func (r *fakeOrderRepo) FindBySessionAndIdempotencyKey(sessionID uuid.UUID, key string) (*models.Order, error) {
	if r.order == nil || r.order.SessionID != sessionID || r.order.IdempotencyKey == nil || *r.order.IdempotencyKey != key {
		return nil, gorm.ErrRecordNotFound
	}
	return r.order, nil
}
func (r *fakeOrderRepo) FindBySession(sessionID uuid.UUID) ([]models.Order, error) { return nil, nil }
func (r *fakeOrderRepo) FindByRestaurant(restaurantID uuid.UUID, status, sessionIDStr, sort string, page, perPage, offset int) ([]models.Order, int64, error) {
	r.sortSeen = sort
	return nil, 0, nil
}
func (r *fakeOrderRepo) UpdateStatus(id uuid.UUID, status string) error {
	r.updateCalled = true
	if r.order != nil {
		r.order.Status = status
	}
	return nil
}
func (r *fakeOrderRepo) CountBySession(sessionID uuid.UUID) (total int, pending int, err error) {
	return 0, 0, nil
}
func (r *fakeOrderRepo) CountOrdersBySession(sessionID uuid.UUID) (total int, pending int, err error) {
	return 0, 0, nil
}
func (r *fakeOrderRepo) FindOrdersBySession(sessionID uuid.UUID) ([]models.Order, error) {
	return nil, nil
}
func (r *fakeOrderRepo) GetDB() *gorm.DB { return nil }

type fakeOrderSessionRepo struct {
	session *models.Session
}

func (r *fakeOrderSessionRepo) FindBySessionToken(token uuid.UUID) (*models.Session, error) {
	if r.session == nil || r.session.SessionToken != token {
		return nil, gorm.ErrRecordNotFound
	}
	return r.session, nil
}
func (r *fakeOrderSessionRepo) FindByID(id uuid.UUID) (*models.Session, error) {
	if r.session == nil || r.session.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return r.session, nil
}
func (r *fakeOrderSessionRepo) TouchLastSeen(sessionID uuid.UUID) error { return nil }

type fakeOrderMenuRepo struct{}

func (r *fakeOrderMenuRepo) FindItemByID(id uuid.UUID) (*models.MenuItem, error) {
	return nil, errors.New("not implemented")
}

func TestGetAdminOrderRejectsForeignRestaurant(t *testing.T) {
	restaurantID := uuid.New()
	foreignRestaurantID := uuid.New()
	orderID := uuid.New()

	repo := &fakeOrderRepo{order: &models.Order{
		ID:           orderID,
		RestaurantID: foreignRestaurantID,
		Status:       "PENDING",
	}}
	svc := NewService(repo, &fakeOrderSessionRepo{}, &fakeOrderMenuRepo{}, nil)

	_, err := svc.GetAdminOrder(restaurantID, orderID)
	assertAppErrorCode(t, err, apperrors.CodeNotFound)
}

func TestUpdateStatusRejectsForeignRestaurantWithoutMutating(t *testing.T) {
	restaurantID := uuid.New()
	foreignRestaurantID := uuid.New()
	orderID := uuid.New()

	repo := &fakeOrderRepo{order: &models.Order{
		ID:           orderID,
		RestaurantID: foreignRestaurantID,
		Status:       "PENDING",
	}}
	svc := NewService(repo, &fakeOrderSessionRepo{}, &fakeOrderMenuRepo{}, nil)

	_, err := svc.UpdateStatus(restaurantID, orderID, "ACCEPTED")
	assertAppErrorCode(t, err, apperrors.CodeNotFound)
	if repo.updateCalled {
		t.Fatal("foreign order status update reached repository mutation")
	}
}

func TestListAdminOrdersRejectsUnsafeSort(t *testing.T) {
	repo := &fakeOrderRepo{}
	svc := NewService(repo, &fakeOrderSessionRepo{}, &fakeOrderMenuRepo{}, nil)

	_, _, err := svc.ListAdminOrders(uuid.New(), "", "", "created_at DESC, (SELECT pg_sleep(2))", 1, 20, 0)
	assertAppErrorCode(t, err, apperrors.CodeValidation)
	if repo.sortSeen != "" {
		t.Fatalf("unsafe sort reached repository: %q", repo.sortSeen)
	}
}

func TestListAdminOrdersNormalizesAllowedSort(t *testing.T) {
	repo := &fakeOrderRepo{}
	svc := NewService(repo, &fakeOrderSessionRepo{}, &fakeOrderMenuRepo{}, nil)

	_, _, err := svc.ListAdminOrders(uuid.New(), "", "", "created_at_asc", 1, 20, 0)
	if err != nil {
		t.Fatalf("expected allowed sort to pass, got %v", err)
	}
	if repo.sortSeen != "created_at ASC" {
		t.Fatalf("expected normalized sort, got %q", repo.sortSeen)
	}
}

func TestPlaceOrderReturnsExistingOrderForRepeatedIdempotencyKey(t *testing.T) {
	sessionToken := uuid.New()
	sessionID := uuid.New()
	key := uuid.New().String()
	orderID := uuid.New()
	repo := &fakeOrderRepo{order: &models.Order{
		ID:             orderID,
		SessionID:      sessionID,
		RestaurantID:   uuid.New(),
		IdempotencyKey: &key,
		Status:         "PENDING",
		TotalCents:     1200,
	}}
	svc := NewService(repo, &fakeOrderSessionRepo{session: &models.Session{
		ID:           sessionID,
		SessionToken: sessionToken,
		Status:       "ACTIVE",
	}}, &fakeOrderMenuRepo{}, nil)

	got, err := svc.PlaceOrder(sessionToken, &PlaceOrderRequest{
		IdempotencyKey: &key,
		Items: []OrderItemRequest{{
			MenuItemID: uuid.New(),
			Quantity:   1,
		}},
	})
	if err != nil {
		t.Fatalf("expected idempotency replay to succeed, got %v", err)
	}
	if got.ID != orderID {
		t.Fatalf("expected existing order %s, got %s", orderID, got.ID)
	}
}

func TestPlaceOrderRejectsSameIdempotencyKeyDifferentRequest(t *testing.T) {
	sessionToken := uuid.New()
	sessionID := uuid.New()
	key := uuid.New().String()
	originalReq := &PlaceOrderRequest{
		IdempotencyKey: &key,
		Items: []OrderItemRequest{{
			MenuItemID: uuid.New(),
			Quantity:   1,
		}},
	}
	repo := &fakeOrderRepo{order: &models.Order{
		ID:             uuid.New(),
		SessionID:      sessionID,
		RestaurantID:   uuid.New(),
		IdempotencyKey: &key,
		RequestHash:    requestFingerprint(originalReq),
		Status:         "PENDING",
	}}
	svc := NewService(repo, &fakeOrderSessionRepo{session: &models.Session{
		ID:           sessionID,
		SessionToken: sessionToken,
		Status:       "ACTIVE",
	}}, &fakeOrderMenuRepo{}, nil)

	_, err := svc.PlaceOrder(sessionToken, &PlaceOrderRequest{
		IdempotencyKey: &key,
		Items: []OrderItemRequest{{
			MenuItemID: uuid.New(),
			Quantity:   2,
		}},
	})
	assertAppErrorCode(t, err, apperrors.CodeConflict)
}

func TestPlaceOrderRejectsExpiredSession(t *testing.T) {
	sessionToken := uuid.New()
	sessionID := uuid.New()
	expiredAt := time.Now().Add(-time.Minute)

	svc := NewService(&fakeOrderRepo{}, &fakeOrderSessionRepo{session: &models.Session{
		ID:           sessionID,
		SessionToken: sessionToken,
		Status:       "ACTIVE",
		ExpiresAt:    &expiredAt,
	}}, &fakeOrderMenuRepo{}, nil)

	_, err := svc.PlaceOrder(sessionToken, &PlaceOrderRequest{
		Items: []OrderItemRequest{{
			MenuItemID: uuid.New(),
			Quantity:   1,
		}},
	})
	assertAppErrorCode(t, err, apperrors.CodeForbidden)
}

func assertAppErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error, got nil", code)
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Code != code {
		t.Fatalf("expected %s, got %s", code, appErr.Code)
	}
}

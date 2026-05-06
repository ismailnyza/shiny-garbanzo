package session

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type fakeSessionRepo struct {
	session                *models.Session
	updateCalled           bool
	findBySessionTokenFunc func(uuid.UUID) (*models.Session, error)
}

func (r *fakeSessionRepo) Create(session *models.Session) error { return nil }
func (r *fakeSessionRepo) FindByID(id uuid.UUID) (*models.Session, error) {
	if r.session == nil || r.session.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return r.session, nil
}
func (r *fakeSessionRepo) FindBySessionToken(token uuid.UUID) (*models.Session, error) {
	if r.findBySessionTokenFunc != nil {
		return r.findBySessionTokenFunc(token)
	}
	return nil, errors.New("not implemented")
}
func (r *fakeSessionRepo) FindActiveByTableID(tableID uuid.UUID) (*models.Session, error) {
	return nil, errors.New("not implemented")
}
func (r *fakeSessionRepo) ListByRestaurant(restaurantID uuid.UUID, status string, page, perPage, offset int) ([]models.Session, int64, error) {
	return nil, 0, nil
}
func (r *fakeSessionRepo) Update(session *models.Session) error {
	r.updateCalled = true
	r.session = session
	return nil
}
func (r *fakeSessionRepo) CountOrdersBySession(sessionID uuid.UUID) (total int, pending int, err error) {
	return 0, 0, nil
}
func (r *fakeSessionRepo) TouchLastSeen(sessionID uuid.UUID) error { return nil }

type fakeSessionOrderRepo struct{}

func (r *fakeSessionOrderRepo) CountOrdersBySession(sessionID uuid.UUID) (int, int, error) {
	return 0, 0, nil
}
func (r *fakeSessionOrderRepo) FindOrdersBySession(sessionID uuid.UUID) ([]models.Order, error) {
	return nil, nil
}

func TestAdminSessionDetailRejectsForeignRestaurant(t *testing.T) {
	restaurantID := uuid.New()
	sessionID := uuid.New()
	repo := &fakeSessionRepo{session: &models.Session{
		ID:           sessionID,
		RestaurantID: uuid.New(),
		Status:       "ACTIVE",
	}}
	svc := NewService(repo, nil, nil, &fakeSessionOrderRepo{})

	_, err := svc.GetByID(restaurantID, sessionID)
	assertSessionAppErrorCode(t, err, apperrors.CodeNotFound)
}

func TestCloseRejectsForeignRestaurantWithoutMutating(t *testing.T) {
	restaurantID := uuid.New()
	sessionID := uuid.New()
	repo := &fakeSessionRepo{session: &models.Session{
		ID:           sessionID,
		RestaurantID: uuid.New(),
		Status:       "ACTIVE",
	}}
	svc := NewService(repo, nil, nil, &fakeSessionOrderRepo{})

	_, err := svc.Close(restaurantID, sessionID, true)
	assertSessionAppErrorCode(t, err, apperrors.CodeNotFound)
	if repo.updateCalled {
		t.Fatal("foreign session close reached repository mutation")
	}
}

func TestUpdatePaymentRejectsForeignRestaurantWithoutMutating(t *testing.T) {
	restaurantID := uuid.New()
	sessionID := uuid.New()
	repo := &fakeSessionRepo{session: &models.Session{
		ID:            sessionID,
		RestaurantID:  uuid.New(),
		Status:        "ACTIVE",
		PaymentStatus: "PENDING",
	}}
	svc := NewService(repo, nil, nil, &fakeSessionOrderRepo{})

	_, err := svc.UpdatePayment(restaurantID, sessionID, "PAID")
	assertSessionAppErrorCode(t, err, apperrors.CodeNotFound)
	if repo.updateCalled {
		t.Fatal("foreign session payment update reached repository mutation")
	}
}

func TestGetPublicSessionRejectsExpiredActiveSession(t *testing.T) {
	sessionToken := uuid.New()
	expiredAt := time.Now().Add(-time.Minute)
	repo := &fakeSessionRepo{session: &models.Session{
		ID:           uuid.New(),
		SessionToken: sessionToken,
		Status:       "ACTIVE",
		ExpiresAt:    &expiredAt,
	}}
	repo.findBySessionTokenFunc = func(token uuid.UUID) (*models.Session, error) {
		if token != sessionToken {
			return nil, gorm.ErrRecordNotFound
		}
		return repo.session, nil
	}
	svc := NewService(repo, nil, nil, &fakeSessionOrderRepo{})

	_, err := svc.GetPublicSession(sessionToken)
	assertSessionAppErrorCode(t, err, apperrors.CodeForbidden)
}

func assertSessionAppErrorCode(t *testing.T, err error, code string) {
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

package staff

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type fakeStaffRepo struct {
	staff        *models.RestaurantStaff
	deleteCalled bool
}

func (r *fakeStaffRepo) Create(staff *models.RestaurantStaff) error { return nil }
func (r *fakeStaffRepo) ListByRestaurant(restaurantID uuid.UUID) ([]models.RestaurantStaff, error) {
	return nil, nil
}
func (r *fakeStaffRepo) FindByID(id uuid.UUID) (*models.RestaurantStaff, error) {
	if r.staff == nil || r.staff.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return r.staff, nil
}
func (r *fakeStaffRepo) Delete(id uuid.UUID) error {
	r.deleteCalled = true
	return nil
}

type fakeStaffUserRepo struct{}

func (r *fakeStaffUserRepo) FindByEmail(email string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (r *fakeStaffUserRepo) Create(user *models.User) error { return nil }

func TestRemoveRejectsForeignRestaurantWithoutDeleting(t *testing.T) {
	restaurantID := uuid.New()
	staffID := uuid.New()
	repo := &fakeStaffRepo{staff: &models.RestaurantStaff{
		ID:           staffID,
		RestaurantID: uuid.New(),
		UserID:       uuid.New(),
	}}
	svc := NewService(repo, &fakeStaffUserRepo{})

	err := svc.Remove(restaurantID, staffID)
	assertStaffAppErrorCode(t, err, apperrors.CodeNotFound)
	if repo.deleteCalled {
		t.Fatal("foreign staff assignment reached repository delete")
	}
}

func assertStaffAppErrorCode(t *testing.T, err error, code string) {
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

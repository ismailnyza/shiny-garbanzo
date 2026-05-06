package staff

import (
	"errors"

	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"github.com/ismael/qr-restaurant/pkg/password"
	"gorm.io/gorm"
)

type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	Create(user *models.User) error
}

type Service interface {
	Create(restaurantID uuid.UUID, req *CreateRequest, bcryptCost int) (*Response, error)
	List(restaurantID uuid.UUID) ([]StaffMemberResponse, error)
	Remove(restaurantID, staffID uuid.UUID) error
}

type service struct {
	repo     Repository
	userRepo UserRepository
}

func NewService(repo Repository, userRepo UserRepository) Service {
	return &service{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *service) Create(restaurantID uuid.UUID, req *CreateRequest, bcryptCost int) (*Response, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewInternal(err)
	}

	var user *models.User
	if existingUser != nil {
		if existingUser.Role != "STAFF" {
			return nil, apperrors.NewConflict("User exists but is not a STAFF role")
		}
		user = existingUser
	} else {
		hash, err := password.Hash(req.Password, bcryptCost)
		if err != nil {
			return nil, apperrors.NewInternal(err)
		}
		user = &models.User{
			Email:        req.Email,
			PasswordHash: hash,
			Role:         "STAFF",
		}
		if err := s.userRepo.Create(user); err != nil {
			return nil, apperrors.NewInternal(err)
		}
	}

	staff := &models.RestaurantStaff{
		RestaurantID: restaurantID,
		UserID:       user.ID,
	}
	if err := s.repo.Create(staff); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return &Response{
		StaffID:      staff.ID,
		UserID:       user.ID,
		Email:        user.Email,
		RestaurantID: restaurantID,
		CreatedAt:    staff.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *service) List(restaurantID uuid.UUID) ([]StaffMemberResponse, error) {
	staffList, err := s.repo.ListByRestaurant(restaurantID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	responses := make([]StaffMemberResponse, len(staffList))
	for i, s := range staffList {
		responses[i] = StaffMemberResponse{
			UserID:    s.User.ID,
			Email:     s.User.Email,
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return responses, nil
}

func (s *service) Remove(restaurantID, staffID uuid.UUID) error {
	staff, err := s.repo.FindByID(staffID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Staff assignment not found")
		}
		return apperrors.NewInternal(err)
	}
	if staff.RestaurantID != restaurantID {
		return apperrors.NewNotFound("Staff assignment not found")
	}

	if err := s.repo.Delete(staffID); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

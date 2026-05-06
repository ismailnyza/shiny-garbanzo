package restaurant

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"gorm.io/gorm"
)

type Service interface {
	Create(req *CreateRequest, ownerID uuid.UUID) (*Response, error)
	GetByID(id uuid.UUID) (*Response, error)
	ListByOwner(ownerID uuid.UUID, page, perPage, offset int, isActive *bool) ([]Response, int64, error)
	Update(id uuid.UUID, req *UpdateRequest) (*Response, error)
	SoftDelete(id uuid.UUID) error
}

type service struct {
	repo        Repository
	userRepo    UserRepository
	staffRepo   StaffRepository
	themeRepo   ThemeRepository
	bcryptCost  int
}

// UserRepository is the subset of auth repo needed for staff creation.
type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uuid.UUID) (*models.User, error)
	Create(user *models.User) error
}

type StaffRepository interface {
	Create(staff *models.RestaurantStaff) error
}

type ThemeRepository interface {
	Create(theme *models.Theme) error
}

func NewService(repo Repository, userRepo UserRepository, staffRepo StaffRepository, themeRepo ThemeRepository, bcryptCost int) Service {
	return &service{
		repo:       repo,
		userRepo:   userRepo,
		staffRepo:  staffRepo,
		themeRepo:  themeRepo,
		bcryptCost: bcryptCost,
	}
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]`)

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonAlphanumericRegex.ReplaceAllString(slug, "")
	slug = strings.Trim(slug, "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	if slug == "" {
		slug = "restaurant"
	}
	return slug
}

func (s *service) Create(req *CreateRequest, ownerID uuid.UUID) (*Response, error) {
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	// Ensure unique slug by appending -N on conflict
	baseSlug := slug
	for counter := 1; ; counter++ {
		existing, err := s.repo.FindBySlug(slug)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return nil, apperrors.NewInternal(err)
		}
		if existing == nil {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
	}

	restaurant := &models.Restaurant{
		OwnerID:     ownerID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Address:     req.Address,
		Phone:       req.Phone,
		IsActive:    true,
	}

	if err := s.repo.Create(restaurant); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	// Auto-create default theme
	theme := &models.Theme{
		RestaurantID: restaurant.ID,
	}
	if err := s.themeRepo.Create(theme); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return toResponse(restaurant), nil
}

func (s *service) GetByID(id uuid.UUID) (*Response, error) {
	restaurant, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Restaurant not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	return toResponse(restaurant), nil
}

func (s *service) ListByOwner(ownerID uuid.UUID, page, perPage, offset int, isActive *bool) ([]Response, int64, error) {
	restaurants, total, err := s.repo.ListByOwner(ownerID, page, perPage, offset, isActive)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err)
	}

	responses := make([]Response, len(restaurants))
	for i, r := range restaurants {
		responses[i] = *toResponse(&r)
	}
	return responses, total, nil
}

func (s *service) Update(id uuid.UUID, req *UpdateRequest) (*Response, error) {
	restaurant, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Restaurant not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	if req.Name != nil {
		restaurant.Name = *req.Name
	}
	if req.Description != nil {
		restaurant.Description = req.Description
	}
	if req.Address != nil {
		restaurant.Address = req.Address
	}
	if req.Phone != nil {
		restaurant.Phone = req.Phone
	}
	if req.IsActive != nil {
		restaurant.IsActive = *req.IsActive
	}

	if err := s.repo.Update(restaurant); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return toResponse(restaurant), nil
}

func (s *service) SoftDelete(id uuid.UUID) error {
	if err := s.repo.SoftDelete(id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

func toResponse(r *models.Restaurant) *Response {
	resp := &Response{
		ID:          r.ID,
		OwnerID:     r.OwnerID,
		Name:        r.Name,
		Slug:        r.Slug,
		Description: r.Description,
		Address:     r.Address,
		Phone:       r.Phone,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.Theme != nil {
		resp.Theme = &ThemeRef{
			PrimaryColor:   r.Theme.PrimaryColor,
			SecondaryColor: r.Theme.SecondaryColor,
			LogoURL:        r.Theme.LogoURL,
			BannerURL:      r.Theme.BannerURL,
			FontStyle:      r.Theme.FontStyle,
			DarkMode:       r.Theme.DarkMode,
		}
	}
	return resp
}

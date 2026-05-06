package theme

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"gorm.io/gorm"
)

type Service interface {
	Get(restaurantID uuid.UUID) (*Response, error)
	Upsert(restaurantID uuid.UUID, req *UpsertRequest) (*Response, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Get(restaurantID uuid.UUID) (*Response, error) {
	theme, err := s.repo.FindByRestaurant(restaurantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Theme not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	return toResponse(theme), nil
}

func (s *service) Upsert(restaurantID uuid.UUID, req *UpsertRequest) (*Response, error) {
	// Try to find existing theme
	theme, err := s.repo.FindByRestaurant(restaurantID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewInternal(err)
		}
		// Create new theme
		theme = &models.Theme{
			RestaurantID: restaurantID,
		}
	}

	// Apply partial updates
	if req.PrimaryColor != nil {
		theme.PrimaryColor = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		theme.SecondaryColor = *req.SecondaryColor
	}
	if req.LogoURL != nil {
		theme.LogoURL = req.LogoURL
	}
	if req.BannerURL != nil {
		theme.BannerURL = req.BannerURL
	}
	if req.FontStyle != nil {
		theme.FontStyle = *req.FontStyle
	}
	if req.DarkMode != nil {
		theme.DarkMode = *req.DarkMode
	}

	if err := s.repo.Upsert(theme); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return toResponse(theme), nil
}

func toResponse(t *models.Theme) *Response {
	return &Response{
		ID:             t.ID,
		RestaurantID:   t.RestaurantID,
		PrimaryColor:   t.PrimaryColor,
		SecondaryColor: t.SecondaryColor,
		LogoURL:        t.LogoURL,
		BannerURL:      t.BannerURL,
		FontStyle:      t.FontStyle,
		DarkMode:       t.DarkMode,
	}
}

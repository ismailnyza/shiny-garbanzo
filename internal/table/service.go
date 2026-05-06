package table

import (
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"github.com/ismael/qr-restaurant/pkg/qr"
	"gorm.io/gorm"
)

type Service interface {
	Create(restaurantID uuid.UUID, req *CreateRequest, frontendBaseURL string) (*Response, error)
	List(restaurantID uuid.UUID, page, perPage, offset int, isActive *bool) ([]Response, int64, error)
	Update(restaurantID uuid.UUID, tableID uuid.UUID, req *UpdateRequest) (*Response, error)
	Delete(restaurantID uuid.UUID, tableID uuid.UUID) error
	RegenerateQR(restaurantID uuid.UUID, tableID uuid.UUID, frontendBaseURL string) (*QRRegenResponse, error)
}

type service struct {
	repo     Repository
	restRepo RestaurantFinder
	cfg      *config.Config
}

// RestaurantRepository is the interface for finding a restaurant (for slug resolution in QR URL)
type RestaurantFinder interface {
	FindByID(id uuid.UUID) (*models.Restaurant, error)
}

func NewService(repo Repository, restRepo ...RestaurantFinder) Service {
	var rf RestaurantFinder
	if len(restRepo) > 0 {
		rf = restRepo[0]
	}
	return &service{repo: repo, restRepo: rf}
}

func (s *service) Create(restaurantID uuid.UUID, req *CreateRequest, frontendBaseURL string) (*Response, error) {
	// Check for duplicate table code
	exists, err := s.repo.ExistsByRestaurantAndCode(restaurantID, req.TableCode)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	if exists {
		return nil, apperrors.NewConflict("Table code already exists in this restaurant")
	}

	token, err := qr.GenerateToken(32)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	table := &models.Table{
		RestaurantID: restaurantID,
		TableCode:    req.TableCode,
		QRToken:      token,
		IsActive:     true,
	}

	if err := s.repo.Create(table); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return s.toResponse(table, frontendBaseURL), nil
}

func (s *service) List(restaurantID uuid.UUID, page, perPage, offset int, isActive *bool) ([]Response, int64, error) {
	tables, total, err := s.repo.FindByRestaurant(restaurantID, page, perPage, offset, isActive)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err)
	}

	responses := make([]Response, len(tables))
	for i, t := range tables {
		resp := s.toResponse(&t, "")
		// Fetch active session for each table
		sessionID, _ := s.repo.FindActiveSessionID(t.ID)
		resp.ActiveSessionID = sessionID
		responses[i] = *resp
	}
	return responses, total, nil
}

func (s *service) Update(restaurantID uuid.UUID, tableID uuid.UUID, req *UpdateRequest) (*Response, error) {
	table, err := s.repo.FindByID(tableID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Table not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	if table.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Table not found")
	}

	if req.TableCode != nil {
		exists, err := s.repo.ExistsByRestaurantAndCode(restaurantID, *req.TableCode)
		if err != nil {
			return nil, apperrors.NewInternal(err)
		}
		if exists && table.TableCode != *req.TableCode {
			return nil, apperrors.NewConflict("Table code already exists")
		}
		table.TableCode = *req.TableCode
	}
	if req.IsActive != nil {
		table.IsActive = *req.IsActive
	}

	if err := s.repo.Update(table); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return s.toResponse(table, ""), nil
}

func (s *service) Delete(restaurantID uuid.UUID, tableID uuid.UUID) error {
	table, err := s.repo.FindByID(tableID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Table not found")
		}
		return apperrors.NewInternal(err)
	}

	if table.RestaurantID != restaurantID {
		return apperrors.NewNotFound("Table not found")
	}

	// Check for active session
	sessionID, err := s.repo.FindActiveSessionID(tableID)
	if err != nil {
		return apperrors.NewInternal(err)
	}
	if sessionID != nil {
		return apperrors.NewConflict("Cannot delete table with an active session")
	}

	if err := s.repo.Delete(tableID); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

func (s *service) RegenerateQR(restaurantID uuid.UUID, tableID uuid.UUID, frontendBaseURL string) (*QRRegenResponse, error) {
	table, err := s.repo.FindByID(tableID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Table not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	if table.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Table not found")
	}

	token, err := qr.GenerateToken(32)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	table.QRToken = token
	if err := s.repo.Update(table); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	qrURL := s.buildQRURLForTable(frontendBaseURL, table)

	return &QRRegenResponse{
		QRToken: token,
		QRURL:   qrURL,
	}, nil
}

func (s *service) toResponse(t *models.Table, frontendBaseURL string) *Response {
	return &Response{
		ID:           t.ID,
		RestaurantID: t.RestaurantID,
		TableCode:    t.TableCode,
		QRToken:      t.QRToken,
		QRURL:        s.buildQRURLForTable(frontendBaseURL, t),
		IsActive:     t.IsActive,
		CreatedAt:    t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *service) buildQRURLForTable(frontendBaseURL string, t *models.Table) string {
	slug := ""
	if s.restRepo != nil && t.RestaurantID != uuid.Nil {
		if restaurant, err := s.restRepo.FindByID(t.RestaurantID); err == nil {
			slug = restaurant.Slug
		}
	}
	return buildQRURL(frontendBaseURL, slug, t.TableCode, t.QRToken)
}

func buildQRURL(frontendBaseURL, slug, tableCode, qrToken string) string {
	if frontendBaseURL == "" {
		return ""
	}
	base := strings.TrimRight(frontendBaseURL, "/")
	return base + "/r/" + url.PathEscape(slug) + "/t/" + url.PathEscape(tableCode) + "?token=" + url.QueryEscape(qrToken)
}

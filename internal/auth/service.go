package auth

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/pkg/jwt"
	"github.com/ismael/qr-restaurant/pkg/password"
	"gorm.io/gorm"
)

type Service interface {
	Register(req *RegisterRequest) (*UserResponse, error)
	Login(req *LoginRequest, jwtSecret string, jwtExpiryHours int) (*AuthResponse, error)
	GetMe(userID uuid.UUID) (*UserResponse, error)
}

type service struct {
	repo       Repository
	bcryptCost int
}

func NewService(repo Repository, bcryptCost int) Service {
	return &service{repo: repo, bcryptCost: bcryptCost}
}

func (s *service) Register(req *RegisterRequest) (*UserResponse, error) {
	// Check if email already exists
	existing, err := s.repo.FindByEmail(req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewInternal(err)
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Email already registered")
	}

	hash, err := password.Hash(req.Password, s.bcryptCost)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return &UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

func (s *service) Login(req *LoginRequest, jwtSecret string, jwtExpiryHours int) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewUnauthorized("Invalid email or password")
		}
		return nil, apperrors.NewInternal(err)
	}

	if !password.Compare(user.PasswordHash, req.Password) {
		return nil, apperrors.NewUnauthorized("Invalid email or password")
	}

	token, err := jwt.Generate(jwtSecret, user.ID, user.Email, user.Role, jwtExpiryHours)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return &AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   jwtExpiryHours * 3600,
		User: UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (s *service) GetMe(userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("User not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	return &UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

// ExtractUserIDFromClaims extracts user ID from JWT claims stored in a map.
func ExtractUserIDFromClaims(claimsMap map[string]interface{}) (uuid.UUID, error) {
	userIDStr, ok := claimsMap["user_id"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("user_id not found in claims")
	}
	return uuid.Parse(userIDStr)
}

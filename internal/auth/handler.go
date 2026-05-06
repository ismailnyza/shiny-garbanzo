package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
)

type Handler struct {
	service Service
	cfg     *config.Config
}

func NewHandler(service Service, cfg *config.Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	user, err := h.service.Register(&req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Created(c, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	authResp, err := h.service.Login(&req, h.cfg.JWTSecret, h.cfg.JWTExpiryHours)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, http.StatusOK, authResp)
}

func (h *Handler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(userID.(string))

	user, err := h.service.GetMe(uid)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, http.StatusOK, user)
}

func handleError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *apperrors.AppError:
		switch e.Code {
		case apperrors.CodeNotFound:
			resp.NotFound(c, e.Message)
		case apperrors.CodeConflict:
			resp.Conflict(c, e.Message)
		case apperrors.CodeUnauthorized:
			resp.Unauthorized(c, e.Message)
		case apperrors.CodeForbidden:
			resp.Forbidden(c, e.Message)
		case apperrors.CodeValidation:
			resp.ValidationError(c, e.Message, e.Details)
		case apperrors.CodeUnprocessable:
			resp.Error(c, http.StatusUnprocessableEntity, e.Code, e.Message)
		case apperrors.CodeFileTooLarge:
			resp.Error(c, http.StatusRequestEntityTooLarge, e.Code, e.Message)
		default:
			resp.InternalError(c)
		}
	default:
		resp.InternalError(c)
	}
}

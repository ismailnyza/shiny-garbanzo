package staff

import (
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

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	result, err := h.service.Create(restaurantID, &req, h.cfg.BCryptCost)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Created(c, result)
}

func (h *Handler) List(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	staffList, err := h.service.List(restaurantID)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, staffList)
}

func (h *Handler) Remove(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	staffID := uuid.MustParse(c.Param("sid"))

	if err := h.service.Remove(restaurantID, staffID); err != nil {
		handleError(c, err)
		return
	}

	resp.NoContent(c)
}

func handleError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *apperrors.AppError:
		switch e.Code {
		case apperrors.CodeNotFound:
			resp.NotFound(c, e.Message)
		case apperrors.CodeConflict:
			resp.Conflict(c, e.Message)
		case apperrors.CodeForbidden:
			resp.Forbidden(c, e.Message)
		case apperrors.CodeValidation:
			resp.ValidationError(c, e.Message, e.Details)
		default:
			resp.InternalError(c)
		}
	default:
		resp.InternalError(c)
	}
}

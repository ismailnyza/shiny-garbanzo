package theme

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Get(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	result, err := h.service.Get(restaurantID)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	result, err := h.service.Upsert(restaurantID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

func handleError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *apperrors.AppError:
		switch e.Code {
		case apperrors.CodeNotFound:
			resp.NotFound(c, e.Message)
		default:
			resp.InternalError(c)
		}
	default:
		resp.InternalError(c)
	}
}

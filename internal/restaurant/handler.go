package restaurant

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/pagination"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	ownerID := uuid.MustParse(c.GetString("user_id"))

	result, err := h.service.Create(&req, ownerID)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Created(c, result)
}

func (h *Handler) ListOwned(c *gin.Context) {
	ownerID := uuid.MustParse(c.GetString("user_id"))
	params := pagination.ExtractPagination(c)

	var isActive *bool
	if c.Query("is_active") != "" {
		val := c.Query("is_active") == "true"
		isActive = &val
	}

	restaurants, total, err := h.service.ListByOwner(ownerID, params.Page, params.PerPage, params.Offset, isActive)
	if err != nil {
		handleError(c, err)
		return
	}

	if restaurants == nil {
		restaurants = []Response{}
	}

	resp.SuccessWithMeta(c, http.StatusOK, restaurants, &resp.Meta{
		Page:    params.Page,
		PerPage: params.PerPage,
		Total:   total,
	})
}

func (h *Handler) Get(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	result, err := h.service.GetByID(restaurantID)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, http.StatusOK, result)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	result, err := h.service.Update(restaurantID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, http.StatusOK, result)
}

func (h *Handler) Delete(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	if err := h.service.SoftDelete(restaurantID); err != nil {
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
		case apperrors.CodeUnauthorized:
			resp.Unauthorized(c, e.Message)
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

package table

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/audit"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/pagination"
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

	result, err := h.service.Create(restaurantID, &req, h.cfg.FrontendBaseURL)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Created(c, result)
}

func (h *Handler) List(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	params := pagination.ExtractPagination(c)

	var isActive *bool
	if c.Query("is_active") != "" {
		val := c.Query("is_active") == "true"
		isActive = &val
	}

	tables, total, err := h.service.List(restaurantID, params.Page, params.PerPage, params.Offset, isActive)
	if err != nil {
		handleError(c, err)
		return
	}

	if tables == nil {
		tables = []Response{}
	}

	resp.SuccessWithMeta(c, 200, tables, &resp.Meta{
		Page:    params.Page,
		PerPage: params.PerPage,
		Total:   total,
	})
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	tableID := uuid.MustParse(c.Param("tableId"))

	result, err := h.service.Update(restaurantID, tableID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

func (h *Handler) Delete(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	tableID := uuid.MustParse(c.Param("tableId"))

	if err := h.service.Delete(restaurantID, tableID); err != nil {
		handleError(c, err)
		return
	}

	resp.NoContent(c)
}

func (h *Handler) RegenerateQR(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	tableID := uuid.MustParse(c.Param("tableId"))

	result, err := h.service.RegenerateQR(restaurantID, tableID, h.cfg.FrontendBaseURL)
	if err != nil {
		handleError(c, err)
		return
	}

	audit.Record(c, "table.qr_regenerate", "table", tableID, nil)
	resp.Success(c, 200, result)
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

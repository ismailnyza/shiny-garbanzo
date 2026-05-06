package session

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/audit"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/pagination"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
	"gorm.io/gorm"
)

type Handler struct {
	service Service
	cfg     *config.Config
	db      *gorm.DB
}

func NewHandler(service Service, cfg *config.Config, db *gorm.DB) *Handler {
	return &Handler{service: service, cfg: cfg, db: db}
}

// Scan handles public QR code scan.
// GET /api/v1/public/scan?token=<qr_token>
func (h *Handler) Scan(c *gin.Context) {
	qrToken := c.Query("token")
	if qrToken == "" {
		resp.ValidationError(c, "token query parameter is required", nil)
		return
	}

	result, err := h.service.Scan(h.db, qrToken)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Set("session_id", result.SessionToken.String())
	resp.Success(c, 200, result)
}

// GetPublicSession handles customer polling session status.
// GET /api/v1/public/sessions/:sessionToken
func (h *Handler) GetPublic(c *gin.Context) {
	sessionToken, err := uuid.Parse(c.Param("sessionToken"))
	if err != nil {
		resp.NotFound(c, "Invalid session token")
		return
	}
	c.Set("session_id", sessionToken.String())

	result, err := h.service.GetPublicSession(sessionToken)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

// ListAdmin handles admin listing of sessions.
// GET /api/v1/restaurants/:restaurantId/sessions
func (h *Handler) ListAdmin(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	params := pagination.ExtractPagination(c)
	status := c.Query("status")

	sessions, total, err := h.service.ListByRestaurant(restaurantID, status, params.Page, params.PerPage, params.Offset)
	if err != nil {
		handleError(c, err)
		return
	}

	if sessions == nil {
		sessions = []AdminSessionListResponse{}
	}

	resp.SuccessWithMeta(c, 200, sessions, &resp.Meta{
		Page:    params.Page,
		PerPage: params.PerPage,
		Total:   total,
	})
}

// GetAdmin handles admin getting session detail.
// GET /api/v1/restaurants/:restaurantId/sessions/:sid
func (h *Handler) GetAdmin(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	sessionID := uuid.MustParse(c.Param("sid"))

	result, err := h.service.GetByID(restaurantID, sessionID)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

// Close handles admin closing a session.
// POST /api/v1/restaurants/:restaurantId/sessions/:sid/close
func (h *Handler) Close(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	sessionID := uuid.MustParse(c.Param("sid"))

	var req CloseSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is acceptable
	}

	force := false
	if req.Force != nil {
		force = *req.Force
	}

	result, err := h.service.Close(restaurantID, sessionID, force)
	if err != nil {
		handleError(c, err)
		return
	}

	audit.Record(c, "session.close", "session", sessionID, map[string]interface{}{
		"force": force,
	})
	resp.Success(c, 200, result)
}

// UpdatePayment handles admin updating payment status.
// PATCH /api/v1/restaurants/:restaurantId/sessions/:sid/payment
func (h *Handler) UpdatePayment(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	sessionID := uuid.MustParse(c.Param("sid"))

	var req UpdatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.UpdatePayment(restaurantID, sessionID, req.PaymentStatus)
	if err != nil {
		handleError(c, err)
		return
	}

	audit.Record(c, "session.payment_update", "session", sessionID, map[string]interface{}{
		"payment_status": req.PaymentStatus,
	})
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
		case apperrors.CodeUnprocessable:
			resp.Error(c, 422, e.Code, e.Message)
		default:
			resp.InternalError(c)
		}
	default:
		resp.InternalError(c)
	}
}

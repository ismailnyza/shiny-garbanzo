package order

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/audit"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/pagination"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
	"github.com/ismael/qr-restaurant/internal/websocket"
)

type Handler struct {
	service Service
	hub     *websocket.Hub
}

func NewHandler(service Service, hub *websocket.Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

// BroadcastOrderUpdate implements OrderStatusBroadcaster for WebSocket pushes.
func (h *Handler) BroadcastOrderUpdate(sessionToken string, order *OrderResponse) {
	data, _ := json.Marshal(order)
	h.hub.BroadcastToSession(sessionToken, &websocket.Message{
		Type: "order_updated",
		Data: json.RawMessage(data),
	})
}

// PlaceOrder handles customer placing an order.
// POST /api/v1/public/sessions/:sessionToken/orders
func (h *Handler) PlaceOrder(c *gin.Context) {
	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	sessionToken, err := uuid.Parse(c.Param("sessionToken"))
	if err != nil {
		resp.NotFound(c, "Invalid session token")
		return
	}
	c.Set("session_id", sessionToken.String())

	result, err := h.service.PlaceOrder(sessionToken, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Created(c, result)
}

// ListCustomerOrders handles customer polling their orders.
// GET /api/v1/public/sessions/:sessionToken/orders
func (h *Handler) ListCustomerOrders(c *gin.Context) {
	sessionToken, err := uuid.Parse(c.Param("sessionToken"))
	if err != nil {
		resp.NotFound(c, "Invalid session token")
		return
	}
	c.Set("session_id", sessionToken.String())

	orders, err := h.service.ListCustomerOrders(sessionToken)
	if err != nil {
		handleError(c, err)
		return
	}

	if orders == nil {
		orders = []OrderResponse{}
	}

	resp.Success(c, 200, orders)
}

// ListAdmin handles admin listing of orders.
// GET /api/v1/restaurants/:restaurantId/orders
func (h *Handler) ListAdmin(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	params := pagination.ExtractPagination(c)
	status := c.Query("status")
	sessionID := c.Query("session_id")
	sort := c.Query("sort")

	orders, total, err := h.service.ListAdminOrders(restaurantID, status, sessionID, sort, params.Page, params.PerPage, params.Offset)
	if err != nil {
		handleError(c, err)
		return
	}

	if orders == nil {
		orders = []AdminOrderListResponse{}
	}

	resp.SuccessWithMeta(c, 200, orders, &resp.Meta{
		Page:    params.Page,
		PerPage: params.PerPage,
		Total:   total,
	})
}

// GetAdmin handles admin getting order detail.
// GET /api/v1/restaurants/:restaurantId/orders/:oid
func (h *Handler) GetAdmin(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	orderID := uuid.MustParse(c.Param("oid"))

	order, err := h.service.GetAdminOrder(restaurantID, orderID)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, order)
}

// UpdateStatus handles admin updating order status.
// PATCH /api/v1/restaurants/:restaurantId/orders/:oid/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	orderID := uuid.MustParse(c.Param("oid"))

	result, err := h.service.UpdateStatus(restaurantID, orderID, req.Status)
	if err != nil {
		handleError(c, err)
		return
	}

	audit.Record(c, "order.status_update", "order", orderID, map[string]interface{}{
		"status": req.Status,
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

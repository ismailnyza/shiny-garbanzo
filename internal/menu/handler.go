package menu

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/audit"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"github.com/ismael/qr-restaurant/internal/shared/pagination"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
)

type Handler struct {
	service       Service
	sessionFinder SessionFinder
	restFinder    RestaurantFinder
}

type SessionFinder interface {
	FindBySessionToken(token uuid.UUID) (*models.Session, error)
	TouchLastSeen(sessionID uuid.UUID) error
}

type RestaurantFinder interface {
	FindByID(id uuid.UUID) (*models.Restaurant, error)
}

func NewHandler(service Service, sessionFinder SessionFinder, restFinder RestaurantFinder) *Handler {
	return &Handler{
		service:       service,
		sessionFinder: sessionFinder,
		restFinder:    restFinder,
	}
}

// --- Category Handlers ---

func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	result, err := h.service.CreateCategory(restaurantID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Created(c, result)
}

func (h *Handler) ListCategories(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	includeHidden := c.Query("include_hidden") == "true"

	categories, err := h.service.ListCategories(restaurantID, includeHidden)
	if err != nil {
		handleError(c, err)
		return
	}

	if categories == nil {
		categories = []CategoryResponse{}
	}

	resp.Success(c, 200, categories)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	categoryID := uuid.MustParse(c.Param("cid"))

	result, err := h.service.UpdateCategory(restaurantID, categoryID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	categoryID := uuid.MustParse(c.Param("cid"))
	force := c.Query("force") == "true"

	if err := h.service.DeleteCategory(restaurantID, categoryID, force); err != nil {
		handleError(c, err)
		return
	}

	resp.NoContent(c)
}

func (h *Handler) ReorderCategories(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))

	result, err := h.service.ReorderCategories(restaurantID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

// --- Item Handlers ---

func (h *Handler) CreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	categoryID := uuid.MustParse(c.Param("cid"))

	result, err := h.service.CreateItem(restaurantID, categoryID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	if req.OptionsConfig != nil {
		audit.Record(c, "menu_item.modifiers_update", "menu_item", result.ID, map[string]interface{}{
			"created": true,
		})
	}
	resp.Created(c, result)
}

func (h *Handler) ListAllItems(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	params := pagination.ExtractPagination(c)

	var categoryID *uuid.UUID
	if c.Query("category_id") != "" {
		id := uuid.MustParse(c.Query("category_id"))
		categoryID = &id
	}

	var isAvailable *bool
	if c.Query("is_available") != "" {
		val := c.Query("is_available") == "true"
		isAvailable = &val
	}

	search := c.Query("search")

	items, total, err := h.service.ListAllItems(restaurantID, categoryID, isAvailable, search, params.Page, params.PerPage, params.Offset)
	if err != nil {
		handleError(c, err)
		return
	}

	if items == nil {
		items = []ItemResponse{}
	}

	resp.SuccessWithMeta(c, 200, items, &resp.Meta{
		Page:    params.Page,
		PerPage: params.PerPage,
		Total:   total,
	})
}

func (h *Handler) ListItemsByCategory(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	categoryID := uuid.MustParse(c.Param("cid"))

	items, err := h.service.ListItemsByCategory(restaurantID, categoryID)
	if err != nil {
		handleError(c, err)
		return
	}

	if items == nil {
		items = []ItemResponse{}
	}

	resp.Success(c, 200, items)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	itemID := uuid.MustParse(c.Param("itemId"))

	result, err := h.service.UpdateItem(restaurantID, itemID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	action := "menu_item.update"
	if req.OptionsConfig != nil {
		action = "menu_item.modifiers_update"
	}
	audit.Record(c, action, "menu_item", itemID, map[string]interface{}{
		"has_options_config": req.OptionsConfig != nil,
	})
	resp.Success(c, 200, result)
}

func (h *Handler) DeleteItem(c *gin.Context) {
	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	itemID := uuid.MustParse(c.Param("itemId"))

	if err := h.service.DeleteItem(restaurantID, itemID); err != nil {
		handleError(c, err)
		return
	}

	resp.NoContent(c)
}

func (h *Handler) ReorderItems(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ValidationError(c, "Invalid request body", err.Error())
		return
	}

	restaurantID := uuid.MustParse(c.Param("restaurantId"))
	categoryID := uuid.MustParse(c.Param("cid"))

	result, err := h.service.ReorderItems(restaurantID, categoryID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, result)
}

// --- Public Handler ---

// GetPublicMenu returns the full menu for a session's restaurant.
// The sessionToken in the URL is resolved to find the restaurant ID.
func (h *Handler) GetPublicMenu(c *gin.Context) {
	sessionTokenStr := c.Param("sessionToken")
	sessionToken, err := uuid.Parse(sessionTokenStr)
	if err != nil {
		resp.NotFound(c, "Invalid session token")
		return
	}

	// Resolve session to restaurant
	session, err := h.sessionFinder.FindBySessionToken(sessionToken)
	if err != nil {
		resp.NotFound(c, "Session not found")
		return
	}

	if session.Status != "ACTIVE" {
		resp.Forbidden(c, "Session is closed")
		return
	}
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		resp.Forbidden(c, "Session has expired")
		return
	}
	c.Set("session_id", sessionToken.String())
	_ = h.sessionFinder.TouchLastSeen(session.ID)

	restaurant, err := h.restFinder.FindByID(session.RestaurantID)
	if err != nil {
		resp.NotFound(c, "Restaurant not found")
		return
	}

	menu, err := h.service.GetPublicMenu(session.RestaurantID, restaurant.Name)
	if err != nil {
		handleError(c, err)
		return
	}

	resp.Success(c, 200, menu)
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

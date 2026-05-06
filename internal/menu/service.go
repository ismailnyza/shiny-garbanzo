package menu

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"github.com/ismael/qr-restaurant/internal/shared/modifiers"
	"gorm.io/gorm"
)

type Service interface {
	// Categories
	CreateCategory(restaurantID uuid.UUID, req *CreateCategoryRequest) (*CategoryResponse, error)
	ListCategories(restaurantID uuid.UUID, includeHidden bool) ([]CategoryResponse, error)
	UpdateCategory(restaurantID uuid.UUID, categoryID uuid.UUID, req *UpdateCategoryRequest) (*CategoryResponse, error)
	DeleteCategory(restaurantID uuid.UUID, categoryID uuid.UUID, force bool) error
	ReorderCategories(restaurantID uuid.UUID, req *ReorderRequest) (*ReorderResponse, error)

	// Items
	CreateItem(restaurantID uuid.UUID, categoryID uuid.UUID, req *CreateItemRequest) (*ItemResponse, error)
	ListAllItems(restaurantID uuid.UUID, categoryID *uuid.UUID, isAvailable *bool, search string, page, perPage, offset int) ([]ItemResponse, int64, error)
	ListItemsByCategory(restaurantID uuid.UUID, categoryID uuid.UUID) ([]ItemResponse, error)
	UpdateItem(restaurantID uuid.UUID, itemID uuid.UUID, req *UpdateItemRequest) (*ItemResponse, error)
	DeleteItem(restaurantID uuid.UUID, itemID uuid.UUID) error
	ReorderItems(restaurantID uuid.UUID, categoryID uuid.UUID, req *ReorderRequest) (*ReorderResponse, error)

	// Public
	GetPublicMenu(restaurantID uuid.UUID, restaurantName string) (*PublicMenuResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// --- Categories ---

func (s *service) CreateCategory(restaurantID uuid.UUID, req *CreateCategoryRequest) (*CategoryResponse, error) {
	isVisible := true
	if req.IsVisible != nil {
		isVisible = *req.IsVisible
	}

	cat := &models.MenuCategory{
		RestaurantID: restaurantID,
		Name:         req.Name,
		SortOrder:    req.SortOrder,
		IsVisible:    isVisible,
	}

	if err := s.repo.CreateCategory(cat); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return categoryToResponse(cat, 0), nil
}

func (s *service) ListCategories(restaurantID uuid.UUID, includeHidden bool) ([]CategoryResponse, error) {
	cats, _, err := s.repo.ListCategories(restaurantID, includeHidden)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	counts, err := s.repo.GetCategoryItemCounts(restaurantID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	responses := make([]CategoryResponse, len(cats))
	for i, cat := range cats {
		count := counts[cat.ID]
		responses[i] = *categoryToResponse(&cat, count)
	}
	return responses, nil
}

func (s *service) UpdateCategory(restaurantID uuid.UUID, categoryID uuid.UUID, req *UpdateCategoryRequest) (*CategoryResponse, error) {
	cat, err := s.repo.FindCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Category not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	if cat.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Category not found")
	}

	if req.Name != nil {
		cat.Name = *req.Name
	}
	if req.SortOrder != nil {
		cat.SortOrder = *req.SortOrder
	}
	if req.IsVisible != nil {
		cat.IsVisible = *req.IsVisible
	}

	if err := s.repo.UpdateCategory(cat); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	count, _ := s.repo.CountItemsInCategory(cat.ID)
	return categoryToResponse(cat, count), nil
}

func (s *service) DeleteCategory(restaurantID uuid.UUID, categoryID uuid.UUID, force bool) error {
	cat, err := s.repo.FindCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Category not found")
		}
		return apperrors.NewInternal(err)
	}

	if cat.RestaurantID != restaurantID {
		return apperrors.NewNotFound("Category not found")
	}

	if !force {
		count, err := s.repo.CountItemsInCategory(categoryID)
		if err != nil {
			return apperrors.NewInternal(err)
		}
		if count > 0 {
			return apperrors.NewConflict("Category has active items. Use force=true to delete anyway")
		}
	}

	if err := s.repo.SoftDeleteCategory(categoryID); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

func (s *service) ReorderCategories(restaurantID uuid.UUID, req *ReorderRequest) (*ReorderResponse, error) {
	// Validate all IDs belong to this restaurant
	for _, item := range req.Order {
		cat, err := s.repo.FindCategoryByID(item.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.NewNotFound("Category not found: " + item.ID.String())
			}
			return nil, apperrors.NewInternal(err)
		}
		if cat.RestaurantID != restaurantID {
			return nil, apperrors.NewNotFound("Category not found: " + item.ID.String())
		}
	}

	if err := s.repo.BatchUpdateSortOrder(req.Order); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return &ReorderResponse{Updated: len(req.Order)}, nil
}

// --- Items ---

func (s *service) CreateItem(restaurantID uuid.UUID, categoryID uuid.UUID, req *CreateItemRequest) (*ItemResponse, error) {
	// Verify category exists and belongs to restaurant
	cat, err := s.repo.FindCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Category not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if cat.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Category not found")
	}

	isAvailable := true
	if req.IsAvailable != nil {
		isAvailable = *req.IsAvailable
	}
	optionsConfig, err := modifiers.DecodeConfigMap(req.OptionsConfig)
	if err != nil {
		return nil, err
	}

	item := &models.MenuItem{
		CategoryID:    categoryID,
		RestaurantID:  restaurantID,
		Name:          req.Name,
		Description:   req.Description,
		ImageURL:      req.ImageURL,
		PriceCents:    req.PriceCents,
		OptionsConfig: optionsConfig,
		IsAvailable:   isAvailable,
		SortOrder:     req.SortOrder,
	}

	if err := s.repo.CreateItem(item); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return itemToResponse(item, cat.Name), nil
}

func (s *service) ListAllItems(restaurantID uuid.UUID, categoryID *uuid.UUID, isAvailable *bool, search string, page, perPage, offset int) ([]ItemResponse, int64, error) {
	items, total, err := s.repo.ListItemsByRestaurant(restaurantID, categoryID, isAvailable, search, page, perPage, offset)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err)
	}

	responses := make([]ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *itemToResponse(&item, "")
	}
	return responses, total, nil
}

func (s *service) ListItemsByCategory(restaurantID uuid.UUID, categoryID uuid.UUID) ([]ItemResponse, error) {
	cat, err := s.repo.FindCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Category not found")
		}
		return nil, apperrors.NewInternal(err)
	}
	if cat.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Category not found")
	}

	items, err := s.repo.ListItemsByCategory(categoryID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	responses := make([]ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *itemToResponse(&item, cat.Name)
	}
	return responses, nil
}

func (s *service) UpdateItem(restaurantID uuid.UUID, itemID uuid.UUID, req *UpdateItemRequest) (*ItemResponse, error) {
	item, err := s.repo.FindItemByID(itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("Item not found")
		}
		return nil, apperrors.NewInternal(err)
	}

	if item.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Item not found")
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = req.Description
	}
	if req.ImageURL != nil {
		item.ImageURL = req.ImageURL
	}
	if req.PriceCents != nil {
		item.PriceCents = *req.PriceCents
	}
	if req.OptionsConfig != nil {
		optionsConfig, err := modifiers.DecodeConfigMap(req.OptionsConfig)
		if err != nil {
			return nil, err
		}
		item.OptionsConfig = optionsConfig
	}
	if req.IsAvailable != nil {
		item.IsAvailable = *req.IsAvailable
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}

	if err := s.repo.UpdateItem(item); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return itemToResponse(item, ""), nil
}

func (s *service) DeleteItem(restaurantID uuid.UUID, itemID uuid.UUID) error {
	item, err := s.repo.FindItemByID(itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.NewNotFound("Item not found")
		}
		return apperrors.NewInternal(err)
	}

	if item.RestaurantID != restaurantID {
		return apperrors.NewNotFound("Item not found")
	}

	if err := s.repo.SoftDeleteItem(itemID); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

func (s *service) ReorderItems(restaurantID uuid.UUID, categoryID uuid.UUID, req *ReorderRequest) (*ReorderResponse, error) {
	cat, err := s.repo.FindCategoryByID(categoryID)
	if err != nil {
		return nil, apperrors.NewNotFound("Category not found")
	}
	if cat.RestaurantID != restaurantID {
		return nil, apperrors.NewNotFound("Category not found")
	}

	// Validate all items belong to this category
	for _, item := range req.Order {
		menuItem, err := s.repo.FindItemByID(item.ID)
		if err != nil {
			return nil, apperrors.NewNotFound("Item not found")
		}
		if menuItem.CategoryID != categoryID {
			return nil, apperrors.NewNotFound("Item not found: " + item.ID.String())
		}
	}

	// Batch update with transaction
	if err := s.repo.BatchUpdateItemSortOrder(req.Order); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	return &ReorderResponse{Updated: len(req.Order)}, nil
}

// --- Public ---

func (s *service) GetPublicMenu(restaurantID uuid.UUID, restaurantName string) (*PublicMenuResponse, error) {
	categories, err := s.repo.ListVisibleCategoriesWithItems(restaurantID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	response := &PublicMenuResponse{
		RestaurantName: restaurantName,
		Categories:     make([]PublicCategoryGroup, 0),
	}

	for _, cat := range categories {
		group := PublicCategoryGroup{
			ID:        cat.ID,
			Name:      cat.Name,
			SortOrder: cat.SortOrder,
			Items:     make([]PublicMenuItem, 0),
		}

		for _, item := range cat.Items {
			group.Items = append(group.Items, PublicMenuItem{
				ID:            item.ID,
				Name:          item.Name,
				Description:   item.Description,
				ImageURL:      item.ImageURL,
				PriceCents:    item.PriceCents,
				OptionsConfig: jsonToMap(item.OptionsConfig),
				IsAvailable:   item.IsAvailable,
				SortOrder:     item.SortOrder,
			})
		}

		response.Categories = append(response.Categories, group)
	}

	return response, nil
}

// --- Helpers ---

func categoryToResponse(cat *models.MenuCategory, itemCount int64) *CategoryResponse {
	return &CategoryResponse{
		ID:           cat.ID,
		RestaurantID: cat.RestaurantID,
		Name:         cat.Name,
		SortOrder:    cat.SortOrder,
		IsVisible:    cat.IsVisible,
		ItemCount:    int(itemCount),
		CreatedAt:    cat.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    cat.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func itemToResponse(item *models.MenuItem, categoryName string) *ItemResponse {
	return &ItemResponse{
		ID:            item.ID,
		CategoryID:    item.CategoryID,
		RestaurantID:  item.RestaurantID,
		Name:          item.Name,
		Description:   item.Description,
		ImageURL:      item.ImageURL,
		PriceCents:    item.PriceCents,
		OptionsConfig: jsonToMap(item.OptionsConfig),
		IsAvailable:   item.IsAvailable,
		SortOrder:     item.SortOrder,
		CategoryName:  categoryName,
		CreatedAt:     item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func jsonToMap(value []byte) map[string]interface{} {
	if len(value) == 0 {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(value, &out); err != nil {
		return nil
	}
	return out
}

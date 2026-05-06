package menu

import (
	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	// Categories
	CreateCategory(cat *models.MenuCategory) error
	FindCategoryByID(id uuid.UUID) (*models.MenuCategory, error)
	ListCategories(restaurantID uuid.UUID, includeHidden bool) ([]models.MenuCategory, int64, error)
	UpdateCategory(cat *models.MenuCategory) error
	SoftDeleteCategory(id uuid.UUID) error
	CountItemsInCategory(categoryID uuid.UUID) (int64, error)
	GetCategoryItemCounts(restaurantID uuid.UUID) (map[uuid.UUID]int64, error)
	BatchUpdateSortOrder(items []ReorderItem) error
	BatchUpdateItemSortOrder(items []ReorderItem) error

	// Items
	CreateItem(item *models.MenuItem) error
	FindItemByID(id uuid.UUID) (*models.MenuItem, error)
	ListItemsByRestaurant(restaurantID uuid.UUID, categoryID *uuid.UUID, isAvailable *bool, search string, page, perPage, offset int) ([]models.MenuItem, int64, error)
	ListItemsByCategory(categoryID uuid.UUID) ([]models.MenuItem, error)
	UpdateItem(item *models.MenuItem) error
	SoftDeleteItem(id uuid.UUID) error

	// Public
	ListVisibleCategoriesWithItems(restaurantID uuid.UUID) ([]models.MenuCategory, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Categories ---

func (r *repository) CreateCategory(cat *models.MenuCategory) error {
	return r.db.Create(cat).Error
}

func (r *repository) FindCategoryByID(id uuid.UUID) (*models.MenuCategory, error) {
	var cat models.MenuCategory
	err := r.db.First(&cat, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) ListCategories(restaurantID uuid.UUID, includeHidden bool) ([]models.MenuCategory, int64, error) {
	var cats []models.MenuCategory
	var total int64

	query := r.db.Model(&models.MenuCategory{}).Where("restaurant_id = ?", restaurantID)
	if !includeHidden {
		query = query.Where("is_visible = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("sort_order ASC, name ASC").Find(&cats).Error; err != nil {
		return nil, 0, err
	}

	return cats, total, nil
}

func (r *repository) UpdateCategory(cat *models.MenuCategory) error {
	return r.db.Save(cat).Error
}

func (r *repository) SoftDeleteCategory(id uuid.UUID) error {
	return r.db.Delete(&models.MenuCategory{}, "id = ?", id).Error
}

func (r *repository) CountItemsInCategory(categoryID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.MenuItem{}).Where("category_id = ?", categoryID).Count(&count).Error
	return count, err
}

func (r *repository) GetCategoryItemCounts(restaurantID uuid.UUID) (map[uuid.UUID]int64, error) {
	type result struct {
		CategoryID uuid.UUID
		Count      int64
	}

	var rows []result
	err := r.db.Model(&models.MenuItem{}).
		Select("category_id, COUNT(*) as count").
		Where("restaurant_id = ?", restaurantID).
		Group("category_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int64)
	for _, row := range rows {
		counts[row.CategoryID] = row.Count
	}
	return counts, nil
}

func (r *repository) BatchUpdateSortOrder(items []ReorderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&models.MenuCategory{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) BatchUpdateItemSortOrder(items []ReorderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&models.MenuItem{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --- Items ---

func (r *repository) CreateItem(item *models.MenuItem) error {
	return r.db.Create(item).Error
}

func (r *repository) FindItemByID(id uuid.UUID) (*models.MenuItem, error) {
	var item models.MenuItem
	err := r.db.First(&item, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) ListItemsByRestaurant(restaurantID uuid.UUID, categoryID *uuid.UUID, isAvailable *bool, search string, page, perPage, offset int) ([]models.MenuItem, int64, error) {
	var items []models.MenuItem
	var total int64

	query := r.db.Model(&models.MenuItem{}).Where("restaurant_id = ?", restaurantID)
	if categoryID != nil {
		query = query.Where("category_id = ?", *categoryID)
	}
	if isAvailable != nil {
		query = query.Where("is_available = ?", *isAvailable)
	}
	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(perPage).Order("sort_order ASC, name ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *repository) ListItemsByCategory(categoryID uuid.UUID) ([]models.MenuItem, error) {
	var items []models.MenuItem
	err := r.db.Where("category_id = ?", categoryID).Order("sort_order ASC, name ASC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) UpdateItem(item *models.MenuItem) error {
	return r.db.Save(item).Error
}

func (r *repository) SoftDeleteItem(id uuid.UUID) error {
	return r.db.Delete(&models.MenuItem{}, "id = ?", id).Error
}

// --- Public ---

func (r *repository) ListVisibleCategoriesWithItems(restaurantID uuid.UUID) ([]models.MenuCategory, error) {
	var categories []models.MenuCategory
	err := r.db.
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_available = ?", true).Order("sort_order ASC")
		}).
		Where("restaurant_id = ? AND is_visible = ?", restaurantID, true).
		Order("sort_order ASC").
		Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

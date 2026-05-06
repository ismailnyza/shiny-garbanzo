package table

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
)

type fakeRestaurantFinder struct {
	restaurant *models.Restaurant
}

func (f *fakeRestaurantFinder) FindByID(id uuid.UUID) (*models.Restaurant, error) {
	return f.restaurant, nil
}

func TestBuildQRURLIncludesSlugAndEscapesParts(t *testing.T) {
	restaurantID := uuid.New()
	svc := NewService(nil, &fakeRestaurantFinder{restaurant: &models.Restaurant{
		ID:   restaurantID,
		Slug: "cafe-test",
	}}).(*service)

	got := svc.buildQRURLForTable("https://app.example.com/", &models.Table{
		RestaurantID: restaurantID,
		TableCode:    "Table 1",
		QRToken:      "abc+123",
	})

	want := "https://app.example.com/r/cafe-test/t/Table%201?token=abc%2B123"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

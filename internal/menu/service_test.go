package menu

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/ismael/qr-restaurant/internal/shared/models"
)

func TestItemResponseIncludesOptionsConfig(t *testing.T) {
	item := &models.MenuItem{
		ID:            uuid.New(),
		CategoryID:    uuid.New(),
		RestaurantID:  uuid.New(),
		Name:          "Rice",
		PriceCents:    1200,
		OptionsConfig: mustJSON(t, map[string]interface{}{"sizes": []interface{}{"small", "large"}}),
		IsAvailable:   true,
	}

	got := itemToResponse(item, "Mains")
	if got.OptionsConfig == nil {
		t.Fatal("expected options_config to be present")
	}
	if _, ok := got.OptionsConfig["sizes"]; !ok {
		t.Fatalf("expected sizes option, got %#v", got.OptionsConfig)
	}
}

func mustJSON(t *testing.T, value map[string]interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

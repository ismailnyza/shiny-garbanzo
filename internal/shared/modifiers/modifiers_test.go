package modifiers

import (
	"encoding/json"
	"errors"
	"testing"

	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"gorm.io/datatypes"
)

func TestBuildSnapshotExactMinMaxAndPricing(t *testing.T) {
	raw := testConfigJSON(t)
	selected := map[string]interface{}{
		"modifier_selections": []interface{}{
			map[string]interface{}{
				"group_id": "size",
				"options": []interface{}{
					map[string]interface{}{"option_id": "large", "quantity": 1},
				},
			},
			map[string]interface{}{
				"group_id": "toppings",
				"options": []interface{}{
					map[string]interface{}{"option_id": "cheese", "quantity": 2},
					map[string]interface{}{"option_id": "bacon", "quantity": 1},
				},
			},
		},
	}

	snapshot, delta, err := BuildSnapshot(raw, selected, 2)
	if err != nil {
		t.Fatalf("expected valid modifiers, got %v", err)
	}
	if len(snapshot) == 0 {
		t.Fatal("expected snapshot")
	}
	if delta != 1800 {
		t.Fatalf("expected delta 1800, got %d", delta)
	}
}

func TestBuildSnapshotRejectsMissingRequiredGroup(t *testing.T) {
	_, _, err := BuildSnapshot(testConfigJSON(t), nil, 1)
	assertValidation(t, err)
}

func TestBuildSnapshotRejectsOverMaximum(t *testing.T) {
	selected := selection("toppings", "cheese", 2, "bacon", 2)
	_, _, err := BuildSnapshot(testConfigJSON(t), selected, 1)
	assertValidation(t, err)
}

func TestBuildSnapshotRejectsDuplicateOption(t *testing.T) {
	selected := map[string]interface{}{
		"modifier_selections": []interface{}{
			map[string]interface{}{
				"group_id": "size",
				"options": []interface{}{
					map[string]interface{}{"option_id": "large", "quantity": 1},
					map[string]interface{}{"option_id": "large", "quantity": 1},
				},
			},
		},
	}
	_, _, err := BuildSnapshot(testConfigJSON(t), selected, 1)
	assertValidation(t, err)
}

func TestBuildSnapshotRejectsInvalidOwnershipAndInactiveOption(t *testing.T) {
	selected := selection("size", "not-from-this-item", 1)
	_, _, err := BuildSnapshot(testConfigJSON(t), selected, 1)
	assertValidation(t, err)

	selected = selection("size", "inactive-size", 1)
	_, _, err = BuildSnapshot(testConfigJSON(t), selected, 1)
	assertValidation(t, err)
}

func TestBuildSnapshotRejectsQuantityExceedingLimit(t *testing.T) {
	selected := selection("toppings", "cheese", 3)
	_, _, err := BuildSnapshot(testConfigJSON(t), selected, 1)
	assertValidation(t, err)
}

func testConfigJSON(t *testing.T) datatypes.JSON {
	t.Helper()
	active := true
	inactive := false
	cfg := &Config{ModifierGroups: []GroupConfig{
		{
			ID:                "size",
			Name:              "Choose 1 size",
			Required:          true,
			MinSelected:       1,
			MaxSelected:       1,
			SelectionStrategy: SingleSelect,
			Active:            &active,
			Options: []OptionConfig{
				{ID: "small", Name: "Small", PriceDelta: 0, Active: &active, MaximumQuantityPerOption: 1},
				{ID: "large", Name: "Large", PriceDelta: 200, Active: &active, MaximumQuantityPerOption: 1},
				{ID: "inactive-size", Name: "Inactive", PriceDelta: 0, Active: &inactive, MaximumQuantityPerOption: 1},
			},
		},
		{
			ID:                      "toppings",
			Name:                    "Choose up to 3 toppings",
			MinSelected:             0,
			MaxSelected:             3,
			SelectionStrategy:       MultiSelect,
			AllowMultipleQuantities: true,
			Active:                  &active,
			Options: []OptionConfig{
				{ID: "cheese", Name: "Extra cheese", PriceDelta: 200, Active: &active, MaximumQuantityPerOption: 2},
				{ID: "bacon", Name: "Bacon", PriceDelta: 300, Active: &active, MaximumQuantityPerOption: 1},
			},
		},
	}}
	raw, err := DecodeConfigMap(mustMap(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func selection(groupID string, optionID string, qty int, rest ...interface{}) map[string]interface{} {
	options := []interface{}{map[string]interface{}{"option_id": optionID, "quantity": qty}}
	for i := 0; i+1 < len(rest); i += 2 {
		options = append(options, map[string]interface{}{"option_id": rest[i], "quantity": rest[i+1]})
	}
	return map[string]interface{}{
		"modifier_selections": []interface{}{
			map[string]interface{}{
				"group_id": groupID,
				"options":  options,
			},
		},
	}
}

func mustMap(t *testing.T, cfg *Config) map[string]interface{} {
	t.Helper()
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertValidation(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeValidation {
		t.Fatalf("expected validation AppError, got %T %v", err, err)
	}
}

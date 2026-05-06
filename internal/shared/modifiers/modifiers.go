package modifiers

import (
	"encoding/json"
	"fmt"

	apperrors "github.com/ismael/qr-restaurant/internal/shared/errors"
	"gorm.io/datatypes"
)

const (
	SingleSelect = "SINGLE_SELECT"
	MultiSelect  = "MULTI_SELECT"
)

type Config struct {
	ModifierGroups []GroupConfig `json:"modifier_groups"`
}

type GroupConfig struct {
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	Description             string         `json:"description,omitempty"`
	DisplayOrder            int            `json:"display_order"`
	Required                bool           `json:"required"`
	MinSelected             int            `json:"min_selected"`
	MaxSelected             int            `json:"max_selected"`
	SelectionStrategy       string         `json:"selection_strategy"`
	AllowMultipleQuantities bool           `json:"allow_multiple_quantities"`
	Active                  *bool          `json:"active,omitempty"`
	Options                 []OptionConfig `json:"options"`
}

type OptionConfig struct {
	ID                       string `json:"id"`
	GroupID                  string `json:"group_id,omitempty"`
	Name                     string `json:"name"`
	Description              string `json:"description,omitempty"`
	DisplayOrder             int    `json:"display_order"`
	PriceDelta               int    `json:"price_delta"`
	Active                   *bool  `json:"active,omitempty"`
	MaximumQuantityPerOption int    `json:"maximum_quantity_per_option"`
}

type SelectionSet struct {
	ModifierSelections []GroupSelection `json:"modifier_selections"`
}

type GroupSelection struct {
	GroupID string            `json:"group_id"`
	Options []OptionSelection `json:"options"`
}

type OptionSelection struct {
	OptionID string `json:"option_id"`
	Quantity int    `json:"quantity"`
}

type Snapshot struct {
	ModifierGroups  []GroupSnapshot `json:"modifier_groups"`
	TotalDeltaCents int             `json:"total_delta_cents"`
}

type GroupSnapshot struct {
	GroupID   string           `json:"group_id"`
	GroupName string           `json:"group_name"`
	Options   []OptionSnapshot `json:"options"`
}

type OptionSnapshot struct {
	OptionID        string `json:"option_id"`
	OptionName      string `json:"option_name"`
	Quantity        int    `json:"quantity"`
	PriceDeltaCents int    `json:"price_delta_cents"`
	TotalDeltaCents int    `json:"total_delta_cents"`
}

func DecodeConfig(raw datatypes.JSON) (*Config, error) {
	if len(raw) == 0 {
		return &Config{}, nil
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, apperrors.NewValidation("Invalid options_config", err.Error())
	}
	return &cfg, nil
}

func DecodeConfigMap(value map[string]interface{}) (datatypes.JSON, error) {
	if value == nil {
		return nil, nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.NewValidation("Invalid options_config", err.Error())
	}
	cfg, err := DecodeConfig(datatypes.JSON(b))
	if err != nil {
		return nil, err
	}
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func ValidateConfig(cfg *Config) error {
	groupIDs := map[string]bool{}
	for _, group := range cfg.ModifierGroups {
		if group.ID == "" {
			return apperrors.NewValidation("Modifier group id is required", nil)
		}
		if group.Name == "" {
			return apperrors.NewValidation("Modifier group name is required", group.ID)
		}
		if groupIDs[group.ID] {
			return apperrors.NewValidation("Duplicate modifier group id", group.ID)
		}
		groupIDs[group.ID] = true
		if group.SelectionStrategy != SingleSelect && group.SelectionStrategy != MultiSelect {
			return apperrors.NewValidation("Invalid modifier selection_strategy", group.ID)
		}
		if group.MinSelected < 0 || group.MaxSelected < 0 || group.MinSelected > group.MaxSelected {
			return apperrors.NewValidation("Invalid modifier min/max selection", group.ID)
		}
		if group.SelectionStrategy == SingleSelect && group.MaxSelected > 1 {
			return apperrors.NewValidation("SINGLE_SELECT modifier groups cannot allow more than one selection", group.ID)
		}

		optionIDs := map[string]bool{}
		for _, option := range group.Options {
			if option.ID == "" {
				return apperrors.NewValidation("Modifier option id is required", group.ID)
			}
			if option.Name == "" {
				return apperrors.NewValidation("Modifier option name is required", option.ID)
			}
			if option.GroupID != "" && option.GroupID != group.ID {
				return apperrors.NewValidation("Modifier option group_id does not match parent group", option.ID)
			}
			if optionIDs[option.ID] {
				return apperrors.NewValidation("Duplicate modifier option id", option.ID)
			}
			optionIDs[option.ID] = true
			if option.MaximumQuantityPerOption < 0 {
				return apperrors.NewValidation("Modifier option maximum_quantity_per_option cannot be negative", option.ID)
			}
		}
	}
	return nil
}

func BuildSnapshot(rawConfig datatypes.JSON, selected map[string]interface{}, itemQuantity int) (datatypes.JSON, int, error) {
	cfg, err := DecodeConfig(rawConfig)
	if err != nil {
		return nil, 0, err
	}
	if err := ValidateConfig(cfg); err != nil {
		return nil, 0, err
	}

	selection, err := decodeSelection(selected)
	if err != nil {
		return nil, 0, err
	}
	groupSelections := map[string]GroupSelection{}
	for _, group := range selection.ModifierSelections {
		if group.GroupID == "" {
			return nil, 0, apperrors.NewValidation("modifier group_id is required", nil)
		}
		if _, exists := groupSelections[group.GroupID]; exists {
			return nil, 0, apperrors.NewValidation("Duplicate modifier group submission", group.GroupID)
		}
		groupSelections[group.GroupID] = group
	}

	snapshot := Snapshot{ModifierGroups: []GroupSnapshot{}}
	for _, group := range cfg.ModifierGroups {
		if !isActive(group.Active) {
			continue
		}
		groupSelection, selected := groupSelections[group.ID]
		if !selected {
			if group.Required || group.MinSelected > 0 {
				return nil, 0, apperrors.NewValidation("Required modifier group missing", group.ID)
			}
			continue
		}
		delete(groupSelections, group.ID)

		optionByID := map[string]OptionConfig{}
		for _, option := range group.Options {
			if isActive(option.Active) {
				optionByID[option.ID] = option
			}
		}

		seenOptions := map[string]bool{}
		selectedUnits := 0
		groupSnapshot := GroupSnapshot{GroupID: group.ID, GroupName: group.Name, Options: []OptionSnapshot{}}
		for _, selectedOption := range groupSelection.Options {
			if selectedOption.OptionID == "" {
				return nil, 0, apperrors.NewValidation("modifier option_id is required", group.ID)
			}
			if seenOptions[selectedOption.OptionID] {
				return nil, 0, apperrors.NewValidation("Duplicate modifier option submission", selectedOption.OptionID)
			}
			seenOptions[selectedOption.OptionID] = true

			option, ok := optionByID[selectedOption.OptionID]
			if !ok {
				return nil, 0, apperrors.NewValidation("Invalid or inactive modifier option", selectedOption.OptionID)
			}

			quantity := selectedOption.Quantity
			if quantity == 0 {
				quantity = 1
			}
			if quantity < 1 {
				return nil, 0, apperrors.NewValidation("Modifier option quantity must be at least 1", selectedOption.OptionID)
			}
			if !group.AllowMultipleQuantities && quantity != 1 {
				return nil, 0, apperrors.NewValidation("Modifier group does not allow multiple quantities", group.ID)
			}
			if option.MaximumQuantityPerOption > 0 && quantity > option.MaximumQuantityPerOption {
				return nil, 0, apperrors.NewValidation("Modifier option quantity exceeds maximum", selectedOption.OptionID)
			}

			if group.AllowMultipleQuantities {
				selectedUnits += quantity
			} else {
				selectedUnits++
			}
			totalDelta := option.PriceDelta * quantity * itemQuantity
			snapshot.TotalDeltaCents += totalDelta
			groupSnapshot.Options = append(groupSnapshot.Options, OptionSnapshot{
				OptionID:        option.ID,
				OptionName:      option.Name,
				Quantity:        quantity,
				PriceDeltaCents: option.PriceDelta,
				TotalDeltaCents: totalDelta,
			})
		}

		if selectedUnits < group.MinSelected {
			return nil, 0, apperrors.NewValidation("Modifier selections below minimum", fmt.Sprintf("%s requires at least %d", group.ID, group.MinSelected))
		}
		if selectedUnits > group.MaxSelected {
			return nil, 0, apperrors.NewValidation("Modifier selections exceed maximum", fmt.Sprintf("%s allows at most %d", group.ID, group.MaxSelected))
		}
		snapshot.ModifierGroups = append(snapshot.ModifierGroups, groupSnapshot)
	}

	for groupID := range groupSelections {
		return nil, 0, apperrors.NewValidation("Modifier group does not belong to this item", groupID)
	}

	if len(snapshot.ModifierGroups) == 0 {
		return nil, 0, nil
	}
	b, _ := json.Marshal(snapshot)
	return datatypes.JSON(b), snapshot.TotalDeltaCents, nil
}

func decodeSelection(selected map[string]interface{}) (*SelectionSet, error) {
	if selected == nil {
		return &SelectionSet{}, nil
	}
	b, err := json.Marshal(selected)
	if err != nil {
		return nil, apperrors.NewValidation("Invalid selected_options", err.Error())
	}
	var selection SelectionSet
	if err := json.Unmarshal(b, &selection); err != nil {
		return nil, apperrors.NewValidation("Invalid selected_options", err.Error())
	}
	return &selection, nil
}

func isActive(value *bool) bool {
	return value == nil || *value
}

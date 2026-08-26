package forms

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type FieldType string

const (
	FieldTypeShortText      FieldType = "SHORT_TEXT"
	FieldTypeLongText       FieldType = "LONG_TEXT"
	FieldTypeNumber         FieldType = "NUMBER"
	FieldTypeSingleChoice   FieldType = "SINGLE_CHOICE"
	FieldTypeMultipleChoice FieldType = "MULTIPLE_CHOICE"
	FieldTypeRating         FieldType = "RATING"
)

const (
	maxShortTextLength = 255
	maxLongTextLength  = 10_000
	maxRatingValue     = 10
)

var ErrInvalidField = errors.New("invalid form field")

type FieldDefinition struct {
	Type          FieldType       `json:"type"`
	Label         string          `json:"label"`
	Description   string          `json:"description,omitempty"`
	Required      bool            `json:"required"`
	Position      int             `json:"position"`
	Configuration json.RawMessage `json:"configuration"`
}

type TextConfiguration struct {
	MinLength *int `json:"min_length,omitempty"`
	MaxLength *int `json:"max_length,omitempty"`
}

type NumberConfiguration struct {
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
	Step *float64 `json:"step,omitempty"`
}

type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type SingleChoiceConfiguration struct {
	Choices []Choice `json:"choices"`
}

type MultipleChoiceConfiguration struct {
	Choices       []Choice `json:"choices"`
	MinSelections *int     `json:"min_selections,omitempty"`
	MaxSelections *int     `json:"max_selections,omitempty"`
}

type RatingConfiguration struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func (t FieldType) IsValid() bool {
	switch t {
	case FieldTypeShortText,
		FieldTypeLongText,
		FieldTypeNumber,
		FieldTypeSingleChoice,
		FieldTypeMultipleChoice,
		FieldTypeRating:
		return true
	default:
		return false
	}
}

func (field FieldDefinition) Validate() error {
	if !field.Type.IsValid() {
		return invalidField("type", "is not supported")
	}

	label := strings.TrimSpace(field.Label)
	if label == "" {
		return invalidField("label", "is required")
	}
	if len([]rune(label)) > 255 {
		return invalidField("label", "must contain at most 255 characters")
	}
	if field.Position < 0 {
		return invalidField("position", "must be zero or greater")
	}

	switch field.Type {
	case FieldTypeShortText:
		return validateTextConfiguration(field.Configuration, maxShortTextLength)
	case FieldTypeLongText:
		return validateTextConfiguration(field.Configuration, maxLongTextLength)
	case FieldTypeNumber:
		return validateNumberConfiguration(field.Configuration)
	case FieldTypeSingleChoice:
		var configuration SingleChoiceConfiguration
		if err := decodeConfiguration(field.Configuration, &configuration); err != nil {
			return err
		}
		return validateChoices(configuration.Choices)
	case FieldTypeMultipleChoice:
		return validateMultipleChoiceConfiguration(field.Configuration)
	case FieldTypeRating:
		return validateRatingConfiguration(field.Configuration)
	default:
		return invalidField("type", "is not supported")
	}
}

func validateTextConfiguration(raw json.RawMessage, maximumAllowed int) error {
	var configuration TextConfiguration
	if err := decodeConfiguration(raw, &configuration); err != nil {
		return err
	}

	if configuration.MinLength != nil && *configuration.MinLength < 0 {
		return invalidField("configuration.min_length", "must be zero or greater")
	}
	if configuration.MaxLength != nil {
		if *configuration.MaxLength < 1 {
			return invalidField("configuration.max_length", "must be greater than zero")
		}
		if *configuration.MaxLength > maximumAllowed {
			return invalidField(
				"configuration.max_length",
				fmt.Sprintf("must be at most %d", maximumAllowed),
			)
		}
	}
	if configuration.MinLength != nil && configuration.MaxLength != nil &&
		*configuration.MinLength > *configuration.MaxLength {
		return invalidField("configuration.min_length", "must not exceed max_length")
	}

	return nil
}

func validateNumberConfiguration(raw json.RawMessage) error {
	var configuration NumberConfiguration
	if err := decodeConfiguration(raw, &configuration); err != nil {
		return err
	}

	if configuration.Min != nil && configuration.Max != nil &&
		*configuration.Min > *configuration.Max {
		return invalidField("configuration.min", "must not exceed max")
	}
	if configuration.Step != nil && *configuration.Step <= 0 {
		return invalidField("configuration.step", "must be greater than zero")
	}

	return nil
}

func validateMultipleChoiceConfiguration(raw json.RawMessage) error {
	var configuration MultipleChoiceConfiguration
	if err := decodeConfiguration(raw, &configuration); err != nil {
		return err
	}
	if err := validateChoices(configuration.Choices); err != nil {
		return err
	}

	if configuration.MinSelections != nil && *configuration.MinSelections < 0 {
		return invalidField("configuration.min_selections", "must be zero or greater")
	}
	if configuration.MaxSelections != nil {
		if *configuration.MaxSelections < 1 {
			return invalidField("configuration.max_selections", "must be greater than zero")
		}
		if *configuration.MaxSelections > len(configuration.Choices) {
			return invalidField("configuration.max_selections", "must not exceed the number of choices")
		}
	}
	if configuration.MinSelections != nil && configuration.MaxSelections != nil &&
		*configuration.MinSelections > *configuration.MaxSelections {
		return invalidField("configuration.min_selections", "must not exceed max_selections")
	}

	return nil
}

func validateRatingConfiguration(raw json.RawMessage) error {
	var configuration RatingConfiguration
	if err := decodeConfiguration(raw, &configuration); err != nil {
		return err
	}

	if configuration.Min < 1 {
		return invalidField("configuration.min", "must be at least 1")
	}
	if configuration.Max <= configuration.Min {
		return invalidField("configuration.max", "must be greater than min")
	}
	if configuration.Max > maxRatingValue {
		return invalidField("configuration.max", "must be at most 10")
	}

	return nil
}

func validateChoices(choices []Choice) error {
	if len(choices) < 2 {
		return invalidField("configuration.choices", "must contain at least two choices")
	}

	values := make(map[string]struct{}, len(choices))
	for index, choice := range choices {
		value := strings.TrimSpace(choice.Value)
		if value == "" {
			return invalidField(fmt.Sprintf("configuration.choices[%d].value", index), "is required")
		}
		if strings.TrimSpace(choice.Label) == "" {
			return invalidField(fmt.Sprintf("configuration.choices[%d].label", index), "is required")
		}
		if _, exists := values[value]; exists {
			return invalidField("configuration.choices", "must have unique values")
		}
		values[value] = struct{}{}
	}

	return nil
}

func decodeConfiguration(raw json.RawMessage, target any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage(`{}`)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalidField("configuration", err.Error())
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return invalidField("configuration", "must contain one JSON object")
	}

	return nil
}

func invalidField(path, message string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalidField, path, message)
}

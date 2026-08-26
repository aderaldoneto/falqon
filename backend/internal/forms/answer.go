package forms

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

var ErrInvalidAnswer = errors.New("invalid answer")

type Answer struct {
	FieldID int64
	Value   json.RawMessage
}

func ValidateAnswer(field PublicField, value any) (json.RawMessage, error) {
	invalid := func(message string) (json.RawMessage, error) {
		return nil, fmt.Errorf("%w: field %d %s", ErrInvalidAnswer, field.ID, message)
	}

	switch field.Type {
	case FieldTypeShortText, FieldTypeLongText:
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return invalid("must be a non-empty text")
		}
		var configuration TextConfiguration
		if err := json.Unmarshal(field.Configuration, &configuration); err != nil {
			return nil, err
		}
		length := utf8.RuneCountInString(text)
		if configuration.MinLength != nil && length < *configuration.MinLength {
			return invalid("is shorter than allowed")
		}
		if configuration.MaxLength != nil && length > *configuration.MaxLength {
			return invalid("is longer than allowed")
		}
	case FieldTypeNumber:
		number, ok := value.(float64)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return invalid("must be a number")
		}
		var configuration NumberConfiguration
		if err := json.Unmarshal(field.Configuration, &configuration); err != nil {
			return nil, err
		}
		if configuration.Min != nil && number < *configuration.Min || configuration.Max != nil && number > *configuration.Max {
			return invalid("is outside the allowed range")
		}
		if configuration.Step != nil {
			base := 0.0
			if configuration.Min != nil {
				base = *configuration.Min
			}
			steps := (number - base) / *configuration.Step
			if math.Abs(steps-math.Round(steps)) > 1e-9 {
				return invalid("does not match the allowed step")
			}
		}
	case FieldTypeSingleChoice:
		choice, ok := value.(string)
		if !ok || !choiceExists(field.Configuration, choice) {
			return invalid("contains an invalid choice")
		}
	case FieldTypeMultipleChoice:
		values, ok := value.([]interface{})
		if !ok || len(values) == 0 {
			return invalid("must contain choices")
		}
		var configuration MultipleChoiceConfiguration
		if err := json.Unmarshal(field.Configuration, &configuration); err != nil {
			return nil, err
		}
		seen := make(map[string]struct{}, len(values))
		for _, item := range values {
			choice, ok := item.(string)
			if !ok || !containsChoice(configuration.Choices, choice) {
				return invalid("contains an invalid choice")
			}
			if _, exists := seen[choice]; exists {
				return invalid("contains duplicate choices")
			}
			seen[choice] = struct{}{}
		}
		if configuration.MinSelections != nil && len(values) < *configuration.MinSelections || configuration.MaxSelections != nil && len(values) > *configuration.MaxSelections {
			return invalid("has an invalid number of choices")
		}
	case FieldTypeRating:
		rating, ok := value.(float64)
		var configuration RatingConfiguration
		if err := json.Unmarshal(field.Configuration, &configuration); err != nil {
			return nil, err
		}
		if !ok || rating != math.Trunc(rating) || rating < float64(configuration.Min) || rating > float64(configuration.Max) {
			return invalid("must be a valid rating")
		}
	default:
		return invalid("has an unsupported type")
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode answer: %w", err)
	}
	return encoded, nil
}

func choiceExists(raw json.RawMessage, value string) bool {
	var configuration SingleChoiceConfiguration
	return json.Unmarshal(raw, &configuration) == nil && containsChoice(configuration.Choices, value)
}

func containsChoice(choices []Choice, value string) bool {
	for _, choice := range choices {
		if choice.Value == value {
			return true
		}
	}
	return false
}

package forms

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field PublicField
		value any
		valid bool
	}{
		{name: "text", field: PublicField{ID: 1, Type: FieldTypeShortText, Configuration: []byte(`{"max_length":5}`)}, value: "ótimo", valid: true},
		{name: "text too long", field: PublicField{ID: 1, Type: FieldTypeShortText, Configuration: []byte(`{"max_length":3}`)}, value: "ótimo"},
		{name: "number step", field: PublicField{ID: 2, Type: FieldTypeNumber, Configuration: []byte(`{"min":0,"max":5,"step":0.5}`)}, value: float64(2.5), valid: true},
		{name: "invalid number step", field: PublicField{ID: 2, Type: FieldTypeNumber, Configuration: []byte(`{"step":0.5}`)}, value: float64(2.2)},
		{name: "choice", field: PublicField{ID: 3, Type: FieldTypeSingleChoice, Configuration: []byte(`{"choices":[{"value":"yes","label":"Yes"}]}`)}, value: "yes", valid: true},
		{name: "unknown choice", field: PublicField{ID: 3, Type: FieldTypeSingleChoice, Configuration: []byte(`{"choices":[{"value":"yes","label":"Yes"}]}`)}, value: "no"},
		{name: "rating", field: PublicField{ID: 4, Type: FieldTypeRating, Configuration: []byte(`{"min":1,"max":5}`)}, value: float64(5), valid: true},
		{name: "fractional rating", field: PublicField{ID: 4, Type: FieldTypeRating, Configuration: []byte(`{"min":1,"max":5}`)}, value: 4.5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateAnswer(test.field, test.value)
			if test.valid && err != nil {
				t.Fatalf("ValidateAnswer() error = %v", err)
			}
			if !test.valid && !errors.Is(err, ErrInvalidAnswer) {
				t.Fatalf("ValidateAnswer() error = %v, want ErrInvalidAnswer", err)
			}
		})
	}
}

func TestValidateAnswerProducesJSON(t *testing.T) {
	t.Parallel()

	value, err := ValidateAnswer(PublicField{ID: 1, Type: FieldTypeShortText, Configuration: []byte(`{}`)}, "review")
	if err != nil || !json.Valid(value) {
		t.Fatalf("ValidateAnswer() = %s, %v, want valid JSON", value, err)
	}
}

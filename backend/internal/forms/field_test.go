package forms

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestFieldDefinitionValidateAcceptsSupportedTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fieldType     FieldType
		configuration string
	}{
		{name: "short text", fieldType: FieldTypeShortText, configuration: `{"min_length":1,"max_length":100}`},
		{name: "long text", fieldType: FieldTypeLongText, configuration: `{"max_length":2000}`},
		{name: "number", fieldType: FieldTypeNumber, configuration: `{"min":0,"max":10,"step":0.5}`},
		{name: "single choice", fieldType: FieldTypeSingleChoice, configuration: `{"choices":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]}`},
		{name: "multiple choice", fieldType: FieldTypeMultipleChoice, configuration: `{"choices":[{"value":"crime","label":"Crime"},{"value":"drama","label":"Drama"}],"min_selections":1,"max_selections":2}`},
		{name: "rating", fieldType: FieldTypeRating, configuration: `{"min":1,"max":10}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			field := FieldDefinition{
				Type:          test.fieldType,
				Label:         "Question",
				Position:      0,
				Configuration: json.RawMessage(test.configuration),
			}

			if err := field.Validate(); err != nil {
				t.Fatalf("Validate() returned an unexpected error: %v", err)
			}
		})
	}
}

func TestFieldDefinitionValidateRejectsInvalidDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fieldType     FieldType
		label         string
		position      int
		configuration string
	}{
		{name: "unknown type", fieldType: "DATE", label: "Date", configuration: `{}`},
		{name: "empty label", fieldType: FieldTypeShortText, label: " ", configuration: `{}`},
		{name: "negative position", fieldType: FieldTypeShortText, label: "Question", position: -1, configuration: `{}`},
		{name: "short text too long", fieldType: FieldTypeShortText, label: "Question", configuration: `{"max_length":256}`},
		{name: "text min exceeds max", fieldType: FieldTypeLongText, label: "Question", configuration: `{"min_length":20,"max_length":10}`},
		{name: "number min exceeds max", fieldType: FieldTypeNumber, label: "Question", configuration: `{"min":11,"max":10}`},
		{name: "invalid number step", fieldType: FieldTypeNumber, label: "Question", configuration: `{"step":0}`},
		{name: "not enough choices", fieldType: FieldTypeSingleChoice, label: "Question", configuration: `{"choices":[{"value":"yes","label":"Yes"}]}`},
		{name: "duplicate choices", fieldType: FieldTypeSingleChoice, label: "Question", configuration: `{"choices":[{"value":"yes","label":"Yes"},{"value":"yes","label":"Also yes"}]}`},
		{name: "too many selections", fieldType: FieldTypeMultipleChoice, label: "Question", configuration: `{"choices":[{"value":"a","label":"A"},{"value":"b","label":"B"}],"max_selections":3}`},
		{name: "rating exceeds limit", fieldType: FieldTypeRating, label: "Question", configuration: `{"min":1,"max":11}`},
		{name: "unknown configuration property", fieldType: FieldTypeShortText, label: "Question", configuration: `{"unknown":true}`},
		{name: "configuration is not object", fieldType: FieldTypeShortText, label: "Question", configuration: `[]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			field := FieldDefinition{
				Type:          test.fieldType,
				Label:         test.label,
				Position:      test.position,
				Configuration: json.RawMessage(test.configuration),
			}

			err := field.Validate()
			if !errors.Is(err, ErrInvalidField) {
				t.Fatalf("Validate() error = %v, want ErrInvalidField", err)
			}
		})
	}
}

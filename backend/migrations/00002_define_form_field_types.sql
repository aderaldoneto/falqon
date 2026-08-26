-- +goose Up
-- +goose StatementBegin
CREATE TYPE form_field_type AS ENUM (
    'SHORT_TEXT',
    'LONG_TEXT',
    'NUMBER',
    'SINGLE_CHOICE',
    'MULTIPLE_CHOICE',
    'RATING'
);

ALTER TABLE form_fields
    ALTER COLUMN field_type TYPE form_field_type
    USING field_type::form_field_type;

ALTER TABLE form_fields
    ADD CONSTRAINT form_fields_configuration_object_check
    CHECK (jsonb_typeof(configuration) = 'object');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE form_fields
    DROP CONSTRAINT IF EXISTS form_fields_configuration_object_check;

ALTER TABLE form_fields
    ALTER COLUMN field_type TYPE VARCHAR(50)
    USING field_type::TEXT;

DROP TYPE IF EXISTS form_field_type;
-- +goose StatementEnd

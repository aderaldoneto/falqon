-- +goose Up
-- +goose StatementBegin
CREATE TYPE form_state AS ENUM ('DRAFT', 'PUBLISHED', 'CANCELED');

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(160) NOT NULL,
    email VARCHAR(320) NOT NULL UNIQUE,
    password_hash TEXT,
    google_subject VARCHAR(255) UNIQUE,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT users_authentication_method_check CHECK (
        password_hash IS NOT NULL OR google_subject IS NOT NULL
    )
);

CREATE TABLE forms (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users (id),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    state form_state NOT NULL DEFAULT 'DRAFT',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE form_fields (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    form_id BIGINT NOT NULL REFERENCES forms (id) ON DELETE CASCADE,
    field_type VARCHAR(50) NOT NULL,
    label VARCHAR(255) NOT NULL,
    description TEXT,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    position INTEGER NOT NULL,
    configuration JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT form_fields_position_check CHECK (position >= 0),
    CONSTRAINT form_fields_form_position_unique UNIQUE (form_id, position)
);

CREATE TABLE submissions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    form_id BIGINT NOT NULL REFERENCES forms (id) ON DELETE CASCADE,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE submission_answers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    submission_id BIGINT NOT NULL REFERENCES submissions (id) ON DELETE CASCADE,
    field_id BIGINT NOT NULL REFERENCES form_fields (id),
    value JSONB NOT NULL,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT submission_answers_submission_field_unique UNIQUE (
        submission_id,
        field_id
    )
);

CREATE INDEX forms_owner_id_idx ON forms (owner_id);
CREATE INDEX form_fields_form_id_idx ON form_fields (form_id);
CREATE INDEX submissions_form_id_idx ON submissions (form_id);
CREATE INDEX submission_answers_submission_id_idx
    ON submission_answers (submission_id);
CREATE INDEX submission_answers_field_id_idx
    ON submission_answers (field_id);

CREATE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER forms_set_updated_at
BEFORE UPDATE ON forms
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER form_fields_set_updated_at
BEFORE UPDATE ON form_fields
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER submissions_set_updated_at
BEFORE UPDATE ON submissions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER submission_answers_set_updated_at
BEFORE UPDATE ON submission_answers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS submission_answers;
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS form_fields;
DROP TABLE IF EXISTS forms;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS set_updated_at();
DROP TYPE IF EXISTS form_state;
-- +goose StatementEnd

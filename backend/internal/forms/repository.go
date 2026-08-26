package forms

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSlugAlreadyExists = errors.New("form slug already exists")
	ErrFormNotFound      = errors.New("form not found")
	ErrInvalidFormState  = errors.New("invalid form state")
)

type Summary struct {
	ID          int64
	Title       string
	Slug        string
	Description *string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Repository struct {
	database *pgxpool.Pool
}

func NewRepository(database *pgxpool.Pool) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) Create(
	ctx context.Context,
	ownerID int64,
	title string,
	slug string,
	description *string,
	fields []FieldDefinition,
) (Summary, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("begin form transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var form Summary
	err = tx.QueryRow(ctx, `
		INSERT INTO forms (owner_id, title, slug, description, state, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'DRAFT', $1, $1)
		RETURNING id, title, slug, description, state::text, created_at, updated_at
	`, ownerID, title, slug, description).Scan(
		&form.ID, &form.Title, &form.Slug, &form.Description,
		&form.State, &form.CreatedAt, &form.UpdatedAt,
	)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			return Summary{}, ErrSlugAlreadyExists
		}
		return Summary{}, fmt.Errorf("create form: %w", err)
	}

	for _, field := range fields {
		_, err := tx.Exec(ctx, `
			INSERT INTO form_fields (
				form_id, field_type, label, description, required, position,
				configuration, created_by, updated_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		`, form.ID, field.Type, field.Label, nullableString(field.Description), field.Required,
			field.Position, field.Configuration, ownerID)
		if err != nil {
			return Summary{}, fmt.Errorf("create form field: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Summary{}, fmt.Errorf("commit form transaction: %w", err)
	}
	return form, nil
}

func (repository *Repository) ListByOwner(ctx context.Context, ownerID int64) ([]Summary, error) {
	rows, err := repository.database.Query(ctx, `
		SELECT id, title, slug, description, state::text, created_at, updated_at
		FROM forms
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC, id DESC
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list forms by owner: %w", err)
	}
	defer rows.Close()

	forms := make([]Summary, 0)
	for rows.Next() {
		var form Summary
		if err := rows.Scan(
			&form.ID,
			&form.Title,
			&form.Slug,
			&form.Description,
			&form.State,
			&form.CreatedAt,
			&form.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan form summary: %w", err)
		}
		forms = append(forms, form)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate form summaries: %w", err)
	}
	return forms, nil
}

func (repository *Repository) Publish(ctx context.Context, formID, ownerID int64) (Summary, error) {
	var form Summary
	err := repository.database.QueryRow(ctx, `
		UPDATE forms
		SET state = 'PUBLISHED', updated_by = $2, updated_at = NOW()
		WHERE id = $1 AND owner_id = $2 AND state = 'DRAFT' AND deleted_at IS NULL
		RETURNING id, title, slug, description, state::text, created_at, updated_at
	`, formID, ownerID).Scan(
		&form.ID, &form.Title, &form.Slug, &form.Description,
		&form.State, &form.CreatedAt, &form.UpdatedAt,
	)
	if err == nil {
		return form, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, fmt.Errorf("publish form: %w", err)
	}

	var state string
	err = repository.database.QueryRow(ctx, `
		SELECT state::text FROM forms
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`, formID, ownerID).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, ErrFormNotFound
	}
	if err != nil {
		return Summary{}, fmt.Errorf("find form for publication: %w", err)
	}
	return Summary{}, ErrInvalidFormState
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

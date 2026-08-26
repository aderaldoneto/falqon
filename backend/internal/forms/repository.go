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

type PublicForm struct {
	ID          int64
	Title       string
	Slug        string
	Description *string
	Fields      []PublicField
}

type PublicField struct {
	ID            int64
	Type          FieldType
	Label         string
	Description   *string
	Required      bool
	Configuration []byte
}

type Submission struct {
	ID        int64
	CreatedAt time.Time
}

type SubmissionAnswerResult struct {
	FieldID   int64
	FieldType FieldType
	Label     string
	Value     []byte
}

type SubmissionResult struct {
	ID        int64
	CreatedAt time.Time
	Answers   []SubmissionAnswerResult
}

type FormSubmissions struct {
	FormID      int64
	Title       string
	Slug        string
	Submissions []SubmissionResult
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

func (repository *Repository) FindPublishedBySlug(ctx context.Context, slug string) (PublicForm, error) {
	var form PublicForm
	err := repository.database.QueryRow(ctx, `
		SELECT id, title, slug, description
		FROM forms
		WHERE slug = $1 AND state = 'PUBLISHED' AND deleted_at IS NULL
	`, slug).Scan(&form.ID, &form.Title, &form.Slug, &form.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicForm{}, ErrFormNotFound
	}
	if err != nil {
		return PublicForm{}, fmt.Errorf("find published form: %w", err)
	}

	rows, err := repository.database.Query(ctx, `
		SELECT id, field_type, label, description, required, configuration
		FROM form_fields
		WHERE form_id = $1 AND deleted_at IS NULL
		ORDER BY position, id
	`, form.ID)
	if err != nil {
		return PublicForm{}, fmt.Errorf("list public form fields: %w", err)
	}
	defer rows.Close()
	form.Fields = make([]PublicField, 0)
	for rows.Next() {
		var field PublicField
		if err := rows.Scan(&field.ID, &field.Type, &field.Label, &field.Description, &field.Required, &field.Configuration); err != nil {
			return PublicForm{}, fmt.Errorf("scan public form field: %w", err)
		}
		form.Fields = append(form.Fields, field)
	}
	if err := rows.Err(); err != nil {
		return PublicForm{}, fmt.Errorf("iterate public form fields: %w", err)
	}
	return form, nil
}

func (repository *Repository) CreateSubmission(ctx context.Context, formID int64, answers []Answer) (Submission, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return Submission{}, fmt.Errorf("begin submission transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var submission Submission
	err = tx.QueryRow(ctx, `
		INSERT INTO submissions (form_id) VALUES ($1)
		RETURNING id, created_at
	`, formID).Scan(&submission.ID, &submission.CreatedAt)
	if err != nil {
		return Submission{}, fmt.Errorf("create submission: %w", err)
	}
	for _, answer := range answers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO submission_answers (submission_id, field_id, value)
			VALUES ($1, $2, $3)
		`, submission.ID, answer.FieldID, answer.Value); err != nil {
			return Submission{}, fmt.Errorf("create submission answer: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Submission{}, fmt.Errorf("commit submission transaction: %w", err)
	}
	return submission, nil
}

func (repository *Repository) ListSubmissions(ctx context.Context, formID, ownerID int64) (FormSubmissions, error) {
	result := FormSubmissions{FormID: formID, Submissions: make([]SubmissionResult, 0)}
	err := repository.database.QueryRow(ctx, `
		SELECT title, slug FROM forms
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`, formID, ownerID).Scan(&result.Title, &result.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return FormSubmissions{}, ErrFormNotFound
	}
	if err != nil {
		return FormSubmissions{}, fmt.Errorf("find form submissions owner: %w", err)
	}

	rows, err := repository.database.Query(ctx, `
		SELECT s.id, s.created_at, ff.id, ff.field_type, ff.label, sa.value
		FROM submissions s
		LEFT JOIN submission_answers sa ON sa.submission_id = s.id AND sa.deleted_at IS NULL
		LEFT JOIN form_fields ff ON ff.id = sa.field_id
		WHERE s.form_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.created_at DESC, s.id DESC, ff.position
	`, formID)
	if err != nil {
		return FormSubmissions{}, fmt.Errorf("list submissions: %w", err)
	}
	defer rows.Close()
	byID := make(map[int64]int)
	for rows.Next() {
		var submissionID int64
		var fieldID *int64
		var createdAt time.Time
		var fieldType, label *string
		var value []byte
		if err := rows.Scan(&submissionID, &createdAt, &fieldID, &fieldType, &label, &value); err != nil {
			return FormSubmissions{}, fmt.Errorf("scan submission answer: %w", err)
		}
		index, exists := byID[submissionID]
		if !exists {
			index = len(result.Submissions)
			byID[submissionID] = index
			result.Submissions = append(result.Submissions, SubmissionResult{ID: submissionID, CreatedAt: createdAt, Answers: make([]SubmissionAnswerResult, 0)})
		}
		if fieldID != nil {
			result.Submissions[index].Answers = append(result.Submissions[index].Answers, SubmissionAnswerResult{FieldID: *fieldID, FieldType: FieldType(*fieldType), Label: *label, Value: value})
		}
	}
	if err := rows.Err(); err != nil {
		return FormSubmissions{}, fmt.Errorf("iterate submissions: %w", err)
	}
	return result, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

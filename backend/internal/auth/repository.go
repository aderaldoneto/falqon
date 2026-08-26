package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUnauthenticated = errors.New("session is missing, expired, or revoked")
var ErrEmailAlreadyExists = errors.New("email already exists")

type User struct {
	ID    int64
	Name  string
	Email string
}

type Repository struct {
	database *pgxpool.Pool
}

func NewRepository(database *pgxpool.Pool) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) CreateEmailUser(
	ctx context.Context,
	name string,
	email string,
	passwordHash string,
) (User, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin user transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user User
	err = tx.QueryRow(ctx, `
        INSERT INTO users (name, email, password_hash)
        VALUES ($1, $2, $3)
        RETURNING id, name, email
    `, name, email, passwordHash).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if constraintError, ok := err.(*pgconn.PgError); ok && constraintError.Code == "23505" {
			return User{}, ErrEmailAlreadyExists
		}
		return User{}, fmt.Errorf("create email user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
        UPDATE users SET created_by = id, updated_by = id WHERE id = $1
    `, user.ID); err != nil {
		return User{}, fmt.Errorf("set user audit fields: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit user transaction: %w", err)
	}
	return user, nil
}

func (repository *Repository) UserByEmail(ctx context.Context, email string) (User, string, error) {
	var user User
	var passwordHash *string
	err := repository.database.QueryRow(ctx, `
		SELECT id, name, email, password_hash
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(&user.ID, &user.Name, &user.Email, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && passwordHash == nil {
		return User{}, "", ErrUnauthenticated
	}
	if err != nil {
		return User{}, "", fmt.Errorf("find user by email: %w", err)
	}
	return user, *passwordHash, nil
}

func (repository *Repository) UpsertGoogleUser(
	ctx context.Context,
	identity GoogleIdentity,
) (User, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin user transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user User
	err = tx.QueryRow(ctx, `
        INSERT INTO users (name, email, google_subject)
        VALUES ($1, $2, $3)
        ON CONFLICT (email) DO UPDATE SET
            name = EXCLUDED.name,
            google_subject = EXCLUDED.google_subject,
            updated_by = users.id,
            deleted_by = NULL,
            deleted_at = NULL
        RETURNING id, name, email
    `, identity.Name, identity.Email, identity.Subject).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		return User{}, fmt.Errorf("upsert Google user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
        UPDATE users
        SET created_by = id, updated_by = id
        WHERE id = $1 AND created_by IS NULL
    `, user.ID); err != nil {
		return User{}, fmt.Errorf("set user audit fields: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit user transaction: %w", err)
	}
	return user, nil
}

func (repository *Repository) CreateSession(
	ctx context.Context,
	userID int64,
	duration time.Duration,
) (string, time.Time, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}
	expiresAt := time.Now().UTC().Add(duration)

	_, err = repository.database.Exec(ctx, `
        INSERT INTO sessions (
            user_id,
            token_hash,
            expires_at,
            created_by,
            updated_by
        ) VALUES ($1, $2, $3, $1, $1)
    `, userID, hashToken(token), expiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}

	return token, expiresAt, nil
}

func (repository *Repository) UserBySession(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}

	var user User
	err := repository.database.QueryRow(ctx, `
        SELECT users.id, users.name, users.email
        FROM sessions
        JOIN users ON users.id = sessions.user_id
        WHERE sessions.token_hash = $1
          AND sessions.expires_at > CURRENT_TIMESTAMP
          AND sessions.deleted_at IS NULL
          AND users.deleted_at IS NULL
    `, hashToken(token)).Scan(&user.ID, &user.Name, &user.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, fmt.Errorf("find session: %w", err)
	}
	return user, nil
}

func (repository *Repository) RevokeSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	_, err := repository.database.Exec(ctx, `
        UPDATE sessions
        SET
            deleted_at = CURRENT_TIMESTAMP,
            deleted_by = user_id,
            updated_by = user_id
        WHERE token_hash = $1 AND deleted_at IS NULL
    `, hashToken(token))
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

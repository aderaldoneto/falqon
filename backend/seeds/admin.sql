\set ON_ERROR_STOP on

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        WHERE id = 1
          AND email <> 'user@falqon.com.br'
    ) THEN
        RAISE EXCEPTION 'user id 1 already belongs to another email';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM users
        WHERE email = 'user@falqon.com.br'
          AND id <> 1
    ) THEN
        RAISE EXCEPTION 'admin email already belongs to another user id';
    END IF;
END;
$$;

INSERT INTO users (
    id,
    name,
    email,
    password_hash,
    created_by,
    updated_by
)
OVERRIDING SYSTEM VALUE
VALUES (
    1,
    'Usuário',
    'user@falqon.com.br',
    crypt('123456', gen_salt('bf', 12)),
    1,
    1
)
ON CONFLICT (id) DO UPDATE
SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    created_by = EXCLUDED.created_by,
    updated_by = EXCLUDED.updated_by,
    deleted_by = NULL,
    deleted_at = NULL;

SELECT setval(
    pg_get_serial_sequence('users', 'id'),
    GREATEST((SELECT MAX(id) FROM users), 1),
    TRUE
);

COMMIT;

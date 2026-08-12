-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, full_name, base_currency)
VALUES (@id, @email, @password_hash, @full_name, @base_currency)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = @id;

-- name: GetUserByEmail :one
-- Cột email là citext nên so sánh ở đây tự động không phân biệt hoa
-- thường, không cần lower() ở hai vế.
SELECT * FROM users
WHERE email = @email;

-- name: EmailExists :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = @email
) AS exists;

-- name: MarkEmailVerified :exec
UPDATE users
SET email_verified_at = now()
WHERE id = @id
  AND email_verified_at IS NULL;

-- name: UpdateUserProfile :one
UPDATE users
SET full_name     = @full_name,
    base_currency = @base_currency
WHERE id = @id
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = @password_hash
WHERE id = @id;

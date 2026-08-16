-- name: CreateUser :one
INSERT INTO users (display_name, given_name, picture_url, preferred_locale)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateUserGoogleProfile :one
-- Refresh picture from Google; fill given_name only when still unset.
UPDATE users
SET
    given_name = CASE
        WHEN given_name IS NULL AND NULLIF(sqlc.arg(given_name)::text, '') IS NOT NULL
            THEN NULLIF(sqlc.arg(given_name)::text, '')
        ELSE given_name
    END,
    picture_url = NULLIF(sqlc.arg(picture_url)::text, '')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateUserPreferredLocale :one
UPDATE users
SET preferred_locale = $2
WHERE id = $1
RETURNING *;

-- name: UpsertAuthIdentity :one
INSERT INTO auth_identities (user_id, provider, provider_subject, email, email_verified)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider, provider_subject) DO UPDATE
SET
    email = EXCLUDED.email,
    email_verified = EXCLUDED.email_verified
RETURNING *;

-- name: GetAuthIdentityByProviderSubject :one
SELECT * FROM auth_identities
WHERE provider = $1 AND provider_subject = $2;

-- name: GetAuthIdentitiesByUser :many
SELECT * FROM auth_identities
WHERE user_id = $1
ORDER BY created_at;

-- name: CreateHousehold :one
INSERT INTO households (name)
VALUES ($1)
RETURNING *;

-- name: GetHousehold :one
SELECT * FROM households
WHERE id = $1;

-- name: ClaimHousehold :one
UPDATE households
SET claimed_at = now()
WHERE id = $1 AND claimed_at IS NULL
RETURNING *;

-- name: GetUnclaimedSeedHousehold :one
SELECT * FROM households
WHERE claimed_at IS NULL
ORDER BY created_at
LIMIT 1;

-- name: GetHouseholdByName :one
SELECT * FROM households
WHERE name = $1
ORDER BY created_at
LIMIT 1;

-- name: GetFirstHousehold :one
SELECT * FROM households
ORDER BY created_at
LIMIT 1;

-- name: CreateMembership :one
INSERT INTO household_memberships (household_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMembershipsByUser :many
SELECT * FROM household_memberships
WHERE user_id = $1
ORDER BY created_at;

-- name: CreateSession :one
INSERT INTO sessions (
    user_id,
    token_hash,
    expires_at,
    absolute_expires_at,
    user_agent,
    ip
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT * FROM sessions
WHERE token_hash = $1;

-- name: TouchSession :one
UPDATE sessions
SET expires_at = $2
WHERE id = $1
  AND revoked_at IS NULL
RETURNING *;

-- name: RevokeSession :one
UPDATE sessions
SET revoked_at = now()
WHERE id = $1
  AND revoked_at IS NULL
RETURNING *;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE revoked_at IS NOT NULL
   OR expires_at < now()
   OR absolute_expires_at < now();

-- name: CreateOAuthLoginState :one
INSERT INTO oauth_login_states (state, nonce, code_verifier, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOAuthLoginState :one
SELECT * FROM oauth_login_states
WHERE state = $1;

-- name: DeleteOAuthLoginState :exec
DELETE FROM oauth_login_states
WHERE state = $1;

-- name: DeleteExpiredOAuthLoginStates :exec
DELETE FROM oauth_login_states
WHERE expires_at < now();

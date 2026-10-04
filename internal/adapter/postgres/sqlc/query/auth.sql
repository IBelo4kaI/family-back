-- name: CreateFamily :one
INSERT INTO families (name) VALUES ($1) RETURNING *;

-- name: GetFamily :one
SELECT * FROM families WHERE id = $1;

-- name: DeleteFamily :exec
DELETE FROM families WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, name, color)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: AddFamilyMember :exec
INSERT INTO family_members (user_id, family_id, role) VALUES ($1, $2, $3);

-- name: GetMembership :one
SELECT user_id, family_id, role, joined_at FROM family_members WHERE user_id = $1;

-- name: ListFamilyMembers :many
SELECT u.id, u.name, u.color, u.email, fm.role, fm.joined_at
FROM family_members fm
JOIN users u ON u.id = fm.user_id
WHERE fm.family_id = $1
ORDER BY fm.joined_at, u.id;

-- name: RemoveFamilyMember :execrows
DELETE FROM family_members WHERE user_id = $1 AND family_id = $2;

-- name: SetMemberRole :exec
UPDATE family_members SET role = $3 WHERE user_id = $1 AND family_id = $2;

-- name: OldestMember :one
SELECT user_id FROM family_members WHERE family_id = $1 ORDER BY joined_at, user_id LIMIT 1;

-- name: CreateInvite :exec
INSERT INTO invites (code, family_id, created_by, expires_at) VALUES ($1, $2, $3, $4);

-- name: GetActiveInvite :one
SELECT code, family_id FROM invites
WHERE code = $1 AND used_at IS NULL AND expires_at > now();

-- name: UseInvite :execrows
UPDATE invites SET used_at = now()
WHERE code = $1 AND used_at IS NULL AND expires_at > now();

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3);

-- name: ConsumeRefreshToken :one
DELETE FROM refresh_tokens
WHERE token_hash = $1 AND expires_at > now()
RETURNING user_id;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE token_hash = $1;

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens WHERE user_id = $1;

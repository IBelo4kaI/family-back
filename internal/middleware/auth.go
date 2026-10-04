package middleware

import (
	"errors"
	"strings"

	repo "family/internal/adapter/postgres/sqlc"
	"family/internal/token"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
)

// Auth проверяет access-токен и кладёт в Locals user_id и family_id.
// Членство читается из БД на каждый запрос, чтобы исключение из семьи
// действовало сразу, а не после истечения токена.
func Auth(tokens *token.Manager, q *repo.Queries) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get("Authorization")
		raw, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || raw == "" {
			return fiber.ErrUnauthorized
		}

		userID, err := tokens.ParseAccess(raw)
		if err != nil {
			return fiber.ErrUnauthorized
		}

		m, err := q.GetMembership(c.RequestCtx(), userID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fiber.ErrInternalServerError
		}

		c.Locals("user_id", userID)
		if err == nil {
			c.Locals("family_id", m.FamilyID)
			c.Locals("role", m.Role)
		}
		return c.Next()
	}
}

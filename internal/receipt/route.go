package receipt

import (
	"family/internal/budget"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(router fiber.Router, service Service, authMW fiber.Handler) {
	h := NewHandler(service)
	g := router.Group("/receipts", authMW, budget.RequireFamily)

	g.Post("/check", h.Check)
	g.Post("/check-image", h.CheckImage)
	g.Post("/", h.Save)
	g.Get("/:id/items", h.Items)
}

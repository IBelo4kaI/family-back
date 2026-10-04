package family

import "github.com/gofiber/fiber/v3"

func SetupRoutes(router fiber.Router, service Service, authMW fiber.Handler) {
	handler := NewHandler(service)
	g := router.Group("/family", authMW)

	g.Get("/", handler.Get)
	g.Post("/invites", handler.CreateInvite)
	g.Post("/leave", handler.Leave)
	g.Delete("/members/:id", handler.Kick)
}

package auth

import (
	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(router fiber.Router, service Service, authMW fiber.Handler) {
	handler := NewHandler(service)
	g := router.Group("/auth")

	g.Post("/register", handler.Register)
	g.Post("/join", handler.Join)
	g.Post("/login", handler.Login)
	g.Post("/refresh", handler.Refresh)
	g.Post("/logout", handler.Logout)
	g.Get("/me", authMW, handler.Me)
}

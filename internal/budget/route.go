package budget

import "github.com/gofiber/fiber/v3"

func SetupRoutes(router fiber.Router, service Service, authMW fiber.Handler) {
	h := NewHandler(service)
	g := router.Group("/budget", authMW, RequireFamily)

	g.Get("/", h.Load)

	g.Post("/categories", create(h, service.CreateCategory))

	g.Post("/transactions", create(h, service.CreateTransaction))
	g.Put("/transactions/:id", update(h, service.UpdateTransaction))
	g.Delete("/transactions/:id", remove(service.DeleteTransaction))

	g.Post("/payments", create(h, service.CreatePayment))
	g.Put("/payments/:id", update(h, service.UpdatePayment))
	g.Post("/payments/:id/close", remove(service.ClosePayment))
	g.Post("/payments/:id/settle", settle(service.SettlePayment))

	g.Post("/cards", create(h, service.CreateCard))
	g.Put("/cards/:id", update(h, service.UpdateCard))
	g.Post("/cards/:id/close", remove(service.CloseCard))
	g.Post("/cards/:id/settle", settle(service.SettleCard))

	g.Post("/limits", create(h, service.CreateLimit))
	g.Put("/limits/:id", update(h, service.UpdateLimit))
	g.Delete("/limits/:id", remove(service.DeleteLimit))

	g.Post("/goals", create(h, service.CreateGoal))
	g.Put("/goals/:id", update(h, service.UpdateGoal))
	g.Delete("/goals/:id", remove(service.DeleteGoal))
	g.Post("/goals/:id/top-up", h.TopUpGoal)
}

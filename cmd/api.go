package main

import (
	"log/slog"

	repo "family/internal/adapter/postgres/sqlc"
	"family/internal/auth"
	"family/internal/budget"
	"family/internal/family"
	"family/internal/middleware"
	"family/internal/receipt"
	"family/internal/token"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

type application struct {
	config config
	pool   *pgxpool.Pool
	logger *slog.Logger
}

type config struct {
	addr          string
	jwtSecret     string
	corsOrigins   []string
	receiptTokens []string
}

func (app *application) mount() *fiber.App {
	fiberApp := fiber.New(fiber.Config{BodyLimit: 10 << 20})

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins: app.config.corsOrigins,
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))
	fiberApp.Use(logger.New(logger.Config{
		Format: "${time} | [${ip}]:${port} | ${latency} | ${status} - ${method} ${path} \n",
	}))

	queries := repo.New(app.pool)
	tokens := token.NewManager(app.config.jwtSecret)
	authMW := middleware.Auth(tokens, queries)

	v1 := fiberApp.Group("v1")

	auth.SetupRoutes(v1, auth.NewService(queries, app.pool, tokens), authMW)
	family.SetupRoutes(v1, family.NewService(queries, app.pool), authMW)
	budget.SetupRoutes(v1, budget.NewService(queries, app.pool), authMW)
	receipt.SetupRoutes(v1, receipt.NewService(queries, app.pool, app.config.receiptTokens), authMW)

	return fiberApp
}

func (app *application) run(f *fiber.App) error {
	return f.Listen(app.config.addr)
}

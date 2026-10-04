package main

import (
	"context"
	"log/slog"
	"os"

	"family/internal/adapter/postgres"
	"family/internal/env"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	e := env.Env{}
	e.Init()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if e.GetJWTSecret() == "" {
		logger.Error("JWT_SECRET is empty")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, e.GetDbString())
	if err != nil {
		logger.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		logger.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	app := application{
		config: config{
			addr:          e.GetAddr(),
			jwtSecret:     e.GetJWTSecret(),
			corsOrigins:   e.GetCORSOrigins(),
			receiptTokens: e.GetProverkachekaTokens(),
		},
		pool:   pool,
		logger: logger,
	}

	if err := app.run(app.mount()); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

package main

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/qwaq-dev/test-api/internal/config"
	"github.com/qwaq-dev/test-api/internal/repository"
	"github.com/qwaq-dev/test-api/internal/route"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func main() {
	app := fiber.New()
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)

	err := repository.NewPostgresDB(cfg.Database, log)
	if err != nil {
		log.Error("Error connecting to database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	route.InitRoutes(app, log, cfg.ExternalAPI)

	log.Info("Server started", slog.String("port", cfg.HTTPServer.Port))
	app.Listen(cfg.HTTPServer.Port)
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

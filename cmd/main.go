// @title			Songs API
// @version		1.0
// @description	Онлайн библиотека песен
// @contact.name QwaQ
// @contact.email qwaq.dev@gmail.com
// @host			localhost:8080
// @BasePath		/api

package main

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	_ "github.com/qwaq-dev/test-api/cmd/docs"
	"github.com/qwaq-dev/test-api/cmd/internal/config"
	"github.com/qwaq-dev/test-api/cmd/internal/handler"
	"github.com/qwaq-dev/test-api/cmd/internal/repository"
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

	api := app.Group("/api")
	h := handler.NewHandler(log, cfg.ExternalAPI)

	api.Get("/songs", h.AllSongs)         //+
	api.Get("/song/:id", h.SongById)      //+
	api.Get("/song/:id/text", h.SongText) //+

	api.Post("/songs", h.CreateSong)       //+
	api.Put("/song/:id", h.UpdateSongInfo) //+
	api.Delete("/song/:id", h.DeleteSong)  //+

	app.Get("/swagger/*", swagger.HandlerDefault) // default

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

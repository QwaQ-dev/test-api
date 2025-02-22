package route

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/qwaq-dev/test-api/internal/handler"
)

func InitRoutes(app *fiber.App, log *slog.Logger, externalApi string) {
	api := app.Group("/api")
	h := handler.NewHandler(log, externalApi)

	api.Get("/songs", h.AllSongs)
	api.Get("/song/:id", h.SongById)
	api.Get("/song/:id/text", h.SongText)

	api.Post("/songs", h.CreateSong)
	api.Put("/song/:id", h.UpdateSong)
	api.Patch("/song/:id", h.PartialUpdateSong)
	api.Delete("/song/:id", h.DeleteSong)
}

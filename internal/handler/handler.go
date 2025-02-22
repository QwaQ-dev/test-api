package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/qwaq-dev/test-api/internal/repository"
	"github.com/qwaq-dev/test-api/structure"
)

type Handler struct {
	log         *slog.Logger
	externalApi string
}

func NewHandler(log *slog.Logger, externalApi string) *Handler {
	return &Handler{log: log, externalApi: externalApi}
}

func (h *Handler) CreateSong(c *fiber.Ctx) error {
	song := new(structure.Song)
	if err := c.BodyParser(song); err != nil {
		h.log.Error("Failed to parse request body", slog.String("error", err.Error()))
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	externalApiUrl := fmt.Sprintf("%s?group=%s&song=%s", h.externalApi, song.Name, song.Group)

	resp, err := http.Get(externalApiUrl)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error connecting to external API"})
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("External API returned status code: %d", resp.StatusCode)})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error reading API response"})
	}

	var songDetails structure.SongDetails
	if err := json.Unmarshal(body, &songDetails); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error decoding API response"})
	}

	_, err = repository.DB.Exec(`INSERT INTO Song (name, "group") VALUES ($1, $2)`,
		song.Name, song.Group)
	if err != nil {
		h.log.Error("Error inserting song", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error with inserting song name and group"})
	}

	_, err = repository.DB.Exec("INSERT INTO SongDetail (song_id, release_date, text, link) VALUES ($1, $2, $3, $4)",
		song.Id, songDetails.Release_date, songDetails.Text, songDetails.Link)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error with inserting SongDetails"})
	}

	h.log.Info("New song created", slog.String("name", song.Name))
	return c.Status(200).JSON(fiber.Map{"message": "New song created", "song": song})
}

func (h *Handler) AllSongs(c *fiber.Ctx) error {

	return c.SendString("All songs data")
}

func (h *Handler) SongText(c *fiber.Ctx) error {
	return c.SendString("Song text by ID")
}

func (h *Handler) SongById(c *fiber.Ctx) error {
	return c.SendString("Song by ID")
}

func (h *Handler) PartialUpdateSong(c *fiber.Ctx) error {
	return c.SendString("Partial update song")
}

func (h *Handler) UpdateSong(c *fiber.Ctx) error {
	return c.SendString("Song updated")
}

func (h *Handler) DeleteSong(c *fiber.Ctx) error {
	return c.SendString("Song deleted")
}

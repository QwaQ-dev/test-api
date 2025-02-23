package handler

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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

	// externalApiUrl := fmt.Sprintf("%s?group=%s&song=%s", h.externalApi, song.Name, song.Group)

	// resp, err := http.Get(externalApiUrl)
	// if err != nil {
	// 	return c.Status(500).JSON(fiber.Map{"error": "Error connecting to external API"})
	// }

	// defer resp.Body.Close()

	// if resp.StatusCode != 200 {
	// 	return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("External API returned status code: %d", resp.StatusCode)})
	// }

	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return c.Status(500).JSON(fiber.Map{"error": "Error reading API response"})
	// }

	// songDetails := new(structure.SongDetails)
	// if err := json.Unmarshal(body, &songDetails); err != nil {
	// 	return c.Status(500).JSON(fiber.Map{"error": "Error decoding API response"})
	// }

	err := repository.DB.QueryRow(`INSERT INTO song (song, "group") VALUES ($1, $2) RETURNING id`, song.Song, song.Group).Scan(&song.Id)
	if err != nil {
		h.log.Error("Error inserting song", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error with inserting song name and group"})
	}

	// _, err = repository.DB.Exec("INSERT INTO SongDetail (song_id, release_date, text, link) VALUES ($1, $2, $3, $4)",
	// 	song.Id, songDetails.Release_date, songDetails.Text, songDetails.Link)
	// if err != nil {
	// 	return c.Status(500).JSON(fiber.Map{"error": "Error with inserting SongDetails"})
	// }

	h.log.Info("New song created", slog.String("song", song.Song))
	return c.Status(200).JSON(fiber.Map{"message": "New song created", "song": song})
}

func (h *Handler) AllSongs(c *fiber.Ctx) error {
	name := c.Query("song")
	group := c.Query("group")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	query := `SELECT id, song, "group" FROM song WHERE 1=1`
	var args []any
	argID := 1

	if name != "" {
		query += fmt.Sprintf(" AND SONG ILIKE $%d", argID)
		args = append(args, "%"+group+"%")
		argID++
	}

	if group != "" {
		query += fmt.Sprintf(` AND "group" ILIKE $%d`, argID)
		args = append(args, "%"+group+"%")
		argID++
	}

	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", argID, argID+1)
	args = append(args, limit, offset)

	rows, err := repository.DB.Query(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get songs"})
	}
	defer rows.Close()

	var songs []structure.Song

	for rows.Next() {
		var song structure.Song
		if err := rows.Scan(&song.Id, &song.Song, &song.Group); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan song"})
		}
		songs = append(songs, song)
	}

	return c.Status(200).JSON(fiber.Map{
		"page":  page,
		"limit": limit,
		"songs": songs,
	})
}

func (h *Handler) SongText(c *fiber.Ctx) error {
	id := c.Params("id")
	var text string

	err := repository.DB.QueryRow("SELECT text FROM SongDetail WHERE song_id = $1", id).Scan(&text)
	if err != nil {

	}

	lines := strings.Split(text, "\n")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "2"))
	offset := (page - 1) * limit

	if offset >= len(lines) {
		return c.Status(400).JSON(fiber.Map{"error": "Page out of range"})
	}

	end := offset + limit
	if end > len(lines) {
		end = len(lines)
	}

	return c.Status(500).JSON(fiber.Map{
		"page":  page,
		"limit": limit,
		"text":  lines[offset:end],
	})
}

func (h *Handler) SongById(c *fiber.Ctx) error {
	id := c.Params("id")

	song := new(structure.Song)

	err := repository.DB.QueryRow(`SELECT id, song, "group" FROM song WHERE id = $1`, id).Scan(&song.Id, &song.Song, &song.Group)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Song not found"})
	}
	return c.JSON(fiber.Map{"Song": song})
}

func (h *Handler) UpdateSongInfo(c *fiber.Ctx) error {
	return c.SendString("Song updated")
}

func (h *Handler) DeleteSong(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := repository.DB.Exec("DELETE FROM song WHERE id = $1", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error with deleting song"})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "Song not found"})
	}

	return c.Status(200).JSON(fiber.Map{"message": fmt.Sprintf("song with id: %s was deleted successfully", id)})
}

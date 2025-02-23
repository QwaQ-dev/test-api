package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/qwaq-dev/test-api/cmd/internal/repository"
	"github.com/qwaq-dev/test-api/cmd/internal/structure"
	"github.com/qwaq-dev/test-api/pkg/logger/sl"
)

type Handler struct {
	log         *slog.Logger
	externalApi string
}

func NewHandler(log *slog.Logger, externalApi string) *Handler {
	return &Handler{log: log, externalApi: externalApi}
}

// @Summary Добавление новой песни
// @Description Позволяет добавить песню с указанием названия и группы
// @Tags Songs
// @Accept json
// @Produce json
// @Param song body structure.Song true "Данные песни"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/songs [post]
func (h *Handler) CreateSong(c *fiber.Ctx) error {
	song := new(structure.Song)
	if err := c.BodyParser(song); err != nil {
		h.log.Error("Failed to parse request body", slog.String("error", err.Error()))
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	externalApiUrl := fmt.Sprintf("%s?group=%s&song=%s", h.externalApi, song.Song, song.Group)

	h.log.Debug("External API url", slog.Any("Api", externalApiUrl))

	resp, err := http.Get(externalApiUrl)
	if err != nil {
		h.log.Error("Connect to external api failed", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error connecting to external API"})
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		h.log.Error("Not OK", slog.Any("status code", fmt.Sprint("External api return status code: %d", resp.StatusCode)))
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("External API returned status code: %d", resp.StatusCode)})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.log.Error("Failed to read API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error reading API response"})
	}

	h.log.Debug("API response", slog.Any("body", body))

	songDetails := new(structure.SongDetails)
	if err := json.Unmarshal(body, &songDetails); err != nil {
		h.log.Error("Error decoding API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error decoding API response"})
	}

	err = repository.DB.QueryRow(`INSERT INTO song (song, "group") VALUES ($1, $2) RETURNING id`, song.Song, song.Group).Scan(&song.Id)
	if err != nil {
		h.log.Error("Error inserting song", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error with inserting song name and group"})
	}

	h.log.Info("Success insert song data")

	_, err = repository.DB.Exec("INSERT INTO SongDetail (song_id, release_date, text, link) VALUES ($1, $2, $3, $4)",
		song.Id, songDetails.Release_date, songDetails.Text, songDetails.Link)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error with inserting SongDetails"})
	}

	h.log.Info("New song created", slog.String("song", song.Song))
	return c.Status(200).JSON(fiber.Map{"message": "New song created", "song": song})
}

// @Summary Получение списка песен
// @Description Можно фильтровать по названию и группе
// @Tags Songs
// @Accept json
// @Produce json
// @Param song query string false "Название песни"
// @Param group query string false "Название группы"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество записей на странице" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/songs [get]
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
		args = append(args, "%"+name+"%")
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
		h.log.Error("Failed to get songs", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get songs"})
	}
	defer rows.Close()

	var songs []structure.Song

	for rows.Next() {
		var song structure.Song
		if err := rows.Scan(&song.Id, &song.Song, &song.Group); err != nil {
			h.log.Error("Failed to scan song", slog.String("error", err.Error()))
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan song"})
		}
		songs = append(songs, song)
	}

	h.log.Debug("Song details", slog.Any("songs", songs))

	return c.Status(200).JSON(fiber.Map{
		"page":  page,
		"limit": limit,
		"songs": songs,
	})
}

// @Summary Получение текста с пагинацией
// @Description Можно фильтровать указать page и limit для текста
// @Tags Songs
// @Accept json
// @Produce json
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество срок песни на странице" default(2)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/song/{id}/text [get]
func (h *Handler) SongText(c *fiber.Ctx) error {
	id := c.Params("id")
	var text string

	err := repository.DB.QueryRow("SELECT text FROM songdetail WHERE song_id = $1", id).Scan(&text)
	if err != nil {
		h.log.Error("Failed get text from database", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error with getting song text from database"})
	}

	h.log.Debug("Song text without pagination", slog.Any("text", text))

	lines := strings.Split(text, "\n")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "2"))
	offset := (page - 1) * limit

	if offset >= len(lines) {
		h.log.Debug("Page out of range")
		return c.Status(400).JSON(fiber.Map{"error": "Page out of range"})
	}

	end := offset + limit
	if end > len(lines) {
		end = len(lines)
	}

	h.log.Info("Text with pagination", slog.Any("text", lines[offset:end]))

	return c.Status(200).JSON(fiber.Map{
		"page":  page,
		"limit": limit,
		"text":  lines[offset:end],
	})
}

// @Summary Получение песни по id
// @Tags Songs
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/song/{id} [get]
func (h *Handler) SongById(c *fiber.Ctx) error {
	id := c.Params("id")

	song := new(structure.Song)

	err := repository.DB.QueryRow(`SELECT id, song, "group" FROM song WHERE id = $1`, id).Scan(&song.Id, &song.Song, &song.Group)
	if err != nil {
		h.log.Error("Song not found", slog.String("error", err.Error()))
		return c.Status(400).JSON(fiber.Map{"error": "Song not found"})
	}

	h.log.Info("Song", slog.Any("song", song))

	return c.Status(200).JSON(fiber.Map{"Song": song})
}

// @Summary Обновление данных о песне
// @Tags Songs
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/song/{id} [put]
func (h *Handler) UpdateSongInfo(c *fiber.Ctx) error {
	idStr := c.Params("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.log.Error("Invalid song ID", sl.Err(err))
		return c.Status(400).JSON(fiber.Map{"error": "Invalid song ID"})
	}

	song := new(structure.Song)
	if err := c.BodyParser(song); err != nil {
		h.log.Error("Failed to parse request body", slog.String("error", err.Error()))
		return c.Status(400).JSON(fiber.Map{"error": "Falied to parse body"})
	}

	res, err := repository.DB.Exec(`UPDATE song SET song = $1, "group" = $2 WHERE id = $3 RETURNING id`, song.Song, song.Group, id)
	if err != nil {
		h.log.Error("Erro updating song", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error updating song"})
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Song not found"})
	}

	externalApiUrl := fmt.Sprintf("%s?group=%s&song=%s", h.externalApi, song.Song, song.Group)

	h.log.Debug("External API url", slog.Any("api", externalApiUrl))

	resp, err := http.Get(externalApiUrl)
	if err != nil {
		h.log.Error("Connect to external api failed", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error connecting to external API"})
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		h.log.Error("Not OK", slog.Any("status code", fmt.Sprint("External api return status code: %d", resp.StatusCode)))
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("External API returned status code: %d", resp.StatusCode)})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.log.Error("Failed to read API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error reading API response"})
	}

	h.log.Debug("API response", slog.Any("body", body))

	songDetails := new(structure.SongDetails)
	if err := json.Unmarshal(body, &songDetails); err != nil {
		h.log.Error("Error decoding API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error decoding API response"})
	}

	song.Id = id

	_, err = repository.DB.Exec("INSERT INTO SongDetail (song_id, release_date, text, link) VALUES ($1, $2, $3, $4)",
		song.Id, songDetails.Release_date, songDetails.Text, songDetails.Link)
	if err != nil {
		h.log.Error("Error with inserting SongDetails", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error with inserting SongDetails"})
	}
	h.log.Info("Updated song", slog.Any("song", song))
	return c.Status(200).JSON(fiber.Map{"message": "Song updated successfully"})
}

// @Summary Удаление песни по id
// @Tags Songs
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/song/{id} [delete]
func (h *Handler) DeleteSong(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := repository.DB.Exec("DELETE FROM song WHERE id = $1", id)
	if err != nil {
		h.log.Error("Error with deleting song", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error with deleting song"})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.log.Debug("Song not foud")
		return c.Status(500).JSON(fiber.Map{"error": "Song not found"})
	}

	h.log.Info("Song was deleted successfully")
	return c.Status(200).JSON(fiber.Map{"message": fmt.Sprintf("song with id: %s was deleted successfully", id)})
}

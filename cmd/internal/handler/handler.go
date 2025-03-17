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
)

type Handler struct {
	log         *slog.Logger
	externalApi string
}

func NewHandler(log *slog.Logger, externalApi string) *Handler {
	return &Handler{log: log, externalApi: externalApi}
}

// @Summary      Добавление новой песни
// @Description  Позволяет добавить песню с указанием названия и группы.
// @Tags         Songs
// @Accept       json
// @Produce      json
// @Param        song  body      structure.Song  true   "Данные песни (название и группа)"
// @Success      201   {object}  map[string]interface{}  "Песня успешно добавлена"
// @Failure      400   {object}  map[string]string  "Некорректные данные"
// @Failure      500   {object}  map[string]string  "Ошибка сервера"
// @Router       /api/songs [post]
func (h *Handler) CreateSong(c *fiber.Ctx) error {
	song := new(structure.Song)
	if err := c.BodyParser(song); err != nil {
		h.log.Error("Failed to parse request body", slog.String("error", err.Error()))
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	var group structure.Group
	if err := repository.DB.First(&group, song.GroupID).Error; err != nil {
		h.log.Error("Group not found", slog.String("error", err.Error()))
		return c.Status(400).JSON(fiber.Map{"error": "Group not found"})
	}

	externalApiUrl := fmt.Sprintf("%s?group=%s&song=%s", h.externalApi, group.Name, song.Song)

	resp, err := http.Get(externalApiUrl)
	if err != nil {
		h.log.Error("Connect to external api failed", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error connecting to external API"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		h.log.Error("External API returned an error", slog.Any("status", resp.StatusCode))
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("External API returned status code: %d", resp.StatusCode)})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.log.Error("Failed to read API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error reading API response"})
	}

	songDetails := new(structure.SongDetails)
	if err := json.Unmarshal(body, &songDetails); err != nil {
		h.log.Error("Error decoding API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error decoding API response"})
	}

	tx := repository.DB.Begin()

	if err := tx.Create(&song).Error; err != nil {
		tx.Rollback()
		h.log.Error("Error inserting song", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error inserting song"})
	}

	songDetails.SongID = uint(song.ID)

	if err := tx.Create(&songDetails).Error; err != nil {
		tx.Rollback()
		h.log.Error("Error inserting SongDetails", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error inserting song details"})
	}

	tx.Commit()

	h.log.Info("New song created", slog.String("song", song.Song))
	return c.Status(200).JSON(fiber.Map{"message": "New song created", "song": song})
}

// @Summary      Получение списка песен
// @Description  Возвращает список песен с возможностью фильтрации по названию и группе, а также с пагинацией.
// @Tags         Songs
// @Accept       json
// @Produce      json
// @Param        song   query     string  false  "Фильтр по названию песни (поиск по подстроке)"
// @Param        group  query     string  false  "Фильтр по названию группы (поиск по подстроке)"
// @Param        page   query     int     false  "Номер страницы"  default(1)
// @Param        limit  query     int     false  "Количество записей на странице"  default(10)
// @Success      200    {object}  map[string]interface{}  "Список песен"
// @Failure      400    {object}  map[string]string  "Некорректный запрос"
// @Failure      500    {object}  map[string]string  "Ошибка сервера"
// @Router       /api/songs [get]
func (h *Handler) AllSongs(c *fiber.Ctx) error {
	name := c.Query("song")
	group := c.Query("group")

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	var songs []structure.Song
	query := repository.DB.Model(&structure.Song{}).Preload("Group")

	if name != "" {
		query = query.Where("song ILIKE ?", "%"+name+"%")
	}

	if group != "" {
		query = query.Joins("JOIN groups ON groups.id = songs.group_id").
			Where("groups.name ILIKE ?", "%"+group+"%")
	}

	if err := query.Order("id DESC").Limit(limit).Offset(offset).Find(&songs).Error; err != nil {
		h.log.Error("Failed to get songs", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get songs"})
	}

	h.log.Debug("Song details", slog.Any("songs", songs))

	return c.Status(200).JSON(fiber.Map{
		"page":  page,
		"limit": limit,
		"songs": songs,
	})
}

// @Summary      Получение текста песни с пагинацией
// @Description  Возвращает текст песни по ID с возможностью указать номер страницы и количество строк на странице.
// @Tags         Songs
// @Accept       json
// @Produce      json
// @Param        id    path      int  true  "ID песни"
// @Param        page  query     int  false  "Номер страницы"  default(1)
// @Param        limit query     int  false  "Количество строк текста на странице"  default(2)
// @Success      200   {object}  map[string]interface{}  "Текст песни с пагинацией"
// @Failure      400   {object}  map[string]string  "Некорректный запрос"
// @Failure      404   {object}  map[string]string  "Песня или текст не найдены"
// @Failure      500   {object}  map[string]string  "Ошибка сервера"
// @Router       /api/song/{id}/text [get]
func (h *Handler) SongText(c *fiber.Ctx) error {
	id := c.Params("id")
	var songDetail structure.SongDetails

	if err := repository.DB.Select("text").Where("song_id = ?", id).First(&songDetail).Error; err != nil {
		h.log.Error("Failed to get text from database", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error getting song text from database"})
	}

	if songDetail.Text == "" {
		h.log.Debug("Song has no text")
		return c.Status(404).JSON(fiber.Map{"error": "No text available for this song"})
	}

	h.log.Debug("Song text without pagination", slog.Any("text", songDetail.Text))

	lines := strings.Split(songDetail.Text, "\n")

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.Query("limit", "2"))
	if err != nil || limit < 1 {
		limit = 2
	}

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

// @Summary      Получение информации о песне
// @Description  Возвращает информацию о песне по её ID, включая название и группу.
// @Tags         Songs
// @Accept       json
// @Produce      json
// @Param        id   path      int                   true  "ID песни"
// @Success      200  {object}  structure.Song        "Данные о песне"
// @Failure      400  {object}  map[string]string     "Некорректный запрос (например, ID не является числом)"
// @Failure      404  {object}  map[string]string     "Песня не найдена"
// @Failure      500  {object}  map[string]string     "Ошибка сервера"
// @Router       /api/song/{id} [get]
func (h *Handler) SongById(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		h.log.Error("Invalid song ID", slog.String("id", c.Params("id")))
		return c.Status(400).JSON(fiber.Map{"error": "Invalid song ID"})
	}

	var song structure.Song

	if err := repository.DB.Preload("Group").Preload("SongDetails").Where("id = ?", id).First(&song).Error; err != nil {
		h.log.Error("Song not found", slog.String("error", err.Error()))
		return c.Status(404).JSON(fiber.Map{"error": "Song not found"})
	}

	h.log.Info("Song found", slog.Any("song", song))

	return c.Status(200).JSON(fiber.Map{"song": song})
}

// @Summary      Обновление данных о песне
// @Description  Обновляет информацию о песне по её ID, включая название и группу.
// @Tags         Songs
// @Accept       json
// @Produce      json
// @Param        id   path      int                   true  "ID песни"
// @Param        song body      structure.Song  true  "Объект с обновлёнными данными"
// @Success      200  {object}  map[string]string     "Песня успешно обновлена"
// @Failure      400  {object}  map[string]string     "Некорректный запрос или ошибка валидации"
// @Failure      404  {object}  map[string]string     "Песня не найдена"
// @Failure      500  {object}  map[string]string     "Внутренняя ошибка сервера"
// @Router       /api/song/{id} [put]
func (h *Handler) UpdateSongInfo(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		h.log.Error("Invalid song ID", slog.String("id", idStr))
		return c.Status(400).JSON(fiber.Map{"error": "Invalid song ID"})
	}

	var song structure.Song
	if err := c.BodyParser(&song); err != nil {
		h.log.Error("Failed to parse request body", slog.String("error", err.Error()))
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse body"})
	}

	var existingSong structure.Song
	if err := repository.DB.Preload("Group").Where("id = ?", id).First(&existingSong).Error; err != nil {
		h.log.Error("Song not found", slog.String("error", err.Error()))
		return c.Status(404).JSON(fiber.Map{"error": "Song not found"})
	}

	result := repository.DB.Model(&existingSong).Updates(map[string]interface{}{
		"song": song.Song,
	})

	if result.Error != nil {
		h.log.Error("Error updating song", slog.String("error", result.Error.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error updating song"})
	}

	externalApiUrl := fmt.Sprintf("%s?group=%s&song=%s", h.externalApi, song.Song, existingSong.Group.Name)
	h.log.Debug("External API URL", slog.String("url", externalApiUrl))

	resp, err := http.Get(externalApiUrl)
	if err != nil {
		h.log.Error("Failed to connect to external API", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error connecting to external API"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		h.log.Error("External API returned an error", slog.Int("status_code", resp.StatusCode))
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("External API returned status code: %d", resp.StatusCode)})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.log.Error("Failed to read API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error reading API response"})
	}

	var songDetails structure.SongDetails
	if err := json.Unmarshal(body, &songDetails); err != nil {
		h.log.Error("Error decoding API response", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error decoding API response"})
	}

	songDetails.SongID = uint(id)

	if err := repository.DB.Where("song_id = ?", id).Delete(&structure.SongDetails{}).Error; err != nil {
		h.log.Error("Error deleting old SongDetails", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error deleting old song details"})
	}

	if err := repository.DB.Create(&songDetails).Error; err != nil {
		h.log.Error("Error inserting SongDetails", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error inserting song details"})
	}

	h.log.Info("Song updated successfully", slog.Any("song", song))
	return c.Status(200).JSON(fiber.Map{"message": "Song updated successfully"})
}

// @Summary      Частичное обновление песни
// @Description  Позволяет обновить одно или несколько полей песни по её ID
// @Tags         Songs
// @Accept       json
// @Produce      json
// @Param        id   path      int                     true  "ID песни"
// @Param        data body      map[string]interface{}  true  "Данные для обновления (только изменяемые поля)"
// @Success      200  {object}  map[string]string       "Песня успешно обновлена"
// @Failure      400  {object}  map[string]string       "Некорректный запрос"
// @Failure      404  {object}  map[string]string       "Песня не найдена"
// @Failure      500  {object}  map[string]string       "Ошибка сервера"
// @Router       /api/song/{id} [patch]
func (h *Handler) PartialUpdateSong(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		h.log.Error("Invalid song ID", slog.String("id", idStr))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid song ID"})
	}

	var updateData map[string]interface{}
	if err := c.BodyParser(&updateData); err != nil {
		h.log.Error("Failed to parse request body", slog.String("error", err.Error()))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(updateData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No data provided for update"})
	}

	var existingSong structure.Song
	if err := repository.DB.Where("id = ?", id).First(&existingSong).Error; err != nil {
		h.log.Error("Song not found", slog.String("error", err.Error()))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Song not found"})
	}

	if groupID, exists := updateData["group_id"]; exists {
		var group structure.Group
		if err := repository.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
			h.log.Error("Group not found", slog.String("group_id", fmt.Sprintf("%v", groupID)))
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid group ID"})
		}
	}

	if err := repository.DB.Model(&structure.Song{}).Where("id = ?", id).Updates(updateData).Error; err != nil {
		h.log.Error("Error updating song", slog.String("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Update failed"})
	}

	h.log.Info("Song updated successfully", slog.Any("song_id", id), slog.Any("updates", updateData))
	return c.JSON(fiber.Map{"message": "Song updated"})
}

// @Summary      Удаление песни по ID
// @Description  Удаляет песню из базы данных по её уникальному идентификатору.
// @Tags         Songs
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "ID песни"
// @Success      200  {object}  map[string]string  "Песня успешно удалена"
// @Failure      400  {object}  map[string]string  "Некорректный ID"
// @Failure      404  {object}  map[string]string  "Песня не найдена"
// @Failure      500  {object}  map[string]string  "Ошибка сервера"
// @Router       /api/song/{id} [delete]
func (h *Handler) DeleteSong(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		h.log.Error("Invalid song ID", slog.String("id", idStr))
		return c.Status(400).JSON(fiber.Map{"error": "Invalid song ID"})
	}

	var song structure.Song
	if err := repository.DB.Where("id = ?", id).First(&song).Error; err != nil {
		h.log.Error("Song not found", slog.String("error", err.Error()))
		return c.Status(404).JSON(fiber.Map{"error": "Song not found"})
	}

	if err := repository.DB.Where("song_id = ?", id).Delete(&structure.SongDetails{}).Error; err != nil {
		h.log.Error("Error deleting song details", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error deleting song details"})
	}

	if err := repository.DB.Delete(&song).Error; err != nil {
		h.log.Error("Error deleting song", slog.String("error", err.Error()))
		return c.Status(500).JSON(fiber.Map{"error": "Error deleting song"})
	}

	h.log.Info("Song deleted successfully", slog.Any("song_id", id))
	return c.Status(200).JSON(fiber.Map{"message": fmt.Sprintf("Song with id %d was deleted successfully", id)})

}

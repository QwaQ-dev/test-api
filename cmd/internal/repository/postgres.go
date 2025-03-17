package repository

import (
	"fmt"
	"log/slog"

	"github.com/qwaq-dev/test-api/cmd/internal/config"
	"github.com/qwaq-dev/test-api/cmd/internal/structure"
	"github.com/qwaq-dev/test-api/pkg/logger/sl"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func NewPostgresDB(cfg config.Database, log *slog.Logger) error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.Username, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Error("No database with this settings", sl.Err(err))
		return err
	}

	DB = db

	err = DB.AutoMigrate(&structure.Song{}, &structure.SongDetails{})
	if err != nil {
		log.Error("Migration failed", sl.Err(err))
		return err
	}

	log.Info("Success connect to database")
	return nil
}

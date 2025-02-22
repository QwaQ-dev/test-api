package repository

import (
	"fmt"
	"log/slog"

	"database/sql"

	_ "github.com/lib/pq"
	"github.com/qwaq-dev/test-api/internal/config"
	"github.com/qwaq-dev/test-api/pkg/logger/sl"
)

var DB *sql.DB

func NewPostgresDB(cfg config.Database, log *slog.Logger) error {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Name, cfg.Password, cfg.SSLMode))
	if err != nil {
		log.Error("No database with this settings", sl.Err(err))
		return err
	}

	err = db.Ping()

	if err != nil {
		log.Error("Can't connect to database", sl.Err(err))
		return err
	}

	DB = db
	log.Info("Success connect to database")
	return nil
}

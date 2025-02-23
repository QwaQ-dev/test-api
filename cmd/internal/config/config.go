package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env" env-default:"dev" env-requried:"true"`
	ExternalAPI string `yaml:"external_api"`
	HTTPServer  `yaml:"http_server"`
	Database    `yaml:"database"`
}

type HTTPServer struct {
	Port string `yaml:"port" env-default:":8080"`
}

type Database struct {
	Name     string `yaml:"db_name"`
	Host     string `yaml:"db_host"`
	Username string `yaml:"db_username"`
	Password string `yaml:"db_password"`
	Port     string `yaml:"db_port"`
	SSLMode  string `yaml:"ssl_mode"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG")

	if configPath == "" {
		log.Fatalf("No env file")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Config file is not exists: %s", err.Error())
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("Cannot read config: %s", err.Error())
	}

	log.Println("Config was read successfully")
	return &cfg
}

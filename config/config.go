package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken    string
	DatabaseURL string
	WebAppURL   string
	ServerPort  string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	return &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		WebAppURL:   os.Getenv("WEBAPP_URL"),
		ServerPort:  port,
	}
}

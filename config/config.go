package config

import (
	"log"
	"os"
	"strconv"
	"github.com/joho/godotenv"
)

type Config struct {
	BotToken    string
	DatabaseURL string
	WebAppURL   string
	ServerPort  string
	AdminChatID int64
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file, using environment variables")
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" { port = "8080" }
	adminID, _ := strconv.ParseInt(os.Getenv("ADMIN_CHAT_ID"), 10, 64)
	return &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		WebAppURL:   os.Getenv("WEBAPP_URL"),
		ServerPort:  port,
		AdminChatID: adminID,
	}
}

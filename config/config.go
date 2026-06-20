package config

import (
	"fmt"
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
	IsLocal     bool
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("[config] .env fayl topilmadi, environment o'zgaruvchilari ishlatiladi")
	}

	adminID, _ := strconv.ParseInt(os.Getenv("ADMIN_CHAT_ID"), 10, 64)
	port := getEnvOrDefault("PORT", getEnvOrDefault("SERVER_PORT", "8080"))
	webAppURL := os.Getenv("WEBAPP_URL")
	isLocal := webAppURL == ""

	if isLocal {
		webAppURL = fmt.Sprintf("http://localhost:%s/webapp/", port)
		log.Printf("[config] WEBAPP_URL yo'q, local mode: %s", webAppURL)
	}

	cfg := &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		WebAppURL:   webAppURL,
		ServerPort:  port,
		AdminChatID: adminID,
		IsLocal:     isLocal,
	}

	cfg.validate()
	return cfg
}

func (c *Config) validate() {
	if c.BotToken == "" {
		log.Fatal("[config] BOT_TOKEN majburiy")
	}
	if c.DatabaseURL == "" {
		log.Fatal("[config] DATABASE_URL majburiy")
	}
}

func (c *Config) IsAdmin(userID int64) bool {
	return c.AdminChatID != 0 && userID == c.AdminChatID
}

func (c *Config) ServerAddr() string {
	return fmt.Sprintf(":%s", c.ServerPort)
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

package db

import (
	"log"
	"mafia-bot/db/models"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectPostgres(dsn string) *gorm.DB {
	var db *gorm.DB
	var err error

	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			break
		}
		log.Printf("[db] postgres waiting... (%d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(3 * time.Second)
		}
	}
	if err != nil {
		log.Fatalf("[db] postgres connection failed: %v", err)
	}

	runMigrations(db)
	seedShopItems(db)
	log.Println("[db] postgres connected")
	return db
}

func runMigrations(db *gorm.DB) {
	tables := []struct {
		model interface{}
		name  string
	}{
		{&models.User{}, "users"},
		{&models.Game{}, "games"},
		{&models.GamePlayer{}, "game_players"},
		{&models.Item{}, "items"},
		{&models.UserItem{}, "user_items"},
	}

	for _, t := range tables {
		if err := db.AutoMigrate(t.model); err != nil {
			log.Printf("[db] migration %s: %v", t.name, err)
		}
	}

	// Fix missing columns on existing tables
	raw := db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name TEXT DEFAULT ''")
	if raw.Error != nil {
		log.Printf("[db] add first_name: %v", raw.Error)
	}
	raw = db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS active_skin TEXT DEFAULT 'default'")
	if raw.Error != nil {
		log.Printf("[db] add active_skin: %v", raw.Error)
	}
}

func seedShopItems(db *gorm.DB) {
	for _, item := range models.DefaultShopItems() {
		db.FirstOrCreate(&item, models.Item{ID: item.ID})
	}
}

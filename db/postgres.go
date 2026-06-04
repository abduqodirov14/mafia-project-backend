package db

import (
	"log"
	"mafia-bot/db/models"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectPostgres(dsn string) *gorm.DB {
	var db *gorm.DB
	var err error
	
	// Retry logic - database tayyor bo'lguncha 30 soniya kutadi
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		
		if err == nil {
			break // Muvaffaqiyatli ulandi
		}
		
		log.Printf("⏳ PostgreSQL kutilmoqda... (%d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(3 * time.Second)
		}
	}
	
	if err != nil {
		log.Fatalf("❌ PostgreSQL ulanish xato: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Game{},
		&models.GamePlayer{}, &models.Item{}, &models.UserItem{},
	); err != nil {
		log.Printf("⚠️ Migration (normal): %v", err)
	}
	items := models.DefaultShopItems()
	for _, item := range items {
		db.FirstOrCreate(&item, models.Item{ID: item.ID})
	}
	log.Println("✅ PostgreSQL ulandi")
	DB = db
	return db
}

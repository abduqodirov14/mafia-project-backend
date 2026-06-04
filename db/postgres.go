package db

import (
	"log"
	"mafia-bot/db/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectPostgres(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("PostgreSQL xato: %v", err)
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

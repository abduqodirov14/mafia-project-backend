package models

import "time"

type Rarity string

const (
	RarityCommon  Rarity = "Common"
	RarityRare    Rarity = "Rare"
	RarityEpic    Rarity = "Epic"
	RarityLegend  Rarity = "Legend"
)

type Item struct {
	ID          uint   `gorm:"primarykey"`
	Name        string `gorm:"not null"`
	Description string
	Category    string
	Rarity      Rarity
	Price       int
	Emoji       string
}

type UserItem struct {
	ID         uint      `gorm:"primarykey"`
	UserID     uint      `gorm:"not null"`
	ItemID     uint      `gorm:"not null"`
	IsEquipped bool      `gorm:"default:false"`
	BoughtAt   time.Time `gorm:"autoCreateTime"`
}

func DefaultShopItems() []Item {
	return []Item{
		{ID: 1, Name: "Bo'ri Alpha", Description: "Mafiya boshlig'i", Category: "character", Rarity: RarityRare, Price: 800, Emoji: "🐺"},
		{ID: 2, Name: "Vampir", Description: "Qotil", Category: "character", Rarity: RarityEpic, Price: 1200, Emoji: "🦇"},
		{ID: 3, Name: "Tulki", Description: "Ayg'ovchi", Category: "character", Rarity: RarityCommon, Price: 300, Emoji: "🦊"},
		{ID: 4, Name: "Ajdaho", Description: "Yetakchi", Category: "character", Rarity: RarityLegend, Price: 2400, Emoji: "🐉"},
	}
}

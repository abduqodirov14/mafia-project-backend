package models

import "time"

type GameStatus string

const (
	StatusWaiting  GameStatus = "waiting"
	StatusPlaying  GameStatus = "playing"
	StatusFinished GameStatus = "finished"
)

type WinnerTeam string

const (
	WinnerMafia    WinnerTeam = "mafia"
	WinnerCivilian WinnerTeam = "civilian"
)

type Game struct {
	ID          uint       `gorm:"primarykey"`
	RoomID      string     `gorm:"uniqueIndex;not null"`
	ChatID      int64      `gorm:"not null"`
	Status      GameStatus `gorm:"default:'waiting'"`
	WinnerTeam  WinnerTeam
	PlayerCount int
	StartedAt   *time.Time
	EndedAt     *time.Time
	CreatedAt   time.Time
}

type GamePlayer struct {
	ID         uint   `gorm:"primarykey"`
	GameID     uint   `gorm:"not null"`
	UserID     uint   `gorm:"not null"`
	TelegramID int64  `gorm:"not null"`
	Username   string
	Role       string
	IsAlive    bool `gorm:"default:true"`
	XPEarned   int  `gorm:"default:0"`
}

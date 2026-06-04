package models

import "time"

type League string

const (
	LeagueBronze   League = "🥉 Bronza"
	LeagueSilver   League = "🥈 Kumush"
	LeagueGold     League = "🥇 Oltin"
	LeagueDiamond  League = "💎 Olmos"
)

type User struct {
	ID          uint      `gorm:"primarykey"`
	TelegramID  int64     `gorm:"uniqueIndex;not null"`
	Username    string    `gorm:"not null"`
	FirstName   string
	XP          int       `gorm:"default:0"`
	Level       int       `gorm:"default:1"`
	Coins       int       `gorm:"default:100"`
	TotalGames  int       `gorm:"default:0"`
	Wins        int       `gorm:"default:0"`
	WinStreak   int       `gorm:"default:0"`
	MaxStreak   int       `gorm:"default:0"`
	League      League    `gorm:"default:'🥉 Bronza'"`
	ActiveSkin  string    `gorm:"default:'default'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (u *User) WinRate() int {
	if u.TotalGames == 0 {
		return 0
	}
	return (u.Wins * 100) / u.TotalGames
}

func (u *User) UpdateLeague() {
	switch {
	case u.XP >= 10000:
		u.League = LeagueDiamond
	case u.XP >= 5000:
		u.League = LeagueGold
	case u.XP >= 2000:
		u.League = LeagueSilver
	default:
		u.League = LeagueBronze
	}
}

func (u *User) AddXP(amount int) {
	u.XP += amount
	u.Level = (u.XP / 500) + 1
	u.UpdateLeague()
}

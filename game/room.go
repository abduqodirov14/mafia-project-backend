package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type RoomStatus string

const (
	RoomWaiting  RoomStatus = "waiting"
	RoomJoining  RoomStatus = "joining"
	RoomPlaying  RoomStatus = "playing"
	RoomFinished RoomStatus = "finished"
)

type Room struct {
	ID         string
	ChatID     int64
	OwnerID    int64
	Players    map[int64]*Player
	MaxPlayers int
	Status     RoomStatus
	CreatedAt  time.Time
	mu         sync.RWMutex
}

func NewRoom(chatID, ownerID int64, ownerName string) *Room {
	roomID := generateID()
	r := &Room{
		ID:         roomID,
		ChatID:     chatID,
		OwnerID:    ownerID,
		Players:    make(map[int64]*Player),
		MaxPlayers: 15,
		Status:     RoomWaiting,
		CreatedAt:  time.Now(),
	}
	r.Players[ownerID] = &Player{
		TelegramID: ownerID,
		Username:   ownerName,
		IsAlive:    true,
		JoinOrder:  1,
	}
	return r
}

func (r *Room) AddPlayer(p *Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Status != RoomWaiting && r.Status != RoomJoining {
		return fmt.Errorf("o'yin allaqachon boshlangan")
	}
	if len(r.Players) >= r.MaxPlayers {
		return fmt.Errorf("xona to'lgan (%d/%d)", len(r.Players), r.MaxPlayers)
	}
	if _, exists := r.Players[p.TelegramID]; exists {
		return fmt.Errorf("siz allaqachon xonаdasiz")
	}
	p.JoinOrder = len(r.Players) + 1
	r.Players[p.TelegramID] = p
	return nil
}

func (r *Room) RemovePlayer(userID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Players, userID)
}

func (r *Room) GetPlayerList() []*Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Player, 0, len(r.Players))
	for _, p := range r.Players {
		list = append(list, p)
	}
	return list
}

func (r *Room) AlivePlayers() []*Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var alive []*Player
	for _, p := range r.Players {
		if p.IsAlive {
			alive = append(alive, p)
		}
	}
	return alive
}

func (r *Room) PlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Players)
}

func (r *Room) AliveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, p := range r.Players {
		if p.IsAlive {
			count++
		}
	}
	return count
}

func generateID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(900000)+100000)
}

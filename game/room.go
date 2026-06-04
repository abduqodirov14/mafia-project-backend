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
	RoomPlaying  RoomStatus = "playing"
	RoomFinished RoomStatus = "finished"
)

type Player struct {
	TelegramID int64
	Username   string
	FirstName  string
	Role       string
	IsAlive    bool
	IsMuted    bool
	JoinOrder  int
}

type Room struct {
	ID        string
	ChatID    int64
	OwnerID   int64
	Players   map[int64]*Player
	Status    RoomStatus
	MaxPlayer int
	CreatedAt time.Time
	mu        sync.RWMutex
}

func NewRoom(chatID, ownerID int64, ownerName string) *Room {
	roomID := generateRoomID()
	room := &Room{
		ID:        roomID,
		ChatID:    chatID,
		OwnerID:   ownerID,
		Players:   make(map[int64]*Player),
		Status:    RoomWaiting,
		MaxPlayer: 12,
		CreatedAt: time.Now(),
	}
	room.Players[ownerID] = &Player{
		TelegramID: ownerID,
		Username:   ownerName,
		IsAlive:    true,
	}
	return room
}

func (r *Room) AddPlayer(p *Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status != RoomWaiting {
		return fmt.Errorf("o'yin boshlangan")
	}
	if len(r.Players) >= r.MaxPlayer {
		return fmt.Errorf("xona to'lgan")
	}
	if _, exists := r.Players[p.TelegramID]; exists {
		return fmt.Errorf("allaqachon xonаdasiz")
	}

	p.JoinOrder = len(r.Players) + 1
	r.Players[p.TelegramID] = p
	return nil
}

func (r *Room) RemovePlayer(telegramID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Players, telegramID)
}

func (r *Room) PlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Players)
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

func (r *Room) GetPlayerList() []*Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*Player
	for _, p := range r.Players {
		list = append(list, p)
	}
	// Sort by join order
	for i := 0; i < len(list)-1; i++ {
		for j := i + 1; j < len(list); j++ {
			if list[i].JoinOrder > list[j].JoinOrder {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	return list
}

func generateRoomID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(999999))
}

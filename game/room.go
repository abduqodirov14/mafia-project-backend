package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	MaxPlayers = 15
)

type RoomStatus string

const (
	RoomWaiting  RoomStatus = "waiting"
	RoomJoining  RoomStatus = "joining"
	RoomPlaying  RoomStatus = "playing"
	RoomFinished RoomStatus = "finished"
)

var (
	ErrRoomFull         = fmt.Errorf("xona to'lgan")
	ErrAlreadyInRoom    = fmt.Errorf("siz allaqachon xonадasisiz")
	ErrGameAlreadyStarted = fmt.Errorf("o'yin allaqachon boshlangan")
	ErrPlayerNotFound   = fmt.Errorf("o'yinchi topilmadi")
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
	r := &Room{
		ID:         generateID(),
		ChatID:     chatID,
		OwnerID:    ownerID,
		Players:    make(map[int64]*Player),
		MaxPlayers: MaxPlayers,
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
		return ErrGameAlreadyStarted
	}
	if len(r.Players) >= r.MaxPlayers {
		return ErrRoomFull
	}
	if _, exists := r.Players[p.TelegramID]; exists {
		return ErrAlreadyInRoom
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

	alive := make([]*Player, 0)
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

func (r *Room) PlayerByID(id int64) (*Player, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.Players[id]
	return p, ok
}

func (r *Room) SetStatus(s RoomStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Status = s
}

func generateID() string {
	return fmt.Sprintf("%06d", rand.Intn(900000)+100000)
}

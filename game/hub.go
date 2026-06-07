package game

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

const (
	MsgTypeRoomInfo    = "room_info"
	MsgTypeRoleReveal  = "role_reveal"
	MsgTypeGameState   = "game_state"
	MsgTypePhaseChange = "phase_change"
	MsgTypePlayerDied  = "player_died"
	MsgTypeGameEnd     = "game_end"
	MsgTypeChat        = "chat"
	MsgTypeVoiceSignal = "voice_signal"
	MsgTypeNightResult = "night_result"
	MsgTypeSheriffResult = "sheriff_result"
)

type WSMessage struct {
	Type    string          `json:"type"`
	RoomID  string          `json:"room_id,omitempty"`
	UserID  int64           `json:"user_id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type PlayerInfo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsAlive  bool   `json:"is_alive"`
	Role     string `json:"role,omitempty"`
	Emoji    string `json:"emoji,omitempty"`
}

type RoleRevealPayload struct {
	Role        string `json:"role"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
}

type GameStatePayload struct {
	Phase   string       `json:"phase"`
	Round   int          `json:"round"`
	Players []PlayerInfo `json:"players"`
	Timer   int          `json:"timer,omitempty"`
}

type PhasePayload struct {
	Phase   string `json:"phase"`
	Round   int    `json:"round"`
	Message string `json:"message"`
	Timer   int    `json:"timer"`
}

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	roomID string
	userID int64
}

type BroadcastMsg struct {
	roomID  string
	message []byte
	exclude int64
}

type DirectMsg struct {
	userID  int64
	message []byte
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMsg
	direct     chan *DirectMsg
	mu         sync.RWMutex

	OnConnect    func(roomID string, userID int64)
	OnStartGame  func(roomID string, ownerID int64) error
	OnNightAction func(roomID, role string, voterID, targetID int64)
	OnDayVote    func(roomID string, voterID, targetID int64)
	OnChat       func(roomID string, userID int64, username, text string)
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *BroadcastMsg, 256),
		direct:     make(chan *DirectMsg, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.roomID] == nil {
				h.rooms[client.roomID] = make(map[*Client]bool)
			}
			h.rooms[client.roomID][client] = true
			h.mu.Unlock()
			log.Printf("✅ WS: user %d → room %s", client.userID, client.roomID)
			if h.OnConnect != nil {
				go h.OnConnect(client.roomID, client.userID)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if room, ok := h.rooms[client.roomID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.send)
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.rooms[msg.roomID] {
				if msg.exclude != 0 && client.userID == msg.exclude {
					continue
				}
				select {
				case client.send <- msg.message:
				default:
				}
			}
			h.mu.RUnlock()

		case msg := <-h.direct:
			h.mu.RLock()
			for _, clients := range h.rooms {
				for client := range clients {
					if client.userID == msg.userID {
						select {
						case client.send <- msg.message:
						default:
						}
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID, msgType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	msg := WSMessage{Type: msgType, RoomID: roomID, Payload: data}
	bytes, _ := json.Marshal(msg)
	h.broadcast <- &BroadcastMsg{roomID: roomID, message: bytes}
}

func (h *Hub) SendToUser(userID int64, msgType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	msg := WSMessage{Type: msgType, UserID: userID, Payload: data}
	bytes, _ := json.Marshal(msg)
	h.direct <- &DirectMsg{userID: userID, message: bytes}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	userIDStr := r.URL.Query().Get("user")
	if roomID == "" || userIDStr == "" {
		http.Error(w, "room and user required", http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256), hub: h, roomID: roomID, userID: userID}
	h.register <- client
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}
		var payload map[string]interface{}
		if msg.Payload != nil {
			json.Unmarshal(msg.Payload, &payload)
		}

		switch msg.Type {
		case MsgTypeVoiceSignal, MsgTypeChat:
			c.hub.broadcast <- &BroadcastMsg{roomID: c.roomID, message: message, exclude: c.userID}

		case "start_game":
			if c.hub.OnStartGame != nil {
				go c.hub.OnStartGame(c.roomID, c.userID)
			}

		case "night_action":
			if c.hub.OnNightAction != nil && payload != nil {
				role, _ := payload["role"].(string)
				targetIDFloat, _ := payload["target_id"].(float64)
				go c.hub.OnNightAction(c.roomID, role, c.userID, int64(targetIDFloat))
			}

		case "day_vote":
			if c.hub.OnDayVote != nil && payload != nil {
				targetIDFloat, _ := payload["target_id"].(float64)
				go c.hub.OnDayVote(c.roomID, c.userID, int64(targetIDFloat))
			}
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

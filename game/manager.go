package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"mafia-bot/game/roles"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Manager struct {
	bot       *tgbotapi.BotAPI
	hub       *Hub
	rooms     map[string]*Room
	chatRooms map[int64]string
	userRooms map[int64]string
	states    map[string]*GameState
	mu        sync.RWMutex
}

func NewManager(bot *tgbotapi.BotAPI, hub *Hub) *Manager {
	return &Manager{
		bot:       bot,
		hub:       hub,
		rooms:     make(map[string]*Room),
		chatRooms: make(map[int64]string),
		userRooms: make(map[int64]string),
		states:    make(map[string]*GameState),
	}
}

func (m *Manager) CreateRoom(chatID, ownerID int64, ownerName string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := NewRoom(chatID, ownerID, ownerName)
	m.rooms[room.ID] = room
	m.chatRooms[chatID] = room.ID
	m.userRooms[ownerID] = room.ID
	
	room.Players[ownerID].JoinOrder = 1
	
	go m.broadcastRoomInfo(room)
	
	return room
}

func (m *Manager) JoinRoom(roomID string, player *Player) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	room, exists := m.rooms[roomID]
	if !exists {
		return fmt.Errorf("xona topilmadi")
	}
	if err := room.AddPlayer(player); err != nil {
		return err
	}
	m.userRooms[player.TelegramID] = roomID

	go m.broadcastRoomInfo(room)
	
	m.hub.SendToUser(player.TelegramID, "join_success", map[string]interface{}{
		"room_id": roomID,
		"status": "joined",
	})
	
	return nil
}

func (m *Manager) GetRoomByUser(userID int64) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	roomID, exists := m.userRooms[userID]
	if !exists {
		return nil
	}
	return m.rooms[roomID]
}

func (m *Manager) GetRoom(roomID string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[roomID]
}

func (m *Manager) StartGame(roomID string) error {
	room := m.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("xona topilmadi")
	}
	players := room.GetPlayerList()
	if len(players) < 1 {
		return fmt.Errorf("kamida 1 o'yinchi kerak")
	}
	room.Status = RoomPlaying
	state := NewGameState(roomID)
	m.mu.Lock()
	m.states[roomID] = state
	m.mu.Unlock()
	m.assignRoles(room)
	go m.runGame(room, state)
	return nil
}

func (m *Manager) StartGameByOwner(roomID string, ownerID int64) error {
	room := m.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("xona topilmadi")
	}
	if room.OwnerID != ownerID {
		return fmt.Errorf("faqat xona egasi boshlashi mumkin")
	}
	return m.StartGame(roomID)
}

func (m *Manager) HandleWebConnect(roomID string, userID int64) {
	room := m.GetRoom(roomID)
	if room == nil {
		m.hub.SendError(userID, "xona topilmadi")
		return
	}
	m.sendRoomInfoToUser(room, userID)
	m.mu.RLock()
	state := m.states[roomID]
	m.mu.RUnlock()
	if state != nil {
		m.sendGameStateToUser(room, state, userID, false)
	}
}

func (m *Manager) SendRoomInfo(roomID string) {
	room := m.GetRoom(roomID)
	if room != nil {
		m.broadcastRoomInfo(room)
	}
}

func (m *Manager) assignRoles(room *Room) {
	players := room.GetPlayerList()
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})
	
	mafiaCount := len(players) / 4
	if mafiaCount == 0 {
		mafiaCount = 1
	}
	
	for i, p := range players {
		switch {
		case i < mafiaCount:
			if i == 0 && mafiaCount > 1 {
				p.Role = string(roles.RoleDon)
			} else {
				p.Role = string(roles.RoleMafia)
			}
		case i == mafiaCount:
			p.Role = string(roles.RoleDoctor)
		case i == mafiaCount+1:
			p.Role = string(roles.RoleSheriff)
		case i == mafiaCount+2 && len(players) >= 7:
			p.Role = string(roles.RoleBodyguard)
		default:
			p.Role = string(roles.RoleCivilian)
		}
		
		roleEmoji, roleDesc := getRoleInfo(roles.RoleName(p.Role))
		roleMsg := fmt.Sprintf("🎭 Sizning rolingiz: <b>%s %s</b>\n\n%s", roleEmoji, p.Role, roleDesc)
		msg := tgbotapi.NewMessage(p.TelegramID, roleMsg)
		msg.ParseMode = "HTML"
		m.bot.Send(msg)
		
		m.hub.SendToUser(p.TelegramID, MsgTypeRoleReveal, RoleRevealPayload{
			Role:        p.Role,
			Description: roleDesc,
			Emoji:       roleEmoji,
		})
	}
}

func getRoleInfo(role roles.RoleName) (string, string) {
	switch role {
	case roles.RoleMafia:
		return "😈", "Har kecha bir tinch aholini o'ldiring. Kunduz kun o'zingizni yashiring!"
	case roles.RoleDon:
		return "👑", "Mafia boshlig'i. Mafia guruhini boshqaring va Sheriff tekshiruvini chalg'iting!"
	case roles.RoleDoctor:
		return "👨‍⚕️", "Har kecha bir o'yinchini davolang. O'zingizni ham davolay olasiz!"
	case roles.RoleSheriff:
		return "🕵️", "Har kecha bir o'yinchining rolini aniqlang. Mafia yoki tinch aholi — bilib oling!"
	case roles.RoleBodyguard:
		return "🛡", "Har kecha bir o'yinchini himoya qiling. Hujum bo'lsa siz ham zarar ko'rasiz!"
	default:
		return "😇", "Mafiyachilarni toping va ovoz berish orqali chiqarib yuboring!"
	}
}

func (m *Manager) runGame(room *Room, state *GameState) {
	for {
		state.Round++
		state.Phase = PhaseNight

		m.sendToChat(room.ChatID, fmt.Sprintf("🌙 <b>TUN %d BOSHLANDI</b>\nBarcha jimlik saqlang...", state.Round))
		m.muteAllPlayers(room)
		m.sendNightActions(room, state)

		// Notify webapp
		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase: "night", Round: state.Round,
			Message: fmt.Sprintf("🌙 TUN %d — Mafia ish boshladi", state.Round),
			Timer:   60,
		})
		m.broadcastGameState(room, state)

		time.Sleep(60 * time.Second)

		// Process night results
		killedID := state.NightKillID
		savedID := state.DoctorSaveID
		guardedID := state.BodyguardID
		var killedName, killedRole string

		if killedID != 0 && killedID != savedID && killedID != guardedID {
			if p, ok := room.Players[killedID]; ok {
				killedName = p.Username
				killedRole = p.Role
				p.IsAlive = false
				m.askLastWords(p)
				m.hub.BroadcastToRoom(room.ID, MsgTypePlayerDied, map[string]interface{}{
					"player_id": killedID, "name": killedName, "role": killedRole,
				})
			}
		}

		// Day phase
		state.Phase = PhaseDay
		m.unmuteAlivePlayers(room)

		var dayMsg string
		if killedName != "" {
			dayMsg = fmt.Sprintf("☀️ <b>KUN %d BOSHLANDI</b>\n💀 Kecha <b>%s</b> o'ldirildi. Rol: %s\n\n90 soniya muhokama qiling!", state.Round, killedName, killedRole)
		} else {
			dayMsg = fmt.Sprintf("☀️ <b>KUN %d BOSHLANDI</b>\n✨ Kecha hech kim o'lmadi!\n\n90 soniya muhokama qiling!", state.Round)
		}
		m.sendToChat(room.ChatID, dayMsg)
		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase: "day", Round: state.Round, Message: dayMsg, Timer: 90,
		})
		m.broadcastGameState(room, state)

		if won, winner := m.checkWin(room); won {
			m.endGame(room, state, winner)
			return
		}

		time.Sleep(90 * time.Second)

		// Voting
		state.Phase = PhaseVoting
		state.ResetDay()
		m.sendVoting(room, state)
		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase: "voting", Round: state.Round,
			Message: "🗳 OVOZ BERISH — Kimni chiqaramiz?", Timer: 30,
		})
		m.broadcastGameState(room, state)

		time.Sleep(30 * time.Second)

		elimID, tie := m.processVotes(state)
		if tie || elimID == 0 {
			m.sendToChat(room.ChatID, "🤝 Ovozlar teng — hech kim chiqarilmadi.")
		} else {
			state.Phase = PhaseConfirm
			state.ConfirmID = elimID
			state.ConfirmVotes = make(map[int64]bool)
			if p, ok := room.Players[elimID]; ok {
				text := fmt.Sprintf("вљ–пёЏ <b>%s</b> eng ko'p ovoz oldi.\nRostdan shu odamni chiqaramizmi? (20 soniya)", p.Username)
				m.sendToChat(room.ChatID, text)
				m.hub.BroadcastToRoom(room.ID, MsgTypeConfirmVote, map[string]interface{}{
					"target_id": p.TelegramID,
					"name":      p.Username,
					"timer":     20,
				})
				m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
					Phase: "confirm", Round: state.Round, Message: text, Timer: 20,
				})
			}
			time.Sleep(20 * time.Second)
			yes, no := m.processConfirmVotes(state)
			if yes <= no {
				if p, ok := room.Players[elimID]; ok {
					m.sendToChat(room.ChatID, fmt.Sprintf("вњ… <b>%s</b> chiqarilmadi. Ha: %d, Yo'q: %d", p.Username, yes, no))
				}
				state.ResetNight()
				continue
			}
			if p, ok := room.Players[elimID]; ok {
				m.sendToChat(room.ChatID, fmt.Sprintf("⚖️ <b>%s</b> chiqarib yuborildi. Rol: %s", p.Username, p.Role))
				p.IsAlive = false
				m.bot.Send(tgbotapi.NewMessage(p.TelegramID, "😢 Siz o'yindan chiqarildingiz. Kuzatishda davom eting!"))
				m.hub.BroadcastToRoom(room.ID, MsgTypePlayerDied, map[string]interface{}{
					"player_id": elimID, "name": p.Username, "role": p.Role, "voted_out": true,
				})
			}
		}

		if won, winner := m.checkWin(room); won {
			m.endGame(room, state, winner)
			return
		}

		state.ResetNight()
	}
}

func (m *Manager) broadcastGameState(room *Room, state *GameState) {
	players := []PlayerInfo{}
	for _, p := range room.Players {
		players = append(players, PlayerInfo{
			ID: p.TelegramID, Name: p.Username, IsAlive: p.IsAlive,
		})
	}
	m.hub.BroadcastToRoom(room.ID, MsgTypeGameState, GameStatePayload{
		Phase: string(state.Phase), Round: state.Round, Players: players,
	})
}

func (m *Manager) broadcastRoomInfo(room *Room) {
	players := []PlayerInfo{}
	for _, p := range room.GetPlayerList() {
		players = append(players, PlayerInfo{
			ID: p.TelegramID, 
			Name: p.Username, 
			IsAlive: true,
			JoinOrder: p.JoinOrder,
		})
	}
	m.hub.BroadcastToRoom(room.ID, MsgTypeRoomInfo, map[string]interface{}{
		"room_id": room.ID, 
		"owner_id": room.OwnerID,
		"players": players,
		"count": len(players), 
		"max": room.MaxPlayer, 
		"status": string(room.Status),
	})
}

func (m *Manager) sendNightActions(room *Room, state *GameState) {
	for _, p := range room.Players {
		if !p.IsAlive {
			continue
		}
		alivePlayers := room.AlivePlayers()
		var keyboard [][]tgbotapi.InlineKeyboardButton
		for _, target := range alivePlayers {
			if target.TelegramID == p.TelegramID {
				continue
			}
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"👤 "+target.Username,
					fmt.Sprintf("%s_%s_%d", p.Role, room.ID, target.TelegramID),
				),
			)
			keyboard = append(keyboard, row)
		}
		if len(keyboard) == 0 {
			continue
		}
		var text string
		switch roles.RoleName(p.Role) {
		case roles.RoleMafia:
			text = "😈 Kimni o'ldirasiz?"
		case roles.RoleDoctor:
			text = "👨‍⚕️ Kimni davolaysiz?"
		case roles.RoleDetective:
			text = "🕵️ Kimning rolini tekshirasiz?"
		default:
			continue
		}
		msg := tgbotapi.NewMessage(p.TelegramID, text)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
		m.bot.Send(msg)
	}
}

func (m *Manager) sendVoting(room *Room, state *GameState) {
	alivePlayers := room.AlivePlayers()
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, p := range alivePlayers {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"👤 "+p.Username,
				fmt.Sprintf("dayVote_%s_%d", room.ID, p.TelegramID),
			),
		)
		keyboard = append(keyboard, row)
	}
	msg := tgbotapi.NewMessage(room.ChatID, "🗳 <b>OVOZ BERISH</b>\nKimni chiqarib yuboramiz? (30 soniya)")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	m.bot.Send(msg)
}

func (m *Manager) processVotes(state *GameState) (int64, bool) {
	counts := make(map[int64]int)
	for _, target := range state.Votes {
		counts[target]++
	}
	var winner int64
	maxVotes := 0
	tie := false
	for target, count := range counts {
		if count > maxVotes {
			maxVotes = count
			winner = target
			tie = false
		} else if count == maxVotes {
			tie = true
		}
	}
	if tie {
		return 0, true
	}
	return winner, false
}

func (m *Manager) checkWin(room *Room) (bool, string) {
	mafiaCount := 0
	civCount := 0
	for _, p := range room.Players {
		if !p.IsAlive {
			continue
		}
		if roles.IsMafiaRole(roles.RoleName(p.Role)) {
			mafiaCount++
		} else {
			civCount++
		}
	}
	if mafiaCount == 0 {
		return true, "civilian"
	}
	if mafiaCount >= civCount {
		return true, "mafia"
	}
	return false, ""
}

func (m *Manager) endGame(room *Room, state *GameState, winner string) {
	room.Status = RoomFinished
	var winMsg, winTitle string
	if winner == "civilian" {
		winTitle = "🎉 TINCH AHOLI YUTDI!"
		winMsg = "🎉 <b>TINCH AHOLI YUTDI!</b>\nBarcha mafiyachilar yo'q qilindi!"
	} else {
		winTitle = "😈 MAFIA YUTDI!"
		winMsg = "😈 <b>MAFIA YUTDI!</b>"
	}
	m.sendToChat(room.ChatID, winMsg)
	m.hub.BroadcastToRoom(room.ID, MsgTypeGameEnd, map[string]interface{}{
		"winner": winner, "title": winTitle,
	})
	m.unmuteAllPlayers(room)
	m.mu.Lock()
	delete(m.states, room.ID)
	for _, p := range room.Players {
		delete(m.userRooms, p.TelegramID)
	}
	delete(m.chatRooms, room.ChatID)
	delete(m.rooms, room.ID)
	m.mu.Unlock()
}

func (m *Manager) HandleNightAction(roomID, role string, voterID, targetID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exists := m.states[roomID]
	if !exists || state.Phase != PhaseNight {
		return
	}
	switch roles.RoleName(role) {
	case roles.RoleMafia:
		state.MafiaVotes[voterID] = targetID
		counts := make(map[int64]int)
		for _, t := range state.MafiaVotes {
			counts[t]++
		}
		maxV := 0
		for t, c := range counts {
			if c > maxV {
				maxV = c
				state.NightKillID = t
			}
		}
	case roles.RoleDoctor:
		state.DoctorSaveID = targetID
	case roles.RoleDetective:
		state.DetectiveCh = targetID
		room := m.rooms[roomID]
		if room != nil {
			if p, ok := room.Players[targetID]; ok {
				result := fmt.Sprintf("🕵️ Tekshiruv natijasi: <b>%s</b> — %s", p.Username, p.Role)
				msg := tgbotapi.NewMessage(voterID, result)
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
			}
		}
	}
}

func (m *Manager) HandleDayVote(roomID string, voterID, targetID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exists := m.states[roomID]
	if !exists || state.Phase != PhaseVoting {
		return
	}
	state.Votes[voterID] = targetID
}

func (m *Manager) muteAllPlayers(room *Room) {
	for _, p := range room.Players {
		if p.IsAlive {
			m.restrictPlayer(room.ChatID, p.TelegramID, false)
		}
	}
}
func (m *Manager) unmuteAlivePlayers(room *Room) {
	for _, p := range room.Players {
		if p.IsAlive {
			m.restrictPlayer(room.ChatID, p.TelegramID, true)
		}
	}
}
func (m *Manager) unmuteAllPlayers(room *Room) {
	for _, p := range room.Players {
		m.restrictPlayer(room.ChatID, p.TelegramID, true)
	}
}
func (m *Manager) restrictPlayer(chatID, userID int64, canSend bool) {
	config := tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: chatID, UserID: userID},
		Permissions:      &tgbotapi.ChatPermissions{CanSendMessages: canSend},
	}
	m.bot.Request(config)
}
func (m *Manager) sendToChat(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	m.bot.Send(msg)
}

func (m *Manager) sendRoomInfoToUser(room *Room, userID int64) {
	players := []PlayerInfo{}
	for _, p := range room.GetPlayerList() {
		players = append(players, PlayerInfo{
			ID: p.TelegramID, 
			Name: p.Username, 
			IsAlive: true,
			JoinOrder: p.JoinOrder,
		})
	}
	m.hub.SendToUser(userID, MsgTypeRoomInfo, map[string]interface{}{
		"room_id": room.ID, "players": players,
		"count": len(players), "max": room.MaxPlayer, "status": string(room.Status),
		"owner_id": room.OwnerID,
	})
}

func (m *Manager) sendGameStateToUser(room *Room, state *GameState, userID int64, revealRoles bool) {
	players := []PlayerInfo{}
	for _, p := range room.Players {
		info := PlayerInfo{ID: p.TelegramID, Name: p.Username, IsAlive: p.IsAlive}
		if revealRoles {
			info.Role = p.Role
		}
		players = append(players, info)
	}
	m.hub.SendToUser(userID, MsgTypeGameState, GameStatePayload{
		Phase: string(state.Phase), Round: state.Round, Players: players,
	})
}

func (m *Manager) HandleConfirmVote(roomID string, userID int64, confirm bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exists := m.states[roomID]
	if !exists || state.Phase != PhaseConfirm {
		return
	}
	state.ConfirmVotes[userID] = confirm
}

func (m *Manager) askLastWords(p *Player) {
	msg := tgbotapi.NewMessage(p.TelegramID, "⚰️ Siz o'ldingiz. So'ng-so'zlaringizni kiriting (20 soniya):")
	msg.ParseMode = "HTML"
	m.bot.Send(msg)
	// Note: In a real implementation, we would wait for the user's response.
	// For simplicity, we just send the prompt and the user can type in the chat.
}

func (m *Manager) processConfirmVotes(state *GameState) (int64, int64) {
	var yes, no int64
	for _, vote := range state.ConfirmVotes {
		if vote {
			yes++
		} else {
			no++
		}
	}
	return yes, no
}

// CreateRoomWeb — WebApp orqali xona yaratish (chatID=0)
func (m *Manager) CreateRoomWeb(ownerID int64, ownerName string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := NewRoom(0, ownerID, ownerName)
	m.rooms[room.ID] = room
	m.userRooms[ownerID] = room.ID
	room.Players[ownerID].JoinOrder = 1
	return room
}

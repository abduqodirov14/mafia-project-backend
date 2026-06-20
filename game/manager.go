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
	bot         *tgbotapi.BotAPI
	hub         *Hub
	rooms       map[string]*Room
	chatRooms   map[int64]string
	userRooms   map[int64]string
	states      map[string]*GameState
	mu          sync.RWMutex
	adminChatID int64
	webAppURL   string
	botUsername string
}

func NewManager(bot *tgbotapi.BotAPI, hub *Hub, adminChatID int64, webAppURL, botUsername string) *Manager {
	m := &Manager{
		bot:         bot,
		hub:         hub,
		rooms:       make(map[string]*Room),
		chatRooms:   make(map[int64]string),
		userRooms:   make(map[int64]string),
		states:      make(map[string]*GameState),
		adminChatID: adminChatID,
		webAppURL:   webAppURL,
		botUsername: botUsername,
	}

	hub.OnConnect = m.HandleWebConnect
	hub.OnStartGame = m.StartGameByOwner
	hub.OnNightAction = m.HandleNightAction
	hub.OnDayVote = m.HandleDayVote

	return m
}

// ─── ROOM MANAGEMENT ───

func (m *Manager) CreateRoom(chatID, ownerID int64, ownerName string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := NewRoom(chatID, ownerID, ownerName)
	m.rooms[room.ID] = room
	if chatID != 0 {
		m.chatRooms[chatID] = room.ID
	}
	m.userRooms[ownerID] = room.ID
	m.notifyAdmin(fmt.Sprintf("🏠 Yangi xona\nEga: <b>%s</b> | ID: <code>%s</code>", ownerName, room.ID))
	return room
}

func (m *Manager) CreateRoomWeb(ownerID int64, ownerName string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := NewRoom(0, ownerID, ownerName)
	m.rooms[room.ID] = room
	m.userRooms[ownerID] = room.ID
	return room
}

func (m *Manager) GetRoom(roomID string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[roomID]
}

func (m *Manager) GetRoomByChat(chatID int64) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, ok := m.chatRooms[chatID]
	if !ok {
		return nil
	}
	return m.rooms[id]
}

func (m *Manager) GetRoomByUser(userID int64) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, ok := m.userRooms[userID]
	if !ok {
		return nil
	}
	return m.rooms[id]
}

func (m *Manager) JoinRoom(roomID string, player *Player) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return fmt.Errorf("xona topilmadi")
	}
	if err := room.AddPlayer(player); err != nil {
		return err
	}
	m.userRooms[player.TelegramID] = roomID
	go m.broadcastRoomInfo(room)
	return nil
}

func (m *Manager) LeaveRoom(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	roomID, ok := m.userRooms[userID]
	if !ok {
		return
	}
	if room, ok := m.rooms[roomID]; ok {
		room.RemovePlayer(userID)
		go m.broadcastRoomInfo(room)
	}
	delete(m.userRooms, userID)
}

// ─── GAME LIFECYCLE ───

func (m *Manager) StartGame(roomID string) error {
	room := m.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("xona topilmadi")
	}

	players := room.GetPlayerList()
	if len(players) < 4 {
		return fmt.Errorf("kamida 4 o'yinchi kerak (hozir %d ta)", len(players))
	}

	room.SetStatus(RoomPlaying)

	state := NewGameState(roomID, room.ChatID)
	m.mu.Lock()
	m.states[roomID] = state
	m.mu.Unlock()

	m.assignRoles(room, state)
	m.notifyAdmin(fmt.Sprintf("🎮 O'yin boshlandi!\nXona: <code>%s</code> | O'yinchilar: %d", roomID, len(players)))

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

func (m *Manager) ForceStopGame(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return
	}

	m.unmuteAll(room)
	delete(m.states, roomID)
	for _, p := range room.Players {
		delete(m.userRooms, p.TelegramID)
	}
	delete(m.chatRooms, room.ChatID)
	delete(m.rooms, roomID)
}

func (m *Manager) GetState(roomID string) *GameState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.states[roomID]
}

// ─── ROLE ASSIGNMENT ───

func (m *Manager) assignRoles(room *Room, state *GameState) {
	players := room.GetPlayerList()
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})

	assigned := roles.AssignRoles(len(players))
	for i, p := range players {
		p.Role = assigned[i]
		if p.Role == roles.RoleKomissar {
			state.KomissarID = p.TelegramID
		}
	}

	for _, p := range players {
		roleInfo := roles.Get(p.Role)
		msg := tgbotapi.NewMessage(p.TelegramID, RolePrivateMsg(p))
		msg.ParseMode = "HTML"

		if m.webAppURL != "" && !isBot(p.TelegramID) {
			webBtn := makeWebAppButton("🎮 O'yinga kirish (WebApp)", m.webAppURL+"?room="+room.ID)
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(webBtn),
			)
		}
		if !isBot(p.TelegramID) {
			go m.bot.Send(msg)
		}

		m.hub.SendToUser(p.TelegramID, MsgTypeRoleReveal, RoleRevealPayload{
			Role:        string(p.Role),
			Description: roleInfo.Description,
			Emoji:       roleInfo.Emoji,
		})
	}

	m.sendRolesSummary(room)
}

func (m *Manager) sendRolesSummary(room *Room) {
	players := room.GetPlayerList()
	roleCount := make(map[roles.RoleName]int)
	for _, p := range players {
		roleCount[p.Role]++
	}

	text := "<b>Rol taqsimlandi!</b>\n\n<b>O'yinchilar:</b>\n"
	for i, p := range players {
		text += fmt.Sprintf("%d. @%s\n", i+1, p.Username)
	}
	text += "\n<b>Ulardan kimlar:</b>\n"

	var parts []string
	for role, count := range roleCount {
		info := roles.Get(role)
		if count > 1 {
			parts = append(parts, fmt.Sprintf("%s %s - %d", info.Emoji, string(role), count))
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", info.Emoji, string(role)))
		}
	}
	text += joinWithComma(parts)
	text += fmt.Sprintf("\nJami: %d kishi.", len(players))
	text += "\n\n<i>Tunda bo'lgan hodisalarni muhokama qilishning ayni vaqti...</i>"

	m.sendToChat(room.ChatID, text)
}

// ─── MAIN GAME LOOP ───

func (m *Manager) runGame(room *Room, state *GameState) {
	for {
		state.Round++
		state.Phase = PhaseNight

		m.sendToChat(room.ChatID, NightStartMsg(state.Round))
		time.Sleep(1 * time.Second)
		m.muteAll(room)
		m.sendNightActions(room, state)
		go m.runBotNightActions(room, state)

		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase:   "night",
			Round:   state.Round,
			Message: fmt.Sprintf("🌙 TUN %d boshlandi", state.Round),
			Timer:   60,
		})
		m.broadcastGameState(room, state)

		time.Sleep(3 * time.Second)
		m.announceNightActions(room, state)
		time.Sleep(57 * time.Second)

		m.processNight(room, state)

		if won, winner := m.checkWin(room); won {
			m.endGame(room, state, winner)
			return
		}

		// DAY PHASE
		state.Phase = PhaseDay
		m.unmuteAlive(room)

		dayMsg := DayStartMsg(state.Round)
		m.sendToChat(room.ChatID, dayMsg)
		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase: "day", Round: state.Round, Message: dayMsg, Timer: 90,
		})
		m.broadcastGameState(room, state)

		time.Sleep(90 * time.Second)

		// VOTING PHASE
		state.Phase = PhaseVoting
		state.ResetVoting()
		m.startVoting(room, state)
		go m.runBotVoting(room, state)

		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase: "voting", Round: state.Round,
			Message: "🗳 Ovoz berish vaqti! (60 soniya)", Timer: 60,
		})
		m.broadcastGameState(room, state)

		time.Sleep(60 * time.Second)

		m.processVoting(room, state)

		if won, winner := m.checkWin(room); won {
			m.endGame(room, state, winner)
			return
		}

		state.ResetNight()
	}
}

// ─── NIGHT PHASE ───

func (m *Manager) announceNightActions(room *Room, state *GameState) {
	var announcements []string

	for _, p := range room.AlivePlayers() {
		switch p.Role {
		case roles.RoleDoctor:
			announcements = append(announcements, "👨‍⚕️ <b>Shifokor</b> tungi navbatchilikka ketdi...")
		case roles.RoleKomissar:
			if state.Round > 1 {
				announcements = append(announcements, "🕵🏼 <b>Komissar Katani</b> yovuzlarni qidirishga ketdi...")
			}
		case roles.RoleDon, roles.RoleMafia:
			if p.Role == roles.RoleDon {
				announcements = append(announcements, "🤵🏻 <b>Mafia</b> qurbonini tanladi...")
			}
		case roles.RoleMashuqa:
			announcements = append(announcements, "💃 <b>Ma'shuqa</b> mehmonxonaga ketdi...")
		case roles.RoleDaydi:
			announcements = append(announcements, "🧙🏻‍♂️ <b>Daydi</b> tungi sayohatga chiqdi...")
		case roles.RoleManyak:
			announcements = append(announcements, "🔪 <b>Manyak</b> tungi ovga chiqdi...")
		}
	}

	for i, ann := range announcements {
		time.Sleep(time.Duration(2+i) * time.Second)
		m.sendToChat(room.ChatID, ann)
	}
}

func (m *Manager) processNight(room *Room, state *GameState) {
	night := state.Night

	// Block target if mashuqa acted
	if night.MashuqaTargetID != 0 {
		if p, ok := room.PlayerByID(night.MashuqaTargetID); ok {
			p.IsBlocked = true
		}
	}

	// Maniac kill
	if night.ManyakTargetID != 0 && night.ManyakTargetID != night.DoctorTargetID {
		if p, ok := room.PlayerByID(night.ManyakTargetID); ok && p.IsAlive {
			p.IsAlive = false
			m.sendToChat(room.ChatID, DeathMessage(p, roles.RoleManyak))
			m.broadcastPlayerDied(room, p)
			if p.Role == roles.RoleKamikaze {
				m.handleKamikazeDeath(room, state, p, false)
			}
		}
	}

	// Mafia kill
	killedID := night.MafiaTargetID
	if killedID != 0 && killedID != night.DoctorTargetID {
		if p, ok := room.PlayerByID(killedID); ok && p.IsAlive {
			killerRole := m.findAliveMafiaRole(room)
			p.IsAlive = false
			m.sendToChat(room.ChatID, DeathMessage(p, killerRole))
			m.broadcastPlayerDied(room, p)

			if p.Role == roles.RoleKamikaze {
				m.handleKamikazeDeath(room, state, p, true)
			}
			if p.Role == roles.RoleDon {
				m.handleDonDeath(room)
			}
			if p.Role == roles.RoleKomissar {
				m.handleKomissarDeath(room, state)
			}
		}
	} else if killedID != 0 && killedID == night.DoctorTargetID {
		m.sendToChat(room.ChatID, SavedMessage(m.getPlayerName(room, killedID)))
	} else {
		m.sendToChat(room.ChatID, NobodyDiedMsg())
	}

	// Daydi result
	if night.DaydiTargetID != 0 {
		if targetPlayer, ok := room.PlayerByID(night.DaydiTargetID); ok {
			visitors := m.getNightVisitors(room, night.DaydiTargetID)
			result := DaydiResultMsg(targetPlayer, visitors)
			if daydi := m.findPlayerByRole(room, roles.RoleDaydi); daydi != nil && !isBot(daydi.TelegramID) {
				msg := tgbotapi.NewMessage(daydi.TelegramID, result)
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
				m.hub.SendToUser(daydi.TelegramID, MsgTypeNightResult, map[string]string{"result": result})
			}
		}
	}

	// Komissar result (round 2+)
	if night.KomissarTargetID != 0 && state.Round > 1 {
		if targetPlayer, ok := room.PlayerByID(night.KomissarTargetID); ok {
			isMafia := roles.IsMafia(targetPlayer.Role)
			if targetPlayer.Role == roles.RoleAdvokat {
				isMafia = false
			}
			result := KomissarResultMsg(targetPlayer, isMafia)
			if komissar, ok := room.PlayerByID(state.KomissarID); ok && !isBot(komissar.TelegramID) {
				msg := tgbotapi.NewMessage(komissar.TelegramID, result)
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
				m.hub.SendToUser(komissar.TelegramID, MsgTypeSheriffResult, map[string]interface{}{
					"target": targetPlayer.Username, "is_mafia": isMafia, "result": result,
				})
			}
		}
	}

	// Reset blocked state
	for _, p := range room.Players {
		p.IsBlocked = false
	}
}

// ─── KAMIKAZE / DON / KOMISSAR DEATH HANDLERS ───

func (m *Manager) handleKamikazeDeath(room *Room, state *GameState, kamikaze *Player, wasShot bool) {
	if wasShot {
		for voterID := range state.Night.MafiaVotes {
			if voter, ok := room.PlayerByID(voterID); ok && voter.IsAlive {
				voter.IsAlive = false
				m.sendToChat(room.ChatID, KamikazeDeathMessage(kamikaze, voter))
				return
			}
		}
	} else {
		alive := room.AlivePlayers()
		if len(alive) > 0 {
			target := alive[rand.Intn(len(alive))]
			target.IsAlive = false
			m.sendToChat(room.ChatID, fmt.Sprintf(
				"💣 <b>Kamikaze</b> osilganda portlab, <b>%s</b> ni ham olib ketdi!", target.Username))
		}
	}
}

func (m *Manager) handleDonDeath(room *Room) {
	for _, p := range room.Players {
		if p.IsAlive && p.Role == roles.RoleMafia {
			p.Role = roles.RoleDon
			if !isBot(p.TelegramID) {
				msg := tgbotapi.NewMessage(p.TelegramID, "👑 Don vafot etdi. Siz yangi Don bo'ldingiz!")
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
			}
			m.sendToChat(room.ChatID, DonInheritMessage(p))
			return
		}
	}
}

func (m *Manager) handleKomissarDeath(room *Room, state *GameState) {
	for _, p := range room.Players {
		if p.IsAlive && p.Role == roles.RoleSerjant {
			p.Role = roles.RoleKomissar
			state.KomissarID = p.TelegramID
			if !isBot(p.TelegramID) {
				msg := tgbotapi.NewMessage(p.TelegramID, "🕵️ Komissar vafot etdi. Siz Komissar bo'ldingiz!")
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
			}
			m.sendToChat(room.ChatID, KomissarInheritMessage(p))
			return
		}
	}
}

// ─── NIGHT ACTION ROUTING ───

func (m *Manager) HandleNightAction(roomID, roleName string, voterID, targetID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.states[roomID]
	if !ok || state.Phase != PhaseNight {
		return
	}
	room := m.rooms[roomID]
	if room == nil {
		return
	}

	player, ok := room.PlayerByID(voterID)
	if !ok || !player.IsAlive {
		return
	}
	if player.IsBlocked {
		if !isBot(voterID) {
			msg := tgbotapi.NewMessage(voterID, "🚫 Siz bu kecha harakat qila olmaysiz (Mashuqa blokadi)")
			m.bot.Send(msg)
		}
		return
	}

	night := state.Night
	role := roles.RoleName(roleName)

	switch role {
	case roles.RoleMafia, roles.RoleDon:
		night.MafiaVotes[voterID] = targetID
		night.MafiaTargetID = m.computeMafiaMajorityTarget(night.MafiaVotes)
		m.sendConfirmation(voterID, room, targetID, "✅ Tanlov qabul qilindi")

	case roles.RoleDoctor:
		night.DoctorTargetID = targetID
		m.sendConfirmation(voterID, room, targetID, "💊 Davolanmoqda...")

	case roles.RoleKomissar, roles.RoleSerjant:
		if state.Round == 1 {
			return
		}
		night.KomissarTargetID = targetID

	case roles.RoleMashuqa:
		night.MashuqaTargetID = targetID
		if target, ok := room.PlayerByID(targetID); ok {
			target.IsBlocked = true
		}
		m.sendConfirmation(voterID, room, targetID, "💃 bilan tunni o'tkazmoqdasiz...")

	case roles.RoleDaydi:
		night.DaydiTargetID = targetID

	case roles.RoleManyak:
		night.ManyakTargetID = targetID

	case roles.RoleTentak:
		night.TentakTargetID = targetID
	}
}

func (m *Manager) computeMafiaMajorityTarget(votes map[int64]int64) int64 {
	counts := make(map[int64]int)
	for _, t := range votes {
		counts[t]++
	}
	var winner int64
	maxVotes := 0
	for id, cnt := range counts {
		if cnt > maxVotes {
			maxVotes = cnt
			winner = id
		}
	}
	return winner
}

func (m *Manager) sendConfirmation(voterID int64, room *Room, targetID int64, prefix string) {
	if isBot(voterID) {
		return
	}
	if target, ok := room.PlayerByID(targetID); ok {
		msg := tgbotapi.NewMessage(voterID, fmt.Sprintf("%s: <b>%s</b>", prefix, target.Username))
		msg.ParseMode = "HTML"
		m.bot.Send(msg)
	}
}

// ─── VOTING ───

func (m *Manager) startVoting(room *Room, state *GameState) {
	alive := room.AlivePlayers()
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, p := range alive {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"👤 "+p.Username,
				fmt.Sprintf("vote_%s_%d", room.ID, p.TelegramID),
			),
		)
		keyboard = append(keyboard, row)
	}
	msg := tgbotapi.NewMessage(room.ChatID, VotingMsg(alive))
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	m.bot.Send(msg)
}

func (m *Manager) HandleDayVote(roomID string, voterID, targetID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.states[roomID]
	if !ok || state.Phase != PhaseVoting {
		return
	}
	state.Voting.Votes[voterID] = targetID

	if !isBot(voterID) {
		if room := m.rooms[roomID]; room != nil {
			if target, ok := room.PlayerByID(targetID); ok {
				m.bot.Request(tgbotapi.NewCallback("", fmt.Sprintf("✅ %s uchun ovoz berildi", target.Username)))
			}
		}
	}
}

func (m *Manager) processVoting(room *Room, state *GameState) {
	votes := state.Voting.Votes
	if len(votes) == 0 {
		m.sendToChat(room.ChatID, NoVoteMessage())
		return
	}

	counts := make(map[int64]int)
	for _, t := range votes {
		counts[t]++
	}

	var winnerID int64
	maxVotes, tie := 0, false
	for id, cnt := range counts {
		if cnt > maxVotes {
			maxVotes = cnt
			winnerID = id
			tie = false
		} else if cnt == maxVotes {
			tie = true
		}
	}

	if tie || winnerID == 0 {
		m.sendToChat(room.ChatID, NoVoteMessage())
		return
	}

	player, ok := room.PlayerByID(winnerID)
	if !ok {
		return
	}

	switch player.Role {
	case roles.RoleSuidsid:
		m.sendToChat(room.ChatID, fmt.Sprintf(
			"🧌 <b>SUIDSID %s chiqarib yuborildi!</b>\n\nU o'yinni YUTDI!", player.Username))
		player.IsAlive = false
		m.broadcastPlayerDied(room, player)
		m.endGame(room, state, "suidsid")
		return

	case roles.RoleKamikaze:
		player.IsAlive = false
		m.sendToChat(room.ChatID, VoteOutMessage(player))
		m.handleKamikazeDeath(room, state, player, false)
		return
	}

	// Normal vote-out
	player.IsAlive = false
	m.sendToChat(room.ChatID, VoteOutMessage(player))
	if !isBot(winnerID) {
		m.bot.Send(tgbotapi.NewMessage(winnerID, "😢 Siz o'yindan chiqarildingiz. Kuzatishda davom eting!"))
	}
	m.broadcastPlayerDied(room, player)

	if player.Role == roles.RoleKomissar {
		m.handleKomissarDeath(room, state)
	}
	if player.Role == roles.RoleDon {
		m.handleDonDeath(room)
	}
}

// ─── BOT AI ───

func (m *Manager) runBotNightActions(room *Room, state *GameState) {
	for _, p := range room.AlivePlayers() {
		if !isBot(p.TelegramID) || !roles.HasNightAction(p.Role) {
			continue
		}
		delay := time.Duration(3000+rand.Intn(8000)) * time.Millisecond
		go func(bot *Player) {
			time.Sleep(delay)
			targets := m.getTargetsForRole(room, bot)
			if len(targets) > 0 {
				target := targets[rand.Intn(len(targets))]
				m.HandleNightAction(room.ID, string(bot.Role), bot.TelegramID, target.TelegramID)
			}
		}(p)
	}
}

func (m *Manager) runBotVoting(room *Room, state *GameState) {
	for _, p := range room.AlivePlayers() {
		if !isBot(p.TelegramID) {
			continue
		}
		go func(bot *Player) {
			time.Sleep(time.Duration(5000+rand.Intn(10000)) * time.Millisecond)
			candidates := filterOut(room.AlivePlayers(), bot.TelegramID)
			if len(candidates) > 0 {
				m.HandleDayVote(room.ID, bot.TelegramID, candidates[rand.Intn(len(candidates))].TelegramID)
			}
		}(p)
	}
}

// ─── WIN CHECK ───

func (m *Manager) checkWin(room *Room) (bool, string) {
	mafiaCount, townCount, manyakAlive := 0, 0, false

	for _, p := range room.Players {
		if !p.IsAlive {
			continue
		}
		switch {
		case roles.IsMafia(p.Role):
			mafiaCount++
		case p.Role == roles.RoleManyak:
			manyakAlive = true
		default:
			townCount++
		}
	}

	total := mafiaCount + townCount
	if manyakAlive {
		total++
	}

	switch {
	case mafiaCount == 0 && !manyakAlive:
		return true, "town"
	case mafiaCount >= townCount+boolToInt(manyakAlive):
		return true, "mafia"
	case manyakAlive && total <= 2:
		return true, "manyak"
	default:
		return false, ""
	}
}

func (m *Manager) endGame(room *Room, state *GameState, winner string) {
	room.SetStatus(RoomFinished)

	players := room.GetPlayerList()
	m.sendToChat(room.ChatID, WinMessage(winner, players))

	m.hub.BroadcastToRoom(room.ID, MsgTypeGameEnd, map[string]interface{}{
		"winner": winner,
		"title":  m.getWinTitle(winner),
	})

	m.notifyAdmin(fmt.Sprintf("🏁 O'yin tugadi\nXona: <code>%s</code> | G'olib: %s", room.ID, winner))
	m.unmuteAll(room)

	m.mu.Lock()
	delete(m.states, room.ID)
	for _, p := range room.Players {
		delete(m.userRooms, p.TelegramID)
	}
	delete(m.chatRooms, room.ChatID)
	delete(m.rooms, room.ID)
	m.mu.Unlock()
}

func (m *Manager) getWinTitle(winner string) string {
	switch winner {
	case "town":
		return "🎉 TINCH AHOLI G'ALABA QOZONDI!"
	case "mafia":
		return "😈 MAFIA G'ALABA QOZONDI!"
	case "manyak":
		return "🔪 MANYAK G'ALABA QOZONDI!"
	case "suidsid":
		return "🧌 SUIDSID G'ALABA QOZONDI!"
	default:
		return "🏁 O'YIN TUGADI"
	}
}

// ─── WEBSOCKET HANDLERS ───

func (m *Manager) HandleWebConnect(roomID string, userID int64) {
	m.mu.RLock()
	room := m.rooms[roomID]
	state := m.states[roomID]
	m.mu.RUnlock()

	if room == nil {
		return
	}
	m.sendRoomInfoToUser(room, userID)
	if state != nil {
		m.sendGameStateToUser(room, state, userID)
	}
}

func (m *Manager) sendRoomInfoToUser(room *Room, userID int64) {
	players := m.buildPlayerInfoList(room)
	m.hub.SendToUser(userID, MsgTypeRoomInfo, map[string]interface{}{
		"room_id":  room.ID,
		"owner_id": room.OwnerID,
		"players":  players,
		"count":    room.PlayerCount(),
		"max":      room.MaxPlayers,
		"status":   string(room.Status),
	})
}

func (m *Manager) sendGameStateToUser(room *Room, state *GameState, userID int64) {
	players := m.buildPlayerInfoList(room)
	m.hub.SendToUser(userID, MsgTypeGameState, GameStatePayload{
		Phase:   string(state.Phase),
		Round:   state.Round,
		Players: players,
	})
}

func (m *Manager) broadcastRoomInfo(room *Room) {
	players := m.buildPlayerInfoList(room)
	m.hub.BroadcastToRoom(room.ID, MsgTypeRoomInfo, map[string]interface{}{
		"room_id":  room.ID,
		"owner_id": room.OwnerID,
		"players":  players,
		"count":    room.PlayerCount(),
		"max":      room.MaxPlayers,
		"status":   string(room.Status),
	})
}

func (m *Manager) broadcastGameState(room *Room, state *GameState) {
	players := m.buildPlayerInfoList(room)
	m.hub.BroadcastToRoom(room.ID, MsgTypeGameState, GameStatePayload{
		Phase:   string(state.Phase),
		Round:   state.Round,
		Players: players,
	})
}

func (m *Manager) broadcastPlayerDied(room *Room, player *Player) {
	m.hub.BroadcastToRoom(room.ID, MsgTypePlayerDied, map[string]interface{}{
		"player_id": player.TelegramID,
		"name":      player.Username,
		"role":      string(player.Role),
	})
}

// ─── HELPERS ───

func (m *Manager) buildPlayerInfoList(room *Room) []PlayerInfo {
	players := room.GetPlayerList()
	infos := make([]PlayerInfo, 0, len(players))
	for _, p := range players {
		info := roles.Get(p.Role)
		infos = append(infos, PlayerInfo{
			ID:      p.TelegramID,
			Name:    p.Username,
			IsAlive: p.IsAlive,
			Emoji:   info.Emoji,
		})
	}
	return infos
}

func (m *Manager) findAliveMafiaRole(room *Room) roles.RoleName {
	for _, p := range room.Players {
		if p.IsAlive && (p.Role == roles.RoleDon || p.Role == roles.RoleMafia) {
			return p.Role
		}
	}
	return roles.RoleMafia
}

func (m *Manager) findPlayerByRole(room *Room, role roles.RoleName) *Player {
	for _, p := range room.Players {
		if p.IsAlive && p.Role == role {
			return p
		}
	}
	return nil
}

func (m *Manager) getPlayerName(room *Room, id int64) string {
	if p, ok := room.PlayerByID(id); ok {
		return p.Username
	}
	return "Noma'lum"
}

func (m *Manager) getTargetsForRole(room *Room, player *Player) []*Player {
	var targets []*Player
	for _, p := range room.AlivePlayers() {
		if p.TelegramID == player.TelegramID {
			continue
		}
		if (player.Role == roles.RoleMafia || player.Role == roles.RoleDon) && roles.IsMafia(p.Role) {
			continue
		}
		targets = append(targets, p)
	}
	return targets
}

func (m *Manager) getNightVisitors(room *Room, targetID int64) []*Player {
	var visitors []*Player
	night := m.getRoomNight(room)
	if night == nil {
		return visitors
	}

	if night.MafiaTargetID == targetID {
		for voterID := range night.MafiaVotes {
			if p, ok := room.PlayerByID(voterID); ok {
				visitors = append(visitors, p)
			}
		}
	}
	if night.DoctorTargetID == targetID {
		if doc := m.findPlayerByRole(room, roles.RoleDoctor); doc != nil {
			visitors = append(visitors, doc)
		}
	}
	return visitors
}

func (m *Manager) getRoomNight(room *Room) *NightState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if state, ok := m.states[room.ID]; ok {
		return state.Night
	}
	return nil
}

func (m *Manager) sendNightActions(room *Room, state *GameState) {
	for _, p := range room.AlivePlayers() {
		if isBot(p.TelegramID) || !roles.HasNightAction(p.Role) {
			continue
		}
		targets := m.getTargetsForRole(room, p)
		if len(targets) == 0 {
			continue
		}

		var keyboard [][]tgbotapi.InlineKeyboardButton
		for _, target := range targets {
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"👤 "+target.Username,
					fmt.Sprintf("%s_%s_%d", string(p.Role), room.ID, target.TelegramID),
				),
			)
			keyboard = append(keyboard, row)
		}

		text := getNightActionText(p.Role, state.Round)
		msg := tgbotapi.NewMessage(p.TelegramID, text)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
		m.bot.Send(msg)
	}
}

func getNightActionText(role roles.RoleName, round int) string {
	switch role {
	case roles.RoleMafia, roles.RoleDon:
		return "😈 <b>Kim sizning qurbonингиз?</b>"
	case roles.RoleDoctor:
		return "👨‍⚕️ <b>Kimni davolaysiz?</b>"
	case roles.RoleKomissar:
		if round == 1 {
			return "🕵️ <b>1-tunda tekshiruv qila olmaysiz. Kuzating.</b>"
		}
		return "🕵️ <b>Kimni tekshirasiz?</b>"
	case roles.RoleSerjant:
		return "👮 <b>Kimni kuzatasiz?</b>"
	case roles.RoleMashuqa:
		return "💃 <b>Kim bilan tunni o'tkazasiz? (U harakat qila olmaydi)</b>"
	case roles.RoleDaydi:
		return "🧙 <b>Kim uyida tunaysiz? (Kelganlarni ko'rasiz)</b>"
	case roles.RoleManyak:
		return "🔪 <b>Kimni o'ldirasiz?</b>"
	case roles.RoleTentak:
		return "👨🏻‍🦲 <b>Kimga borasiz?</b>"
	default:
		return "🎯 <b>Maqsadingizni tanlang</b>"
	}
}

// ─── MUTE / UNMUTE / SEND ───

func (m *Manager) muteAll(room *Room) {
	for _, p := range room.Players {
		if p.IsAlive && !isBot(p.TelegramID) && room.ChatID != 0 {
			m.restrictPlayer(room.ChatID, p.TelegramID, false)
		}
	}
}

func (m *Manager) unmuteAlive(room *Room) {
	for _, p := range room.Players {
		if p.IsAlive && !isBot(p.TelegramID) && room.ChatID != 0 {
			m.restrictPlayer(room.ChatID, p.TelegramID, true)
		}
	}
}

func (m *Manager) unmuteAll(room *Room) {
	for _, p := range room.Players {
		if !isBot(p.TelegramID) && room.ChatID != 0 {
			m.restrictPlayer(room.ChatID, p.TelegramID, true)
		}
	}
}

func (m *Manager) restrictPlayer(chatID, userID int64, canSend bool) {
	cfg := tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: chatID, UserID: userID},
		Permissions:      &tgbotapi.ChatPermissions{CanSendMessages: canSend},
	}
	m.bot.Request(cfg)
}

func (m *Manager) sendToChat(chatID int64, text string) {
	if chatID == 0 {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	m.bot.Send(msg)
}

func (m *Manager) notifyAdmin(text string) {
	if m.adminChatID == 0 {
		return
	}
	msg := tgbotapi.NewMessage(m.adminChatID, text)
	msg.ParseMode = "HTML"
	m.bot.Send(msg)
}

// ─── UTILITY ───

func isBot(userID int64) bool {
	return userID < 0
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func makeWebAppButton(text, url string) tgbotapi.InlineKeyboardButton {
	return tgbotapi.InlineKeyboardButton{Text: text, URL: &url}
}

func joinWithComma(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func filterOut(players []*Player, excludeID int64) []*Player {
	var result []*Player
	for _, p := range players {
		if p.TelegramID != excludeID {
			result = append(result, p)
		}
	}
	return result
}

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
	// Hub callbacklarini ulash
	hub.OnConnect     = m.HandleWebConnect
	hub.OnStartGame   = m.StartGameByOwner
	hub.OnNightAction = m.HandleNightAction
	hub.OnDayVote     = m.HandleDayVote
	return m
}

// ─── XONA BOSHQARUVI ───

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

// ─── O'YIN BOSHLASH ───

func (m *Manager) StartGame(roomID string) error {
	room := m.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("xona topilmadi")
	}
	players := room.GetPlayerList()
	if len(players) < 4 {
		return fmt.Errorf("kamida 4 o'yinchi kerak (hozir %d ta)", len(players))
	}
	room.Status = RoomPlaying
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

// ─── ROL TAQSIMLASH ───

func (m *Manager) assignRoles(room *Room, state *GameState) {
	players := room.GetPlayerList()
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})

	assigned := roles.AssignRoles(len(players))
	for i, p := range players {
		p.Role = assigned[i]
		// Komissar ID ni saqlash
		if p.Role == roles.RoleKomissar {
			state.KomissarID = p.TelegramID
		}
	}

	// Har bir o'yinchiga rol yuborish
	for _, p := range players {
		roleInfo := roles.Get(p.Role)
		privateMsg := RolePrivateMsg(p)

		if !isBot(p.TelegramID) {
			msg := tgbotapi.NewMessage(p.TelegramID, privateMsg)
			msg.ParseMode = "HTML"
			// WebApp tugmasi
			if m.webAppURL != "" {
				webAppBtn := makeWebAppButton(
					"🎮 O'yinga kirish (WebApp)",
					m.webAppURL+"?room="+room.ID,
				)
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(webAppBtn),
				)
			}
			m.bot.Send(msg)
		}

		// WebApp ga ham yuborish
		m.hub.SendToUser(p.TelegramID, MsgTypeRoleReveal, RoleRevealPayload{
			Role:        string(p.Role),
			Description: roleInfo.Description,
			Emoji:       roleInfo.Emoji,
		})
	}

	// Guruhga o'yinchilar ro'yxati va rollar sonini yuborish
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

	roleSummary := ""
	for role, count := range roleCount {
		info := roles.Get(role)
		if count > 1 {
			roleSummary += fmt.Sprintf("%s %s - %d, ", info.Emoji, string(role), count)
		} else {
			roleSummary += fmt.Sprintf("%s %s, ", info.Emoji, string(role))
		}
	}
	if len(roleSummary) > 2 {
		roleSummary = roleSummary[:len(roleSummary)-2]
	}
	text += roleSummary
	text += fmt.Sprintf("\nJami: %d kishi.", len(players))
	text += "\n\n<i>Tunda bo'lgan hodisalarni muhokama qilishning ayni vaqti...</i>"

	m.sendToChat(room.ChatID, text)
}

// ─── ASOSIY O'YIN TSIKLI ───

func (m *Manager) runGame(room *Room, state *GameState) {
	for {
		state.Round++
		state.Phase = PhaseNight

		// Tun boshlanishi
		m.sendToChat(room.ChatID, NightStartMsg(state.Round))
		time.Sleep(1 * time.Second)
		m.muteAll(room)
		m.sendNightActions(room, state)
		go m.runBotNightActions(room, state)

		// WebApp ga tun fazasini yuborish
		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase:   "night",
			Round:   state.Round,
			Message: fmt.Sprintf("🌙 TUN %d boshlandi", state.Round),
			Timer:   60,
		})
		m.broadcastGameState(room, state)

		// Tun harakatlarini e'lon qilish (True Mafia Black uslubi)
		time.Sleep(3 * time.Second)
		m.announceNightActions(room, state)

		time.Sleep(57 * time.Second) // Jami 60 soniya

		// Tun natijalarini qayta ishlash
		m.processNight(room, state)

		// Win tekshiruvi
		if won, winner := m.checkWin(room); won {
			m.endGame(room, state, winner)
			return
		}

		// Kun fazasi
		state.Phase = PhaseDay
		m.unmuteAlive(room)

		dayMsg := DayStartMsg(state.Round)
		m.sendToChat(room.ChatID, dayMsg)
		m.hub.BroadcastToRoom(room.ID, MsgTypePhaseChange, PhasePayload{
			Phase: "day", Round: state.Round, Message: dayMsg, Timer: 90,
		})
		m.broadcastGameState(room, state)

		time.Sleep(90 * time.Second)

		// Ovoz berish
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

		// Ovoz natijasi
		m.processVoting(room, state)

		if won, winner := m.checkWin(room); won {
			m.endGame(room, state, winner)
			return
		}

		state.ResetNight()
	}
}

// ─── TUN HARAKATLARINI E'LON QILISH ───

func (m *Manager) announceNightActions(room *Room, state *GameState) {
	// Har bir rol harakatini ketma-ket e'lon qilish (True Mafia Black uslubi)
	announcements := []string{}

	for _, p := range room.AlivePlayers() {
		switch p.Role {
		case roles.RoleDoctor:
			announcements = append(announcements, "👨‍⚕️ <b>Shifokor</b> tungi navbatchilikka ketdi...")
		case roles.RoleKomissar:
			if state.Round > 1 { // 1-tunda tekshiruv qila olmaydi
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

// ─── TUN NATIJALARINI QAYTA ISHLASH ───

func (m *Manager) processNight(room *Room, state *GameState) {
	night := state.Night

	// Mashuqa bloklangan odamlarni aniqlash
	blockedID := night.MashuqaTargetID
	if blockedID != 0 {
		if p, ok := room.Players[blockedID]; ok {
			p.IsBlocked = true
		}
	}

	// Asosiy o'ldirish
	killedID := night.MafiaTargetID
	manyakKilledID := night.ManyakTargetID

	// Shifokor saqladi?
	savedByDoctor := night.DoctorTargetID

	// Manyak jabrlanuvchisi
	if manyakKilledID != 0 && manyakKilledID != savedByDoctor {
		if p, ok := room.Players[manyakKilledID]; ok && p.IsAlive {
			p.IsAlive = false
			m.sendToChat(room.ChatID, DeathMessage(p, roles.RoleManyak))
			m.hub.BroadcastToRoom(room.ID, MsgTypePlayerDied, map[string]interface{}{
				"player_id": p.TelegramID, "name": p.Username, "role": string(p.Role),
			})
			// Kamikaze
			if p.Role == roles.RoleKamikaze {
				m.handleKamikazeDeath(room, state, p, false)
			}
		}
	}

	// Mafia jabrlanuvchisi
	if killedID != 0 && killedID != savedByDoctor {
		if p, ok := room.Players[killedID]; ok && p.IsAlive {
			// Mashuqa bloklagan odamni o'ldirish mumkinmi?
			killerRole := m.getMafiaRole(room)

			p.IsAlive = false
			m.sendToChat(room.ChatID, DeathMessage(p, killerRole))
			time.Sleep(500 * time.Millisecond)

			m.hub.BroadcastToRoom(room.ID, MsgTypePlayerDied, map[string]interface{}{
				"player_id": p.TelegramID, "name": p.Username, "role": string(p.Role),
			})

			// Kamikaze
			if p.Role == roles.RoleKamikaze {
				m.handleKamikazeDeath(room, state, p, true)
			}

			// Don o'lsa — meros
			if p.Role == roles.RoleDon {
				m.handleDonDeath(room)
			}

			// Komissar o'lsa — Serjant meros oladi
			if p.Role == roles.RoleKomissar {
				m.handleKomissarDeath(room, state)
			}
		}
	} else if killedID != 0 && killedID == savedByDoctor {
		m.sendToChat(room.ChatID, SavedMessage(m.getPlayerName(room, killedID)))
	} else {
		m.sendToChat(room.ChatID, NobodyDiedMsg())
	}

	// Daydi natijasi
	if night.DaydiTargetID != 0 && state.KomissarID != 0 {
		if targetPlayer, ok := room.Players[night.DaydiTargetID]; ok {
			visitors := m.getNightVisitors(room, state, night.DaydiTargetID)
			result := DaydiResultMsg(targetPlayer, visitors)
			// Daydi ga shaxsan yuborish
			if daydiPlayer := m.findPlayerByRole(room, roles.RoleDaydi); daydiPlayer != nil {
				if !isBot(daydiPlayer.TelegramID) {
					msg := tgbotapi.NewMessage(daydiPlayer.TelegramID, result)
					msg.ParseMode = "HTML"
					m.bot.Send(msg)
				}
				m.hub.SendToUser(daydiPlayer.TelegramID, MsgTypeNightResult, map[string]string{"result": result})
			}
		}
	}

	// Komissar natijasi (2-tun va undan keyin)
	if night.KomissarTargetID != 0 && state.Round > 1 {
		if targetPlayer, ok := room.Players[night.KomissarTargetID]; ok {
			isMafia := roles.IsMafia(targetPlayer.Role)
			// Advokat himoyasi
			if targetPlayer.Role == roles.RoleAdvokat {
				isMafia = false // Advokat Komissar oldida tinch aholi ko'rinadi
			}
			result := KomissarResultMsg(targetPlayer, isMafia)
			komissarID := state.KomissarID
			if komissarPlayer, ok := room.Players[komissarID]; ok {
				if !isBot(komissarPlayer.TelegramID) {
					msg := tgbotapi.NewMessage(komissarPlayer.TelegramID, result)
					msg.ParseMode = "HTML"
					m.bot.Send(msg)
				}
				m.hub.SendToUser(komissarPlayer.TelegramID, MsgTypeSheriffResult, map[string]interface{}{
					"target": targetPlayer.Username, "is_mafia": isMafia, "result": result,
				})
			}
		}
	}

	// Bloklangan holatni tozalash
	for _, p := range room.Players {
		p.IsBlocked = false
	}
}

func (m *Manager) getMafiaRole(room *Room) roles.RoleName {
	for _, p := range room.Players {
		if p.IsAlive && (p.Role == roles.RoleDon || p.Role == roles.RoleMafia) {
			return p.Role
		}
	}
	return roles.RoleMafia
}

func (m *Manager) handleKamikazeDeath(room *Room, state *GameState, kamikaze *Player, wasShot bool) {
	// Otib o'ldirilsa — otgan odam ham o'ladi
	if wasShot {
		killerID := state.Night.MafiaTargetID
		if killer, ok := room.Players[killerID]; ok {
			_ = killer // handled above, but kamikaze kills the SHOOTER
			// Find who voted for kamikaze
			for voterID := range state.Night.MafiaVotes {
				if voter, ok := room.Players[voterID]; ok && voter.IsAlive {
					voter.IsAlive = false
					m.sendToChat(room.ChatID, KamikazeDeathMessage(kamikaze, voter))
					break
				}
			}
		}
	} else {
		// Osilsa — bitta odamni olib ketadi
		alive := room.AlivePlayers()
		if len(alive) > 0 {
			target := alive[rand.Intn(len(alive))]
			target.IsAlive = false
			m.sendToChat(room.ChatID, fmt.Sprintf("💣 <b>Kamikaze</b> osilganda portlab, <b>%s</b> ni ham olib ketdi!", target.Username))
		}
	}
}

func (m *Manager) handleDonDeath(room *Room) {
	// Don o'lganda — Mafia Don bo'ladi
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
	// Komissar o'lganda — Serjant uni o'rnini oladi
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

func (m *Manager) getNightVisitors(room *Room, state *GameState, targetID int64) []*Player {
	var visitors []*Player
	night := state.Night
	if night.MafiaTargetID == targetID {
		for voterID := range night.MafiaVotes {
			if p, ok := room.Players[voterID]; ok {
				visitors = append(visitors, p)
			}
		}
	}
	if night.DoctorTargetID == targetID {
		if p := m.findPlayerByRole(room, roles.RoleDoctor); p != nil {
			visitors = append(visitors, p)
		}
	}
	return visitors
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
	if p, ok := room.Players[id]; ok {
		return p.Username
	}
	return "Noma'lum"
}

// ─── TUN HARAKATLARI ───

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
			info := roles.Get(target.Role)
			_ = info
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"👤 "+target.Username,
					fmt.Sprintf("%s_%s_%d", string(p.Role), room.ID, target.TelegramID),
				),
			)
			keyboard = append(keyboard, row)
		}
		text := m.getNightActionText(p.Role, state.Round)
		msg := tgbotapi.NewMessage(p.TelegramID, text)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
		m.bot.Send(msg)
	}
}

func (m *Manager) getTargetsForRole(room *Room, player *Player) []*Player {
	var targets []*Player
	switch player.Role {
	case roles.RoleMafia, roles.RoleDon:
		// Mafia mafia bo'lmaganlarni tanlaydi
		for _, p := range room.AlivePlayers() {
			if p.TelegramID != player.TelegramID && !roles.IsMafia(p.Role) {
				targets = append(targets, p)
			}
		}
	default:
		// Boshqalar istalgan odamni tanlashi mumkin (o'zlari tashqari)
		for _, p := range room.AlivePlayers() {
			if p.TelegramID != player.TelegramID {
				targets = append(targets, p)
			}
		}
	}
	return targets
}

func (m *Manager) getNightActionText(role roles.RoleName, round int) string {
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

// ─── NIGHT ACTION HANDLER ───

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

	player, ok := room.Players[voterID]
	if !ok || !player.IsAlive {
		return
	}

	// Bloklangan holda harakat qila olmaydi
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
		// Ko'pchilik ovozi
		counts := make(map[int64]int)
		for _, t := range night.MafiaVotes {
			counts[t]++
		}
		maxV := 0
		for t, c := range counts {
			if c > maxV {
				maxV = c
				night.MafiaTargetID = t
			}
		}
		if !isBot(voterID) {
			target := room.Players[targetID]
			if target != nil {
				msg := tgbotapi.NewMessage(voterID, fmt.Sprintf("✅ Tanlov qabul qilindi: <b>%s</b>", target.Username))
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
			}
		}

	case roles.RoleDoctor:
		night.DoctorTargetID = targetID
		if !isBot(voterID) {
			target := room.Players[targetID]
			if target != nil {
				msg := tgbotapi.NewMessage(voterID, fmt.Sprintf("💊 <b>%s</b> davolanmoqda...", target.Username))
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
			}
		}

	case roles.RoleKomissar, roles.RoleSerjant:
		if state.Round == 1 {
			return // 1-tunda tekshiruv qila olmaydi
		}
		night.KomissarTargetID = targetID

	case roles.RoleMashuqa:
		night.MashuqaTargetID = targetID
		if target, ok := room.Players[targetID]; ok {
			target.IsBlocked = true
		}
		if !isBot(voterID) {
			target := room.Players[targetID]
			if target != nil {
				msg := tgbotapi.NewMessage(voterID, fmt.Sprintf("💃 <b>%s</b> bilan tunni o'tkazmoqdasiz...", target.Username))
				msg.ParseMode = "HTML"
				m.bot.Send(msg)
			}
		}

	case roles.RoleDaydi:
		night.DaydiTargetID = targetID

	case roles.RoleManyak:
		night.ManyakTargetID = targetID

	case roles.RoleTentak:
		night.TentakTargetID = targetID
	}
}

// ─── OVOZ BERISH ───

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
		room := m.rooms[roomID]
		if room != nil {
			if target, ok := room.Players[targetID]; ok {
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

	player, ok := room.Players[winnerID]
	if !ok {
		return
	}

	// Suidsid — ovoz bilan chiqarilsa u yutadi!
	if player.Role == roles.RoleSuidsid {
		m.sendToChat(room.ChatID, fmt.Sprintf("🧌 <b>SUIDSID %s chiqarib yuborildi!</b>\n\nU o'yinni YUTDI! Bu uning maqsadi edi!", player.Username))
		player.IsAlive = false
		m.hub.BroadcastToRoom(room.ID, MsgTypePlayerDied, map[string]interface{}{
			"player_id": winnerID, "name": player.Username, "role": string(player.Role), "voted_out": true,
		})
		m.endGame(room, state, "suidsid")
		return
	}

	// Kamikaze — osilsa bitta odamni olib ketadi
	if player.Role == roles.RoleKamikaze {
		player.IsAlive = false
		m.sendToChat(room.ChatID, VoteOutMessage(player))
		m.handleKamikazeDeath(room, state, player, false)
		return
	}

	// Oddiy chiqarish
	player.IsAlive = false
	m.sendToChat(room.ChatID, VoteOutMessage(player))

	if !isBot(winnerID) {
		m.bot.Send(tgbotapi.NewMessage(winnerID, "😢 Siz o'yindan chiqarildingiz. Kuzatishda davom eting!"))
	}

	m.hub.BroadcastToRoom(room.ID, MsgTypePlayerDied, map[string]interface{}{
		"player_id": winnerID, "name": player.Username, "role": string(player.Role), "voted_out": true,
	})

	// Komissar o'lsa meros
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
			m.doBotNightAction(room, state, bot)
		}(p)
	}
}

func (m *Manager) doBotNightAction(room *Room, state *GameState, bot *Player) {
	targets := m.getTargetsForRole(room, bot)
	if len(targets) == 0 {
		return
	}
	target := targets[rand.Intn(len(targets))]
	m.HandleNightAction(room.ID, string(bot.Role), bot.TelegramID, target.TelegramID)
}

func (m *Manager) runBotVoting(room *Room, state *GameState) {
	for _, p := range room.AlivePlayers() {
		if !isBot(p.TelegramID) {
			continue
		}
		go func(bot *Player) {
			time.Sleep(time.Duration(5000+rand.Intn(10000)) * time.Millisecond)
			alive := room.AlivePlayers()
			var candidates []*Player
			for _, c := range alive {
				if c.TelegramID != bot.TelegramID {
					candidates = append(candidates, c)
				}
			}
			if len(candidates) > 0 {
				m.HandleDayVote(room.ID, bot.TelegramID, candidates[rand.Intn(len(candidates))].TelegramID)
			}
		}(p)
	}
}

// ─── G'ALABA TEKSHIRUVI ───

func (m *Manager) checkWin(room *Room) (bool, string) {
	mafiaCount, townCount, manyakAlive, suidsidAlive := 0, 0, false, false
	for _, p := range room.Players {
		if !p.IsAlive {
			continue
		}
		switch {
		case roles.IsMafia(p.Role):
			mafiaCount++
		case p.Role == roles.RoleManyak:
			manyakAlive = true
		case p.Role == roles.RoleSuidsid:
			suidsidAlive = true
		default:
			townCount++
		}
	}
	total := mafiaCount + townCount
	if manyakAlive {
		total++
	}

	if mafiaCount == 0 && !manyakAlive {
		return true, "town"
	}
	if mafiaCount >= townCount+boolToInt(manyakAlive) {
		return true, "mafia"
	}
	if manyakAlive && total <= 2 {
		return true, "manyak"
	}
	_ = suidsidAlive
	return false, ""
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (m *Manager) endGame(room *Room, state *GameState, winner string) {
	room.Status = RoomFinished
	players := room.GetPlayerList()
	winMsg := WinMessage(winner, players)
	m.sendToChat(room.ChatID, winMsg)

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
	players := []PlayerInfo{}
	for _, p := range room.GetPlayerList() {
		info := roles.Get(p.Role)
		players = append(players, PlayerInfo{
			ID:    p.TelegramID,
			Name:  p.Username,
			IsAlive: p.IsAlive,
			Emoji: info.Emoji,
		})
	}
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
	players := []PlayerInfo{}
	for _, p := range room.GetPlayerList() {
		info := roles.Get(p.Role)
		players = append(players, PlayerInfo{
			ID:      p.TelegramID,
			Name:    p.Username,
			IsAlive: p.IsAlive,
			Emoji:   info.Emoji,
		})
	}
	m.hub.SendToUser(userID, MsgTypeGameState, GameStatePayload{
		Phase:   string(state.Phase),
		Round:   state.Round,
		Players: players,
	})
}

func (m *Manager) broadcastRoomInfo(room *Room) {
	players := []PlayerInfo{}
	for _, p := range room.GetPlayerList() {
		players = append(players, PlayerInfo{ID: p.TelegramID, Name: p.Username, IsAlive: p.IsAlive})
	}
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
	players := []PlayerInfo{}
	for _, p := range room.GetPlayerList() {
		players = append(players, PlayerInfo{ID: p.TelegramID, Name: p.Username, IsAlive: p.IsAlive})
	}
	m.hub.BroadcastToRoom(room.ID, MsgTypeGameState, GameStatePayload{
		Phase:   string(state.Phase),
		Round:   state.Round,
		Players: players,
	})
}

// ─── YORDAMCHI FUNKSIYALAR ───

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

func isBot(userID int64) bool {
	return userID < 0
}

func makeWebAppButton(text, url string) tgbotapi.InlineKeyboardButton {
	return tgbotapi.InlineKeyboardButton{
		Text: text,
		URL:  &url,
	}
}

// GetRoom for Web API
func (m *Manager) GetState(roomID string) *GameState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.states[roomID]
}

func (m *Manager) CreateRoomWeb(ownerID int64, ownerName string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := NewRoom(0, ownerID, ownerName)
	m.rooms[room.ID] = room
	m.userRooms[ownerID] = room.ID
	return room
}

func (m *Manager) ForceStopGame(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room, ok := m.rooms[roomID]
	if !ok { return }
	m.unmuteAll(room)
	delete(m.states, roomID)
	for _, p := range room.Players { delete(m.userRooms, p.TelegramID) }
	delete(m.chatRooms, room.ChatID)
	delete(m.rooms, roomID)
}

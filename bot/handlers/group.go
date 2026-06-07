package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"mafia-bot/db/repositories"
	"mafia-bot/game"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type GroupHandler struct {
	bot       *tgbotapi.BotAPI
	manager   *game.Manager
	userRepo  *repositories.UserRepository
	webAppURL string
}

func NewGroupHandler(bot *tgbotapi.BotAPI, manager *game.Manager, userRepo *repositories.UserRepository, webAppURL string) *GroupHandler {
	return &GroupHandler{bot: bot, manager: manager, userRepo: userRepo, webAppURL: webAppURL}
}

func (h *GroupHandler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	chatID := msg.Chat.ID

	if !msg.Chat.IsGroup() && !msg.Chat.IsSuperGroup() {
		return
	}

	switch msg.Command() {
	case "start", "newgame", "startgame":
		h.handleStart(msg, chatID)
	case "stopgame", "stop":
		h.handleStop(msg, chatID)
	case "stat", "stats":
		h.handleStat(msg, chatID)
	}

	// /+ qo'shilish
	if msg.Text == "/+" || msg.Text == "/join" || strings.HasPrefix(msg.Text, "/+") {
		h.handleJoin(msg, chatID)
	}
	// /- chiqish
	if msg.Text == "/-" || msg.Text == "/leave" {
		h.handleLeave(msg, chatID)
	}
}

func (h *GroupHandler) HandleCallback(query *tgbotapi.CallbackQuery) bool {
	data := query.Data
	switch {
	case strings.HasPrefix(data, "join_"):
		roomID := strings.TrimPrefix(data, "join_")
		h.joinByCallback(query, roomID)
		return true
	case strings.HasPrefix(data, "leave_"):
		roomID := strings.TrimPrefix(data, "leave_")
		h.leaveByCallback(query, roomID)
		return true
	case strings.HasPrefix(data, "startgame_"):
		roomID := strings.TrimPrefix(data, "startgame_")
		h.startByCallback(query, roomID)
		return true
	case strings.HasPrefix(data, "vote_"):
		parts := strings.Split(strings.TrimPrefix(data, "vote_"), "_")
		if len(parts) == 2 {
			var roomID, targetIDStr = parts[0], parts[1]
			var targetID int64
			fmt.Sscanf(targetIDStr, "%d", &targetID)
			h.manager.HandleDayVote(roomID, query.From.ID, targetID)
			h.bot.Request(tgbotapi.NewCallback(query.ID, "✅ Ovoz berildi"))
		}
		return true
	}
	// Night actions: "RoleName_roomID_targetID"
	return h.handleNightActionCallback(query, data)
}

func (h *GroupHandler) handleStart(msg *tgbotapi.Message, chatID int64) {
	from := msg.From

	// Guruhda mavjud o'yin bormi?
	existingRoom := h.manager.GetRoomByChat(chatID)
	if existingRoom != nil {
		h.bot.Send(tgbotapi.NewMessage(chatID,
			fmt.Sprintf("⚠️ Guruhda allaqachon o'yin bor!\nXona ID: <code>%s</code>", existingRoom.ID)))
		return
	}

	// Yangi xona yaratish
	room := h.manager.CreateRoom(chatID, from.ID, from.UserName)

	text := game.JoinMessage(1, room.MaxPlayers)

	outMsg := tgbotapi.NewMessage(chatID, text)
	outMsg.ParseMode = "HTML"
	outMsg.ReplyMarkup = h.buildJoinKeyboard(room.ID)
	sentMsg, err := h.bot.Send(outMsg)
	if err == nil {
		// Pin the message
		h.bot.Request(tgbotapi.PinChatMessageConfig{
			ChatID:    chatID,
			MessageID: sentMsg.MessageID,
		})
	}
}

func (h *GroupHandler) buildJoinKeyboard(roomID string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// WebApp tugmasi
	if h.webAppURL != "" {
		webBtn := tgbotapi.InlineKeyboardButton{
			Text:   "🎮 O'yinga kirish (WebApp)",
			WebApp: &tgbotapi.WebAppInfo{URL: h.webAppURL + "?room=" + roomID},
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(webBtn))
	}

	// Qo'shilish / Chiqish / Boshlash
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Qo'shilish", "join_"+roomID),
			tgbotapi.NewInlineKeyboardButtonData("❌ Chiqish", "leave_"+roomID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("▶️ O'yinni boshlash", "startgame_"+roomID),
		),
	)
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (h *GroupHandler) handleJoin(msg *tgbotapi.Message, chatID int64) {
	from := msg.From
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ Aktiv o'yin yo'q. /start bilan boshlang."))
		return
	}
	player := &game.Player{
		TelegramID: from.ID,
		Username:   from.UserName,
		FirstName:  from.FirstName,
		IsAlive:    true,
	}
	if err := h.manager.JoinRoom(room.ID, player); err != nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "⚠️ "+err.Error()))
		return
	}
	joinMsg := tgbotapi.NewMessage(chatID, game.PlayerJoinedMsg(from.UserName, room.PlayerCount(), room.MaxPlayers))
	joinMsg.ParseMode = "HTML"
	h.bot.Send(joinMsg)
}

func (h *GroupHandler) handleLeave(msg *tgbotapi.Message, chatID int64) {
	from := msg.From
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		return
	}
	h.manager.LeaveRoom(from.ID)
	leaveMsg := tgbotapi.NewMessage(chatID, game.PlayerLeftMsg(from.UserName, room.PlayerCount(), room.MaxPlayers))
	leaveMsg.ParseMode = "HTML"
	h.bot.Send(leaveMsg)
}

func (h *GroupHandler) handleStop(msg *tgbotapi.Message, chatID int64) {
	from := msg.From
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ Aktiv o'yin yo'q."))
		return
	}
	if room.OwnerID != from.ID {
		// Admin tekshiruvi
		member, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: chatID, UserID: from.ID},
		})
		if err != nil || (member.Status != "administrator" && member.Status != "creator") {
			h.bot.Send(tgbotapi.NewMessage(chatID, "❌ Faqat guruh admini yoki o'yin egasi to'xtatishi mumkin."))
			return
		}
	}
	h.manager.ForceStopGame(room.ID)
	h.bot.Send(tgbotapi.NewMessage(chatID, "🛑 O'yin to'xtatildi."))
}

func (h *GroupHandler) handleStat(msg *tgbotapi.Message, chatID int64) {
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "ℹ️ Hozir aktiv o'yin yo'q."))
		return
	}
	players := room.GetPlayerList()
	text := fmt.Sprintf("📊 <b>O'yin holati</b>\nXona: <code>%s</code>\nHolat: %s\nO'yinchilar: <b>%d</b>\n\n",
		room.ID, string(room.Status), len(players))
	for i, p := range players {
		text += fmt.Sprintf("%d. @%s\n", i+1, p.Username)
	}
	statMsg := tgbotapi.NewMessage(chatID, text)
	statMsg.ParseMode = "HTML"
	h.bot.Send(statMsg)
}

// ─── CALLBACK HANDLERS ───

func (h *GroupHandler) joinByCallback(query *tgbotapi.CallbackQuery, roomID string) {
	from := query.From
	player := &game.Player{
		TelegramID: from.ID,
		Username:   from.UserName,
		FirstName:  from.FirstName,
		IsAlive:    true,
	}
	if err := h.manager.JoinRoom(roomID, player); err != nil {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "⚠️ "+err.Error()))
		return
	}
	room := h.manager.GetRoom(roomID)
	if room == nil {
		return
	}
	h.bot.Request(tgbotapi.NewCallback(query.ID, "✅ O'yinga qo'shildingiz!"))

	// Guruhga xabar
	if query.Message != nil {
		joinMsg := tgbotapi.NewMessage(query.Message.Chat.ID,
			game.PlayerJoinedMsg(from.UserName, room.PlayerCount(), room.MaxPlayers))
		joinMsg.ParseMode = "HTML"
		h.bot.Send(joinMsg)

		// Tugmani yangilash
		edit := tgbotapi.NewEditMessageReplyMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			h.buildJoinKeyboard(roomID),
		)
		h.bot.Request(edit)
	}
}

func (h *GroupHandler) leaveByCallback(query *tgbotapi.CallbackQuery, roomID string) {
	from := query.From
	h.manager.LeaveRoom(from.ID)
	h.bot.Request(tgbotapi.NewCallback(query.ID, "👋 O'yindan chiqdingiz"))

	room := h.manager.GetRoom(roomID)
	if room != nil && query.Message != nil {
		leaveMsg := tgbotapi.NewMessage(query.Message.Chat.ID,
			game.PlayerLeftMsg(from.UserName, room.PlayerCount(), room.MaxPlayers))
		leaveMsg.ParseMode = "HTML"
		h.bot.Send(leaveMsg)
	}
}

func (h *GroupHandler) startByCallback(query *tgbotapi.CallbackQuery, roomID string) {
	from := query.From
	room := h.manager.GetRoom(roomID)
	if room == nil {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ Xona topilmadi"))
		return
	}

	// Faqat egasi yoki admin boshlashi mumkin
	if room.OwnerID != from.ID {
		if query.Message != nil {
			member, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
				ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
					ChatID: query.Message.Chat.ID, UserID: from.ID,
				},
			})
			if err != nil || (member.Status != "administrator" && member.Status != "creator") {
				h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ Faqat xona egasi boshlashi mumkin"))
				return
			}
		}
	}

	if err := h.manager.StartGame(roomID); err != nil {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ "+err.Error()))
		return
	}
	h.bot.Request(tgbotapi.NewCallback(query.ID, "🎮 O'yin boshlandi!"))
}

func (h *GroupHandler) handleNightActionCallback(query *tgbotapi.CallbackQuery, data string) bool {
	// Format: "RoleName_roomID_targetID"
	parts := strings.SplitN(data, "_", 3)
	if len(parts) != 3 {
		return false
	}

	roleName := parts[0]
	roomID := parts[1]
	var targetID int64
	fmt.Sscanf(parts[2], "%d", &targetID)

	// Rol tekshiruvi
	validRoles := map[string]bool{
		"Don": true, "Mafiya": true, "Shifokor": true,
		"Komissar": true, "Serjant": true, "Mashuqa": true,
		"Daydi": true, "Manyak": true, "Tentak": true, "Bodyguard": true,
	}
	if !validRoles[roleName] {
		return false
	}

	from := query.From
	room := h.manager.GetRoom(roomID)
	if room == nil {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ Xona topilmadi"))
		return true
	}

	target, ok := room.Players[targetID]
	if !ok {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ O'yinchi topilmadi"))
		return true
	}

	h.manager.HandleNightAction(roomID, roleName, from.ID, targetID)
	h.bot.Request(tgbotapi.NewCallback(query.ID, fmt.Sprintf("✅ %s tanlandi", target.Username)))
	return true
}

// Inline keyboard uchun JSON helper
func makeInlineKbd(rows ...[]map[string]interface{}) interface{} {
	type kbdBtn struct {
		Text         string `json:"text"`
		CallbackData string `json:"callback_data,omitempty"`
	}
	type kbd struct {
		InlineKeyboard [][]kbdBtn `json:"inline_keyboard"`
	}
	_ = rows
	return nil
}

func jsonEncode(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

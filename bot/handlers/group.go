package handlers

import (
	"fmt"
	"strings"

	"mafia-bot/config"
	"mafia-bot/db/repositories"
	"mafia-bot/game"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type GroupHandler struct {
	bot      *tgbotapi.BotAPI
	cfg      *config.Config
	manager  *game.Manager
	userRepo *repositories.UserRepository
}

func NewGroupHandler(bot *tgbotapi.BotAPI, cfg *config.Config, manager *game.Manager, userRepo *repositories.UserRepository) *GroupHandler {
	return &GroupHandler{bot: bot, cfg: cfg, manager: manager, userRepo: userRepo}
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

	if msg.Text == "/+" || msg.Text == "/join" || strings.HasPrefix(msg.Text, "/+") {
		h.handleJoin(msg, chatID)
	}
	if msg.Text == "/-" || msg.Text == "/leave" {
		h.handleLeave(msg, chatID)
	}
}

func (h *GroupHandler) HandleCallback(query *tgbotapi.CallbackQuery) bool {
	data := query.Data
	switch {
	case strings.HasPrefix(data, "join_"):
		h.joinByCallback(query, strings.TrimPrefix(data, "join_"))
		return true
	case strings.HasPrefix(data, "leave_"):
		h.leaveByCallback(query, strings.TrimPrefix(data, "leave_"))
		return true
	case strings.HasPrefix(data, "startgame_"):
		h.startByCallback(query, strings.TrimPrefix(data, "startgame_"))
		return true
	case strings.HasPrefix(data, "vote_"):
		h.handleVoteCallback(query, strings.TrimPrefix(data, "vote_"))
		return true
	}
	return h.handleNightActionCallback(query, data)
}

func (h *GroupHandler) handleStart(msg *tgbotapi.Message, chatID int64) {
	existingRoom := h.manager.GetRoomByChat(chatID)
	if existingRoom != nil {
		h.send(chatID, fmt.Sprintf("⚠️ Guruhda allaqachon o'yin bor!\nXona ID: <code>%s</code>", existingRoom.ID))
		return
	}

	room := h.manager.CreateRoom(chatID, msg.From.ID, msg.From.UserName)
	outMsg := tgbotapi.NewMessage(chatID, game.JoinMessage(1, room.MaxPlayers))
	outMsg.ParseMode = "HTML"
	outMsg.ReplyMarkup = h.buildJoinKeyboard(room.ID)

	sentMsg, err := h.bot.Send(outMsg)
	if err == nil {
		h.bot.Request(tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: sentMsg.MessageID})
	}
}

func (h *GroupHandler) buildJoinKeyboard(roomID string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	if h.cfg.WebAppURL != "" {
		joinURL := h.cfg.WebAppURL + "?room=" + roomID
		webBtn := tgbotapi.InlineKeyboardButton{Text: "🎮 O'yinga kirish (WebApp)", URL: &joinURL}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(webBtn))
	}

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
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		h.send(chatID, "❌ Aktiv o'yin yo'q. /start bilan boshlang.")
		return
	}

	player := &game.Player{
		TelegramID: msg.From.ID,
		Username:   msg.From.UserName,
		FirstName:  msg.From.FirstName,
		IsAlive:    true,
	}
	if err := h.manager.JoinRoom(room.ID, player); err != nil {
		h.send(chatID, "⚠️ "+err.Error())
		return
	}
	h.send(chatID, game.PlayerJoinedMsg(msg.From.UserName, room.PlayerCount(), room.MaxPlayers))
}

func (h *GroupHandler) handleLeave(msg *tgbotapi.Message, chatID int64) {
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		return
	}
	h.manager.LeaveRoom(msg.From.ID)
	h.send(chatID, game.PlayerLeftMsg(msg.From.UserName, room.PlayerCount(), room.MaxPlayers))
}

func (h *GroupHandler) handleStop(msg *tgbotapi.Message, chatID int64) {
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		h.send(chatID, "❌ Aktiv o'yin yo'q.")
		return
	}

	if room.OwnerID != msg.From.ID && !h.isGroupAdmin(chatID, msg.From.ID) {
		h.send(chatID, "❌ Faqat guruh admini yoki o'yin egasi to'xtatishi mumkin.")
		return
	}

	h.manager.ForceStopGame(room.ID)
	h.send(chatID, "🛑 O'yin to'xtatildi.")
}

func (h *GroupHandler) handleStat(msg *tgbotapi.Message, chatID int64) {
	room := h.manager.GetRoomByChat(chatID)
	if room == nil {
		h.send(chatID, "ℹ️ Hozir aktiv o'yin yo'q.")
		return
	}

	players := room.GetPlayerList()
	text := fmt.Sprintf("📊 <b>O'yin holati</b>\nXona: <code>%s</code>\nHolat: %s\nO'yinchilar: <b>%d</b>\n\n",
		room.ID, string(room.Status), len(players))
	for i, p := range players {
		text += fmt.Sprintf("%d. @%s\n", i+1, p.Username)
	}
	h.send(chatID, text)
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

	if query.Message != nil {
		h.send(query.Message.Chat.ID, game.PlayerJoinedMsg(from.UserName, room.PlayerCount(), room.MaxPlayers))
		edit := tgbotapi.NewEditMessageReplyMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			h.buildJoinKeyboard(roomID),
		)
		h.bot.Request(edit)
	}
}

func (h *GroupHandler) leaveByCallback(query *tgbotapi.CallbackQuery, roomID string) {
	h.manager.LeaveRoom(query.From.ID)
	h.bot.Request(tgbotapi.NewCallback(query.ID, "👋 O'yindan chiqdingiz"))

	if room := h.manager.GetRoom(roomID); room != nil && query.Message != nil {
		h.send(query.Message.Chat.ID, game.PlayerLeftMsg(query.From.UserName, room.PlayerCount(), room.MaxPlayers))
	}
}

func (h *GroupHandler) startByCallback(query *tgbotapi.CallbackQuery, roomID string) {
	room := h.manager.GetRoom(roomID)
	if room == nil {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ Xona topilmadi"))
		return
	}

	if room.OwnerID != query.From.ID && !h.isGroupAdmin(query.Message.Chat.ID, query.From.ID) {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ Faqat xona egasi boshlashi mumkin"))
		return
	}

	if err := h.manager.StartGame(roomID); err != nil {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ "+err.Error()))
		return
	}
	h.bot.Request(tgbotapi.NewCallback(query.ID, "🎮 O'yin boshlandi!"))
}

func (h *GroupHandler) handleVoteCallback(query *tgbotapi.CallbackQuery, data string) {
	parts := strings.Split(data, "_")
	if len(parts) != 2 {
		return
	}

	roomID := parts[0]
	var targetID int64
	fmt.Sscanf(parts[1], "%d", &targetID)

	h.manager.HandleDayVote(roomID, query.From.ID, targetID)
	h.bot.Request(tgbotapi.NewCallback(query.ID, "✅ Ovoz berildi"))
}

func (h *GroupHandler) handleNightActionCallback(query *tgbotapi.CallbackQuery, data string) bool {
	parts := strings.SplitN(data, "_", 3)
	if len(parts) != 3 {
		return false
	}

	roleName, roomID := parts[0], parts[1]
	var targetID int64
	fmt.Sscanf(parts[2], "%d", &targetID)

	validRoles := map[string]bool{
		"Don": true, "Mafiya": true, "Shifokor": true,
		"Komissar": true, "Serjant": true, "Mashuqa": true,
		"Daydi": true, "Manyak": true, "Tentak": true,
	}
	if !validRoles[roleName] {
		return false
	}

	room := h.manager.GetRoom(roomID)
	if room == nil {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ Xona topilmadi"))
		return true
	}

	if _, ok := room.PlayerByID(targetID); !ok {
		h.bot.Request(tgbotapi.NewCallback(query.ID, "❌ O'yinchi topilmadi"))
		return true
	}

	h.manager.HandleNightAction(roomID, roleName, query.From.ID, targetID)

	target, _ := room.PlayerByID(targetID)
	h.bot.Request(tgbotapi.NewCallback(query.ID, fmt.Sprintf("✅ %s tanlandi", target.Username)))
	return true
}

func (h *GroupHandler) isGroupAdmin(chatID, userID int64) bool {
	member, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: chatID, UserID: userID},
	})
	if err != nil {
		return false
	}
	return member.Status == "administrator" || member.Status == "creator"
}

func (h *GroupHandler) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

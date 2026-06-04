package handlers

import (
	"mafia-bot/game"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type GameHandler struct {
	bot     *tgbotapi.BotAPI
	manager *game.Manager
}

func NewGameHandler(bot *tgbotapi.BotAPI, manager *game.Manager) *GameHandler {
	return &GameHandler{bot: bot, manager: manager}
}

func (h *GameHandler) HandleCallback(update tgbotapi.Update) {
	callback := update.CallbackQuery
	if callback == nil {
		return
	}

	data := callback.Data
	from := callback.From

	// startgame_ROOMID
	if strings.HasPrefix(data, "startgame_") {
		roomID := strings.TrimPrefix(data, "startgame_")
		room := h.manager.GetRoom(roomID)
		if room != nil && room.OwnerID == from.ID {
			if err := h.manager.StartGame(roomID); err != nil {
				h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ "+err.Error()))
			} else {
				h.bot.Request(tgbotapi.NewCallback(callback.ID, "✅ O'yin boshlandi!"))
			}
		}
		return
	}

	// Night action: ROLE_ROOMID_TARGETID
	// e.g. "Mafia_123456_987654321"
	parts := strings.Split(data, "_")
	if len(parts) == 3 {
		role := parts[0]
		roomID := parts[1]
		targetID, err := strconv.ParseInt(parts[2], 10, 64)
		if err == nil {
			h.manager.HandleNightAction(roomID, role, from.ID, targetID)
			h.bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Tanlov qabul qilindi"))
			return
		}
	}

	// Day vote: dayVote_ROOMID_TARGETID
	if strings.HasPrefix(data, "dayVote_") {
		rest := strings.TrimPrefix(data, "dayVote_")
		parts := strings.SplitN(rest, "_", 2)
		if len(parts) == 2 {
			roomID := parts[0]
			targetID, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil {
				h.manager.HandleDayVote(roomID, from.ID, targetID)
				h.bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Ovoz berildi"))
			}
		}
		return
	}
}

package handlers

import (
	"fmt"
	"mafia-bot/config"
	"mafia-bot/db/models"
	"mafia-bot/db/repositories"
	"mafia-bot/game"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StartHandler struct {
	bot      *tgbotapi.BotAPI
	userRepo *repositories.UserRepository
	manager  *game.Manager
}

func NewStartHandler(bot *tgbotapi.BotAPI, userRepo *repositories.UserRepository, manager *game.Manager) *StartHandler {
	return &StartHandler{bot: bot, userRepo: userRepo, manager: manager}
}

func (h *StartHandler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	user, _ := h.userRepo.GetOrCreate(update.Message.From.ID, update.Message.From.UserName, update.Message.From.FirstName)
	switch update.Message.Command() {
	case "start":
		h.handleStart(update, user)
	case "profile":
		h.handleProfile(update, user)
	case "rating":
		h.handleRating(update)
	case "testgame":
		h.handleTestGame(update)
	}
}

func (h *StartHandler) handleStart(update tgbotapi.Update, user *models.User) {
	args := update.Message.CommandArguments()
	if strings.HasPrefix(args, "ref_") {
		roomID := strings.TrimPrefix(args, "ref_")
		room := h.manager.GetRoom(roomID)
		if room != nil {
			from := update.Message.From
			player := &game.Player{TelegramID: from.ID, Username: from.UserName, FirstName: from.FirstName, IsAlive: true}
			if err := h.manager.JoinRoom(roomID, player); err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ "+err.Error())
				h.bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("✅ Xonaga qo'shildingiz!\nXona ID: <b>%s</b>", roomID))
				msg.ParseMode = "HTML"
				h.bot.Send(msg)
			}
			return
		}
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, config.MsgWelcome)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *StartHandler) handleProfile(update tgbotapi.Update, user *models.User) {
	text := fmt.Sprintf(config.MsgProfileCard, user.Username, user.Level, user.XP, user.TotalGames, user.Wins, user.WinRate(), user.WinStreak, string(user.League), user.Coins)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *StartHandler) handleRating(update tgbotapi.Update) {
	users, err := h.userRepo.GetTopUsers(10)
	if err != nil {
		return
	}
	text := "🏆 <b>TOP-10 REYTING</b>\n\n"
	medals := []string{"🥇", "🥈", "🥉"}
	for i, u := range users {
		medal := fmt.Sprintf("%d.", i+1)
		if i < 3 {
			medal = medals[i]
		}
		text += fmt.Sprintf("%s <b>%s</b> — %d XP\n", medal, u.Username, u.XP)
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *StartHandler) handleTestGame(update tgbotapi.Update) {
	from := update.Message.From
	chatID := update.Message.Chat.ID

	// CreateRoom egani o'zi qo'shadi (1-o'yinchi)
	room := h.manager.CreateRoom(chatID, from.ID, from.UserName)

	// 4 ta bot qo'shish
	bots := []struct{ id int64; name string }{
		{-3001, "🤖 Sardor"}, {-3002, "🤖 Dilnoza"},
		{-3003, "🤖 Komil"},  {-3004, "🤖 Shaxboz"},
	}
	for _, b := range bots {
		h.manager.JoinRoom(room.ID, &game.Player{TelegramID: b.id, Username: b.name, IsAlive: true})
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"🧪 <b>TEST O'YIN</b>\n\nXona: <code>%s</code>\n5 o'yinchi: Siz + 4 bot\n\n🤖 Sardor | 🤖 Dilnoza\n🤖 Komil | 🤖 Shaxboz\n\nRollar yuborilmoqda...", room.ID))
	msg.ParseMode = "HTML"
	h.bot.Send(msg)

	if err := h.manager.StartGame(room.ID); err != nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ "+err.Error()))
	}
}

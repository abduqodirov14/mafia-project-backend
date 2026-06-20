package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"mafia-bot/config"
	"mafia-bot/db/repositories"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AdminHandler struct {
	bot      *tgbotapi.BotAPI
	cfg      *config.Config
	userRepo *repositories.UserRepository
}

func NewAdminHandler(bot *tgbotapi.BotAPI, cfg *config.Config, userRepo *repositories.UserRepository) *AdminHandler {
	return &AdminHandler{bot: bot, cfg: cfg, userRepo: userRepo}
}

func (h *AdminHandler) IsAdmin(userID int64) bool {
	return h.cfg.IsAdmin(userID)
}

func (h *AdminHandler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	if !h.IsAdmin(update.Message.From.ID) {
		return
	}

	switch update.Message.Command() {
	case "admin":
		h.showPanel(update.Message.Chat.ID)
	case "stats":
		h.showStats(update.Message.Chat.ID)
	case "addcoins":
		h.addCoins(update.Message)
	case "ban":
		h.banUser(update.Message)
	case "broadcast":
		h.broadcast(update.Message)
	}
}

func (h *AdminHandler) HandleCallback(query *tgbotapi.CallbackQuery) bool {
	if !h.IsAdmin(query.From.ID) {
		return false
	}
	switch query.Data {
	case "admin_stats":
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		h.showStats(query.Message.Chat.ID)
		return true
	case "admin_users":
		h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		h.showUsers(query.Message.Chat.ID)
		return true
	}
	return false
}

func (h *AdminHandler) showPanel(chatID int64) {
	users, _ := h.userRepo.GetAllUsers()
	total := len(users)
	active := 0
	for _, u := range users {
		if u.TotalGames > 0 {
			active++
		}
	}

	text := fmt.Sprintf(
		"🔐 <b>ADMIN PANEL</b>\n\n"+
			"👥 Jami foydalanuvchilar: <b>%d</b>\n"+
			"🎮 Aktiv o'yinchilar: <b>%d</b>\n\n"+
			"Buyruqlar:\n"+
			"/stats — batafsil statistika\n"+
			"/addcoins @user miqdor — tanga berish\n"+
			"/ban @user — foydalanuvchini bloklash\n"+
			"/broadcast xabar — hammaga xabar yuborish",
		total, active)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Statistika", "admin_stats"),
			tgbotapi.NewInlineKeyboardButtonData("👥 Foydalanuvchilar", "admin_users"),
		),
	)
	h.bot.Send(msg)
}

func (h *AdminHandler) showStats(chatID int64) {
	users, _ := h.userRepo.GetAllUsers()

	var totalGames, totalWins, totalCoins int
	for _, u := range users {
		totalGames += u.TotalGames
		totalWins += u.Wins
		totalCoins += u.Coins
	}

	text := fmt.Sprintf(
		"📊 <b>BATAFSIL STATISTIKA</b>\n\n"+
			"👥 Jami users: <b>%d</b>\n"+
			"🎮 Jami o'yinlar: <b>%d</b>\n"+
			"🏆 Jami g'alabalar: <b>%d</b>\n"+
			"💰 Jami tangalar: <b>%d</b>\n",
		len(users), totalGames, totalWins, totalCoins)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *AdminHandler) showUsers(chatID int64) {
	users, _ := h.userRepo.GetTopUsers(20)
	text := "👥 <b>TOP FOYDALANUVCHILAR</b>\n\n"
	for i, u := range users {
		text += fmt.Sprintf("%d. @%s — %d XP | %d 🎮\n", i+1, u.Username, u.XP, u.TotalGames)
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *AdminHandler) addCoins(msg *tgbotapi.Message) {
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 2 {
		h.send(msg.Chat.ID, "❌ /addcoins @username miqdor")
		return
	}

	username := strings.TrimPrefix(args[0], "@")
	amount, err := strconv.Atoi(args[1])
	if err != nil {
		h.send(msg.Chat.ID, "❌ Noto'g'ri miqdor")
		return
	}

	users, _ := h.userRepo.GetAllUsers()
	for _, u := range users {
		if strings.EqualFold(u.Username, username) {
			u.Coins += amount
			h.userRepo.Update(&u)
			h.send(msg.Chat.ID, fmt.Sprintf("✅ @%s ga <b>%d tanga</b> qo'shildi.\nJami: <b>%d tanga</b>",
				u.Username, amount, u.Coins))
			return
		}
	}
	h.send(msg.Chat.ID, "❌ Foydalanuvchi topilmadi")
}

func (h *AdminHandler) banUser(msg *tgbotapi.Message) {
	h.send(msg.Chat.ID, "🚧 Ban funksiyasi ishlab chiqilmoqda")
}

func (h *AdminHandler) broadcast(msg *tgbotapi.Message) {
	text := strings.TrimPrefix(msg.CommandArguments(), "broadcast ")
	if text == "" {
		h.send(msg.Chat.ID, "❌ Xabar matnini kiriting")
		return
	}

	users, _ := h.userRepo.GetAllUsers()
	sent := 0
	for _, u := range users {
		broadMsg := tgbotapi.NewMessage(u.TelegramID, "📢 <b>E'LON</b>\n\n"+text)
		broadMsg.ParseMode = "HTML"
		if _, err := h.bot.Send(broadMsg); err == nil {
			sent++
		}
	}
	h.send(msg.Chat.ID, fmt.Sprintf("✅ Xabar <b>%d</b> foydalanuvchiga yuborildi", sent))
}

func (h *AdminHandler) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

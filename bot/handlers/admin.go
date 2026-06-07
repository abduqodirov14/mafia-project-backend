package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"mafia-bot/db/repositories"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const SuperAdminID = int64(7972843834)

type AdminHandler struct {
	bot      *tgbotapi.BotAPI
	userRepo *repositories.UserRepository
}

func NewAdminHandler(bot *tgbotapi.BotAPI, userRepo *repositories.UserRepository) *AdminHandler {
	return &AdminHandler{bot: bot, userRepo: userRepo}
}

func (h *AdminHandler) IsAdmin(userID int64) bool {
	return userID == SuperAdminID
}

func (h *AdminHandler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	from := update.Message.From
	if !h.IsAdmin(from.ID) {
		return
	}
	switch update.Message.Command() {
	case "admin":
		h.showPanel(update.Message.Chat.ID)
	case "stats", "stat":
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
		h.showStatsCallback(query)
		return true
	case "admin_users":
		h.showUsersCallback(query)
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
	total := len(users)
	totalGames := 0
	totalWins := 0
	totalCoins := 0
	for _, u := range users {
		totalGames += int(u.TotalGames)
		totalWins += int(u.Wins)
		totalCoins += int(u.Coins)
	}

	text := fmt.Sprintf(
		"📊 <b>BATAFSIL STATISTIKA</b>\n\n"+
			"👥 Jami users: <b>%d</b>\n"+
			"🎮 Jami o'yinlar: <b>%d</b>\n"+
			"🏆 Jami g'alabalar: <b>%d</b>\n"+
			"💰 Jami tangalar: <b>%d</b>\n",
		total, totalGames, totalWins, totalCoins)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *AdminHandler) showStatsCallback(query *tgbotapi.CallbackQuery) {
	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	h.showStats(query.Message.Chat.ID)
}

func (h *AdminHandler) showUsersCallback(query *tgbotapi.CallbackQuery) {
	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	users, _ := h.userRepo.GetTopUsers(20)
	text := "👥 <b>TOP FOYDALANUVCHILAR</b>\n\n"
	for i, u := range users {
		text += fmt.Sprintf("%d. @%s — %d XP | %d 🎮\n", i+1, u.Username, u.XP, u.TotalGames)
	}
	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *AdminHandler) addCoins(msg *tgbotapi.Message) {
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 2 {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ /addcoins @username miqdor"))
		return
	}
	username := strings.TrimPrefix(args[0], "@")
	amount, err := strconv.Atoi(args[1])
	if err != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Noto'g'ri miqdor"))
		return
	}
	users, _ := h.userRepo.GetAllUsers()
	for _, u := range users {
		if strings.EqualFold(u.Username, username) {
			u.Coins += amount
			h.userRepo.Update(&u)
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
				fmt.Sprintf("✅ @%s ga <b>%d tanga</b> qo'shildi.\nJami: <b>%d tanga</b>",
					u.Username, amount, u.Coins)))
			return
		}
	}
	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Foydalanuvchi topilmadi"))
}

func (h *AdminHandler) banUser(msg *tgbotapi.Message) {
	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🚧 Ban funksiyasi ishlab chiqilmoqda"))
}

func (h *AdminHandler) broadcast(msg *tgbotapi.Message) {
	text := strings.TrimPrefix(msg.CommandArguments(), "broadcast ")
	if text == "" {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Xabar matnini kiriting"))
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
	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
		fmt.Sprintf("✅ Xabar <b>%d</b> foydalanuvchiga yuborildi", sent)))
}

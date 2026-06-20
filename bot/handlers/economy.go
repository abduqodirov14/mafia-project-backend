package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"mafia-bot/db/repositories"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type EconomyHandler struct {
	bot      *tgbotapi.BotAPI
	userRepo *repositories.UserRepository
}

func NewEconomyHandler(bot *tgbotapi.BotAPI, userRepo *repositories.UserRepository) *EconomyHandler {
	return &EconomyHandler{bot: bot, userRepo: userRepo}
}

func (h *EconomyHandler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	switch update.Message.Command() {
	case "money":
		h.handleMoney(update.Message)
	case "give":
		h.handleGive(update.Message)
	case "balance", "bal":
		h.handleBalance(update.Message)
	case "top", "leaderboard":
		h.handleTop(update.Message)
	}
}

func (h *EconomyHandler) handleMoney(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	if msg.ReplyToMessage == nil {
		h.send(chatID, "❌ <b>Foydalanish:</b>\nBirovga reply qilib: /money 100")
		return
	}

	args := strings.Fields(msg.CommandArguments())
	if len(args) == 0 {
		h.send(chatID, "❌ Miqdorni kiriting: /money 100")
		return
	}

	amount, err := strconv.Atoi(args[0])
	if err != nil || amount <= 0 {
		h.send(chatID, "❌ Noto'g'ri miqdor. Musbat son kiriting.")
		return
	}

	from := msg.From
	toUser := msg.ReplyToMessage.From

	if toUser.ID == from.ID {
		h.send(chatID, "❌ O'zingizga pul o'tkaza olmaysiz!")
		return
	}
	if toUser.IsBot {
		h.send(chatID, "❌ Botga pul o'tkaza olmaysiz!")
		return
	}

	sender, err := h.userRepo.GetOrCreate(from.ID, from.UserName, from.FirstName)
	if err != nil {
		return
	}
	if sender.Coins < amount {
		h.send(chatID, fmt.Sprintf("❌ Balansingiz yetarli emas!\n💰 Sizda: <b>%d tanga</b>", sender.Coins))
		return
	}

	sender.Coins -= amount
	h.userRepo.Update(sender)

	receiver, err := h.userRepo.GetOrCreate(toUser.ID, toUser.UserName, toUser.FirstName)
	if err != nil {
		return
	}
	receiver.Coins += amount
	h.userRepo.Update(receiver)

	h.send(chatID, fmt.Sprintf(
		"💸 <b>Muvaffaqiyatli o'tkazildi!</b>\n\n"+
			"👤 Jo'natuvchi: @%s\n"+
			"👤 Qabul qiluvchi: @%s\n"+
			"💰 Miqdor: <b>%d tanga</b>\n\n"+
			"💰 Sizda qoldi: <b>%d tanga</b>",
		from.UserName, toUser.UserName, amount, sender.Coins))
}

func (h *EconomyHandler) handleGive(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	if msg.ReplyToMessage == nil {
		h.send(chatID, "❌ <b>Foydalanish:</b>\nBirovga reply qilib: /give 10")
		return
	}

	args := strings.Fields(msg.CommandArguments())
	if len(args) == 0 {
		h.send(chatID, "❌ Miqdorni kiriting: /give 10")
		return
	}

	amount, err := strconv.Atoi(args[0])
	if err != nil || amount <= 0 {
		h.send(chatID, "❌ Noto'g'ri miqdor.")
		return
	}

	from := msg.From
	toUser := msg.ReplyToMessage.From

	sender, err := h.userRepo.GetOrCreate(from.ID, from.UserName, from.FirstName)
	if err != nil {
		return
	}

	cost := amount * 10
	if sender.Coins < cost {
		h.send(chatID, fmt.Sprintf("❌ Yetarli tanga yo'q!\n💰 Sizda: <b>%d tanga</b>\n💎 Kerak: <b>%d tanga</b>",
			sender.Coins, cost))
		return
	}

	sender.Coins -= cost
	h.userRepo.Update(sender)

	h.send(chatID, fmt.Sprintf(
		"💎 <b>Olmos o'tkazildi!</b>\n\n"+
			"👤 @%s → @%s\n"+
			"💎 Miqdor: <b>%d olmos</b>\n"+
			"💰 Sizdan: <b>%d tanga</b>",
		from.UserName, toUser.UserName, amount, cost))
}

func (h *EconomyHandler) handleBalance(msg *tgbotapi.Message) {
	user, err := h.userRepo.GetOrCreate(msg.From.ID, msg.From.UserName, msg.From.FirstName)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"💰 <b>Hisobingiz</b>\n\n"+
			"👤 @%s\n"+
			"──────────────\n"+
			"💰 Tanga: <b>%d</b>\n"+
			"🏆 Daraja: <b>%d</b>\n"+
			"⭐ XP: <b>%d</b>\n"+
			"──────────────\n"+
			"🎮 O'yinlar: <b>%d</b> | G'alabalar: <b>%d</b>",
		msg.From.UserName, user.Coins, user.Level, user.XP, user.TotalGames, user.Wins)

	balMsg := tgbotapi.NewMessage(msg.Chat.ID, text)
	balMsg.ParseMode = "HTML"
	balMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ Tanga sotib olish", "buy_coins_500"),
		),
	)
	h.bot.Send(balMsg)
}

func (h *EconomyHandler) handleTop(msg *tgbotapi.Message) {
	users, err := h.userRepo.GetTopUsers(10)
	if err != nil {
		return
	}

	text := "🏆 <b>TOP-10 O'YINCHILAR</b>\n\n"
	medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	for i, u := range users {
		text += fmt.Sprintf("%s @%s — <b>%d XP</b> | %d g'alaba\n", medals[i], u.Username, u.XP, u.Wins)
	}
	h.send(msg.Chat.ID, text)
}

func (h *EconomyHandler) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

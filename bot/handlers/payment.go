package handlers

import (
	"fmt"
	"log"
	"mafia-bot/db/repositories"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Product struct {
	ID          string
	Title       string
	Description string
	Emoji       string
	Stars       int
	Coins       int
}

var Products = []Product{
	{"coins_500",  "500 Tanga",     "💰", "O'yinda ishlatish uchun 500 tanga",    15,  500},
	{"coins_1500", "1500 Tanga",    "💎", "O'yinda ishlatish uchun 1500 tanga",   40,  1500},
	{"coins_5000", "5000 Tanga",    "👑", "O'yinda ishlatish uchun 5000 tanga",  120, 5000},
	{"xp_boost",   "2x XP (7 kun)", "⚡", "7 kun davomida 2 baravar XP yig'asiz", 50,  100},
	{"vip_badge",  "VIP Nishon",    "🏆", "VIP nishon va bonuslar",              100, 200},
}

type PaymentHandler struct {
	bot      *tgbotapi.BotAPI
	userRepo *repositories.UserRepository
}

func NewPaymentHandler(bot *tgbotapi.BotAPI, userRepo *repositories.UserRepository) *PaymentHandler {
	return &PaymentHandler{bot: bot, userRepo: userRepo}
}

func (h *PaymentHandler) Handle(update tgbotapi.Update) {
	if update.PreCheckoutQuery != nil {
		h.bot.Request(tgbotapi.PreCheckoutConfig{PreCheckoutQueryID: update.PreCheckoutQuery.ID, OK: true})
		return
	}
	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		h.handleSuccess(update.Message)
		return
	}
	if update.Message != nil && update.Message.IsCommand() {
		switch update.Message.Command() {
		case "buy", "shop_stars":
			h.showShop(update.Message.Chat.ID)
		case "donate":
			h.showDonate(update.Message.Chat.ID)
		}
	}
}

func (h *PaymentHandler) showShop(chatID int64) {
	text := "🛍 <b>STARS DO'KONI</b>\n\nTelegram Stars orqali xarid qiling:\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range Products {
		text += fmt.Sprintf("%s <b>%s</b> — %d ⭐\n%s\n\n", p.Emoji, p.Title, p.Stars, p.Description)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %s — %d ⭐", p.Emoji, p.Title, p.Stars),
				"buy_"+p.ID,
			),
		))
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.bot.Send(msg)
}

func (h *PaymentHandler) showDonate(chatID int64) {
	msg := tgbotapi.NewMessage(chatID,
		"❤️ <b>BOTNI QO'LLAB-QUVVATLASH</b>\n\nBotni rivojlantirish uchun yordam bering!\nTelegram Stars orqali xarid qiling 🙏")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ 15 Stars", "buy_coins_500"),
			tgbotapi.NewInlineKeyboardButtonData("⭐ 40 Stars", "buy_coins_1500"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👑 120 Stars", "buy_coins_5000"),
		),
	)
	h.bot.Send(msg)
}

func (h *PaymentHandler) SendInvoice(chatID int64, productID string) {
	var p *Product
	for i := range Products {
		if Products[i].ID == productID { p = &Products[i]; break }
	}
	if p == nil { return }
	invoice := tgbotapi.InvoiceConfig{
		BaseChat:             tgbotapi.BaseChat{ChatID: chatID},
		Title:                p.Emoji + " " + p.Title,
		Description:          p.Description,
		Payload:              p.ID,
		Currency:             "XTR",
		Prices:               []tgbotapi.LabeledPrice{{Label: p.Title, Amount: p.Stars}},
		SuggestedTipAmounts: []int{p.Stars},
	}
	if _, err := h.bot.Send(invoice); err != nil {
		log.Printf("Invoice xato: %v", err)
	}
}

func (h *PaymentHandler) handleSuccess(msg *tgbotapi.Message) {
	pay := msg.SuccessfulPayment
	user, err := h.userRepo.GetOrCreate(msg.From.ID, msg.From.UserName, msg.From.FirstName)
	if err != nil { return }
	var text string
	for _, p := range Products {
		if p.ID == pay.InvoicePayload {
			user.Coins += p.Coins
			h.userRepo.Update(user)
			text = fmt.Sprintf("✅ <b>%s %s</b> sotib olindi!\n💰 Hisobingiz: <b>%d tanga</b>",
				p.Emoji, p.Title, user.Coins)
			break
		}
	}
	if text == "" { text = "✅ Xarid muvaffaqiyatli!" }
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "HTML"
	h.bot.Send(reply)
}

func (h *PaymentHandler) HandleCallback(query *tgbotapi.CallbackQuery) bool {
	if len(query.Data) < 4 || query.Data[:4] != "buy_" { return false }
	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	h.SendInvoice(query.Message.Chat.ID, query.Data[4:])
	return true
}

package handlers

import (
	"fmt"
	"log"
	"mafia-bot/db/repositories"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Mahsulotlar ro'yxati (Telegram Stars bilan)
type Product struct {
	ID          string
	Title       string
	Description string
	Emoji       string
	Stars       int // Telegram Stars narxi
	CoinsGiven  int
}

var Products = []Product{
	{ID: "coins_500",  Title: "500 Tanga",      Emoji: "💰", Description: "O'yinda teri va buyumlar uchun 500 tanga",    Stars: 15,  CoinsGiven: 500},
	{ID: "coins_1500", Title: "1500 Tanga",     Emoji: "💎", Description: "O'yinda teri va buyumlar uchun 1500 tanga",   Stars: 40,  CoinsGiven: 1500},
	{ID: "coins_5000", Title: "5000 Tanga",     Emoji: "👑", Description: "O'yinda teri va buyumlar uchun 5000 tanga",   Stars: 120, CoinsGiven: 5000},
	{ID: "xp_boost",   Title: "2x XP (7 kun)",  Emoji: "⚡", Description: "1 hafta davomida 2 baravar XP yig'asiz",      Stars: 50,  CoinsGiven: 0},
	{ID: "vip_badge",  Title: "VIP Nishon",      Emoji: "🏆", Description: "Profilingizda VIP nishon va maxsus ranglar", Stars: 100, CoinsGiven: 200},
}

type PaymentHandler struct {
	bot      *tgbotapi.BotAPI
	userRepo *repositories.UserRepository
}

func NewPaymentHandler(bot *tgbotapi.BotAPI, userRepo *repositories.UserRepository) *PaymentHandler {
	return &PaymentHandler{bot: bot, userRepo: userRepo}
}

// Handle — barcha to'lov hodisalarini boshqaradi
func (h *PaymentHandler) Handle(update tgbotapi.Update) {
	// 1. To'lov tasdiqlash (pre_checkout)
	if update.PreCheckoutQuery != nil {
		h.handlePreCheckout(update.PreCheckoutQuery)
		return
	}
	// 2. Muvaffaqiyatli to'lov
	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		h.handleSuccess(update.Message)
		return
	}
	// 3. Buyruqlar
	if update.Message != nil && update.Message.IsCommand() {
		switch update.Message.Command() {
		case "buy", "shop_stars":
			h.showShop(update.Message.Chat.ID)
		case "donate":
			h.showDonate(update.Message.Chat.ID)
		}
	}
}

// Do'kon ko'rsatish
func (h *PaymentHandler) showShop(chatID int64) {
	text := "🛍 <b>DO'KON — Telegram Stars</b>\n\n" +
		"Telegram Stars orqali xarid qilishingiz mumkin:\n\n"

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range Products {
		text += fmt.Sprintf("%s <b>%s</b> — %d ⭐\n%s\n\n", p.Emoji, p.Title, p.Stars, p.Description)
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %s — %d ⭐", p.Emoji, p.Title, p.Stars),
				"buy_"+p.ID,
			),
		)
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.bot.Send(msg)
}

// Xarid uchun invoice yuborish
func (h *PaymentHandler) SendInvoice(chatID int64, productID string) {
	var product *Product
	for i := range Products {
		if Products[i].ID == productID {
			product = &Products[i]
			break
		}
	}
	if product == nil {
		return
	}

	invoice := tgbotapi.InvoiceConfig{
		BaseChat:    tgbotapi.BaseChat{ChatID: chatID},
		Title:       product.Emoji + " " + product.Title,
		Description: product.Description,
		Payload:     product.ID,
		Currency:    "XTR", // Telegram Stars
		Prices: []tgbotapi.LabeledPrice{
			{Label: product.Title, Amount: product.Stars},
		},
	}

	if _, err := h.bot.Send(invoice); err != nil {
		log.Printf("Invoice xato: %v", err)
	}
}

// Pre-checkout: doim tasdiqlash
func (h *PaymentHandler) handlePreCheckout(q *tgbotapi.PreCheckoutQuery) {
	cfg := tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: q.ID,
		OK:                 true,
	}
	h.bot.Request(cfg)
}

// Muvaffaqiyatli to'lovni qayta ishlash
func (h *PaymentHandler) handleSuccess(msg *tgbotapi.Message) {
	pay := msg.SuccessfulPayment
	userID := msg.From.ID

	log.Printf("✅ To'lov: user=%d, product=%s, stars=%d", userID, pay.InvoicePayload, pay.TotalAmount)

	user, err := h.userRepo.GetOrCreate(userID, "", "")
	if err != nil {
		log.Printf("User topilmadi: %v", err)
		return
	}

	var thankMsg string

	switch pay.InvoicePayload {
	case "coins_500":
		user.Coins += 500
		thankMsg = "✅ 500 tanga hisobingizga qo'shildi!"
	case "coins_1500":
		user.Coins += 1500
		thankMsg = "✅ 1500 tanga hisobingizga qo'shildi!"
	case "coins_5000":
		user.Coins += 5000
		thankMsg = "✅ 5000 tanga hisobingizga qo'shildi!"
	case "xp_boost":
		thankMsg = "⚡ 2x XP 7 kun davomida faollashtirildi!"
		// TODO: XP boost logicini qo'shish
	case "vip_badge":
		thankMsg = "👑 VIP nishon qo'shildi! /profile ni tekshiring"
		// TODO: VIP badge logicini qo'shish
	default:
		thankMsg = "✅ Xarid muvaffaqiyatli!"
	}

	h.userRepo.Update(user)

	reply := tgbotapi.NewMessage(msg.Chat.ID, thankMsg)
	h.bot.Send(reply)
}

// Callback: "buy_coins_500" kabi
func (h *PaymentHandler) HandleCallback(query *tgbotapi.CallbackQuery) bool {
	data := query.Data
	if len(data) < 4 || data[:4] != "buy_" {
		return false
	}
	productID := data[4:]
	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	h.SendInvoice(query.Message.Chat.ID, productID)
	return true
}

func (h *PaymentHandler) showDonate(chatID int64) {
	msg := tgbotapi.NewMessage(chatID,
		"❤️ <b>BOTNI QO'LLAB-QUVVATLASH</b>\n\n"+
			"Agar bot sizga yoqqan bo'lsa, rivojlantirish uchun yordam bering:\n\n"+
			"⭐ Telegram Stars orqali istalgan miqdor yuborishingiz mumkin\n\n"+
			"Rahmat! 🙏")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ 50 Stars yuborish", "buy_coins_500"),
			tgbotapi.NewInlineKeyboardButtonData("⭐ 100 Stars yuborish", "buy_coins_1500"),
		),
	)
	h.bot.Send(msg)
}

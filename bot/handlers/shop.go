package handlers

import (
	"fmt"
	"mafia-bot/config"
	"mafia-bot/db/models"
	"mafia-bot/db/repositories"
	"strconv"
	"strings"

	"gorm.io/gorm"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ShopHandler struct {
	bot      *tgbotapi.BotAPI
	userRepo *repositories.UserRepository
	db       *gorm.DB
}

func NewShopHandler(bot *tgbotapi.BotAPI, userRepo *repositories.UserRepository, db *gorm.DB) *ShopHandler {
	return &ShopHandler{bot: bot, userRepo: userRepo, db: db}
}

func (h *ShopHandler) Handle(update tgbotapi.Update) {
	if update.Message != nil {
		switch update.Message.Command() {
		case "shop":
			h.handleShop(update)
		case "inventory":
			h.handleInventory(update)
		}
	}

	if update.CallbackQuery != nil {
		data := update.CallbackQuery.Data
		if strings.HasPrefix(data, "buy_") {
			h.handleBuy(update)
		}
	}
}

func (h *ShopHandler) handleShop(update tgbotapi.Update) {
	from := update.Message.From
	chatID := update.Message.Chat.ID

	user, err := h.userRepo.GetOrCreate(from.ID, from.UserName, from.FirstName)
	if err != nil {
		return
	}

	var items []models.Item
	h.db.Find(&items)

	text := fmt.Sprintf(config.MsgShopHeader, user.Coins)
	text += "\n\n"

	rarityEmoji := map[models.Rarity]string{
		models.RarityCommon: "⚪",
		models.RarityRare:   "🔵",
		models.RarityEpic:   "🟣",
		models.RarityLegend: "🟡",
	}

	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, item := range items {
		owns := h.userRepo.HasItem(user.ID, item.ID)
		status := ""
		if owns {
			status = " ✅"
		}

		text += fmt.Sprintf("%s %s <b>%s</b>%s\n💰 %d tanga — %s\n\n",
			item.Emoji,
			rarityEmoji[item.Rarity],
			item.Name,
			status,
			item.Price,
			item.Description,
		)

		if !owns {
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("💰 %s sotib ol (%d)", item.Name, item.Price),
					fmt.Sprintf("buy_%d", item.ID),
				),
			)
			keyboard = append(keyboard, row)
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if len(keyboard) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	}
	h.bot.Send(msg)
}

func (h *ShopHandler) handleBuy(update tgbotapi.Update) {
	callback := update.CallbackQuery
	from := callback.From

	parts := strings.Split(callback.Data, "_")
	if len(parts) < 2 {
		return
	}

	itemID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return
	}

	user, err := h.userRepo.GetOrCreate(from.ID, from.UserName, from.FirstName)
	if err != nil {
		return
	}

	var item models.Item
	if err := h.db.First(&item, itemID).Error; err != nil {
		return
	}

	if h.userRepo.HasItem(user.ID, uint(itemID)) {
		h.bot.Request(tgbotapi.NewCallback(callback.ID, config.MsgAlreadyOwned))
		return
	}

	if user.Coins < item.Price {
		h.bot.Request(tgbotapi.NewCallback(callback.ID,
			fmt.Sprintf("❌ Yetarli tanga yo'q! Kerak: %d, Sizda: %d", item.Price, user.Coins)))
		return
	}

	if err := h.userRepo.BuyItem(user.ID, uint(itemID), item.Price); err != nil {
		return
	}

	h.bot.Request(tgbotapi.NewCallback(callback.ID,
		fmt.Sprintf("✅ %s sotib olindi!", item.Name)))
}

func (h *ShopHandler) handleInventory(update tgbotapi.Update) {
	from := update.Message.From
	chatID := update.Message.Chat.ID

	user, err := h.userRepo.GetOrCreate(from.ID, from.UserName, from.FirstName)
	if err != nil {
		return
	}

	items, err := h.userRepo.GetUserItems(user.ID)
	if err != nil || len(items) == 0 {
		msg := tgbotapi.NewMessage(chatID, "🎒 Inventaringiz bo'sh. /shop dan narsa sotib oling!")
		h.bot.Send(msg)
		return
	}

	text := "🎒 <b>INVENTAR</b>\n\n"
	for _, item := range items {
		text += fmt.Sprintf("%s <b>%s</b> — %s\n", item.Emoji, item.Name, item.Description)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

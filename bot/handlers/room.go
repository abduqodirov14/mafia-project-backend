package handlers

import (
	"encoding/json"
	"fmt"
	"mafia-bot/config"
	"mafia-bot/db/repositories"
	"mafia-bot/game"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type webAppBtn struct {
	Text   string     `json:"text"`
	WebApp struct {
		URL string `json:"url"`
	} `json:"web_app"`
}
type normalBtn struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}
type inlineKbd struct {
	InlineKeyboard [][]json.RawMessage `json:"inline_keyboard"`
}

func makeKbd(webURL, webText, btnText, btnData string) interface{} {
	var b webAppBtn
	b.Text = webText
	b.WebApp.URL = webURL
	wb, _ := json.Marshal(b)
	nb, _ := json.Marshal(normalBtn{Text: btnText, CallbackData: btnData})
	return inlineKbd{InlineKeyboard: [][]json.RawMessage{{wb}, {nb}}}
}

type RoomHandler struct {
	bot       *tgbotapi.BotAPI
	manager   *game.Manager
	userRepo  *repositories.UserRepository
	webAppURL string
}

func NewRoomHandler(bot *tgbotapi.BotAPI, manager *game.Manager, userRepo *repositories.UserRepository, webAppURL string) *RoomHandler {
	return &RoomHandler{bot: bot, manager: manager, userRepo: userRepo, webAppURL: webAppURL}
}

func (h *RoomHandler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	switch update.Message.Command() {
	case "newroom":
		h.handleNewRoom(update)
	case "join":
		h.handleJoin(update)
	case "startgame":
		h.handleStartGame(update)
	case "leave":
		h.handleLeave(update)
	}
}

func (h *RoomHandler) handleNewRoom(update tgbotapi.Update) {
	from := update.Message.From
	chatID := update.Message.Chat.ID
	room := h.manager.CreateRoom(chatID, from.ID, from.UserName)

	botInfo, _ := h.bot.GetMe()
	refLink := fmt.Sprintf("https://t.me/%s?start=ref_%s", botInfo.UserName, room.ID)

	text := fmt.Sprintf("🏠 <b>Xona yaratildi!</b>\n\nXona ID: <code>%s</code>\nO'yinchilar: %d/%d\n\n📎 Taklif:\n%s",
		room.ID, room.PlayerCount(), room.MaxPlayer, refLink)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	if h.webAppURL != "" {
		msg.ReplyMarkup = makeKbd(
			h.webAppURL+"?room="+room.ID,
			"🎮 O'yinga kirish (WebApp)",
			"▶️ O'yinni boshlash",
			"startgame_"+room.ID,
		)
	} else {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("▶️ O'yinni boshlash", "startgame_"+room.ID),
			),
		)
	}
	h.bot.Send(msg)
}

func (h *RoomHandler) handleJoin(update tgbotapi.Update) {
	args := strings.Fields(update.Message.CommandArguments())
	chatID := update.Message.Chat.ID
	from := update.Message.From

	if len(args) == 0 {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ /join 123456"))
		return
	}

	roomID := args[0]
	player := &game.Player{TelegramID: from.ID, Username: from.UserName, FirstName: from.FirstName, IsAlive: true}

	if err := h.manager.JoinRoom(roomID, player); err != nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ "+err.Error()))
		return
	}

	room := h.manager.GetRoom(roomID)
	text := fmt.Sprintf(config.MsgJoinedRoom, from.UserName, room.PlayerCount(), room.MaxPlayer)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	if h.webAppURL != "" {
		var wb webAppBtn
		wb.Text = "🎮 O'yinga kirish"
		wb.WebApp.URL = h.webAppURL + "?room=" + roomID
		wbj, _ := json.Marshal(wb)
		msg.ReplyMarkup = inlineKbd{InlineKeyboard: [][]json.RawMessage{{wbj}}}
	}
	h.bot.Send(msg)
}

func (h *RoomHandler) handleStartGame(update tgbotapi.Update) {
	from := update.Message.From
	chatID := update.Message.Chat.ID
	room := h.manager.GetRoomByUser(from.ID)
	if room == nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ Siz hech qaysi xonada emassiz!"))
		return
	}
	if room.OwnerID != from.ID {
		h.bot.Send(tgbotapi.NewMessage(chatID, config.MsgNotOwner))
		return
	}
	if err := h.manager.StartGame(room.ID); err != nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ "+err.Error()))
		return
	}
	h.bot.Send(tgbotapi.NewMessage(chatID, config.MsgGameStarted))
}

func (h *RoomHandler) handleLeave(update tgbotapi.Update) {
	from := update.Message.From
	chatID := update.Message.Chat.ID
	room := h.manager.GetRoomByUser(from.ID)
	if room == nil {
		h.bot.Send(tgbotapi.NewMessage(chatID, "❌ Xonada emassiz!"))
		return
	}
	room.RemovePlayer(from.ID)
	h.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("👋 %s xonadan chiqdi.", from.UserName)))
}

package handlers

import (
	"fmt"
	"strings"

	"mafia-bot/db/repositories"
	"mafia-bot/game"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StartHandler struct {
	bot         *tgbotapi.BotAPI
	userRepo    *repositories.UserRepository
	manager     *game.Manager
	webAppURL   string
	botUsername string
}

func NewStartHandler(bot *tgbotapi.BotAPI, userRepo *repositories.UserRepository, manager *game.Manager, webAppURL, botUsername string) *StartHandler {
	return &StartHandler{bot: bot, userRepo: userRepo, manager: manager, webAppURL: webAppURL, botUsername: botUsername}
}

func (h *StartHandler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	from := msg.From

	// DB ga saqlash
	user, _ := h.userRepo.GetOrCreate(from.ID, from.UserName, from.FirstName)
	_ = user

	switch msg.Command() {
	case "start":
		h.handleStart(msg)
	case "help":
		h.handleHelp(msg)
	case "profile", "me":
		h.handleProfile(msg)
	case "rating", "top":
		h.handleRating(msg)
	case "rules":
		h.handleRules(msg)
	case "roles":
		h.handleRoles(msg)
	case "support":
		h.handleSupport(msg)
	case "testgame":
		h.handleTestGame(msg)
	}
}

func (h *StartHandler) handleStart(msg *tgbotapi.Message) {
	from := msg.From
	args := msg.CommandArguments()

	// Referral / xonaga qo'shilish
	if strings.HasPrefix(args, "ref_") {
		roomID := strings.TrimPrefix(args, "ref_")
		h.joinRoomByRef(msg, roomID)
		return
	}

	name := from.FirstName
	if from.UserName != "" {
		name = "@" + from.UserName
	}

	text := fmt.Sprintf(
		"🎭 <b>MAFIA GAME</b> ga xush kelibsiz, %s!\n\n"+
			"🎮 Bu klassik Mafia o'yinining Telegram versiyasi!\n\n"+
			"<b>Qanday o'ynash:</b>\n"+
			"1️⃣ Guruhingizga botni qo'shing\n"+
			"2️⃣ /start yuboring — o'yin boshlanadi\n"+
			"3️⃣ Do'stlaringiz qo'shiladi\n"+
			"4️⃣ Rollar tarqatiladi — o'ynang!\n\n"+
			"<b>Yoki WebApp orqali o'ynang:</b>", name)

	outMsg := tgbotapi.NewMessage(msg.Chat.ID, text)
	outMsg.ParseMode = "HTML"

	var rows [][]tgbotapi.InlineKeyboardButton
	if h.webAppURL != "" {
		webBtn := tgbotapi.InlineKeyboardButton{
			Text: "🎮 O'yinni boshlash (WebApp)",
			URL:  &h.webAppURL,
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(webBtn))
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📖 Qoidalar", "rules"),
			tgbotapi.NewInlineKeyboardButtonData("👥 Rollar", "roles_info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Profilim", "my_profile"),
			tgbotapi.NewInlineKeyboardButtonData("🏆 Reyting", "top_players"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ Do'kon", "shop_main"),
			tgbotapi.NewInlineKeyboardButtonData("🆘 Yordam", "support_main"),
		),
	)
	outMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.bot.Send(outMsg)
}

func (h *StartHandler) joinRoomByRef(msg *tgbotapi.Message, roomID string) {
	from := msg.From
	room := h.manager.GetRoom(roomID)
	if room == nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Xona topilmadi yoki o'yin tugagan."))
		return
	}
	player := &game.Player{
		TelegramID: from.ID,
		Username:   from.UserName,
		FirstName:  from.FirstName,
		IsAlive:    true,
	}
	if err := h.manager.JoinRoom(roomID, player); err != nil {
		h.send(msg.Chat.ID, "⚠️ "+err.Error())
		return
	}
	text := fmt.Sprintf("✅ Xonaga qo'shildingiz!\nXona ID: <code>%s</code>\nO'yinchilar: <b>%d/%d</b>",
		roomID, room.PlayerCount(), room.MaxPlayers)
	outMsg := tgbotapi.NewMessage(msg.Chat.ID, text)
	outMsg.ParseMode = "HTML"
	if h.webAppURL != "" {
		joinURL := h.webAppURL + "?room=" + roomID
		webBtn := tgbotapi.InlineKeyboardButton{
			Text: "🎮 O'yinga kirish (WebApp)",
			URL:  &joinURL,
		}
		outMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(webBtn),
		)
	}
	h.bot.Send(outMsg)
}

func (h *StartHandler) handleHelp(msg *tgbotapi.Message) {
	text := `🎭 <b>MAFIA GAME — YORDAM</b>

<b>Asosiy buyruqlar:</b>
/start — Botni ishga tushirish
/profile — Profilingiz
/rating — TOP o'yinchilar
/rules — O'yin qoidalari
/roles — Rollar haqida

<b>Guruhda:</b>
/start — O'yinni boshlash
/+ — O'yinga qo'shilish
/- — O'yindan chiqish
/stat — O'yin holati

<b>Iqtisodiyot:</b>
/balance — Hisobingiz
/money [miqdor] — Pul o'tkazish (reply)
/give [miqdor] — Olmos o'tkazish (reply)
/buy — Stars do'koni

<b>Qo'llab-quvvatlash:</b>
/support — Yordam markazi`

	h.send(msg.Chat.ID, text)
}

func (h *StartHandler) handleProfile(msg *tgbotapi.Message) {
	from := msg.From
	user, err := h.userRepo.GetOrCreate(from.ID, from.UserName, from.FirstName)
	if err != nil {
		return
	}
	winRate := 0.0
	if user.TotalGames > 0 {
		winRate = float64(user.Wins) / float64(user.TotalGames) * 100
	}
	text := fmt.Sprintf(
		"👤 <b>PROFIL</b>\n\n"+
			"📛 Ism: @%s\n"+
			"🏅 Liga: %s\n"+
			"⭐ Daraja: <b>%d</b>\n"+
			"📊 XP: <b>%d</b>\n\n"+
			"🎮 O'yinlar: <b>%d</b>\n"+
			"🏆 G'alabalar: <b>%d</b>\n"+
			"📈 G'alaba ulushi: <b>%.1f%%</b>\n"+
			"🔥 Seriya: <b>%d</b>\n\n"+
			"💰 Tangalar: <b>%d</b>",
		user.Username, string(user.League), user.Level, user.XP,
		user.TotalGames, user.Wins, winRate, user.WinStreak, user.Coins)

	profMsg := tgbotapi.NewMessage(msg.Chat.ID, text)
	profMsg.ParseMode = "HTML"
	profMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ Tanga sotib olish", "buy_coins_500"),
		),
	)
	h.bot.Send(profMsg)
}

func (h *StartHandler) handleRating(msg *tgbotapi.Message) {
	users, err := h.userRepo.GetTopUsers(10)
	if err != nil {
		return
	}
	text := "🏆 <b>TOP-10 O'YINCHILAR</b>\n\n"
	medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	for i, u := range users {
		text += fmt.Sprintf("%s @%s — <b>%d XP</b> | %d 🎮\n", medals[i], u.Username, u.XP, u.TotalGames)
	}
	h.send(msg.Chat.ID, text)
}

func (h *StartHandler) handleRules(msg *tgbotapi.Message) {
	text := `🎭 <b>O'YIN QOIDALARI</b>

🌙 <b>Tun fazasi:</b>
• Mafia qurbon tanlaydi
• Shifokor kimnidir davolaydi
• Komissar kimnidir tekshiradi
• Boshqa rollar ham harakat qiladi

☀️ <b>Kun fazasi:</b>
• 90 soniya muhokama
• Mafiyani ovoz bilan chiqaring

🏆 <b>G'alaba sharti:</b>
• <b>Shahar:</b> Barcha mafia o'lsa
• <b>Mafia:</b> Mafia soni ≥ shahar soni

💡 <b>Maslahat:</b>
Rollar shaxsiy xabarda yuboriladi.
WebApp orqali osonroq o'ynang!`

	h.send(msg.Chat.ID, text)
}

func (h *StartHandler) handleRoles(msg *tgbotapi.Message) {
	text := `🎭 <b>ROLLAR HAQIDA</b>

🔵 <b>Shahar tomonlari:</b>
👨🏼 Tinch aholi — Mafia topadi
👨‍⚕️ Shifokor — Birini davolaydi
🕵🏼 Komissar — Birini tekshiradi
👮‍♂️ Serjant — Komissar o'lsa o'rnini oladi
🧙🏻‍♂️ Daydi — Kechasi kimlar kelganini ko'radi
💃 Mashuqa — Birini bloklaydi

🔴 <b>Mafia tomonlari:</b>
🤵🏻 Don — Mafia boshlig'i
🤵🏼 Mafiya — Don ko'rsatmasini bajaradi
👨‍💼 Advokat — Komissardan yashiradi

⚫ <b>Yakka rollar:</b>
🔪 Manyak — Hammani o'ldiradi
🧌 Suidsid — Ovoz bilan chiqarilsa yutadi
💣 Kamikaze — O'lsa bitta odamni olib ketadi`

	h.send(msg.Chat.ID, text)
}

func (h *StartHandler) handleSupport(msg *tgbotapi.Message) {
	text := fmt.Sprintf(
		"🆘 <b>YORDAM MARKAZI</b>\n\n"+
			"Savolingiz bormi? Admin bilan bog'laning:\n"+
			"📞 @abduqodirov_14\n\n"+
			"<b>Ko'p so'raladigan savollar:</b>\n\n"+
			"❓ O'yin qanday boshlanadi?\n"+
			"➡️ Guruhda /start yuboring\n\n"+
			"❓ Kamida nechta kishi kerak?\n"+
			"➡️ 4 ta o'yinchi\n\n"+
			"❓ WebApp ishlamayapti?\n"+
			"➡️ Telegramni yangilang\n\n"+
			"❓ Tangalar qanday olinadi?\n"+
			"➡️ O'yin o'ynang va g'alaba qozongan @%s", h.botUsername)

	suppMsg := tgbotapi.NewMessage(msg.Chat.ID, text)
	suppMsg.ParseMode = "HTML"
	suppMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📞 Admin", "https://t.me/abduqodirov_14"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ Botni qo'llab-quvvatlash", "buy_coins_500"),
		),
	)
	h.bot.Send(suppMsg)
}

func (h *StartHandler) handleTestGame(msg *tgbotapi.Message) {
	from := msg.From
	chatID := msg.Chat.ID

	room := h.manager.CreateRoom(chatID, from.ID, from.UserName)
	bots := []struct{ id int64; name string }{
		{-1001, "🤖 Sardor"}, {-1002, "🤖 Dilnoza"},
		{-1003, "🤖 Komil"},  {-1004, "🤖 Shaxboz"},
	}
	for _, b := range bots {
		h.manager.JoinRoom(room.ID, &game.Player{TelegramID: b.id, Username: b.name, IsAlive: true})
	}
	h.send(chatID, fmt.Sprintf("🧪 <b>TEST O'YIN</b>\nXona: <code>%s</code>\n5 o'yinchi (siz + 4 bot)\n\nRollar yuborilmoqda...", room.ID))
	if err := h.manager.StartGame(room.ID); err != nil {
		h.send(chatID, "❌ "+err.Error())
	}
}

func (h *StartHandler) HandleCallback(query *tgbotapi.CallbackQuery) bool {
	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	switch query.Data {
	case "rules":
		h.send(query.Message.Chat.ID, "Qoidalarni /rules buyrug'i bilan ko'ring")
		return true
	case "roles_info":
		msg := &tgbotapi.Message{Chat: query.Message.Chat, From: query.From}
		h.handleRoles(msg)
		return true
	case "my_profile":
		msg := &tgbotapi.Message{Chat: query.Message.Chat, From: query.From}
		h.handleProfile(msg)
		return true
	case "top_players":
		msg := &tgbotapi.Message{Chat: query.Message.Chat, From: query.From}
		h.handleRating(msg)
		return true
	case "support_main":
		msg := &tgbotapi.Message{Chat: query.Message.Chat, From: query.From}
		h.handleSupport(msg)
		return true
	}
	return false
}

func (h *StartHandler) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

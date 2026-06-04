package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func MainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎮 Yangi xona", "newroom"),
			tgbotapi.NewInlineKeyboardButtonData("🚪 Xonaga kirish", "joinroom"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Profil", "profile"),
			tgbotapi.NewInlineKeyboardButtonData("🛒 Do'kon", "shop"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏆 Reyting", "rating"),
		),
	)
}

func StartGame() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("▶️ O'yinni boshlash", "startgame"),
		),
	)
}

func PlayerVote(players []struct{ ID int64; Name string }) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range players {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"👤 "+p.Name,
				"vote_"+string(rune(p.ID)),
			),
		)
		rows = append(rows, row)
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func ShopCategories() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎭 Personajlar", "shop_character"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎒 Aksessuarlar", "shop_accessory"),
		),
	)
}

func RatingTabs(active string) tgbotapi.InlineKeyboardMarkup {
	weekBtn := "Haftalik"
	monthBtn := "Oylik"
	allBtn := "Barcha vaqt"

	if active == "weekly" {
		weekBtn = "✅ Haftalik"
	} else if active == "monthly" {
		monthBtn = "✅ Oylik"
	} else {
		allBtn = "✅ Barcha vaqt"
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(weekBtn, "rating_weekly"),
			tgbotapi.NewInlineKeyboardButtonData(monthBtn, "rating_monthly"),
			tgbotapi.NewInlineKeyboardButtonData(allBtn, "rating_all"),
		),
	)
}

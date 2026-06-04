package voice

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MutePlayer(bot *tgbotapi.BotAPI, chatID, userID int64) error {
	config := tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		Permissions: &tgbotapi.ChatPermissions{
			CanSendMessages: false,
		},
	}
	_, err := bot.Request(config)
	return err
}

func UnmutePlayer(bot *tgbotapi.BotAPI, chatID, userID int64) error {
	config := tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		Permissions: &tgbotapi.ChatPermissions{
			CanSendMessages: true,
		},
	}
	_, err := bot.Request(config)
	return err
}

func MuteAll(bot *tgbotapi.BotAPI, chatID int64, playerIDs []int64) {
	for _, id := range playerIDs {
		MutePlayer(bot, chatID, id)
	}
}

func UnmuteAll(bot *tgbotapi.BotAPI, chatID int64, playerIDs []int64) {
	for _, id := range playerIDs {
		UnmutePlayer(bot, chatID, id)
	}
}

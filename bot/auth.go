package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/reddit-bot/config"
)

func auth(update tgbotapi.Update) bool {
	id := update.Message.Chat.ID

	for _, user := range config.Users {
		if user == id {
			return true
		}
	}

	return false
}

package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/reddit-bot/config"
)

func auth(update tgbotapi.Update) bool {
	var id int64
	if from := update.Message.From; from != nil {
		id = from.ID
	} else if from := update.CallbackQuery.From; from != nil {
		id = from.ID
	}

	for _, user := range config.Users {
		if user == id {
			return true
		}
	}

	return false
}

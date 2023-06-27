package main

import (
	"github.com/mariownyou/reddit-bot/bot"
	"github.com/mariownyou/reddit-bot/config"
)

func main() {
	b, err := bot.NewBot(config.TelegramToken)
	if err != nil {
		panic(err)
	}

	b.Debug = config.Debug

	manager := bot.NewManager(*b)
	manager.Construct()
	manager.Run(bot.AuthMiddleware)
}

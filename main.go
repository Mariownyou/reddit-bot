package main

import (
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	RedditID       string
	RedditSecret   string
	RedditUsername string
	RedditPassword string
	ImgurClientID  string
	TelegramToken  string
	Subreddits     []string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	RedditID = os.Getenv("REDDIT_CLIENT_ID")
	RedditSecret = os.Getenv("REDDIT_CLIENT_SECRET")
	RedditUsername = os.Getenv("REDDIT_USERNAME")
	RedditPassword = os.Getenv("REDDIT_PASSWORD")
	ImgurClientID = os.Getenv("IMGUR_CLIENT_ID")
	TelegramToken = os.Getenv("TELEGRAM_TOKEN")
	if subreddits := os.Getenv("SUBREDDITS"); subreddits != "" {
		Subreddits = strings.Split(subreddits, ",")
	} else {
		Subreddits = []string{"test"}
	}
}

func main() {
	bot, err := NewBot(TelegramToken)
	if err != nil {
		panic(err)
	}

	bot.Debug = true
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		bot.UpdateHandler(update)
	}
}

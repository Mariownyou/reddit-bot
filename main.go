package main

import (
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	Debug          bool
	RedditID       string
	RedditSecret   string
	RedditUsername string
	RedditPassword string
	ImgurClientID  string
	TelegramToken  string
	Subreddits     []string
	Users          []int64
)

func init() {
	// check if .env file exists
	_, err := os.Stat(".env")
	if err == nil {
		err := godotenv.Load()
		if err != nil {
			panic(err)
		}
	}

	if os.Getenv("DEBUG") == "true" {
		Debug = true
	} else {
		Debug = false
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

	if users := os.Getenv("USERS"); users != "" {
		for _, user := range strings.Split(users, ",") {
			usr, _ := strconv.Atoi(user)
			Users = append(Users, int64(usr))
		}
	} else {
		Users = []int64{0}
	}
}

func main() {
	bot, err := NewBot(TelegramToken)
	if err != nil {
		panic(err)
	}

	bot.Debug = Debug
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		bot.UpdateHandler(update)
	}
}

package main

import (
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/mariownyou/reddit-bot/fsm"
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

	BOT     *Bot
	MANAGER *fsm.Manager
)

const EnvFile = ".env.local"

func init() {
	// check if .env file exists
	_, err := os.Stat(EnvFile)
	if err == nil {
		err := godotenv.Load(EnvFile)
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
	var err error

	BOT, err = NewBot(TelegramToken)
	if err != nil {
		panic(err)
	}

	MANAGER = fsm.NewManager(BOT.BotAPI)

	// Submit post states
	MANAGER.Handle(fsm.OnPhoto, fsm.DefaultState, postHandler)
	MANAGER.Handle(fsm.OnVideo, fsm.DefaultState, postHandler)

	MANAGER.Handle(fsm.OnText, fsm.AwaitFlairMessageState, awaitFlairMessageBind)
	MANAGER.Bind(fsm.CreateFlairMessageState, createFlairMessageBind)
	MANAGER.Bind(fsm.SubmitPostState, submitPostBind)

	// Helpers
	MANAGER.Handle(fsm.OnText, fsm.AnyState, func(u tgbotapi.Update) {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Please send a photo or video with caption")
		BOT.Send(msg)
	})

	MANAGER.Run()
}

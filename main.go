package main

import (
	"os"
	"strconv"
	"strings"

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

	MANAGER = &fsm.Manager{
		BotAPI: BOT.BotAPI,
		State:  fsm.DefaultState,
		Data:   fsm.Context{},
	}

	MANAGER.Data.Set("flairs", map[string]string{})

	// Submit post states
	MANAGER.Handle(fsm.OnPhoto, fsm.DefaultState, postHandler)
	MANAGER.Handle(fsm.OnVideo, fsm.DefaultState, postHandler)

	MANAGER.Handle(fsm.OnText, fsm.AwaitFlairMessageState, awaitFlairMessage)
	MANAGER.Bind(fsm.CreateFlairMessageState, createFlairMessage)
	MANAGER.Bind(fsm.SubmitPostState, submitPostBind)

	MANAGER.Run()
}

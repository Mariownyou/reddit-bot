package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/mariownyou/reddit-bot/bot"
	"github.com/mariownyou/reddit-bot/config"
)

var (
	Debug bool
	// RedditID       string
	// RedditSecret   string
	// RedditUsername string
	// RedditPassword string
	// ImgurClientID  string
	TelegramToken string
	// Subreddits    []string
	// Users         []int64
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

	// RedditID = os.Getenv("REDDIT_CLIENT_ID")
	// RedditSecret = os.Getenv("REDDIT_CLIENT_SECRET")
	// RedditUsername = os.Getenv("REDDIT_USERNAME")
	// RedditPassword = os.Getenv("REDDIT_PASSWORD")
	// ImgurClientID = os.Getenv("IMGUR_CLIENT_ID")
	TelegramToken = os.Getenv("TELEGRAM_TOKEN")
	// if subreddits := os.Getenv("SUBREDDITS"); subreddits != "" {
	// 	Subreddits = strings.Split(subreddits, ",")
	// } else {
	// 	Subreddits = []string{"test"}
	// }

	// if users := os.Getenv("USERS"); users != "" {
	// 	for _, user := range strings.Split(users, ",") {
	// 		usr, _ := strconv.Atoi(user)
	// 		Users = append(Users, int64(usr))
	// 	}
	// } else {
	// 	Users = []int64{0}
	// }
}

func main() {
	fmt.Println("token", config.TelegramToken)
	b, err := bot.NewBot(config.TelegramToken)
	if err != nil {
		panic(err)
	}

	b.Debug = config.Debug

	manager := bot.NewManager(*b)
	manager.Construct()
	manager.Run(bot.AuthMiddleware)
}

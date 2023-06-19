package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const EnvFile = ".env.local"

var (
	RedditID       string
	RedditSecret   string
	RedditUsername string
	RedditPassword string

	ImgurClientID string

	TelegramToken string
	Debug         bool
	Subreddits    = []string{"test"}
	Users         = []int64{0}
)

func init() {
	// get cwd for .env.local
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// check if .env file exists
	path := cwd + "/" + EnvFile
	fmt.Println(path)
	_, err = os.Stat(path)
	if err == nil {
		err := godotenv.Load(path)
		if err != nil {
			panic(err)
		}
	}

	RedditID = os.Getenv("REDDIT_CLIENT_ID")
	RedditSecret = os.Getenv("REDDIT_CLIENT_SECRET")
	RedditUsername = os.Getenv("REDDIT_USERNAME")
	RedditPassword = os.Getenv("REDDIT_PASSWORD")

	ImgurClientID = os.Getenv("IMGUR_CLIENT_ID")

	TelegramToken = os.Getenv("TELEGRAM_TOKEN")
	Debug = os.Getenv("DEBUG") == "true"

	if users := os.Getenv("USERS"); users != "" {
		for _, user := range strings.Split(users, ",") {
			usr, _ := strconv.Atoi(user)
			Users = append(Users, int64(usr))
		}
	}

	if subreddits := os.Getenv("SUBREDDITS"); subreddits != "" {
		Subreddits = strings.Split(subreddits, ",")
	}
}

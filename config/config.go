package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	EnvFile = ".env.local"
	// EnvFile          = ".env"
	DriveDeleteAfter = 180
)

var (
	RedditID       string
	RedditSecret   string
	RedditUsername string
	RedditPassword string

	ImgurClientID string

	DriveCredentials []byte

	TwitterConsumerKey       string
	TwitterConsumerSecret    string
	TwitterAccessToken       string
	TwitterAccessTokenSecret string
	TwitterHashtags          string
	TwitterReplyText         string

	UseNativeUplaoder bool
	SendPreview       bool

	ExternalServiceURL string

	TelegramToken string
	Debug         bool
	Subreddits    = []string{"test"}
	Users         = []int64{0}
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// check if .env file exists
	path := cwd + "/" + EnvFile
	log.Println(path)
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

	UseNativeUplaoder = os.Getenv("USE_NATIVE_UPLOADER") == "true"
	SendPreview = os.Getenv("SendPreview") == "true"

	if users := os.Getenv("USERS"); users != "" {
		for _, user := range strings.Split(users, ",") {
			usr, _ := strconv.Atoi(user)
			Users = append(Users, int64(usr))
		}
	}

	if subreddits := os.Getenv("SUBREDDITS"); subreddits != "" {
		Subreddits = strings.Split(subreddits, ",")
	}

	DriveCredentials = []byte(os.Getenv("DRIVE_CREDENTIALS"))

	TwitterConsumerKey = os.Getenv("TWITTER_CONSUMER_KEY")
	TwitterConsumerSecret = os.Getenv("TWITTER_CONSUMER_SECRET")
	TwitterAccessToken = os.Getenv("TWITTER_ACCESS_TOKEN")
	TwitterAccessTokenSecret = os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")
	TwitterHashtags = os.Getenv("TWITTER_HASHTAGS")
	TwitterReplyText = os.Getenv("TWITTER_REPLY_TEXT")

	ExternalServiceURL = os.Getenv("EXTERNAL_SERVICE_URL")
}

package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/go-twitter-uploader/twitter_uploader"
	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/reddit-bot/upload"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

const (
	BotStateNone = iota
	BotStateFlair
)

type Bot struct {
	tgbotapi.BotAPI
	Client        *upload.RedditClient
	TwitterClient *twitter_uploader.Uploader
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	reddit := upload.NewRedditClient()
	twitter := twitter_uploader.New(
		config.TwitterConsumerKey,
		config.TwitterConsumerSecret,
		config.TwitterAccessToken,
		config.TwitterAccessTokenSecret,
	)

	return &Bot{*bot, reddit, twitter}, nil
}

func (bot *Bot) RefreshRedditClient() {
	bot.Client = upload.NewRedditClient()
}

func findSubredditsInMessage(message string) (bool, string, []string) {
	subreddits := []string{}

	for _, word := range strings.Split(message, " ") {
		if strings.HasPrefix(word, "@") {
			message = strings.Replace(message, word, "", 1)
			subreddits = append(subreddits, word[1:])
		}
	}

	if len(subreddits) > 0 {
		return true, message, subreddits
	}
	return false, message, subreddits
}

func NewFlairMessage(flairs []*reddit.Flair, subreddit string, chatID int64) tgbotapi.MessageConfig {
	if len(flairs) == 0 || (len(flairs) == 1 && flairs[0].Text == "") {
		return tgbotapi.NewMessage(chatID, "Click /next to continue")
	}

	buttons := make([][]tgbotapi.KeyboardButton, len(flairs))
	for i, flair := range flairs {
		buttons[i] = []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(flair.Text)}
	}
	keyboard := tgbotapi.NewOneTimeReplyKeyboard(buttons...)
	msg := tgbotapi.NewMessage(chatID, "Please select a flair for subreddit "+subreddit)
	msg.ReplyMarkup = keyboard
	return msg
}

func (bot *Bot) GetFileURL(m *tgbotapi.Message) string {
	var fileID string

	if m.Photo != nil {
		fileID = m.Photo[len(m.Photo)-1].FileID
	}
	if m.Video != nil {
		fileID = m.Video.FileID
	}
	if m.Animation != nil {
		fileID = m.Animation.FileID
	}

	url, err := bot.GetFileDirectURL(fileID)
	if err != nil {
		panic(err)
	}

	return url
}

func (bot *Bot) SendPhoto(chatID int64, file []byte) {
	fb := tgbotapi.FileBytes{
		Name:  "Photo",
		Bytes: file,
	}

	photo := tgbotapi.NewPhoto(chatID, fb)
	if _, err := bot.Send(photo); err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BotStateNone = iota
	BotStateFlair
)

type Context struct {
	state      int
	subreddit  string
	subreddits []string
	flairs     map[string]string
	caption    string
	link       string
	chat       int64
}

type Bot struct {
	tgbotapi.BotAPI
	Ctx    Context
	Client *RedditClient
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	ctx := Context{state: BotStateNone, flairs: map[string]string{}}
	reddit := NewRedditClient()

	return &Bot{*bot, ctx, reddit}, nil
}

func (bot *Bot) ChangeContext(state int, subs []string, flairs map[string]string, sub, link, caption string) {
	bot.Ctx.state = state
	bot.Ctx.subreddit = sub
	bot.Ctx.subreddits = subs
	bot.Ctx.flairs = flairs
	bot.Ctx.link = link
	bot.Ctx.caption = caption
}

func (bot *Bot) auth(update tgbotapi.Update) bool {
	id := update.Message.Chat.ID
	bot.Ctx.chat = update.Message.Chat.ID
	for _, user := range Users {
		if user == id {
			return true
		}
	}

	return false
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

func (bot *Bot) UpdateHandler(update tgbotapi.Update) {
	if !bot.auth(update) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You are not authorized to use this bot "+fmt.Sprint(update.Message.Chat.ID))
		bot.Send(msg)
		return
	}

	if update.Message == nil {
		return
	}

	// check ctx state
	if bot.Ctx.state == BotStateFlair && update.Message.Text != "" {
		subreddits := bot.Ctx.subreddits
		if len(subreddits) == 0 {
			prev := bot.Ctx.subreddit
			if update.Message.Text == "/next" {
				bot.Ctx.flairs[prev] = "None"
			} else {
				bot.Ctx.flairs[prev] = update.Message.Text
			}

			m := bot.Client.SubmitPosts(bot.Ctx.flairs, bot.Ctx.caption, bot.Ctx.link, bot.Ctx.subreddit)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Posting content to the following subreddits:\n"+m)
			bot.Send(msg)

			bot.ChangeContext(BotStateNone, []string{}, map[string]string{}, "", "", "")
			return
		}

		prevSub := bot.Ctx.subreddit
		if update.Message.Text == "/next" {
			bot.Ctx.flairs[prevSub] = "None"
		} else {
			bot.Ctx.flairs[prevSub] = update.Message.Text
			m := fmt.Sprintf("Selected Flair for subreddit: %s -- %s", prevSub, update.Message.Text)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, m)
			bot.Send(msg)
		}

		subreddit := subreddits[0]
		// add custom keyboard with flairs
		flairs := bot.Client.GetPostFlairs(subreddit)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Click /next to continue")

		if len(flairs) > 0 {
			buttons := make([][]tgbotapi.KeyboardButton, len(flairs))
			for i, flair := range flairs {
				buttons[i] = []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(flair.Text)}
			}

			keyboard := tgbotapi.NewOneTimeReplyKeyboard(buttons...)
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Please select a flair for subreddit "+subreddit)
			msg.ReplyMarkup = keyboard
		}

		bot.Send(msg)

		bot.Ctx.subreddits = subreddits[1:]
		bot.Ctx.subreddit = subreddit
		return
	}

	// If message is a photo or a video, download it
	switch {
	case update.Message.Photo != nil && update.Message.Caption == "":
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please add a caption to your post")
		bot.Send(msg)
	case update.Message.Photo != nil, update.Message.Video != nil, update.Message.Text == "test":
		var link string
		switch {
		case update.Message.Text == "test":
			link = "test link"
		case update.Message.Photo != nil:
			url, err := bot.GetFileDirectURL(update.Message.Photo[len(update.Message.Photo)-1].FileID)
			if err != nil {
				panic(err)
			}

			file := DownloadFile(url)
			link = ImgurUpload(file, "image")
		case update.Message.Video != nil:
			url, err := bot.GetFileDirectURL(update.Message.Video.FileID)
			if err != nil {
				panic(err)
			}

			file := DownloadFile(url)
			link = ImgurUpload(file, "video")
		}

		caption := update.Message.Caption
		if caption == "" {
			caption = update.Message.Text
		}

		isSubs, newCaption, Subs := findSubredditsInMessage(caption)
		if !isSubs {
			Subs = Subreddits
		} else {
			caption = newCaption
		}

		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Posting to "+strings.Join(Subs, ", ")))
		bot.ChangeContext(BotStateFlair, Subs[1:], bot.Ctx.flairs, Subs[0], link, caption)

		// add custom keyboard with flairs
		flairs := bot.Client.GetPostFlairs(Subs[0])
		buttons := make([][]tgbotapi.KeyboardButton, len(flairs))
		for i, flair := range flairs {
			buttons[i] = []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(flair.Text)}
		}

		keyboard := tgbotapi.NewOneTimeReplyKeyboard(buttons...)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please select a flair for subreddit "+Subs[0])
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please send a photo or a video")
		bot.Send(msg)
	}
}

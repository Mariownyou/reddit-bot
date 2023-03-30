package main

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BotStateNone = iota
	BotStateFlair
)

var SelectedFlairs = map[string]string{}

type ContextKey string

type Bot struct {
	tgbotapi.BotAPI
	Ctx    context.Context
	Client *RedditClient
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("state"), BotStateNone)

	reddit := NewRedditClient()

	return &Bot{*bot, ctx, reddit}, nil
}

func (bot *Bot) auth(update tgbotapi.Update) bool {
	id := update.Message.Chat.ID
	bot.Ctx = context.WithValue(bot.Ctx, ContextKey("chat"), update.Message.Chat.ID)
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
	if state := bot.Ctx.Value(ContextKey("state")).(int); state == BotStateFlair && update.Message.Text != "" {
		subreddits := bot.Ctx.Value(ContextKey("subreddits")).([]string)
		if len(subreddits) == 0 {
			prev := bot.Ctx.Value(ContextKey("subreddit")).(string)
			if update.Message.Text == "/next" {
				SelectedFlairs[prev] = "None"
			} else {
				SelectedFlairs[prev] = update.Message.Text
			}

			// post content
			m := ""
			for sub, flair := range SelectedFlairs {
				if flair == "None" {
					flair = ""
				}
				bot.Client.SubmitLink(bot.Ctx.Value(ContextKey("caption")).(string), bot.Ctx.Value(ContextKey("link")).(string), sub, flair)

				if flair == "" {
					flair = "None"
				}
				m += fmt.Sprintf("Subreddit: %s -- Flair: %s\n", sub, flair)
			}

			SelectedFlairs = map[string]string{}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Posting content to the following subreddits:\n"+m)
			bot.Send(msg)

			bot.Ctx = context.WithValue(bot.Ctx, ContextKey("state"), BotStateNone)
			bot.Ctx = context.WithValue(bot.Ctx, ContextKey("subreddit"), nil)
			bot.Ctx = context.WithValue(bot.Ctx, ContextKey("subreddits"), []string{})
			bot.Ctx = context.WithValue(bot.Ctx, ContextKey("caption"), nil)
			bot.Ctx = context.WithValue(bot.Ctx, ContextKey("link"), nil)

			return
		}

		prevSub := bot.Ctx.Value(ContextKey("subreddit")).(string)
		if update.Message.Text == "/next" {
			SelectedFlairs[prevSub] = "None"
		} else {
			SelectedFlairs[prevSub] = update.Message.Text
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

		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("subreddits"), subreddits[1:])
		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("subreddit"), subreddit)
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

		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("link"), link)
		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("caption"), caption)
		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("subreddits"), Subs[1:])
		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("subreddit"), Subs[0])
		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("state"), BotStateFlair)

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

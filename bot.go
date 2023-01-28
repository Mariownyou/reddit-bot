package main

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BotStateNone = iota
	BotStateFlair
)

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

func (b *Bot) PostContent(subreddits []string, title, link string) bool {
	for i, subreddit := range Subreddits {
		if flair := b.Ctx.Value(ContextKey("flair")); flair != nil {
			err := b.Client.SubmitLinkFlair(title, link, subreddit, flair.(string))
			if err != nil {
				fmt.Println(err)
				return false
			}
			fmt.Println("submitting with flair", flair.(string))

			// reset flair
			b.Ctx = context.WithValue(b.Ctx, ContextKey("flair"), nil)
			continue
		}

		err := b.Client.SubmitLink(title, link, subreddit)

		if err != nil {
			if errors.Is(err, ErrFlairRequired) {
				b.Ctx = context.WithValue(b.Ctx, ContextKey("subreddit"), subreddit)
				b.Ctx = context.WithValue(b.Ctx, ContextKey("link"), link)
				b.Ctx = context.WithValue(b.Ctx, ContextKey("caption"), title)
				b.Ctx = context.WithValue(b.Ctx, ContextKey("subreddits"), Subreddits[i:])
				b.Ctx = context.WithValue(b.Ctx, ContextKey("state"), BotStateFlair)

				// add custom keyboard with flairs
				flairs := b.Client.GetPostFlairs(subreddit)
				buttons := make([][]tgbotapi.KeyboardButton, len(flairs))
				for i, flair := range flairs {
					buttons[i] = []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(flair.Text)}
				}

				keyboard := tgbotapi.NewOneTimeReplyKeyboard(buttons...)
				chat := b.Ctx.Value(ContextKey("chat")).(int64)
				msg := tgbotapi.NewMessage(chat, "Please select a flair")
				msg.ReplyMarkup = keyboard
				b.Send(msg)
			}
			return false
		}
	}

	// clear context
	b.Ctx = context.WithValue(b.Ctx, ContextKey("state"), BotStateNone)
	b.Ctx = context.WithValue(b.Ctx, ContextKey("subreddit"), nil)
	b.Ctx = context.WithValue(b.Ctx, ContextKey("link"), nil)
	b.Ctx = context.WithValue(b.Ctx, ContextKey("caption"), nil)
	b.Ctx = context.WithValue(b.Ctx, ContextKey("flair"), nil)
	b.Ctx = context.WithValue(b.Ctx, ContextKey("subreddits"), nil)

	return true
}

func (bot *Bot) UpdateHandler(update tgbotapi.Update) {
	bot.Ctx = context.WithValue(bot.Ctx, ContextKey("chat"), update.Message.Chat.ID)

	if update.Message == nil {
		return
	}

	// check ctx state
	state := bot.Ctx.Value(ContextKey("state")).(int)
	if state == BotStateFlair && update.Message.Text != "" {
		state = BotStateNone
		bot.Ctx = context.WithValue(bot.Ctx, ContextKey("flair"), update.Message.Text)
		bot.PostContent(bot.Ctx.Value(ContextKey("subreddits")).([]string), bot.Ctx.Value(ContextKey("caption")).(string), bot.Ctx.Value(ContextKey("link")).(string))
		return
	}

	// If message is a photo or a video, download it
	switch {
	case update.Message.Video != nil:
		url, err := bot.GetFileDirectURL(update.Message.Video.FileID)
		if err != nil {
			panic(err)
		}

		file := DownloadFile(url)
		link := ImgurUpload(file, "video")

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, link)
		bot.Send(msg)
	case update.Message.Photo != nil:
		url, err := bot.GetFileDirectURL(update.Message.Photo[len(update.Message.Photo)-1].FileID)
		if err != nil {
			panic(err)
		}

		file := DownloadFile(url)
		link := ImgurUpload(file, "image")

		bot.PostContent(Subreddits, update.Message.Caption, link)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, link)
		bot.Send(msg)
	case update.Message.Caption == "":
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please add a caption to your post")
		bot.Send(msg)
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please send a photo or a video")
		bot.Send(msg)
	}
}

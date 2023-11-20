package main

import (
	"fmt"
	"sort"
	"time"
	"context"
	"strings"

	"github.com/mariownyou/reddit-bot/upload"
	"github.com/mariownyou/reddit-bot/logger"
	"github.com/mariownyou/go-reddit-uploader/reddit_uploader"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	EmptySub = "None"
	NoFlair  = ""
)

type Post struct {
	cancel   context.CancelFunc
	Title    string
	Subs     map[string]string
	File     []byte
	FileType string
}

func NewPost(bot *Bot, update tgbotapi.Update) *Post {
	text, msg := GetMessageText(update)
	title, subs := ParsePostMessage(text)

	subMap := make(map[string]string)
	for _, sub := range subs {
		subMap[sub] = EmptySub
	}

	var fileURL string
	var filetype string

	switch {
	case msg.Photo != nil:
		filetype = upload.SubmissionTypeImage
		fileURL = msg.Photo[len(msg.Photo)-1].FileID
	case msg.Video != nil:
		filetype = upload.SubmissionTypeVideo
		fileURL = msg.Video.FileID
	case msg.Animation != nil:
		filetype = upload.SubmissionTypeGif
		fileURL = msg.Animation.FileID
	}

	var file []byte
	if fileURL != "" {
		fileURL, err := bot.GetFileDirectURL(fileURL)
		if err != nil {
			panic(err)
		}

		file = upload.DownloadFile(fileURL)
	}

	return &Post{
		Title: title,
		Subs: subMap,
		File: file,
		FileType: filetype,
	}
}

func NewRepost(bot *Bot, update tgbotapi.Update) *Post {
	post := NewPost(bot, update)

	subs := ParseRepostMessage(update.CallbackQuery.Message.Text, update.CallbackQuery.Message.Entities)
	post.Subs = subs

	logger.Red(update.CallbackQuery.Message.Text, update.CallbackQuery.Message.Entities)

	logger.Green("Repost:", post, subs)

	return post
}

func (p *Post) Submit() chan map[string]upload.SubmitStatus {
	status := make(chan map[string]upload.SubmitStatus)
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		defer close(status)

		m := make(map[string]upload.SubmitStatus)
		for sub, flair := range p.Subs {
			select {
			case <-ctx.Done():
				// @TODO send status, remove
				return
			default:
				m[sub] = p.SubmitOne(sub, flair)
				status <- m
				time.Sleep(time.Second * 1)
			}
		}
	}()
	// @TODO try to post first time, if something fails, create buttons for each failed sub to retry

	return status
}

func (p *Post) SubmitOne(sub, flair string) upload.SubmitStatus {
	submission := upload.NewSubmission(
		reddit_uploader.Submission{
			Title: p.Title,
			Subreddit: sub,
			NSFW: true,
		},
		p.File,
		p.FileType,
	)

	flairID := upload.GetFlairID(sub, flair)
	if flairID != "" {
		submission.Submission.FlairID = flairID
	}

	return submission.Submit()
}

func (p *Post) Cancel() {
	if p.cancel == nil {
		return
	}

	p.cancel()
}

func (p *Post) NewStatusMessage(status map[string]upload.SubmitStatus) (string, [][]tgbotapi.InlineKeyboardButton) {
	text := fmt.Sprintf("*%s*\n", EscapeString(p.Title))

	buttons := make([][]tgbotapi.InlineKeyboardButton, 0)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("repost", "repost"),
		tgbotapi.NewInlineKeyboardButtonData("cancel", "cancel"),
	))

	if status == nil {
		return "Please wait, upload in progress🥹", buttons
	}

	for sub, s := range status {
		text += fmt.Sprintf(
			"[%s](%s): %s\n",
			EscapeString(sub),
			EscapeString("https://www.google.com/" + p.Subs[sub]),
			EscapeString(TruncateString(s.Message)),
		)
		if !s.Success {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("repost: "+sub, sub),
			))
		}
	}

	return text, buttons
}

func (p *Post) GetNextEmptySub() string {
	sorted := make([]string, 0, len(p.Subs))
	for sub := range p.Subs {
		sorted = append(sorted, sub)
	}
	sort.Strings(sorted)

	for _, sub := range sorted {
		flair := p.Subs[sub]
		if flair == EmptySub {
			return sub
		}
	}
	return ""
}

func (p *Post) SetFlair(sub string, flair string) {
	p.Subs[sub] = flair
}

func ParsePostMessage(text string) (string, []string) {
	text = strings.Replace(text, "\n", " ", -1)
	words := strings.Split(text, " ")
	subs := []string{}

	for _, word := range words {
		if strings.HasPrefix(word, "@") {
			text = strings.Replace(text, word, "", 1)
			subs = append(subs, word[1:])
		}
	}
	return strings.TrimSpace(text), subs
}

func ParseRepostMessage(text string, styles []tgbotapi.MessageEntity) map[string]string {
	subs := make(map[string]string)

	for _, style := range styles {
		if style.Type != "text_link" {
			continue
		}

		sub := text[style.Offset:style.Offset+style.Length]
		splited := strings.Split(style.URL, "/")
		if len(splited) == 3 {
			subs[sub] = splited[2]
		} else {
			subs[sub] = ""
		}
	}
	return subs
}

func GetMessageText(update tgbotapi.Update) (string, *tgbotapi.Message) {
	message := update.Message
	if message == nil {
		message = update.CallbackQuery.Message
	}

	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil && update.CallbackQuery.Message.ReplyToMessage != nil {
		message = update.CallbackQuery.Message.ReplyToMessage
	}

	text := message.Text
	if text == "" {
		text = message.Caption
	}

	return text, message
}

func TruncateString(s string) string {
	if len(s) <= 30 {
		return s
	}

	return s[:15] + "..." + s[len(s)-15:]
}

func EscapeString(s string) string {
	return tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, s)
}

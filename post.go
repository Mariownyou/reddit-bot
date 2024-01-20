package main

import (
	"fmt"
	"sort"
	"time"
	"context"
	"strings"
	"math/rand"

	"github.com/mariownyou/reddit-bot/upload"
	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/go-reddit-uploader/reddit_uploader"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	EmptySub = "None"
	NoFlair  = ""

	SubmitStatusWait = "Waiting..."
)

type RepostInfo struct {
	Sub string
	Flair string
}

type Post struct {
	repostCh chan RepostInfo
	cancel   context.CancelFunc
	Title    string
	Subs     map[string]string
	File     []byte
	FileType string
	FileName string
	Tag      string
}

func NewPost(bot *Bot, update tgbotapi.Update) *Post {
	text, msg := GetMessageText(update)
	title, subs := ParsePostMessage(text)

	subMap := make(map[string]string)
	for _, sub := range subs {
		subMap[sub] = EmptySub
	}

	var (
		fileURL string
		filetype string
		filename string
	)

	switch {
	case msg.Photo != nil:
		filetype = upload.SubmissionTypeImage
		fileURL = msg.Photo[len(msg.Photo)-1].FileID
		filename = msg.Photo[len(msg.Photo)-1].FileUniqueID
	case msg.Video != nil:
		filetype = upload.SubmissionTypeVideo
		fileURL = msg.Video.FileID
		filename = msg.Video.FileName
	case msg.Animation != nil:
		filetype = upload.SubmissionTypeGif
		fileURL = msg.Animation.FileID
		filename = msg.Animation.FileUniqueID
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
		FileName: filename,
		Tag:  "\n\\#post",
	}
}

func NewRepost(bot *Bot, update tgbotapi.Update) *Post {
	post := NewPost(bot, update)

	subs := ParseRepostMessage(update.CallbackQuery.Message.Text, update.CallbackQuery.Message.Entities)
	post.Subs = subs

	post.Tag = "\n\\#repost"
	return post
}

func (p *Post) Submit() chan map[string]upload.SubmitStatus {
	status := make(chan map[string]upload.SubmitStatus)
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		defer close(status)
		// @TODO add sorting by sub

		m := make(map[string]upload.SubmitStatus)
		for sub := range p.Subs {
			m[sub] = upload.SubmitStatus{Success: false, Message: SubmitStatusWait}
		}
		status <- m

		for sub, flair := range p.Subs {
			select {
			case <-ctx.Done():
				// @TODO send status, remove
				return
			default:
			}

			m[sub] = p.SubmitOne(sub, flair)
			status <- m
			time.Sleep(time.Second * 1)
		}
	}()

	return status
}

func (p *Post) SubmitOne(sub, flair string) upload.SubmitStatus {
	if config.Debug {
		return randomSubmitStatus()
	}

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

	p.Tag = ""
	p.cancel()
}

func (p *Post) NewStatusMessage(status map[string]upload.SubmitStatus) (string, [][]tgbotapi.InlineKeyboardButton) {
	text := fmt.Sprintf("*%s*\n", EscapeString(p.Title))

	buttons := make([][]tgbotapi.InlineKeyboardButton, 0)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("repost", CallbackRepost),
		tgbotapi.NewInlineKeyboardButtonData("cancel", CallbackCancel),
	))

	if status == nil {
		return "Please wait, upload in progress🥹", buttons
	}

	sorted := make([]string, 0, len(status))
	for sub := range status {
		sorted = append(sorted, sub)
	}
	sort.Strings(sorted)

	hasFailed := false
	for _, sub := range sorted {
		flair := p.Subs[sub]
		s := status[sub]

		text += fmt.Sprintf(
			"[%s](%s): %s\n",
			EscapeString(sub),
			EscapeString("https://www.google.com/" + flair),
			EscapeString(TruncateString(s.Message)),
		)
		if !s.Success {
			hasFailed = true
		}
	}

	if hasFailed {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("repost failed", CallbackRepostFailed),
		))
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

		unicodeText := []rune(text)
		sub := string(unicodeText[style.Offset:style.Offset+style.Length])
		splited := strings.Split(style.URL, "/")
		if len(splited) == 3 {
			subs[sub] = splited[2]
		} else {
			subs[sub] = ""
		}
	}
	return subs
}

func ParseFaieldSubs(text string, styles []tgbotapi.MessageEntity) map[string]upload.SubmitStatus {
	status := make(map[string]upload.SubmitStatus)
	t := []rune(text)

	for _, style := range styles {
		if style.Type != "text_link" {
			continue
		}

		start := style.Offset
		sub := string(t[start:start+style.Length])
		line := t[start:]

		check := string(t[start:start+GetEndOfLine(line)])
		if strings.Contains(check, "❌") || strings.Contains(check, SubmitStatusWait) {
			status[sub] = upload.SubmitStatus{Success: false}
		}
	}

	return status
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

func GetEndOfLine(text []rune) int {
	for i, char := range text {
		if char == '\n' {
			return i
		}
	}
	return len(text)
}

func randomSubmitStatus() upload.SubmitStatus {
	r := rand.Intn(2)
	if r == 0 {
		return upload.SubmitStatus{Success: true, Message: upload.StatusOK + " debug"}
	}
	return upload.SubmitStatus{Success: false, Message: upload.StatusError + " debug"}
}

package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/reddit-bot/upload"
)

func PostHandler(m *Manager, u tgbotapi.Update) {
	caption := u.Message.Caption

	isSubs, newCaption, Subs := findSubredditsInMessage(caption)
	if !isSubs {
		Subs = config.Subreddits
	} else {
		caption = strings.TrimSpace(newCaption)
	}

	fileURL := m.GetFileURL(u)
	file := upload.DownloadFile(fileURL)

	var filetype string

	switch {
	case u.Message.Photo != nil:
		filetype = "photo.jpg"
	case u.Message.Video != nil:
		filetype = "video.mp4"
	}

	m.Data.file = file
	m.Data.filetype = filetype
	m.Data.caption = caption
	m.Data.subs = Subs

	m.SetState(CreateFlairMessageState)
}

func AwaitFlairMessageBind(m *Manager, u tgbotapi.Update) {
	flair := u.Message.Text
	flairMap := m.Data.flairs
	subs := m.Data.subs
	sub := subs[0]

	m.Data.subs = subs[1:]
	flairMap[sub] = flair
	m.Data.flairs = flairMap

	if len(subs[1:]) == 0 {
		m.SetState(SubmitPostState)
		return
	}

	text := fmt.Sprintf("You choose %s for sub: %s", flair, sub)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, text)
	m.Send(msg)
	m.SetState(CreateFlairMessageState)
}

func CreateFlairMessageBind(m *Manager, u tgbotapi.Update) State {
	subs := m.Data.subs
	sub := subs[0]

	flairs := m.Client.GetPostFlairs(sub)

	if len(flairs) <= 1 {
		m.Data.subs = subs[1:]
		flairsMap := m.Data.flairs

		flairsMap[sub] = "None"
		m.Data.flairs = flairsMap

		if len(m.Data.subs) == 0 {
			fmt.Println("map", m.Data.flairs)
			text := fmt.Sprintf("No flairs found for sub %s, posting without flair", sub)

			msg := tgbotapi.NewMessage(u.Message.Chat.ID, text)
			m.Send(msg)

			return SubmitPostState
		}

		text := fmt.Sprintf("No flairs found for sub %s, go to the next subreddit", sub)
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, text)
		m.Send(msg)
		return CreateFlairMessageBind(m, u)
	}

	msg := NewFlairMessage(flairs, sub, u.Message.Chat.ID)
	m.Send(msg)

	return AwaitFlairMessageState
}

func SubmitPostBind(m *Manager, u tgbotapi.Update) State {
	var text string
	out := make(chan string)
	flairs := m.Data.flairs
	caption := m.Data.caption

	for sub, flair := range m.Data.flairs {
		text += fmt.Sprintf("%s - %s awaiting...\n", sub, flair)
	}

	text += fmt.Sprintf("Title: %s\n", caption)
	// text += fmt.Sprintf("Content Link: %s\n", link)

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Posting content to the following subreddits with flairs:\n"+text)
	msgObj, _ := m.Send(msg)
	mID := msgObj.MessageID

	go m.Client.SubmitPosts(out, flairs, caption, m.Data.file, m.Data.filetype)

	for text := range out {
		text += fmt.Sprintf("Title: %s\n", caption)
		// text += fmt.Sprintf("Content Link: %s\n", link)
		editMsg := tgbotapi.NewEditMessageText(u.Message.Chat.ID, mID, text)

		m.Send(editMsg)
	}

	// go func() {
	// 	if config.Debug {
	// 		return
	// 	}
	// 	m.TwitterClient.Upload(caption, m.Data.file, m.Data.filetype)
	// }()
	if !config.Debug {
		m.TwitterClient.Upload(caption, m.Data.file, m.Data.filetype)
	}

	m.Data = NewContext()

	return DefaultState
}

func DriveUplaodHandler(m *Manager, u tgbotapi.Update) {
	if u.Message.Photo == nil && u.Message.Video == nil {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Please send a photo or video")
		m.Send(msg)
		return
	}

	file := upload.DownloadFile(m.GetFileURL(u))
	link := upload.DriveUpload(file, u.Message.Caption)

	text := fmt.Sprintf("File will be deleted in %d minutes: %s", config.DriveDeleteAfter, link)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, text)
	m.Send(msg)
}

func ImgurUploadHandler(m *Manager, u tgbotapi.Update) {
	if u.Message.Photo == nil && u.Message.Video == nil {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Please send a photo or video")
		m.Send(msg)
		return
	}

	file := upload.DownloadFile(m.GetFileURL(u))
	var link string

	switch {
	case u.Message.Photo != nil:
		link = upload.ImgurUpload(file, "image")
	case u.Message.Video != nil:
		link = upload.ImgurUpload(file, "video")
	}

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, link)
	m.Send(msg)
}

func AuthMiddleware(m *Manager, u tgbotapi.Update, p processFunc) processFunc {
	if auth(u) {
		return p
	}

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "You are not authorized to use this bot")
	m.Send(msg)
	return nil
}

func (m *Manager) Construct() {
	m.Handle("/imgur", DefaultState, ImgurUploadHandler)
	m.Handle("/drive", DefaultState, DriveUplaodHandler)

	// Submit post states
	m.Handle(OnPhoto, DefaultState, PostHandler)
	m.Handle(OnVideo, DefaultState, PostHandler)

	m.Handle(OnText, AwaitFlairMessageState, AwaitFlairMessageBind)
	m.Bind(CreateFlairMessageState, CreateFlairMessageBind)
	m.Bind(SubmitPostState, SubmitPostBind)

	// Helpers
	m.Handle(OnText, AnyState, func(m *Manager, u tgbotapi.Update) {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Please send a photo or video with caption")
		m.Send(msg)
	})
}

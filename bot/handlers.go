package bot

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/reddit-bot/upload"
)

func PostHandler(m *Manager, u tgbotapi.Update) {
	caption := m.Data.caption
	if caption == "" {
		caption = u.Message.Caption
	}

	// remove #offtweet
	if strings.Contains(caption, "#offtweet") {
		caption = strings.ReplaceAll(caption, "#offtweet", "")
		m.Data.tweet = false
	}

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
		filetype = "image.jpg"
	case u.Message.Video != nil:
		filetype = "video.mp4"
	case u.Message.Animation != nil:
		filetype = "gif.mp4"
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
			log.Println("map", m.Data.flairs)
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

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Posting content to the following subreddits with flairs:\n"+text)
	msgObj, _ := m.Send(msg)
	mID := msgObj.MessageID

	go m.Client.SubmitPosts(out, flairs, caption, m.Data.file, m.Data.filetype)

	for text := range out {
		text += fmt.Sprintf("Title: %s\n", caption)
		editMsg := tgbotapi.NewEditMessageText(u.Message.Chat.ID, mID, text)

		m.Send(editMsg)
	}

	if !config.Debug && m.Data.tweet {
		log.Println("Posting to twitter")
		text := fmt.Sprintf("%s\n%s", caption, config.TwitterHashtags)
		m.TwitterClient.Upload(text, m.Data.file, m.Data.filetype)
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
	link := upload.DriveShareFile(file, u.Message.Caption)

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
	var filetype string

	switch {
	case u.Message.Photo != nil:
		filetype = "image.jpg"
	case u.Message.Video != nil:
		filetype = "video.mp4"
	}

	link := upload.ImgurUpload(file, filetype)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, link)
	m.Send(msg)
}

func FlairsHandler(m *Manager, u tgbotapi.Update) {
	words := strings.Split(u.Message.Text, " ")
	if len(words) < 2 {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Please provide a subreddit")
		m.Send(msg)
		return
	}

	flairs := m.Client.GetPostFlairs(words[1])

	var text string
	for _, flair := range flairs[:10] {
		text += fmt.Sprintf("%s -- %s\n", flair.Text, flair.ID)
	}

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, text)
	m.Send(msg)
}

func UploadAlbumHandler(m *Manager, u tgbotapi.Update) {
	// check that the message contains more than one photo

	log.Println("Photos Len:", len(u.Message.Photo))
}

func (m *Manager) Construct() {
	m.Handle("/album", DefaultState, UploadAlbumHandler)
	m.Handle("/flairs", DefaultState, FlairsHandler)
	m.Handle("/imgur", DefaultState, ImgurUploadHandler)
	m.Handle("/drive", DefaultState, DriveUplaodHandler)

	// Submit post states
	m.Handle(OnPhoto, DefaultState, PostHandler)
	m.Handle(OnVideo, DefaultState, PostHandler)
	m.Handle(OnAnimation, DefaultState, PostHandler)

	m.Handle(OnText, AwaitFlairMessageState, AwaitFlairMessageBind)
	m.Bind(CreateFlairMessageState, CreateFlairMessageBind)
	m.Bind(SubmitPostState, SubmitPostBind)

	// Helpers
	m.Handle(OnText, AnyState, func(m *Manager, u tgbotapi.Update) {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Please send a photo or video with caption")
		m.Send(msg)
		m.Data.caption = u.Message.Text
	})
}

func AuthMiddleware(m *Manager, u tgbotapi.Update, p processFunc) processFunc {
	if auth(u) {
		return p
	}

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "You are not authorized to use this bot")
	m.Send(msg)
	return nil
}

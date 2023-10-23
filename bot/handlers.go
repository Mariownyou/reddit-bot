package bot

import (
	"fmt"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/reddit-bot/logger"
	"github.com/mariownyou/reddit-bot/upload"
)

func GetChatID(u tgbotapi.Update) int64 {
	if u.Message != nil {
		return u.Message.Chat.ID
	}

	if u.CallbackQuery != nil {
		return u.CallbackQuery.Message.Chat.ID
	}

	return 0
}

func (m *Manager) PreparePost(msg *tgbotapi.Message) {
	caption := m.Data.caption
	if caption == "" {
		caption = msg.Caption
	}

	caption = strings.ReplaceAll(caption, "#post", "")

	// remove #offtweet
	if strings.Contains(caption, "#offtweet") {
		caption = strings.ReplaceAll(caption, "#offtweet", "")
		m.Data.tweet = false
	}

	if strings.Contains(caption, "#offext") {
		caption = strings.ReplaceAll(caption, "#offtext", "")
		m.Data.externalSrv = false
	}

	found, newCaption, Subs := findSubredditsInMessage(caption)
	if !found {
		Subs = config.Subreddits
	}

	fileURL := m.GetFileURL(msg)
	file := upload.DownloadFile(fileURL)

	var filetype string

	switch {
	case msg.Photo != nil:
		filetype = "image.jpg"
	case msg.Video != nil:
		filetype = "video.mp4"
	case msg.Animation != nil:
		filetype = "gif.mp4"
	}

	m.Data.file = file
	m.Data.filetype = filetype
	m.Data.caption = caption
	m.Data.subs = Subs
}

func (m *Manager) ParsePost(msg string) {
	for _, line := range strings.Split(msg, "\n") {
		pattern := `(?P<sub>\w+): (?P<flair>\w+), (?P<msg>.+)`
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(line)

		if len(match) == 0 {
			logger.Yellow("No matches found, skiping line: %s", line)
			continue
		}

		sub := match[1]
		flair := match[2]

		m.Data.flairs[sub] = flair
	}
}

func PostHandler(m *Manager, u tgbotapi.Update) {
	m.PreparePost(u.Message)
	m.Data.replyToMsg = u.Message.MessageID

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

func PostCallbackHandler(m *Manager, u tgbotapi.Update) {
	switch u.CallbackQuery.Data {
	case "repost":
		// get msg by id
		// id := u.CallbackQuery.Message.ReplyToMessage
		m.PreparePost(u.CallbackQuery.Message.ReplyToMessage)
		m.Data.replyToMsg = u.CallbackQuery.Message.ReplyToMessage.MessageID
		m.ParsePost(u.CallbackQuery.Message.Text)
		m.SetState(SubmitPostState)
		// fmt.Println("message photo", id.Photo)
		return
		// fmt.Println(u.CallbackQuery.Message.ReplyToMessage)
	default:
		return
	}

	fmt.Println("from callback handler", u.CallbackQuery.Data, u.CallbackQuery.Message)
}

func SubmitPostBind(m *Manager, u tgbotapi.Update) State {
	var text string
	flairs := m.Data.flairs
	caption := m.Data.caption

	text = fmt.Sprintf("Title: %s\n", caption) // @TODO add #post and #repost for better navigation
	for sub, flair := range m.Data.flairs {
		text += fmt.Sprintf("%s: %s, awaiting...\n", sub, flair)
	}

	// callback := tgbotapi.NewInlineKeyboardMarkup(
	// 	tgbotapi.NewInlineKeyboardRow(
	// 		tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel"), // @TODO add Post button
	// 	),
	// )

	msg := tgbotapi.NewMessage(GetChatID(u), text)
	// msg.ReplyMarkup = callback
	msg.ReplyToMessageID = m.Data.replyToMsg // @TODO cleanup after using

	msgObj, _ := m.Send(msg)
	mID := msgObj.MessageID

	if !config.Debug {
		out := make(chan string)

		m.RefreshRedditClient()
		go m.Client.SubmitPosts(out, flairs, caption, m.Data.file, m.Data.filetype)

		for text = range out {
			text = fmt.Sprintf("Title: %s\n%s", caption, text)
			editMsg := tgbotapi.NewEditMessageText(GetChatID(u), mID, text)

			m.Send(editMsg)
		}
	}

	callback := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Repost", "repost"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(GetChatID(u), mID, text)
	editMsg.ReplyMarkup = &callback
	m.Send(editMsg)

	return TwitterAskState
}

const (
	YesOption = "Yes"
	NoOption  = "No"
)

func TwiterSendHandler(m *Manager, u tgbotapi.Update) {
	text := u.Message.Text

	if text == YesOption {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "OK")
		m.Send(msg)

		logger.Yellow("Posting to twitter")
		text := fmt.Sprintf("%s\n%s", m.Data.caption, config.TwitterHashtags)
		id := m.TwitterClient.Upload(text, m.Data.file, m.Data.filetype)
		m.TwitterClient.UploadText(config.TwitterReplyText, id)
	}

	m.SetState(ExtAskState)
}

func TwiterAskBind(m *Manager, u tgbotapi.Update) State {
	buttons := [][]tgbotapi.KeyboardButton{
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton(YesOption),
			tgbotapi.NewKeyboardButton(NoOption),
		},
	}
	keyboard := tgbotapi.NewOneTimeReplyKeyboard(buttons...)

	msg := tgbotapi.NewMessage(GetChatID(u), "Do you want to send this post to twitter?")
	msg.ReplyMarkup = keyboard
	m.Send(msg)

	return TwitterSendState
}

func ExtSendHandler(m *Manager, u tgbotapi.Update) {
	text := u.Message.Text

	if text == YesOption {
		msg := tgbotapi.NewMessage(GetChatID(u), "OK")
		m.Send(msg)

		logger.Yellow("Posting to external service")
		ft := upload.GetMimetype(m.Data.filetype)
		upload.UploadFile(config.ExternalServiceURL, m.Data.caption, ft, m.Data.file)
	}

	m.Data = NewContext()
	m.SetState(DefaultState)
}

func ExtAskBind(m *Manager, u tgbotapi.Update) State {
	buttons := [][]tgbotapi.KeyboardButton{
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton(YesOption),
			tgbotapi.NewKeyboardButton(NoOption),
		},
	}
	keyboard := tgbotapi.NewOneTimeReplyKeyboard(buttons...)

	msg := tgbotapi.NewMessage(GetChatID(u), "Do you want to send this post to f?")
	msg.ReplyMarkup = keyboard
	m.Send(msg)

	return ExtSendState
}

func DriveUplaodHandler(m *Manager, u tgbotapi.Update) {
	if u.Message.Photo == nil && u.Message.Video == nil {
		msg := tgbotapi.NewMessage(GetChatID(u), "Please send a photo or video")
		m.Send(msg)
		return
	}

	file := upload.DownloadFile(m.GetFileURL(u.Message))
	link := upload.DriveShareFile(file, u.Message.Caption)

	text := fmt.Sprintf("File will be deleted in %d minutes: %s", config.DriveDeleteAfter, link)
	msg := tgbotapi.NewMessage(GetChatID(u), text)
	m.Send(msg)
}

func ImgurUploadHandler(m *Manager, u tgbotapi.Update) {
	if u.Message.Photo == nil && u.Message.Video == nil {
		msg := tgbotapi.NewMessage(GetChatID(u), "Please send a photo or video")
		m.Send(msg)
		return
	}

	file := upload.DownloadFile(m.GetFileURL(u.Message))
	var filetype string

	switch {
	case u.Message.Photo != nil:
		filetype = "image.jpg"
	case u.Message.Video != nil:
		filetype = "video.mp4"
	}

	link := upload.ImgurUpload(file, filetype)
	msg := tgbotapi.NewMessage(GetChatID(u), link)
	m.Send(msg)
}

func FlairsHandler(m *Manager, u tgbotapi.Update) {
	words := strings.Split(u.Message.Text, " ")
	if len(words) < 2 {
		msg := tgbotapi.NewMessage(GetChatID(u), "Please provide a subreddit")
		m.Send(msg)
		return
	}

	flairs := m.Client.GetPostFlairs(words[1])

	var text string
	for _, flair := range flairs[:10] {
		text += fmt.Sprintf("%s -- %s\n", flair.Text, flair.ID)
	}

	msg := tgbotapi.NewMessage(GetChatID(u), text)
	m.Send(msg)
}

func (m *Manager) Construct() {
	m.Handle("/flairs", DefaultState, FlairsHandler)
	m.Handle("/imgur", DefaultState, ImgurUploadHandler)
	m.Handle("/drive", DefaultState, DriveUplaodHandler)
	m.Handle("/copy", DefaultState, func(m *Manager, u tgbotapi.Update) {
		t := u.Message.Text
		t = strings.ReplaceAll(t, "/copy ", "")
		m.Data.caption = t
	})

	// Submit post states
	m.Handle(OnPhoto, DefaultState, PostHandler)
	m.Handle(OnVideo, DefaultState, PostHandler)
	m.Handle(OnAnimation, DefaultState, PostHandler)

	m.Handle(OnText, AwaitFlairMessageState, AwaitFlairMessageBind)
	m.Handle(OnCallbackQuery, AnyState, PostCallbackHandler)
	m.Bind(CreateFlairMessageState, CreateFlairMessageBind)
	m.Bind(SubmitPostState, SubmitPostBind)

	m.Handle(OnText, TwitterSendState, TwiterSendHandler)
	m.Bind(TwitterAskState, TwiterAskBind)

	m.Handle(OnText, ExtSendState, ExtSendHandler)
	m.Bind(ExtAskState, ExtAskBind)

	// Helpers
	m.Handle(OnText, AnyState, func(m *Manager, u tgbotapi.Update) {
		msg := tgbotapi.NewMessage(GetChatID(u), "Please send a photo or video with caption")
		m.Send(msg)
	})
}

func AuthMiddleware(m *Manager, u tgbotapi.Update, p processFunc) processFunc {
	if auth(u) {
		return p
	}

	msg := tgbotapi.NewMessage(GetChatID(u), "You are not authorized to use this bot")
	m.Send(msg)
	return nil
}

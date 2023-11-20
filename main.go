package main

import (
	"fmt"
	"log"
	"strings"
	"context"

	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/reddit-bot/upload"
	"github.com/mariownyou/reddit-bot/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Session struct {
	post    *Post
	state   State
	cancel  context.CancelFunc
	replyID int
}

func NewSession() *Session {
	return &Session{
		post: &Post{
			Subs: make(map[string]string),
		},
		state: StateDefault,
	}
}

type Bot struct {
	*tgbotapi.BotAPI
	redditClient *upload.RedditClient
	sessions map[int64]*Session
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		BotAPI: bot,
		redditClient: upload.NewRedditClient(),
		sessions: make(map[int64]*Session),
	}, nil
}

func (b *Bot) GetChatID(u tgbotapi.Update) int64 {
	if u.Message != nil {
		return u.Message.Chat.ID
	}

	if u.CallbackQuery != nil {
		return u.CallbackQuery.Message.Chat.ID
	}

	return 0
}

func (b *Bot) GetSession(update tgbotapi.Update) *Session {
	session, ok := b.sessions[b.GetChatID(update)]
	if !ok {
		session = NewSession()
		b.sessions[b.GetChatID(update)] = session
	}

	return session
}

func (b *Bot) GetPost(update tgbotapi.Update) *Post {
	return b.GetSession(update).post
}

func (b *Bot) SetPost(update tgbotapi.Update, post *Post) {
	b.GetSession(update).post = post
}

func (b *Bot) SetState(update tgbotapi.Update, state State) {
	session := b.GetSession(update)
	session.state = state
}

func main() {
	bot, err := NewBot(config.TelegramToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		session := bot.GetSession(update)

		if update.Message != nil && update.Message.Caption != "" {
			// always reset state on new post
			bot.SetState(update, StateDefault)
		}

		if update.Message != nil {
			switch session.state {
			case StateDefault:
				bot.handleStateDefault(update)
			case StateFlairSelect:
				bot.handleStateFlairSelect(update)
			case StateFlairConfirm:
				bot.handleStateFlairConfirm(update)
			}
		}

		if update.CallbackQuery != nil {
			switch update.CallbackQuery.Data {
			case "repost":
				session.replyID = update.CallbackQuery.Message.ReplyToMessage.MessageID
				post := NewRepost(bot, update)
				session.post = post
				bot.SetState(update, StatePostSending)
				go bot.handleStatePostSending(update)
			case "cancel":
				session.post.Cancel()
			default:
				if strings.HasPrefix(update.CallbackQuery.Data, "repost:") {
					// @TODO
					args := strings.Split(strings.TrimPrefix(update.CallbackQuery.Data, "repost:"), ":")
					sub := args[0]
					flair := args[1]

					post := NewRepost(bot, update)
					post.SubmitOne(sub, flair)
				}
			}
		}
	}
}

func (b *Bot) handleStateDefault(update tgbotapi.Update) {
	b.GetSession(update).replyID = update.Message.MessageID
	b.SetPost(update, NewPost(b, update))
	b.SetState(update, StateFlairSelect)
	b.handleStateFlairSelect(update)
}

func (b *Bot) handleStateFlairSelect(update tgbotapi.Update) {
	post := b.GetPost(update)
	sub := post.GetNextEmptySub()
	if sub != "" {
		flairs := b.redditClient.GetPostFlairs(sub);
		if len(flairs) == 0 || (len(flairs) == 1 && flairs[0].Text == "") {
			post.SetFlair(sub, NoFlair)
			b.handleStateFlairSelect(update)
			return
		}

		m := fmt.Sprintf("Select flair for subreddit %s", sub)

		buttons := make([][]tgbotapi.KeyboardButton, len(flairs))
		for i, flair := range flairs {
			buttons[i] = []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(flair.Text)}
		}

		keyboard := tgbotapi.NewOneTimeReplyKeyboard(buttons...)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, m)
		msg.ReplyMarkup = keyboard
		b.Send(msg)
		b.SetState(update, StateFlairConfirm)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Preparing post...")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	b.Send(msg)

	b.SetState(update, StatePostSending)
	go b.handleStatePostSending(update)
}

func (b *Bot) handleStateFlairConfirm(update tgbotapi.Update) {
	sub := b.GetPost(update).GetNextEmptySub()
	if sub != "" {
		b.GetPost(update).SetFlair(sub, update.Message.Text)
	}
	b.handleStateFlairSelect(update)
}

func (b *Bot) handleStatePostSending(update tgbotapi.Update) {
	post := b.GetPost(update)
	chatID := b.GetChatID(update)
	logger.Green("Sending post to", chatID)
	m, buttons := post.NewStatusMessage(nil)
	msg := tgbotapi.NewMessage(chatID, m)
	msg.ReplyToMessageID = b.GetSession(update).replyID
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyMarkup = &markup
	msgInfo, _ := b.Send(msg)
	msgID := msgInfo.MessageID

	ch := post.Submit()
	var status map[string]upload.SubmitStatus
	var updated tgbotapi.EditMessageTextConfig

	for status = range ch {
		m, buttons = post.NewStatusMessage(status)

		updated = tgbotapi.NewEditMessageText(chatID, msgID, m)
		updated.ParseMode = tgbotapi.ModeMarkdownV2
		markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)
		updated.ReplyMarkup = &markup

		b.Send(updated)
	}

	m, buttons = post.NewStatusMessage(status)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("twitter", "twitter"),
		tgbotapi.NewInlineKeyboardButtonData("f...", "f-service"),
	))

	markup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	updated.ReplyMarkup = &markup
	updated.Text = m + "\n\\#post"
	b.Send(updated)

	b.SetState(update, StateDefault)
}

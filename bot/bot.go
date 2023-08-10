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

// type Context struct {
// 	state      int
// 	subreddit  string
// 	subreddits []string
// 	flairs     map[string]string
// 	caption    string
// 	link       string
// 	chat       int64
// }

type Bot struct {
	tgbotapi.BotAPI
	// Ctx    Context
	Client        *upload.RedditClient
	TwitterClient *twitter_uploader.Uploader
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	// ctx := Context{state: BotStateNone, flairs: map[string]string{}}
	reddit := upload.NewRedditClient()
	twitter := twitter_uploader.New(
		config.TwitterConsumerKey,
		config.TwitterConsumerSecret,
		config.TwitterAccessToken,
		config.TwitterAccessTokenSecret,
	)

	return &Bot{*bot, reddit, twitter}, nil
}

// func (bot *Bot) ChangeContext(state int, subs []string, flairs map[string]string, sub, link, caption string) {
// 	bot.Ctx.state = state
// 	bot.Ctx.subreddit = sub
// 	bot.Ctx.subreddits = subs
// 	bot.Ctx.flairs = flairs
// 	bot.Ctx.link = link
// 	bot.Ctx.caption = caption
// }

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

func (bot *Bot) GetFileURL(u tgbotapi.Update) string {
	var fileID string

	if u.Message.Photo != nil {
		fileID = u.Message.Photo[len(u.Message.Photo)-1].FileID
	}
	if u.Message.Video != nil {
		fileID = u.Message.Video.FileID
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

// func (bot *Bot) UpdateHandler(update tgbotapi.Update) {
// 	if !bot.auth(update) {
// 		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You are not authorized to use this bot "+fmt.Sprint(update.Message.Chat.ID))
// 		bot.Send(msg)
// 		return
// 	}

// 	if update.Message == nil {
// 		return
// 	}

// 	// check ctx state
// 	if bot.Ctx.state == BotStateFlair && update.Message.Text != "" {
// 		subreddits := bot.Ctx.subreddits
// 		if len(subreddits) == 0 {
// 			prev := bot.Ctx.subreddit
// 			if update.Message.Text == "/next" {
// 				bot.Ctx.flairs[prev] = "None"
// 			} else {
// 				bot.Ctx.flairs[prev] = update.Message.Text
// 			}

// 			out := make(chan string)
// 			var message string

// 			for key, value := range bot.Ctx.flairs {
// 				message += fmt.Sprintf("%s: %s\n Awaiting...", key, value)
// 			}

// 			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Posting content to the following subreddits:\n"+message)
// 			bot.Send(msg)

// 			go bot.Client.SubmitPosts(out, bot.Ctx.flairs, bot.Ctx.caption, bot.Ctx.link)

// 			for m := range out {
// 				msg := tgbotapi.NewEditMessageText(update.Message.Chat.ID, update.Message.MessageID+1, m)
// 				bot.Send(msg)
// 			}

// 			bot.ChangeContext(BotStateNone, []string{}, map[string]string{}, "", "", "")
// 			return
// 		}

// 		prevSub := bot.Ctx.subreddit
// 		if update.Message.Text == "/next" {
// 			bot.Ctx.flairs[prevSub] = "None"
// 		} else {
// 			bot.Ctx.flairs[prevSub] = update.Message.Text
// 			m := fmt.Sprintf("Selected Flair for subreddit: %s -- %s", prevSub, update.Message.Text)
// 			msg := tgbotapi.NewMessage(update.Message.Chat.ID, m)
// 			bot.Send(msg)
// 		}

// 		subreddit := subreddits[0]
// 		flairs := bot.Client.GetPostFlairs(subreddit)
// 		msg := NewFlairMessage(flairs, subreddit, update.Message.Chat.ID)
// 		bot.Send(msg)

// 		bot.Ctx.subreddits = subreddits[1:]
// 		bot.Ctx.subreddit = subreddit
// 		return
// 	}

// 	// If message is a photo or a video, download it
// 	switch {
// 	case update.Message.Photo != nil && update.Message.Caption == "":
// 		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please add a caption to your post")
// 		bot.Send(msg)
// 	case update.Message.Photo != nil, update.Message.Video != nil, update.Message.Text == "test":
// 		var link string
// 		switch {
// 		case update.Message.Text == "test":
// 			link = "test link"
// 		case update.Message.Photo != nil:
// 			url, err := bot.GetFileDirectURL(update.Message.Photo[len(update.Message.Photo)-1].FileID)
// 			if err != nil {
// 				panic(err)
// 			}

// 			file := upload.DownloadFile(url)
// 			link = upload.RedditUpload(file, "image")
// 		case update.Message.Video != nil:
// 			url, err := bot.GetFileDirectURL(update.Message.Video.FileID)
// 			if err != nil {
// 				panic(err)
// 			}

// 			file := upload.DownloadFile(url)
// 			link = upload.ImgurUpload(file, "video")
// 		}

// 		caption := update.Message.Caption
// 		if caption == "" {
// 			caption = update.Message.Text
// 		}

// 		isSubs, newCaption, Subs := findSubredditsInMessage(caption)
// 		if !isSubs {
// 			Subs = Subreddits
// 		} else {
// 			caption = newCaption
// 		}

// 		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Posting to "+strings.Join(Subs, ", ")))
// 		bot.ChangeContext(BotStateFlair, Subs[1:], bot.Ctx.flairs, Subs[0], link, caption)

// 		// add custom keyboard with flairs
// 		flairs := bot.Client.GetPostFlairs(Subs[0])
// 		msg := NewFlairMessage(flairs, Subs[0], update.Message.Chat.ID)
// 		bot.Send(msg)
// 	default:
// 		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please send a photo or a video")
// 		bot.Send(msg)
// 	}
// }

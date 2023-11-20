package bot

import (
	"fmt"
	"encoding/json"

	// "github.com/mariownyou/reddit-bot/upload"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackData struct {
	Action 	   string `json:"a"`
	Sub    	   string `json:"s"`
	Flair  	   string `json:"f"`
}

func NewCallbackData(data string) *CallbackData {
	cl := &CallbackData{}
	cl.FromJson(data) // @TODO: handle error
	return cl
}

func (cl *CallbackData) ToJson() (string, error) {
	bs, err := json.Marshal(cl)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func (cl *CallbackData) FromJson(data string) error {
	return json.Unmarshal([]byte(data), cl)
}

func PostCallbackHandler(m *Manager, u tgbotapi.Update) {
	data := NewCallbackData(u.CallbackQuery.Data)

	switch data.Action {
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
	case "repost-sub":
		post := m.CreatePostMessage(u.CallbackQuery.Message.ReplyToMessage)
		post.flairs = map[string]string{
			data.Sub: data.Flair,
		}
		fmt.Println("POST MESSAGE:", post)
	default:
		return
	}

	fmt.Println("from callback handler", u.CallbackQuery.Data, u.CallbackQuery.Message)
}

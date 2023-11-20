package bot

import (
	"testing"
)

func TestCallbackData(t *testing.T) {
	data := CallbackData{
		Action: "repost",
		Sub: "test",
		Flair: "test",
		ReplyToMsg: 1,
	}

	json, err := data.ToJson()
	if err != nil {
		t.Error(err)
	}

	if json != `{"action":"repost","sub":"test","flair":"test","replyToMsg":1}` {
		t.Error("json is not correct", json)
	}
}

func TestPostCallbackHandler(t *testing.T) {

}

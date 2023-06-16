package main

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/reddit-bot/fsm"
)

func postHandler(u tgbotapi.Update) {
	caption := u.Message.Text

	isSubs, newCaption, Subs := findSubredditsInMessage(caption)
	if !isSubs {
		Subs = Subreddits
	} else {
		caption = strings.TrimSpace(newCaption)
	}

	MANAGER.Data.Set("caption", caption)
	MANAGER.Data.Set("subs", Subs)

	MANAGER.SetState(fsm.CreateFlairMessageState)
}

func awaitFlairMessage(u tgbotapi.Update) {
	flair := u.Message.Text
	flairMap := MANAGER.Data.Get("flairs").(map[string]string)
	subs := MANAGER.Data.Get("subs").([]string)
	sub := subs[0]

	MANAGER.Data.Set("subs", subs[1:])
	flairMap[sub] = flair
	MANAGER.Data.Set("flairs", flairMap)

	if len(subs[1:]) == 0 {
		MANAGER.SetState(fsm.SubmitPostState)
		return
	}

	m := fmt.Sprintf("You choose %s for sub: %s", flair, sub)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, m)
	BOT.Send(msg)
	MANAGER.SetState(fsm.CreateFlairMessageState)
}

func createFlairMessage(u tgbotapi.Update) fsm.State {
	// caption := manager.Data.Get("caption").(string)
	subs := MANAGER.Data.Get("subs").([]string)
	sub := subs[0]

	flairs := BOT.Client.GetPostFlairs(sub)
	fmt.Println(flairs)

	if len(flairs) == 0 {
		MANAGER.Data.Set("subs", subs[1:])
		flairsMap := MANAGER.Data.Get("flairs").(map[string]string)

		flairsMap[sub] = "None"
		MANAGER.Data.Set("flairs", flairsMap)

		if len(MANAGER.Data.Get("subs").([]string)) == 0 {
			fmt.Println("map", MANAGER.Data.Get("flairs").(map[string]string))

			msg := tgbotapi.NewMessage(u.Message.Chat.ID, "No flairs found, posting without flair")
			BOT.Send(msg)

			return fsm.SubmitPostState
		}

		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "No flairs found, go to the next subreddit")
		BOT.Send(msg)
		return createFlairMessage(u)
	}

	msg := NewFlairMessage(flairs, sub, u.Message.Chat.ID)
	BOT.Send(msg)

	return fsm.AwaitFlairMessageState
}

func submitPostBind(u tgbotapi.Update) fsm.State {
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Posting image...")
	BOT.Send(msg)
	return fsm.DefaultState
}

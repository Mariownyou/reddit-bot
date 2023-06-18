package main

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mariownyou/reddit-bot/fsm"
)

func postHandler(u tgbotapi.Update) {
	caption := u.Message.Caption

	isSubs, newCaption, Subs := findSubredditsInMessage(caption)
	if !isSubs {
		Subs = Subreddits
	} else {
		caption = strings.TrimSpace(newCaption)
	}

	fileURL := BOT.GetFileURL(u)
	file := DownloadFile(fileURL)
	var link string

	switch {
	case u.Message.Photo != nil:
		link = RedditUpload(file, "jpg")
	case u.Message.Video != nil:
		link = ImgurUpload(file, "video")
	}

	MANAGER.Data.Set("link", link)
	MANAGER.Data.Set("caption", caption)
	MANAGER.Data.Set("subs", Subs)

	MANAGER.SetState(fsm.CreateFlairMessageState)
}

func awaitFlairMessageBind(u tgbotapi.Update) {
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

func createFlairMessageBind(u tgbotapi.Update) fsm.State {
	// caption := manager.Data.Get("caption").(string)
	subs := MANAGER.Data.Get("subs").([]string)
	sub := subs[0]

	flairs := BOT.Client.GetPostFlairs(sub)

	if len(flairs) <= 1 {
		MANAGER.Data.Set("subs", subs[1:])
		flairsMap := MANAGER.Data.Get("flairs").(map[string]string)

		flairsMap[sub] = "None"
		MANAGER.Data.Set("flairs", flairsMap)

		if len(MANAGER.Data.Get("subs").([]string)) == 0 {
			fmt.Println("map", MANAGER.Data.Get("flairs").(map[string]string))
			m := fmt.Sprintf("No flairs found for sub %s, posting without flair", sub)

			msg := tgbotapi.NewMessage(u.Message.Chat.ID, m)
			BOT.Send(msg)

			return fsm.SubmitPostState
		}

		m := fmt.Sprintf("No flairs found for sub %s, go to the next subreddit", sub)
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, m)
		BOT.Send(msg)
		return createFlairMessageBind(u)
	}

	msg := NewFlairMessage(flairs, sub, u.Message.Chat.ID)
	BOT.Send(msg)

	return fsm.AwaitFlairMessageState
}

func submitPostBind(u tgbotapi.Update) fsm.State {
	var m string
	out := make(chan string)
	flairs := MANAGER.Data.Get("flairs").(map[string]string)
	caption := MANAGER.Data.Get("caption").(string)
	link := MANAGER.Data.Get("link").(string)

	for sub, flair := range MANAGER.Data.Get("flairs").(map[string]string) {
		m += fmt.Sprintf("%s - %s awaiting...\n", sub, flair)
	}

	m += fmt.Sprintf("Title: %s\n", caption)
	m += fmt.Sprintf("Content Link: %s\n", link)

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Posting content to the following subreddits with flairs:\n"+m)
	msgObj, _ := BOT.Send(msg)
	mID := msgObj.MessageID

	go BOT.Client.SubmitPosts(out, flairs, caption, link)

	for m := range out {
		m += fmt.Sprintf("Title: %s\n", caption)
		m += fmt.Sprintf("Content Link: %s\n", link)
		msg := tgbotapi.NewEditMessageText(u.Message.Chat.ID, mID, m)

		BOT.Send(msg)
	}

	MANAGER.Data = fsm.NewContext()

	return fsm.DefaultState
}

package bot

import (
	"strings"
	"regexp"

	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/reddit-bot/upload"
	"github.com/mariownyou/reddit-bot/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type PostMessage struct {
	flairs   map[string]string
	title    string
	file     []byte
	filetype string
}

func (m *Manager) CreatePostMessage(msg *tgbotapi.Message) PostMessage {
	// caption := m.Data.caption
	// if caption == "" {
	// 	caption = msg.Caption
	// } // @TODO remove along with copy handler

	caption := msg.Caption
	caption = strings.ReplaceAll(caption, "#post", "")

	found, newCaption, subs := findSubredditsInMessage(caption)
	if !found {
		subs = config.Subreddits
	} else {
		caption = newCaption
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

	flairs := map[string]string{}
	for _, sub := range subs {
		flairs[sub] = "None"
	}

	return PostMessage{
		title:    caption,
		flairs:   flairs,
		file:     file,
		filetype: filetype,
	}
}

func (m *Manager) PreparePost(msg *tgbotapi.Message) {
	post := m.CreatePostMessage(msg)

	subs := []string{}
	for sub := range post.flairs {
		subs = append(subs, sub)
	}

	m.Data.file = post.file
	m.Data.filetype = post.filetype
	m.Data.caption = post.title
	m.Data.subs = subs
}

func (m *Manager) ParsePost(msg string) {
	pattern := `(?P<sub>\w+): (.*), (?P<msg>.+)`
	re := regexp.MustCompile(pattern)

	for _, line := range strings.Split(msg, "\n") {
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

func (m *Manager) ParseFailedPost(msg string) [][]string {
	pattern := `(?P<sub>\w+): (.*?), (?P<msg>.+)`
	re := regexp.MustCompile(pattern)
	failed := [][]string{}

	for _, line := range strings.Split(msg, "\n") {
		if !strings.Contains(line, "❌") {
			continue
		}

		match := re.FindStringSubmatch(line)
		if len(match) == 0 {
			logger.Yellow("No matches found, skiping line: %s", line)
			continue
		}

		sub := match[1]
		flair := match[2]

		failed = append(failed, []string{sub, flair})
	}

	return failed
}

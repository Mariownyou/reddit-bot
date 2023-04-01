package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

var FlairRequiredText = `POST https://oauth.reddit.com/api/submit: 200 field "flair" caused SUBMIT_VALIDATION_FLAIR_REQUIRED: Your post must contain post flair.`
var ErrFlairRequired = errors.New("flair is required")

type RedditClient struct {
	Client *reddit.Client
	Ctx    context.Context
}

func NewRedditClient() *RedditClient {
	credentials := reddit.Credentials{ID: RedditID, Secret: RedditSecret, Username: RedditUsername, Password: RedditPassword}
	client, err := reddit.NewClient(credentials)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	return &RedditClient{Client: client, Ctx: ctx}
}

func (c *RedditClient) NewSubmitLinkRequest(title, url, subreddit, flair string) reddit.SubmitLinkRequest {
	if flair == "" {
		return reddit.SubmitLinkRequest{
			Subreddit: subreddit,
			Title:     title,
			URL:       url,
		}
	}

	flairs := map[string]string{}
	for _, flair := range c.GetPostFlairs(subreddit) {
		flairs[flair.Text] = flair.ID
	}

	return reddit.SubmitLinkRequest{
		Subreddit: subreddit,
		Title:     title,
		URL:       url,
		FlairID:   flairs[flair],
	}
}

func (c *RedditClient) SubmitLink(title, url, subreddit, flair string) error {
	var post *reddit.Submitted
	var err error

	submitLinkRequest := c.NewSubmitLinkRequest(title, url, subreddit, flair)
	post, _, err = c.Client.Post.SubmitLink(c.Ctx, submitLinkRequest)

	if err != nil {
		// try to send post 3 more times
		// TODO handle rate limit
		return err
	}

	fmt.Printf("The link post is available at: %s\n", post.URL)
	return nil
}

func (c *RedditClient) GetPostFlairs(subreddit string) []*reddit.Flair {
	flairs, _, err := c.Client.Flair.GetPostFlairs(c.Ctx, subreddit)
	if err != nil {
		fmt.Printf("Error getting flairs for subreddit: %s -- %s\n", subreddit, err)
		return []*reddit.Flair{}
	}

	return flairs
}

func (c *RedditClient) SubmitPosts(flairs map[string]string, caption, link, sub string) string {
	m := ""
	for sub, flair := range flairs {
		if flair == "None" {
			flair = ""
		}
		err := c.SubmitLink(caption, link, sub, flair)
		if err != nil {
			if strings.Contains(err.Error(), "RATELIMIT") {
				errorWords := strings.Split(err.Error(), " ")
				minStr := errorWords[len(errorWords)-5]
				min, _ := strconv.Atoi(minStr)
				// bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Rate limit reached for %s, retrying in %d minutes", sub, min)))
				time.Sleep(time.Duration(min+2) * time.Minute)
				err = c.SubmitLink(caption, link, sub, flair)

				if err != nil {
					m += fmt.Sprintf("Error posting to subreddit: %s -- %s\n", sub, err)
					fmt.Printf("Error posting to subreddit: %s -- %s\n", sub, err)
				} else {
					// bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Successfully posted to %s", sub)))
				}
			} else {
				m += fmt.Sprintf("Error posting to subreddit: %s -- %s\n", sub, err)
				fmt.Printf("Error posting to subreddit: %s -- %s\n", sub, err)
			}
		}

		if flair == "" {
			flair = "None"
		}
		m += fmt.Sprintf("Subreddit: %s -- Flair: %s\n", sub, flair)

		time.Sleep(time.Second)
	}

	return m
}

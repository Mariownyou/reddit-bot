package main

import (
	"context"
	"errors"
	"fmt"

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

func (c *RedditClient) SubmitText(title, text string) {
	post, _, err := c.Client.Post.SubmitText(c.Ctx, reddit.SubmitTextRequest{
		Subreddit: "test",
		Title:     title,
		Text:      text,
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("The text post is available at: %s\n", post.URL)
}

func (c *RedditClient) SubmitLink(title, url, subreddit, flair string) error {
	// flairs map
	var post *reddit.Submitted
	var err error

	if flair == "" {
		post, _, err = c.Client.Post.SubmitLink(c.Ctx, reddit.SubmitLinkRequest{
			Subreddit: subreddit,
			Title:     title,
			URL:       url,
		})
	} else {
		flairs := map[string]string{}
		for _, flair := range c.GetPostFlairs(subreddit) {
			flairs[flair.Text] = flair.ID
		}

		post, _, err = c.Client.Post.SubmitLink(c.Ctx, reddit.SubmitLinkRequest{
			Subreddit: subreddit,
			Title:     title,
			URL:       url,
			FlairID:   flairs[flair],
		})
	}

	if err != nil {
		return err
	}

	fmt.Printf("The link post is available at: %s\n", post.URL)
	return nil
}

func (c *RedditClient) GetPostFlairs(subreddit string) []*reddit.Flair {
	flairs, _, err := c.Client.Flair.GetPostFlairs(c.Ctx, subreddit)
	if err != nil {
		return []*reddit.Flair{}
	}

	return flairs
}

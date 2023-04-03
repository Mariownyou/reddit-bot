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

type Progress map[string]string

func (p Progress) String() string {
	var str string
	for k, v := range p {
		str += fmt.Sprintf("%s: %s\n", k, v)
	}
	return str
}

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

func (c *RedditClient) submitLink(submitLinkRequest reddit.SubmitLinkRequest, out chan string, retry int) {
	post, _, err := c.Client.Post.SubmitLink(c.Ctx, submitLinkRequest)
	// err = errors.New("RATELIMIT: you are doing that too much. try again in 2 min min min minutes.")

	if err != nil {
		if strings.Contains(err.Error(), "RATELIMIT") && retry < 3 {
			errorWords := strings.Split(err.Error(), " ")
			minStr := errorWords[len(errorWords)-5]
			min, _ := strconv.Atoi(minStr)

			for i := 0; i <= min+1; i++ {
				out <- fmt.Sprintf("Waiting %d minutes to retry post\n", min-i)
				time.Sleep(time.Minute)
			}

			c.submitLink(submitLinkRequest, out, retry+1)
		} else {
			out <- fmt.Sprintf("Error submitting post: %s\n", err)
			return
		}
	} else {
		out <- fmt.Sprintf("The post is available at: %s\n", post.URL)
	}
}

func (c *RedditClient) SubmitLink(out chan string, title, url, subreddit, flair string) {
	submitLinkRequest := c.NewSubmitLinkRequest(title, url, subreddit, flair)
	c.submitLink(submitLinkRequest, out, 0)
	close(out)
}

func (c *RedditClient) GetPostFlairs(subreddit string) []*reddit.Flair {
	flairs, _, err := c.Client.Flair.GetPostFlairs(c.Ctx, subreddit)
	if err != nil {
		fmt.Printf("Error getting flairs for subreddit: %s -- %s\n", subreddit, err)
		return []*reddit.Flair{}
	}

	return flairs
}

func (c *RedditClient) SubmitPosts(out chan string, flairs map[string]string, caption, link, sub string) {
	progress := flairs

	for sub, flair := range flairs {
		if flair == "None" {
			flair = ""
		}

		submitChan := make(chan string)
		go c.SubmitLink(submitChan, caption, link, sub, flair)

		for msg := range submitChan {
			progress[sub] = msg
			out <- Progress(progress).String()
		}

		time.Sleep(time.Second * 3)
	}

	out <- Progress(progress).String()
	close(out)
}

package upload

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mariownyou/go-reddit-uploader/reddit_uploader"
	"github.com/mariownyou/reddit-bot/config"
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
	credentials := reddit.Credentials{ID: config.RedditID, Secret: config.RedditSecret, Username: config.RedditUsername, Password: config.RedditPassword}
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

func (c *RedditClient) submitLink(submitLinkRequest reddit.SubmitLinkRequest, out chan string, retry int) error {
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
			fmt.Printf("Error submitting post: %s\n", err)
			out <- fmt.Sprintf("Error submitting post: %s\n", err)
			return err
		}
	} else {
		out <- fmt.Sprintf("The post is available at: %s\n", post.URL)
	}
	return nil
}

func (c *RedditClient) Submit(out chan string, p reddit_uploader.Submission, file []byte, filetype, imgurLink string) {
	var redditPreviewLink, redditLink string

	if filetype == "image.jpg" {
		redditPreviewLink = ""
		redditLink = RedditUpload(file, "image")
	} else {
		redditPreviewLink, _ = GetRedditPreviewLink(file)
		redditLink = RedditUpload(file, "video")
	}

	client, err := reddit_uploader.New(config.RedditUsername, config.RedditPassword, config.RedditID, config.RedditSecret)
	if err != nil {
		fmt.Printf("Error creating reddit uploader client: %s\n", err)
		close(out)
		return
	}

	redditRes := func() (string, error) {
		if redditPreviewLink == "" {
			return client.SubmitImageLink(p, redditLink, "image.jpg")
		} else {
			return client.SubmitVideoLink(p, redditLink, redditPreviewLink, "video.mp4")
		}
	}

	imgurRes := func() (string, error) {
		if redditPreviewLink == "" {
			return client.SubmitImageLink(p, imgurLink, "image.jpg")
		} else {
			return client.SubmitImageLink(p, imgurLink, "video.mp4")
		}
	}

	r, err := redditRes()
	if err != nil {
		out <- fmt.Sprintf("Error submitting post using reddit native api❌: %s", err)
		fmt.Println("Error submitting post using reddit native api", p.Subreddit, r)

		time.Sleep(time.Second * 1)

		r, err = imgurRes()
		if err != nil {
			out <- fmt.Sprintf("Error submitting post using imgur api❌: %s", err)
			fmt.Println("Error submitting post using imgur api", p.Subreddit, r)
		} else {
			out <- "Post submitted successfully using imgur ✅"
			fmt.Println("Post submitted successfully using imgur api", p.Subreddit)
		}
	} else {
		out <- "Post submitted successfully ✅"
		fmt.Println("Post submitted successfully using reddit native api", p.Subreddit)
	}

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

func (c *RedditClient) SubmitPosts(out chan string, flairs map[string]string, caption string, file []byte, filetype string) {
	progress := flairs

	var imgurLink string

	if filetype == "image.jpg" {
		imgurLink = ImgurUpload(file, "image")
	} else {
		imgurLink = ImgurUpload(file, "video")
	}

	for sub, flair := range flairs {
		if flair == "None" {
			flair = ""
		}

		submitChan := make(chan string)
		params := reddit_uploader.Submission{Title: caption, Subreddit: sub, FlairID: flair}
		go c.Submit(submitChan, params, file, filetype, imgurLink)

		for msg := range submitChan {
			progress[sub] = msg
			out <- Progress(progress).String()
		}

		time.Sleep(time.Second * 3)
	}

	out <- Progress(progress).String()
	close(out)
}

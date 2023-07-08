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

func (c *RedditClient) Submit(out chan string, p reddit_uploader.Submission, redditLink, redditPreviewLink, imgurLink string) {
	client, err := reddit_uploader.New(config.RedditUsername, config.RedditPassword, config.RedditID, config.RedditSecret)
	if err != nil {
		fmt.Printf("Error creating reddit uploader client: %s\n", err)
		close(out)
		return
	}

	if redditPreviewLink == "" {
		r, err := client.SubmitImageLink(p, redditLink, "image.jpg")
		fmt.Println("Response", r)
		if err != nil {
			out <- fmt.Sprintf("Error submitting image link using reddit native api❌: %s", err)
			_, err := client.SubmitImageLink(p, imgurLink, "image.jpg")
			if err != nil {
				out <- fmt.Sprintf("Error submitting image link using imgur api❌: %s", err)
			} else {
				fmt.Println("Image submitted successfully using imgur api", p.Subreddit)
				out <- "Post submitted successfully using imgur ✅"
			}
		} else {
			fmt.Println("Image submitted successfully using reddit native api", p.Subreddit)
			out <- "Post submitted successfully ✅"
		}
	} else {
		r, err := client.SubmitVideoLink(p, redditLink, redditPreviewLink, "video.mp4")
		fmt.Println("Response", r)
		if err != nil {
			out <- fmt.Sprintf("Error submitting video link using reddit native api❌: %s", err)
			_, err := client.SubmitImageLink(p, imgurLink, "image.jpg")
			if err != nil {
				out <- fmt.Sprintf("Error submitting image link using imgur api❌: %s", err)
			} else {
				fmt.Println("Video submitted successfully using imgur api", p.Subreddit)
				out <- "Post submitted successfully using imgur ✅"
			}
		} else {
			fmt.Println("Video submitted successfully using reddit native api", p.Subreddit)
			out <- "Post submitted successfully ✅"
		}
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

	var (
		redditPreviewLink string
		redditLink        string
		imgurLink         string
	)

	if filetype == "image.jpg" {
		redditPreviewLink = ""
		redditLink = RedditUpload(file, "image")
		imgurLink = ImgurUpload(file, "image")
	} else {
		redditPreviewLink, _ = GetRedditPreviewLink(file)
		redditLink = RedditUpload(file, "video")
		imgurLink = ImgurUpload(file, "video")
	}

	for sub, flair := range flairs {
		if flair == "None" {
			flair = ""
		}

		submitChan := make(chan string)
		params := reddit_uploader.Submission{Title: caption, Subreddit: sub, FlairID: flair}
		go c.Submit(submitChan, params, redditLink, redditPreviewLink, imgurLink)

		for msg := range submitChan {
			progress[sub] = msg
			out <- Progress(progress).String()
		}

		time.Sleep(time.Second * 3)
	}

	out <- Progress(progress).String()
	close(out)
}

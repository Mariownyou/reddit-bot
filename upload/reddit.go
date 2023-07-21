package upload

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/mariownyou/go-reddit-uploader/reddit_uploader"
	"github.com/mariownyou/reddit-bot/config"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type Progress map[string]string

func (p Progress) String() string {
	var str string
	for k, v := range p {
		str += fmt.Sprintf("%s: %s\n", k, v)
	}
	return str
}

type RedditClient struct {
	Client   *reddit.Client
	Uploader *reddit_uploader.RedditUplaoder
	Ctx      context.Context
}

func NewRedditClient() *RedditClient {
	credentials := reddit.Credentials{ID: config.RedditID, Secret: config.RedditSecret, Username: config.RedditUsername, Password: config.RedditPassword}
	client, err := reddit.NewClient(credentials)
	if err != nil {
		panic(err)
	}

	uploader, err := reddit_uploader.New(config.RedditUsername, config.RedditPassword, config.RedditID, config.RedditSecret)
	if err != nil {
		panic(err)
	}

	return &RedditClient{
		Client:   client,
		Uploader: uploader,
		Ctx:      context.Background(),
	}
}

func (c *RedditClient) Submit(out chan string, p reddit_uploader.Submission, file []byte, filetype, imgurLink string) {
	var redditPreviewLink string

	defer close(out)

	log.Println("Submitting post", p, filetype)

	redditLink, err := c.Uploader.UploadMedia(file, filetype)
	if err != nil {
		out <- fmt.Sprintf("Error uploading media to reddit ❌: %s", err)
		log.Println("Error uploading media to reddit", p.Subreddit, redditLink, err)
		return
	}

	if filetype == "image.jpg" {
		redditPreviewLink = ""
	} else {
		redditPreviewLink, err = c.GetRedditPreviewLink(file)
		if err != nil {
			out <- fmt.Sprintf("Error getting reddit preview link ❌: %s", err)
			log.Println("Error getting reddit preview link", p.Subreddit, redditLink, err)
			return
		}
	}

	redditRes := func() (string, error) {
		if redditPreviewLink == "" {
			return c.Uploader.SubmitImageLink(p, redditLink, "image.jpg")
		} else {
			return c.Uploader.SubmitVideoLink(p, redditLink, redditPreviewLink, "video.mp4")
		}
	}

	imgurRes := func() (string, error) {
		if redditPreviewLink == "" {
			return c.Uploader.SubmitImageLink(p, imgurLink, "image.jpg")
		} else {
			return c.Uploader.SubmitImageLink(p, imgurLink, "video.mp4")
		}
	}

	r, err := redditRes()
	if err != nil {
		out <- fmt.Sprintf("Error submitting post using reddit native api ❌: %s", err)
		log.Println("Error submitting post using reddit native api", p.Subreddit, r, err)

		time.Sleep(time.Second * 1)

		r, err = imgurRes()
		if err != nil {
			out <- fmt.Sprintf("Error submitting post using imgur api ❌: %s", err)
			log.Println("Error submitting post using imgur api", p.Subreddit, r, err)
		} else {
			out <- "Post submitted successfully using imgur ✅"
			log.Println("Post submitted successfully using imgur api", p.Subreddit)
		}
	} else {
		out <- "Post submitted successfully ✅"
		log.Println("Post submitted successfully using reddit native api", p.Subreddit)
	}
}

func (c *RedditClient) SubmitPosts(out chan string, flairs map[string]string, caption string, file []byte, filetype string) {
	progress := flairs

	defer close(out)

	var imgurLink string

	imgurLink = ImgurUpload(file, filetype)

	for sub, flair := range flairs {
		if flair == "None" {
			flair = ""
		}

		submitChan := make(chan string)

		params := c.NewSubmission(caption, sub, flair)
		go c.Submit(submitChan, params, file, filetype, imgurLink)

		for msg := range submitChan {
			progress[sub] = msg
			out <- Progress(progress).String()
		}

		time.Sleep(time.Second * 3)
	}

	out <- Progress(progress).String()
}

func (c *RedditClient) NewSubmission(text, sub, flair string) reddit_uploader.Submission {
	ids := map[string]string{}
	for _, flair := range c.GetPostFlairs(sub) {
		ids[flair.Text] = flair.ID
	}

	params := reddit_uploader.Submission{Title: text, Subreddit: sub}
	if len(ids) > 0 {
		params.FlairID = ids[flair]
	}
	return params
}

func (c *RedditClient) GetRedditPreviewLink(video []byte) (string, error) {
	name := getRandomName()
	vName := name + ".mp4"
	pName := name + ".jpg"

	err := os.WriteFile(vName, video, 0644)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("ffmpeg", "-i", vName, "-vframes", "1", pName)
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	preview, err := os.ReadFile(pName)
	if err != nil {
		return "", err
	}

	link, err := c.Uploader.UploadMedia(preview, "preview.jpg")
	if err != nil {
		panic(err)
	}

	os.Remove(vName)
	os.Remove(pName)

	return link, nil
}

func (c *RedditClient) GetPostFlairs(subreddit string) []*reddit.Flair {
	flairs, _, err := c.Client.Flair.GetPostFlairs(c.Ctx, subreddit)
	if err != nil {
		fmt.Printf("Error getting flairs for subreddit: %s -- %s\n", subreddit, err)
		return []*reddit.Flair{}
	}

	return flairs
}

func getRandomName() string {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(randomBytes)
}

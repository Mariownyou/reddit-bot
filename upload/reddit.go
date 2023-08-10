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
	Uploader *reddit_uploader.Uploader
	Ctx      context.Context
}

func NewRedditClient() *RedditClient {
	credentials := reddit.Credentials{ID: config.RedditID, Secret: config.RedditSecret, Username: config.RedditUsername, Password: config.RedditPassword}
	client, err := reddit.NewClient(credentials)
	if err != nil {
		panic(err)
	}

	uploader, err := reddit_uploader.New(config.RedditUsername, config.RedditPassword, config.RedditID, config.RedditSecret, "u/mariwonyou reddit bot")
	if err != nil {
		panic(err)
	}

	return &RedditClient{
		Client:   client,
		Uploader: uploader,
		Ctx:      context.Background(),
	}
}

type Uploader interface {
	Upload() error
	PrintResponse(out chan string, resp string)
	PrintError(out chan string, err error, resp string)
}

type RedditUploader struct {
	srv       *reddit_uploader.Uploader
	post      reddit_uploader.Submission
	mediaPath string
	isVideo   bool
}

func (u *RedditUploader) PrintResponse(out chan string, resp string) {
	out <- "Post submitted successfully ✅"
	log.Println("Post submitted successfully using reddit native api", u.post.Subreddit)
}

func (u *RedditUploader) PrintError(out chan string, err error, resp string) {
	out <- fmt.Sprintf("Error submitting post using reddit native api ❌: %s", err)
	log.Println("Error submitting post using reddit native api", u.post.Subreddit, resp, err)
}

// func (u *RedditUploader) GetRedditPreviewLink(video []byte) (string, error) {
// 	preview, err := GetPreviewFile(video)
// 	if err != nil {
// 		panic(err)
// 	}

// 	link, err := u.srv.UploadMedia(preview, "preview.jpg")
// 	if err != nil {
// 		panic(err)
// 	}

// 	return link, nil
// }

func (u *RedditUploader) Upload() error {
	if u.isVideo {
		previewPath, err := GetPreviewFile(u.mediaPath)
		if err != nil {
			panic(err)
		}
		return u.srv.SubmitVideo(u.post, u.mediaPath, previewPath)
	}
	return u.srv.SubmitImage(u.post, u.mediaPath)
}

// type ImgurUploader struct {
// 	srv       *reddit_uploader.Uploader
// 	post      reddit_uploader.Submission
// 	mediaLink string
// 	isVideo   bool
// }

// func (u *ImgurUploader) PrintResponse(out chan string, resp string) {
// 	out <- "Post submitted successfully using imgur ✅"
// 	log.Println("Post submitted successfully using imgur api", u.post.Subreddit)
// }

// func (u *ImgurUploader) PrintError(out chan string, err error, resp string) {
// 	out <- fmt.Sprintf("Error submitting post using imgur api ❌: %s", err)
// 	log.Println("Error submitting post using imgur api", u.post.Subreddit, resp, err)
// }

// func (u *ImgurUploader) Upload() (string, error) {
// 	if u.isVideo {
// 		return u.srv.SubmitImageLink(u.post, u.mediaLink, "video.mp4")
// 	}
// 	return u.srv.SubmitImageLink(u.post, u.mediaLink, "image.jpg")
// }

func (c *RedditClient) Submit(out chan string, p reddit_uploader.Submission, filetype, imgurLink string) {
	defer close(out)

	log.Println("Submitting post", p, filetype)

	// redditLink, err := c.Uploader.UploadMedia(file, filetype)
	// if err != nil {
	// 	out <- fmt.Sprintf("Error uploading media to reddit ❌: %s", err)
	// 	log.Println("Error uploading media to reddit", p.Subreddit, redditLink, err)
	// 	return
	// }

	redditUploader := &RedditUploader{
		srv:       c.Uploader,
		post:      p,
		mediaPath: filetype,
		isVideo:   !(filetype == "image.jpg"),
	}

	// imgurUploader := &ImgurUploader{
	// 	srv:       c.Uploader,
	// 	post:      p,
	// 	mediaLink: imgurLink,
	// 	isVideo:   !(filetype == "image.jpg"),
	// }

	uploaders := []Uploader{redditUploader}
	for _, upl := range uploaders {
		err := upl.Upload()
		if err == nil {
			redditUploader.PrintResponse(out, "")
			break
		}

		redditUploader.PrintError(out, err, "")
		time.Sleep(time.Second * 1)
	}
}

func GetPreviewFile(filename string) (string, error) {
	// name := getRandomName()
	// vName := name + ".mp4"
	// pName := name + ".jpg"

	// err := os.WriteFile(vName, video, 0644)
	// if err != nil {
	// 	return nil, err
	// }

	cmd := exec.Command("ffmpeg", "-i", filename, "-vframes", "1", "preview.jpg")
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// preview, err := os.ReadFile(pName)
	// if err != nil {
	// 	return nil, err
	// }

	// defer os.Remove(vName)
	// defer os.Remove(pName)

	return "preview.jpg", nil
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func (c *RedditClient) SubmitPosts(out chan string, flairs map[string]string, caption string, file []byte, filetype string) {
	progress := flairs

	defer close(out)

	// imgurLink := ImgurUpload(file, filetype)

    err := os.WriteFile(filetype, file, 0644)
    check(err)

	for sub, flair := range flairs {
		if flair == "None" {
			flair = ""
		}

		submitChan := make(chan string)

		params := c.NewSubmission(caption, sub, flair)
		go c.Submit(submitChan, params, filetype, imgurLink)

		for msg := range submitChan {
			progress[sub] = msg
			out <- Progress(progress).String()
		}

		time.Sleep(time.Second * 3)
	}

	err = os.Remove(filetype)
	check(err)
	os.Remove("preview.jpg")

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

func (c *RedditClient) GetPreviewFile(video []byte) ([]byte, error) {
	name := getRandomName()
	vName := name + ".mp4"
	pName := name + ".jpg"

	err := os.WriteFile(vName, video, 0644)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("ffmpeg", "-i", vName, "-vframes", "1", pName)
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	preview, err := os.ReadFile(pName)
	if err != nil {
		return nil, err
	}

	defer os.Remove(vName)
	defer os.Remove(pName)

	return preview, nil
}

// func (c *RedditClient) GetRedditPreviewLink(video []byte) (string, error) {
// 	preview, err := c.GetPreviewFile(video)
// 	if err != nil {
// 		panic(err)
// 	}

// 	link, err := c.Uploader.UploadMedia(preview, "preview.jpg")
// 	if err != nil {
// 		panic(err)
// 	}

// 	return link, nil
// }

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

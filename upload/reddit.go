package upload

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"strings"
	"regexp"
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
	Success(resp string) string
	Error(err error, resp string) string
}

type RedditUploader struct {
	srv       *reddit_uploader.Uploader
	post      reddit_uploader.Submission
	mediaPath string
	media     []byte
	filetype  string
	isVideo   bool
}

func NewRedditUploader(srv *reddit_uploader.Uploader, post reddit_uploader.Submission, mediaPath string, media []byte, filetype string, isVideo bool) *RedditUploader {
	return &RedditUploader{
		srv:       srv,
		post:      post,
		mediaPath: mediaPath,
		media:     media,
		filetype:  filetype,
		isVideo:   isVideo,
	}
}

func (u *RedditUploader) Success(resp string) string {
	log.Println("Post submitted successfully using reddit native api", u.post.Subreddit)
	return "Post submitted successfully ✅"
}

func (u *RedditUploader) Error(err error, resp string) string {
	log.Println("Error submitting post using reddit native api", u.post.Subreddit, resp, err)
	return fmt.Sprintf("Error submitting post using reddit native api ❌: %s: %s", err, resp)
}

func (u *RedditUploader) ConvertToGif() {
	command := exec.Command("ffmpeg", "-i", u.mediaPath, "-r", "20", "gif.gif")
	out, err := command.Output()
	if err != nil {
		panic(string(out))
	}
}

func (u *RedditUploader) Upload() error {
	if u.isVideo {
		previewPath, err := GetPreviewFile(u.mediaPath)
		if err != nil {
			panic(err)
		}
		return u.srv.SubmitVideo(u.post, u.mediaPath, previewPath)
	}

	if u.filetype == "gif.mp4" {
		os.Remove("gif.gif")
		u.ConvertToGif()
		u.mediaPath = "gif.gif"
		defer os.Remove("gif.gif")
	}

	return u.srv.SubmitImage(u.post, u.mediaPath)
}

type ImgurUploader struct {
	srv      *reddit_uploader.Uploader
	post     reddit_uploader.Submission
	media    []byte
	filename string
}

func (u *ImgurUploader) Success(resp string) string {
	msg := fmt.Sprintf("Post submitted successfully using imgur ✅ %s", u.post.Subreddit)
	log.Println(msg)
	return msg
}

func (u *ImgurUploader) Error(err error, resp string) string {
	msg := fmt.Sprintf("Error submitting post using imgur api ❌: %s: %s", err, resp)
	log.Println(msg)
	return msg
}

func (u *ImgurUploader) Upload() error {
	fmt.Printf("Uploading to imgur: %s\n", u.filename)
	link := ImgurUpload(u.media, u.filename)
	return u.srv.SubmitLink(u.post, link)
}

func (c *RedditClient) Submit(out chan string, p reddit_uploader.Submission, file []byte, filetype, imgurLink string) {
	defer close(out)

	log.Println("Submitting post", p, filetype)

	name := getRandomName() + filetype
	os.Remove(name)
	os.Remove("preview.jpg")
	err := os.WriteFile(name, file, 0644)
	check(err)

	redditUploader := &RedditUploader{
		srv:       c.Uploader,
		post:      p,
		mediaPath: name,
		media:     file,
		filetype:  filetype,
		isVideo:   filetype == "video.mp4",
	}

	// imgurUploader := &ImgurUploader{
	// 	srv:      c.Uploader,
	// 	post:     p,
	// 	media:    file,
	// 	filename: filetype,
	// }

	uploaders := []Uploader{redditUploader}
	for _, upl := range uploaders {
		err := upl.Upload()
		if err == nil {
			out <- redditUploader.Success("uploaded")
			break
		}

		// simple load balancer
		if strings.Contains(err.Error(), "Take a break for") {
			re := regexp.MustCompile(`(\d+)`)
			m := re.FindAllString(err.Error(), -1)
			if len(m) > 0 {
				mins, _ := strconv.Atoi(m[0])
				out <- redditUploader.Error(err, fmt.Sprintf("will repeat in %d minutes", mins))

				for i:=1; i<=mins+1; i++ {
					out <- fmt.Sprintf("🕣 Waiting to send post again in %d minutes", mins-i)
					time.Sleep(time.Minute * 1)
				}

				err = upl.Upload()
				if err == nil {
					out <- redditUploader.Success("uploaded")
					break
				}
			}
		}

		out <- redditUploader.Error(err, "Could not submit post: " + p.Subreddit)
		time.Sleep(time.Second * 1)
	}

	os.Remove(name)
	os.Remove("preview.jpg")
}

func GetPreviewFile(filename string) (string, error) {
	cmd := exec.Command("ffmpeg", "-i", filename, "-vframes", "1", "preview.jpg")
	err := cmd.Run()
	if err != nil {
		return "", err
	}

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

	for sub, flair := range flairs {
		if flair == "None" {
			flair = ""
		}

		if flair == "skip" {
			continue
		}

		submitChan := make(chan string)

		params := c.NewSubmission(caption, sub, flair)
		go c.Submit(submitChan, params, file, filetype, imgurLink)

		for msg := range submitChan {
			progress[sub] = msg
			out <- Progress(progress).String()
		}

		time.Sleep(time.Second * 2)
	}

	out <- Progress(progress).String()
}

func (c *RedditClient) NewSubmission(text, sub, flair string) reddit_uploader.Submission {
	ids := map[string]string{}
	for _, flair := range c.GetPostFlairs(sub) {
		ids[flair.Text] = flair.ID
	}

	params := reddit_uploader.Submission{Title: text, Subreddit: sub, NSFW: true}
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

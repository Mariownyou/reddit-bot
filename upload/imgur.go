package upload

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"

	"github.com/mariownyou/go-reddit-uploader/reddit_uploader"
	"github.com/mariownyou/reddit-bot/config"
)

const (
	imgurLink      = "https://api.imgur.com/3/image"
	imgurVideoLink = "https://api.imgur.com/3/upload"
)

func DownloadFile(link string) []byte {
	resp, err := http.Get(link)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}

func ImgurUpload(file []byte, filetype string) string {
	filename := "test." + filetype

	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)

	var fw io.Writer
	var err error

	if filetype == "image" {
		fw, err = w.CreateFormFile("image", filename)
	} else {
		fw, err = w.CreateFormFile("video", filename)
	}
	if err != nil {
		panic(err)
	}

	// Copy the file to the form file
	if _, err = io.Copy(fw, bytes.NewReader(file)); err != nil {
		panic(err)
	}

	// Close the multipart writer
	if err = w.Close(); err != nil {
		panic(err)
	}

	// Create a new request
	var req *http.Request
	if filetype == "image" {
		req, err = http.NewRequest("POST", imgurLink, buf)
	} else {
		req, err = http.NewRequest("POST", imgurVideoLink, buf)
	}

	if err != nil {
		panic(err)
	}

	// Set the content type
	req.Header.Set("Content-Type", w.FormDataContentType())
	// Set the authorization header
	req.Header.Set("Authorization", "Client-ID "+config.ImgurClientID)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Unmarshal the response
	var data map[string]interface{}
	if err = json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	// Get the link
	link := data["data"].(map[string]interface{})["link"].(string)

	if filetype == "video" {
		link = link[:len(link)-3] + "gifv"
	}

	// Return the link
	fmt.Printf("Imgur Link: %s\nResponse: %s\n", link, string(body))
	return link
}

func RedditUpload(file []byte, filename string) string {
	client := reddit_uploader.New(config.RedditUsername, config.RedditPassword, config.RedditID, config.RedditSecret)

	link, err := client.UploadMedia(file, filename)
	if err != nil {
		panic(err)
	}

	return link
}

func GetRedditPreviewLink(video []byte) (string, error) {
	client := reddit_uploader.New(config.RedditUsername, config.RedditPassword, config.RedditID, config.RedditSecret)

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

	link, err := client.UploadMedia(preview, "preview.jpg")
	if err != nil {
		panic(err)
	}

	os.Remove(vName)
	os.Remove(pName)

	return link, nil
}

func getRandomName() string {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(randomBytes)
}

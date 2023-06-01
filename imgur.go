package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/mariownyou/go-reddit-uploader/reddit_uploader"
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
	req.Header.Set("Authorization", "Client-ID "+ImgurClientID)

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

func RedditUpload(file []byte, filetype string) string {
	client := reddit_uploader.New(RedditUsername, RedditPassword, RedditID, RedditSecret)

	link, err := client.UploadMedia(file, filetype)
	if err != nil {
		panic(err)
	}

	return link
}

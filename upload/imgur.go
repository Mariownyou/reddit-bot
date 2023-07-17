package upload

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"

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

	type ImgurResponse struct {
		Data struct {
			Link string `json:"link"`
		} `json:"data"`
	}

	// Unmarshal the response
	var data ImgurResponse
	if err = json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	// Get the link
	link := data.Data.Link
	if link == "" {
		log.Println("imgur response:", filetype, string(body))
		panic("Imgur link is empty")
	}

	if filetype == "video" {
		link = link[:len(link)-3] + "gifv"
	}

	// Return the link
	log.Printf("Imgur Link: %s\nResponse: %s\n", link, string(body))
	return link
}

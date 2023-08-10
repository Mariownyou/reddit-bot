package twitter_uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/dghubble/oauth1"
)

const BatchSize = 5 * 1024 * 1024

var types = map[string]string{
	".jpg":  "tweet_image",
	".jpeg": "tweet_image",
	".png":  "tweet_image",
	".gif":  "tweet_gif",
	".mp4":  "amplify_video",
	".mov":  "amplify_video",
}

type Uploader struct {
	Client *http.Client
}

func New(consumerKey, consumerSecret, accessToken, accessTokenSecret string) *Uploader {
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	return &Uploader{
		Client: httpClient,
	}
}

type Media struct {
	MediaIDs []string `json:"media_ids"`
}

type Tweet struct {
	Text  string `json:"text"`
	Media *Media `json:"media,omitempty"`
}

func (u *Uploader) Downlaod() {} // TODO

func (u *Uploader) Upload(text string, file []byte, filename string) {
	path := "https://api.twitter.com/2/tweets"

	t := types[filepath.Ext(filename)]
	var mediaID string
	if t == "tweet_image" {
		mediaID = u.uploadImage(file, filename)
	} else {
		mediaID = u.uploadVideo(file, filename)
	}

	tweet := Tweet{
		Text:  text,
		Media: &Media{MediaIDs: []string{mediaID}},
	}

	payload, _ := json.Marshal(tweet)
	reader := bytes.NewReader(payload)
	resp, _ := u.Client.Post(path, "application/json", reader)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Raw Response Body:\n%v\n", string(body))
}

func (u *Uploader) uploadImage(file []byte, filename string) string {
	// amplify_video, tweet_gif, tweet_image, and tweet_video
	path := "https://upload.twitter.com/1.1/media/upload.json"

	// create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// add image to multipart form
	part, _ := writer.CreateFormFile("media", filename)
	reader := bytes.NewReader(file)
	io.Copy(part, reader)
	writer.Close()
	// build request
	req, _ := http.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// send request
	resp, _ := u.Client.Do(req)
	defer resp.Body.Close()
	// read response
	respBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Raw Response Body:\n%v\n", string(respBody))

	type MediaResponse struct {
		MediaID string `json:"media_id_string"`
	}

	var mediaResponse MediaResponse
	// unmarshal response
	json.Unmarshal(respBody, &mediaResponse)
	fmt.Printf("Media ID: %v\n", mediaResponse.MediaID)

	return mediaResponse.MediaID
}

func createBatches(l int) []int {
	times := l / BatchSize
	reminder := l % BatchSize
	batches := make([]int, times)
	for i := 0; i < times; i++ {
		batches[i] = BatchSize
	}
	if reminder > 0 {
		batches = append(batches, reminder)
	}
	return batches
}

func (u *Uploader) uploadVideo(file []byte, filename string) string {
	mediaID := u.initUpload(len(file))
	fmt.Println(mediaID)

	start := 0
	for i, batch := range createBatches(len(file)) {
		u.appendUpload(file[start:start+batch], filename, i, mediaID)
		start += batch
	}

	u.finalizeUpload(mediaID)
	return mediaID
}

func (u *Uploader) initUpload(size int) string {
	path := fmt.Sprintf("https://upload.twitter.com/1.1/media/upload.json?command=INIT&total_bytes=%v&media_type=video/mp4", size)

	// create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	form := url.Values{}
	form.Add("command", "INIT")
	form.Add("total_bytes", fmt.Sprintf("%v", size))
	form.Add("media_type", "video/mp4")

	// create url variable
	url, _ := url.Parse(path)

	// set query params
	url.RawQuery = form.Encode()

	req, _ := http.NewRequest("POST", url.String(), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// send request
	resp, _ := u.Client.Do(req)
	defer resp.Body.Close()
	// read response
	respBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Raw Response Body:\n%v\n", string(respBody))

	type InitResponse struct {
		MediaID string `json:"media_id_string"`
	}

	var initResponse InitResponse
	// unmarshal response
	json.Unmarshal(respBody, &initResponse)
	fmt.Printf("Media ID: %v\n", initResponse.MediaID)

	return initResponse.MediaID
}

func (u *Uploader) appendUpload(file []byte, filename string, segmentIndex int, mediaID string) {
	path := "https://upload.twitter.com/1.1/media/upload.json"
	// multipart/form-data

	// create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	form := url.Values{}
	form.Add("command", "APPEND")
	form.Add("media_id", mediaID)
	form.Add("segment_index", fmt.Sprintf("%v", segmentIndex))

	url, _ := url.Parse(path)
	url.RawQuery = form.Encode()

	// add image to multipart form
	part, _ := writer.CreateFormFile("media", filename)
	reader := bytes.NewReader(file)
	io.Copy(part, reader)
	writer.Close()
	// build request
	req, _ := http.NewRequest("POST", url.String(), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// send request
	resp, _ := u.Client.Do(req)
	defer resp.Body.Close()
	// read response
	respBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Raw Response Body:\n%v\n", string(respBody))
}

func (u *Uploader) finalizeUpload(mediaID string) {
	path := "https://upload.twitter.com/1.1/media/upload.json?command=FINALIZE"

	// create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	form := url.Values{}
	form.Add("command", "FINALIZE")
	form.Add("media_id", mediaID)

	// create url variable
	url, _ := url.Parse(path)

	// set query params
	url.RawQuery = form.Encode()

	req, _ := http.NewRequest("POST", url.String(), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// send request
	resp, _ := u.Client.Do(req)
	defer resp.Body.Close()
	// read response
	respBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Raw Response Body:\n%v\n", string(respBody))

	resp, err := u.Client.Post(path, "application/json", nil)
	if err != nil {
		fmt.Println(err)
	}
}

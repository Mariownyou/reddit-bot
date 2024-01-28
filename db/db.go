package db

import (
	"io"
	"fmt"
	"log"
	"bytes"
	"strings"
	"net/url"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"mime/multipart"

	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/reddit-bot/upload"
	"github.com/mariownyou/reddit-bot/logger"
)

var (
	root = config.PocketHostURL
	records = "/api/collections/uploads/records"
)

type Record struct {
	ID 	       string `json:"id"`

	MediaName  string `json:"media_name"`
	Media      string `json:"media"`
	Title      string `json:"title"`

	File       []byte
	Preview    []byte

	Data       map[string]interface{} `json:"data"`
}

func (r Record) Upload() {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	filename := GetFileName(r.MediaName)
	if strings.Contains(r.MediaName, ".") {
		preview, err := upload.GetPreviewFileFromBytes(r.File)
		if err != nil {
			log.Fatal(err)
		}

		r.Preview = preview
	}

	if record := IsFileUploaded(filename); record != nil {
		r.Update(record.ID)
		return
	}

	part, _ := writer.CreateFormFile("media", filename)
	io.Copy(part, bytes.NewReader(r.File))

	if r.Preview != nil {
		part, _ := writer.CreateFormFile("preview", "preview.jpg")
		io.Copy(part, bytes.NewReader(r.Preview))
	}

	writer.WriteField("title", r.Title)
	writer.WriteField("media_name", filename)

	data, _ := json.Marshal(r.Data)
	writer.WriteField("data", string(data))

	writer.Close()

	req, _ := http.NewRequest("POST", root + records, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("X-Token", config.PocketHostToken)
	client := &http.Client{}
	client.Do(req)

	defer req.Body.Close()

	b, _ := ioutil.ReadAll(req.Body)
	logger.Yellow(string(b))
}

func (r Record) Update(id string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("title", r.Title)

	data, _ := json.Marshal(r.Data)
	writer.WriteField("data", string(data))

	writer.Close()

	req, _ := http.NewRequest("PATCH", root + records + "/" + id, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	req.Header.Add("X-Token", config.PocketHostToken)
	client := &http.Client{}
	client.Do(req)

	defer req.Body.Close()

	b, _ := ioutil.ReadAll(req.Body)
	logger.Yellow(string(b))
}

type Response struct {
	Items []Record
}

func IsFileUploaded(name string) *Record {
	name = GetFileName(name)
	params := url.Values{}
	params.Add("filter", fmt.Sprintf("(media_name='%s')", name))

	url := root + records + "?" + params.Encode()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("X-Token", config.PocketHostToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	var response Response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(body, &response); err != nil {
		log.Fatal(err)
	}

	if len(response.Items) == 0 {
		return nil
	}
	return &response.Items[0]
}

func GetFileName(n string) string {
	if !strings.Contains(n, ".") {
		n += ".jpg"
	}

	return n
}

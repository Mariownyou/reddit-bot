package upload

import (
    "bytes"
    "io"
    "strings"
    "mime"
    "mime/multipart"
    "net/http"
)

func GetMimetype(filename string) string {
    splitted := strings.Split(filename, ".")
    extension := splitted[len(splitted)-1]
    return mime.TypeByExtension("." + extension)
}

func UploadFile(targetURL string, title string, mimetype string, b []byte) error {
    var err error
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    file := bytes.NewReader(b)

	err = writer.WriteField("title", title)
	if err != nil {
		return err
	}

	err = writer.WriteField("mimetype", mimetype)
	if err != nil {
		return err
	}

    part, err := writer.CreateFormFile("image", "image.jpg")
    if err != nil {
        return err
    }

    _, err = io.Copy(part, file)
    if err != nil {
        return err
    }

    err = writer.Close()
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", targetURL, body)
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", writer.FormDataContentType())

    client := &http.Client{}
    _, err = client.Do(req)
    if err != nil {
        return err
    }

    return nil
}

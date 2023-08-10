package drive_uploader

import (
	"context"
	"fmt"
	"log"
	"mime"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Uploader struct {
	credentialsJSON []byte
	service         *drive.Service
}

func New(credentialsJSON []byte) (*Uploader, error) {
	ctx := context.Background()
	options := option.WithCredentialsJSON(credentialsJSON)
	service, err := drive.NewService(ctx, options)

	if err != nil {
		log.Fatalf("Unable to create Drivade service: %v", err)
		return nil, err
	}

	return &Uploader{
		credentialsJSON: credentialsJSON,
		service:         service,
	}, nil
}

func (u *Uploader) ShareFile(file []byte, filename string) (string, string, error) {
	splitted := strings.Split(filename, ".")
	extension := splitted[len(splitted)-1]

	mimeType := mime.TypeByExtension("." + extension)

	f := &drive.File{
		Name:     filename,
		MimeType: mimeType,
	}

	p := &drive.Permission{
		Type:               "anyone",
		Role:               "reader",
		AllowFileDiscovery: false,
	}

	return u.Upload(file, f, p)
}

func (u *Uploader) Upload(b []byte, f *drive.File, p *drive.Permission) (string, string, error) {
	file := strings.NewReader(string(b))
	fileSize := len(b)

	var res *drive.File
	var err error

	if fileSize > 5*1024*1024 {
		res, err = u.service.Files.
			Create(f).
			ResumableMedia(context.Background(), file, int64(fileSize), f.MimeType).
			ProgressUpdater(func(now, size int64) { fmt.Printf("%d, %d\r", now, size) }).
			Do()
	} else {
		res, err = u.service.Files.Create(f).Media(file).Do()
	}

	if err != nil {
		log.Fatalf("Unable to upload media file: %v", err)
		return "", "", err
	}

	if p != nil {
		_, err = u.service.Permissions.Create(res.Id, p).Do()
		if err != nil {
			log.Fatalf("Failed to create permission for the file: %v", err)
			return "", "", err
		}
	}

	fileLink := fmt.Sprintf("https://drive.google.com/file/d/%s/view?usp=sharing", res.Id)
	return fileLink, res.Id, nil
}

func (u *Uploader) CreateFolder(name string, parents ...string) (string, error) {

	f := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  parents,
	}

	createdFolder, err := u.service.Files.Create(f).Do()
	if err != nil {
		log.Fatalf("Unable to create folder: %v", err)
		return "", err
	}

	return createdFolder.Id, nil
}

func (u *Uploader) Delete(fileID string) error {
	err := u.service.Files.Delete(fileID).Do()
	if err != nil {
		return err
	}

	return nil
}

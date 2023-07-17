package upload

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"path/filepath"
	"time"

	"github.com/mariownyou/go-drive-uploader/drive_uploader"
	"github.com/mariownyou/reddit-bot/config"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type DriveUploader struct {
	srv *drive_uploader.Uploader
}

func NewDriveUploader() *DriveUploader {
	uploader, _ := drive_uploader.New(config.DriveCredentials)
	return &DriveUploader{
		srv: uploader,
	}
}

func (d *DriveUploader) GetImage(subreddit string) ([]byte, error) {
	return d.GetFirstFileInFolder(subreddit, true)
}

func (d *DriveUploader) UploadFile(file []byte, filename string, folder string) error {
	mime := mime.TypeByExtension(filepath.Ext(filename))

	f := &drive.File{
		Name:     filename,
		MimeType: mime,
	}

	if folder != "" {
		f.Parents = []string{folder}
	}

	_, _, err := d.srv.Upload(file, f, nil)
	return err
}

func (d *DriveUploader) CreateFolder(name string) (string, error) {
	return d.srv.CreateFolder(name)
}

func (d *DriveUploader) GetFirstFileInFolder(folderName string, deleteAfter bool) ([]byte, error) {
	ctx := context.Background()
	options := option.WithCredentialsJSON(config.DriveCredentials)
	srv, err := drive.NewService(ctx, options)

	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("mimeType='application/vnd.google-apps.folder' and trashed=false and name='%s'", folderName)
	folders, err := srv.Files.List().Q(query).Fields("files(id)").Do()
	if err != nil {
		return nil, err
	}

	if len(folders.Files) == 0 {
		return nil, fmt.Errorf("folder not found: %s", folderName)
	}

	files, err := srv.Files.List().Q(fmt.Sprintf("'%s' in parents and trashed=false", folders.Files[0].Id)).Fields("files(id, name)").Do()
	if err != nil {
		return nil, err
	}

	if len(files.Files) == 0 {
		return nil, fmt.Errorf("no files found in folder: %s", folderName)
	}

	resp, err := srv.Files.Get(files.Files[0].Id).Download()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if deleteAfter {
		DriveDelete(files.Files[0].Id, 0)
	}

	return body, nil
}

func DriveShareFile(file []byte, filename string) string {
	uploader, _ := drive_uploader.New(config.DriveCredentials)
	link, fileID, err := uploader.ShareFile(file, filename)
	if err != nil {
		log.Println(err)
		return ""
	}

	DriveDelete(fileID, config.DriveDeleteAfter)

	return link
}

func DriveDelete(fileID string, mins int) {
	timer := time.NewTimer(time.Duration(mins) * time.Minute)

	go func() {
		<-timer.C // Wait for the timer to expire
		uploader, _ := drive_uploader.New(config.DriveCredentials)
		err := uploader.Delete(fileID)
		if err != nil {
			fmt.Printf("Failed to delete file: %s\n", fileID)
			return
		}

		fmt.Printf("Deleted file: %s\n", fileID)
	}()
}

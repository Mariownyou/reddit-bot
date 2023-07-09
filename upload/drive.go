package upload

import (
	"fmt"
	"time"

	"github.com/mariownyou/go-drive-uploader/drive_uploader"
	"github.com/mariownyou/reddit-bot/config"
)

func DriveUpload(file []byte, filename string) string {
	uploader, _ := drive_uploader.New(config.DriveCredentials)
	link, fileID, err := uploader.ShareFile(file, filename)
	if err != nil {
		fmt.Println(err)
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

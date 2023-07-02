package upload

import (
	"github.com/mariownyou/go-drive-uploader/drive_uploader"
	"github.com/mariownyou/reddit-bot/config"
)

func DriveUpload(file []byte, filename string) string {
	uploader := drive_uploader.New(config.DriveCredentials)
	return uploader.Upload(file, filename)
}

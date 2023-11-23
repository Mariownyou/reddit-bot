package upload

import (
	"os"
	"strings"
	"os/exec"

	"github.com/mariownyou/reddit-bot/config"
	"github.com/mariownyou/go-reddit-uploader/reddit_uploader"
)

const (
	SubmissionTypeImage = "image.jpg"
	SubmissionTypeVideo = "video.mp4"
	SubmissionTypeGif   = "gif.mp4"

	PreviewFilename = "preview.jpg"
	GifFilename     = "gif.gif"
)

type Submission struct {
	reddit_uploader.Submission

	rupl *reddit_uploader.Uploader

	File     []byte
	FileType string
}

func NewSubmission(submission reddit_uploader.Submission, file []byte, fileType string) *Submission {
	redditUploader, err := reddit_uploader.New(config.RedditUsername, config.RedditPassword, config.RedditID, config.RedditSecret, "u/mariownyou reddit bot")
	if err != nil {
		panic(err) // @TODO add panic to logger
	}

	return &Submission{
		Submission: submission,
		rupl: 	    redditUploader,
		File:       file,
		FileType:   fileType,
	}
}

type SubmitStatus struct {
	Success bool
	Message string
}

func (s *Submission) Submit() SubmitStatus {
	var err error

	deleteFile(s.FileType)
	deleteFile(PreviewFilename)
	deleteFile(GifFilename)

	saveFile(s.File, s.FileType)
	defer deleteFile(s.FileType)

	switch s.FileType {
	case SubmissionTypeImage:
		err = s.rupl.SubmitImage(s.Submission, s.FileType)
	case SubmissionTypeVideo:
		err = s.submitVideo()
	case SubmissionTypeGif:
		// try send as video
		err = s.submitVideo()
		if err == nil {
			break
		}

		// convert to gif
		var out []byte
		command := exec.Command("ffmpeg", "-i", s.FileType, "-r", "20", GifFilename)
		out, err = command.Output()
		if err != nil {
			return SubmitStatus{
				Success: false,
				Message: "❌" + string(out) + err.Error(),
			}
		}

		defer deleteFile(GifFilename)
		err = s.rupl.SubmitImage(s.Submission, GifFilename)
	}

	if err != nil {
		return SubmitStatus{
			Success: false,
			Message: "❌" + CleanString(err.Error()),
		}
	}

	return SubmitStatus{
		Success: true,
		Message: "✅",
	}
}

func (s *Submission) submitVideo() error {
	_, err := GetPreviewFile(s.FileType)
	if err != nil {
		return err
	}

	defer deleteFile(PreviewFilename)
	return s.rupl.SubmitVideo(s.Submission, s.FileType, PreviewFilename)
}

func saveFile(f []byte, n string) {
	os.WriteFile(n, f, 0644)
}

func deleteFile(n string) {
	os.Remove(n)
}

func CleanString(s string) string {
	// @TODO remove newlines in package, not here
	s = strings.Replace(s, "\n", " ", -1)
	s = strings.TrimSpace(s)
	return s
}

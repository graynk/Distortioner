package tools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	tb "gopkg.in/telebot.v3"
)

const (
	progress        = "Processing frames...\n<code>[----------] %d%%</code>"
	MaxSizeMb       = 20_000_000
	NotEnoughRights = "The bot does not have enough rights to send media to your chat"
	Failed          = "Failed"
)

// why isn't this an separate type declared in the telebot?
const (
	Animation = "animation"
	Audio     = "audio"
	Document  = "document"
	Photo     = "photo"
	Sticker   = "sticker"
	Video     = "video"
	VideoNote = "videoNote"
	Voice     = "voice"
)

var NoFileErr = errors.New("no file found")
var TooBigErr = errors.New("Senpai, it's too big..")
var FailedToDownloadErr = errors.New("Failed to download")
var NotSupportedErr = errors.New("Not supported yet, sorry")

func GetUserFriendlyErr(err error) (string, bool) {
	switch err {
	case TooBigErr:
	case FailedToDownloadErr:
	case NotSupportedErr:
		return err.Error(), true
	}
	return Failed, false
}

func GenerateProgressMessage(done, total int) string {
	fraction := float64(done) / float64(total)
	message := fmt.Sprintf(progress, int(fraction*100))
	return strings.Replace(message, "-", "=", int(fraction*10))
}

func IsMedia(m *tb.Message) bool {
	if m == nil {
		return false
	}
	return m.Photo != nil || m.Video != nil || m.VideoNote != nil || m.Voice != nil
}

func IsNonMediaMedia(m *tb.Message) bool {
	if m == nil {
		return false
	}
	return m.Animation != nil || m.Sticker != nil
}

// there's a bug in telebot where media.MediaFile() does not check for Sticker.
func JustGetTheMedia(m *tb.Message) tb.Media {
	media := m.Media()
	if media == nil && m.Sticker != nil {
		return m.Sticker
	}
	return media
}

func JustGetTheFile(b *tb.Bot, media tb.Media) (string, error) {
	if media == nil {
		return "", NoFileErr
	}
	file := media.MediaFile()
	if file == nil {
		return "", NoFileErr
	}
	if file.FileSize > MaxSizeMb {
		return "", TooBigErr
	}
	filename := uuid.New().String()
	if err := b.Download(file, filename); err != nil {
		return "", FailedToDownloadErr
	}

	return filename, nil
}

func ExtractPossibleTimeout(err error) (int, error) {
	// format: "telegram: retry after x (429)"
	errorString := err.Error()
	if strings.Contains(errorString, "kicked") {
		return 0, err
	}
	after := "after "
	retryAfterStringEnd := strings.LastIndex(errorString, after)
	if retryAfterStringEnd == -1 {
		return 0, err
	}
	timeoutEnd := strings.LastIndex(errorString, " (")
	if timeoutEnd == -1 {
		timeoutEnd = len(errorString)
	}
	return strconv.Atoi(errorString[retryAfterStringEnd+len(after) : timeoutEnd])
}

func FormatRateLimitResponse(diff int64) string {
	return fmt.Sprintf("Please, not so often. Try again in %d seconds", diff)
}

package tools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	tb "gopkg.in/telebot.v3"
)

const progress = "Processing frames...\n<code>[----------] %d%%</code>"

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

func JustGetTheFile(b *tb.Bot, m *tb.Message) (string, error) {
	filename := uuid.New().String()
	file := m.Media().MediaFile()
	err := b.Download(file, filename)
	if err != nil {
		b.Reply(m, "Failed to download media")
	}

	return filename, err
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

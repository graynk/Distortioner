package main

import (
	"fmt"
	"github.com/google/uuid"
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"strconv"
	"strings"
)

const progress = "Processing frames...\n<code>[----------] %d%%</code>"

func generateProgressMessage(done, total int) string {
	fraction := float64(done) / float64(total)
	message := fmt.Sprintf(progress, int(fraction*100))
	return strings.Replace(message, "-", "=", int(fraction*10))
}

func justGetTheFile(b *tb.Bot, m *tb.Message) (string, error) {
	var file tb.File
	filename := uuid.New().String()
	switch {
	case m.Animation != nil:
		file = m.Animation.File
	case m.Photo != nil:
		file = m.Photo.File
	case m.Sticker != nil:
		file = m.Sticker.File
	case m.Video != nil:
		file = m.Video.File
	case m.VideoNote != nil:
		file = m.VideoNote.File
	case m.Voice != nil:
		file = m.Voice.File
	}
	err := b.Download(&file, filename)
	if err != nil {
		log.Println(err)
		b.Send(m.Chat, "Failed to download media")
	}

	return filename, err
}

func extractPossibleTimeout(err error) (int, error) {
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

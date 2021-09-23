package main

import (
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"strconv"
	"strings"
)

const progress = "Processing frames...\n<code>[----------] %d%%</code>"

func uniqueFileName(fileId string, timestamp int64) string {
	return fileId + strconv.FormatInt(timestamp, 10)
}

func generateProgressMessage(done, total int) string {
	fraction := float64(done) / float64(total)
	message := fmt.Sprintf(progress, int(fraction*100))
	return strings.Replace(message, "-", "=", int(fraction*10))
}

func justGetTheFile(b *tb.Bot, m *tb.Message) (string, error) {
	var filename string
	var file tb.File
	switch {
	case m.Animation != nil:
		filename = uniqueFileName(m.Animation.FileID, m.Unixtime)
		file = m.Animation.File
	case m.Photo != nil:
		filename = uniqueFileName(m.Photo.FileID, m.Unixtime)
		file = m.Photo.File
	case m.Sticker != nil:
		filename = uniqueFileName(m.Sticker.FileID, m.Unixtime)
		file = m.Sticker.File
	case m.Video != nil:
		filename = uniqueFileName(m.Video.FileID, m.Unixtime)
		file = m.Video.File
	case m.VideoNote != nil:
		filename = uniqueFileName(m.VideoNote.FileID, m.Unixtime)
		file = m.VideoNote.File
	case m.Voice != nil:
		filename = uniqueFileName(m.Voice.FileID, m.Unixtime)
		file = m.Voice.File
	}
	err := b.Download(&file, filename)
	if err != nil {
		log.Println(err)
		b.Send(m.Chat, "Failed to download media")
	}

	return filename, err
}

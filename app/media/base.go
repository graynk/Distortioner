package media

import (
	"os"

	tb "gopkg.in/telebot.v3"
)

type Base struct {
	File     *tb.File
	Filename string
}

func (b Base) GetFile() *tb.File {
	return b.File
}

func (b Base) GetFilename() string {
	return b.Filename
}

func (b Base) Cleanup() {
	os.Remove(b.Filename)
}

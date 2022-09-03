package media

import tb "gopkg.in/telebot.v3"

type Type int

// why isn't this a separate type declared in the telebot?
const (
	AnimationType Type = iota
	AudioType
	DocumentType
	PhotoType
	StickerType
	VideoType
	VideoNoteType
	VoiceType
)

type Media interface {
	Distort() (tb.Sendable, error)
	GetFile() *tb.File
	GetFilename() string
	GetType() Type
	Cleanup()
}

package media

import (
	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/tools"
)

type Sticker struct {
	Base
	Animated bool
	Video    bool
}

func (s Sticker) Distort() (tb.Sendable, error) {
	if s.Animated || s.Video {
		return nil, tools.NotSupportedErr
	}
	err := distorters.DistortImage(s.Filename)
	return &tb.Sticker{File: tb.FromDisk(s.Filename)}, err
}

func (s Sticker) GetType() Type {
	return StickerType
}

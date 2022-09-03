package media

import (
	"os"

	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/distorters"
)

type Voice struct {
	Base
	output string
}

func (v Voice) Distort() (tb.Sendable, error) {
	var err error
	v.output, err = distorters.DistortSound(v.Filename)
	return &tb.Voice{File: tb.FromDisk(v.output)}, err
}

func (v Voice) GetType() Type {
	return VoiceType
}

func (v Voice) Cleanup() {
	v.Base.Cleanup()
	os.Remove(v.output)
}

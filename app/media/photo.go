package media

import (
	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/distorters"
)

type Photo struct {
	Base
	Caption string
}

func (p Photo) Distort() (tb.Sendable, error) {
	err := distorters.DistortImage(p.Filename)
	if err != nil {
		return nil, err
	}
	photo := &tb.Photo{File: tb.FromDisk(p.Filename)}
	if p.Caption != "" {
		photo.Caption = distorters.DistortText(p.Caption)
	}

	return photo, nil
}

func (p Photo) GetType() Type {
	return PhotoType
}

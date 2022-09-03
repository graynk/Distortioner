package bot

import (
	"github.com/google/uuid"
	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/media"
	"github.com/graynk/distortioner/tools"
)

const mediaKey = "media"

func (d distorterBot) GetTheFileOrErrorOutMiddleware(h tb.HandlerFunc) tb.HandlerFunc {
	//TODO: handle replies
	return func(c tb.Context) error {
		mediaFile, err := JustGetTheMedia(c.Bot(), c.Message())
		if err != nil {
			return err
		}
		c.Set(mediaKey, mediaFile)
		return h(c)
	}
}

func (d distorterBot) ErrMiddleware(h tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) error {
		err := h(c)
		if err != nil {
			errStr, isFriendly := tools.GetUserFriendlyErr(err)
			if !isFriendly {
				d.logger.Error(err)
			}
			if sentErr := d.SendMessageWithRepeater(c, errStr); sentErr != nil {
				d.logger.Error(sentErr)
			}
		}
		return err
	}
}

func (d distorterBot) ShutdownMiddleware(h tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) error {
		d.graceWg.Add(1)
		err := h(c)
		d.graceWg.Done()
		return err
	}
}

// there's a bug in telebot where media.MediaFile() does not check for Sticker.
func JustGetTheMedia(b *tb.Bot, m *tb.Message) (media.Media, error) {
	filename := uuid.New().String()
	var mediaFile media.Media
	switch {
	case m.Photo != nil:
		mediaFile = media.Photo{
			Base:    media.Base{File: m.Photo.MediaFile(), Filename: filename},
			Caption: m.Photo.Caption,
		}
	case m.Voice != nil:
		mediaFile = media.Voice{
			Base: media.Base{File: m.Voice.MediaFile(), Filename: filename},
		}
	//case m.Audio != nil:
	//	return m.Audio
	//case m.Animation != nil:
	//	return m.Animation
	//case m.Document != nil:
	//	return m.Document
	//case m.Video != nil:
	//	return m.Video
	//case m.VideoNote != nil:
	//	return m.VideoNote
	case m.Sticker != nil:
		mediaFile = media.Sticker{
			Base:     media.Base{File: m.Sticker.MediaFile(), Filename: filename},
			Animated: m.Sticker.Animated,
			Video:    m.Sticker.Video,
		}
	}

	return mediaFile, DownloadFile(b, mediaFile)
}

func DownloadFile(b *tb.Bot, media media.Media) error {
	if media == nil {
		return tools.NoFileErr
	}
	file := media.GetFile()
	if file == nil {
		return tools.NoFileErr
	}
	if file.FileSize > tools.MaxSizeMb {
		return tools.TooBigErr
	}
	if err := b.Download(file, media.GetFilename()); err != nil {
		return tools.FailedToDownloadErr
	}

	return nil
}

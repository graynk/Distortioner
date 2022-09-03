package bot

import (
	"os"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/media"
	"github.com/graynk/distortioner/tools"
)

func (d distorterBot) HandleAnimationDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
		return d.SendMessageWithRepeater(c, tools.FormatRateLimitResponse(diff))
	}

	d.videoWorker.Submit(m.Chat.ID, func() {
		progressMessage, filename, output, err := d.HandleAnimationCommon(c)
		failed := err != nil
		if failed {
			if progressMessage != nil && progressMessage.Text != distorters.TooLong {
				d.DoneMessageWithRepeater(b, progressMessage, failed)
			}
			d.logger.Error(err)
			return
		}
		defer os.Remove(filename)
		defer os.Remove(output)

		// not sure why, but now I'm forced to specify filename manually
		distorted := &tb.Animation{File: tb.FromDisk(output), FileName: output}
		if m.Caption != "" {
			distorted.Caption = distorters.DistortText(m.Caption)
		}
		err = d.SendMessageWithRepeater(c, distorted)
		d.DoneMessageWithRepeater(b, progressMessage, failed)
	})
	if d.videoWorker.IsBusy() {
		d.SendMessageWithRepeater(c, distorters.Queued)
	}
	return nil
}

func (d distorterBot) HandleSimpleMediaDistortion(c tb.Context) error {
	mediaFile := c.Get(mediaKey).(media.Media)
	defer mediaFile.Cleanup()
	distorted, err := mediaFile.Distort()
	if err != nil {
		return err
	}
	return d.SendMessageWithRepeater(c, distorted)
}

func (d distorterBot) HandleTextDistortion(c tb.Context) error {
	return d.SendMessageWithRepeater(c, distorters.DistortText(c.Text()))
}

func (d distorterBot) HandleVideoDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
		return d.SendMessageWithRepeater(c, tools.FormatRateLimitResponse(diff))
	}

	d.videoWorker.Submit(m.Chat.ID, func() {
		output, progressMessage, err := d.HandleVideoCommon(c)
		failed := err != nil
		if failed {
			if progressMessage != nil && progressMessage.Text != distorters.TooLong {
				d.DoneMessageWithRepeater(b, progressMessage, failed)
			}
			d.logger.Error(err)
			return
		}
		defer os.Remove(output)

		distorted := &tb.Video{File: tb.FromDisk(output)}
		err = d.SendMessageWithRepeater(c, distorted)
		d.DoneMessageWithRepeater(b, progressMessage, failed)
		if err != nil {
			d.logger.Error(err)
		}
	})
	if d.videoWorker.IsBusy() {
		d.SendMessageWithRepeater(c, distorters.Queued)
	}
	return nil
}

func (d distorterBot) HandleVideoNoteDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
		return d.SendMessageWithRepeater(c, tools.FormatRateLimitResponse(diff))
	}

	d.videoWorker.Submit(m.Chat.ID, func() {
		output, progressMessage, err := d.HandleVideoCommon(c)
		failed := err != nil
		if failed {
			if progressMessage != nil && progressMessage.Text != distorters.TooLong {
				d.DoneMessageWithRepeater(b, progressMessage, failed)
			}
			d.logger.Error(err)
			return
		}
		defer os.Remove(output)
		distorted := &tb.VideoNote{File: tb.FromDisk(output)}
		err = d.SendMessageWithRepeater(c, distorted)
		d.DoneMessageWithRepeater(b, progressMessage, failed)
	})
	if d.videoWorker.IsBusy() {
		d.SendMessageWithRepeater(c, distorters.Queued)
	}
	return nil
}

func (d distorterBot) HandleReplyDistortion(c tb.Context) error {
	//	m := c.Message()
	//	if m.ReplyTo == nil {
	//		msg := "You need to reply with this command to the media you want distorted."
	//		if m.FromGroup() {
	//			msg += "\nYou might also need to make chat history visible for new members if your group is private."
	//		}
	//		return c.Send(msg)
	//	}
	//	original := m.ReplyTo
	//	update := c.Update()
	//	update.Message = original
	//	tweakedContext := c.Bot().NewContext(update)
	//	originalMedia := tools.JustGetTheMedia(original)
	//	switch originalMedia.MediaType() {
	//	case tools.Animation:
	//		return d.HandleAnimationDistortion(tweakedContext)
	//	case tools.Sticker:
	//		fallthrough
	//	case tools.Photo:
	//		fallthrough
	//	case tools.Voice:
	//		return d.HandleSimpleMediaDistortion(tweakedContext)
	//	case tools.Video:
	//		return d.HandleVideoDistortion(tweakedContext)
	//	case tools.VideoNote:
	//		return d.HandleVideoNoteDistortion(tweakedContext)
	//	default:
	//		if original.Text != "" {
	//			return d.HandleTextDistortion(tweakedContext)
	//		}
	//	}
	return nil
}

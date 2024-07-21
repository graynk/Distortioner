package main

import (
	"os"
	"strings"
	"sync"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/tools"
)

const (
	NotEnoughRights = "The bot does not have enough rights to send media to your chat"
	NotSupported    = "Not supported yet, sorry"
)

type MethodOfResponding = int

const (
	Reply MethodOfResponding = iota
	Send
)

func (d DistorterBot) HandleAnimationCommon(c tb.Context) (*tb.Message, string, string, error) {
	m := c.Message()
	b := c.Bot()
	progressMessage, err := d.SendMessage(c, "Downloading...", Reply)
	if err != nil {
		d.logger.Error(err)
		return nil, "", "", err
	}
	filename, err := tools.JustGetTheFile(b, m)
	if err != nil {
		d.logger.Error(err)
		return nil, "", "", err
	}
	animationOutput := filename + ".mp4"
	progressChan := make(chan string, 3)
	go distorters.DistortVideo(filename, d.codec, animationOutput, progressChan)
	for report := range progressChan {
		if progressMessage == nil {
			continue
		}
		msg, err := b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
		if err == nil && msg != nil {
			progressMessage = msg
		}
	}
	_, err = os.Stat(animationOutput)
	return progressMessage, filename, animationOutput, err
}

func (d DistorterBot) HandleVideoCommon(c tb.Context) (string, *tb.Message, error) {
	progressMessage, filename, animationOutput, err := d.HandleAnimationCommon(c)
	defer os.Remove(filename)
	if err != nil {
		if progressMessage != nil && progressMessage.Text != distorters.TooLong {
			d.DoneMessageWithRepeater(c.Bot(), progressMessage, true)
		}
		return "", progressMessage, err
	}
	defer os.Remove(animationOutput)
	soundOutput := filename + ".ogg"
	err = distorters.DistortSound(filename, soundOutput)
	if err != nil {
		soundOutput = ""
	} else {
		defer os.Remove(soundOutput)
	}
	output := filename + "Final.mp4"
	if progressMessage != nil {
		// intentionally not updating progressMessage variable
		c.Edit(progressMessage, "Muxing frames with sound back together...")
	}
	err = distorters.CollectAnimationAndSound(animationOutput, soundOutput, output)
	return output, progressMessage, err
}

func (d DistorterBot) HandleVideoSticker(c tb.Context) (string, string, error) {
	filename, err := tools.JustGetTheFile(c.Bot(), c.Message())
	if err != nil {
		d.logger.Error(err)
		return "", "", err
	}
	animationOutput := filename + ".webm"
	group := sync.WaitGroup{}
	group.Add(1)
	go distorters.DistortVideoSticker(filename, animationOutput, &group)
	group.Wait()
	_, err = os.Stat(animationOutput)
	return filename, animationOutput, err
}

func (d DistorterBot) dealWithStatusMessage(b *tb.Bot, m *tb.Message, failed bool) error {
	if m == nil {
		return nil
	}
	var err error
	if failed {
		_, err = b.Edit(m, distorters.Failed)
	} else {
		err = b.Delete(m)
	}
	return err
}

func (d DistorterBot) DoneMessageWithRepeater(b *tb.Bot, m *tb.Message, failed bool) {
	err := d.dealWithStatusMessage(b, m, failed)
	for err != nil {
		var timeout int
		timeout, err = tools.ExtractPossibleTimeout(err)
		if err != nil {
			return
		}
		time.Sleep(time.Duration(timeout) * time.Second)
		err = d.dealWithStatusMessage(b, m, failed)
	}
}

func (d DistorterBot) SendMessage(c tb.Context, toSend interface{}, method MethodOfResponding) (*tb.Message, error) {
	b := c.Bot()
	message := c.Message()

	var m *tb.Message
	var err error
	if method == Reply {
		m, err = b.Reply(message, toSend)
	} else {
		m, err = b.Send(message.Chat, toSend)
	}
	for err != nil {
		switch {
		case strings.Contains(err.Error(), "not enough rights to send"):
			b.Reply(message, NotEnoughRights)
		case strings.Contains(err.Error(), "bot was blocked by the user (403)"):
			d.videoWorker.BanUser(message.Chat.ID)
			return nil, nil
		case strings.Contains(err.Error(), "telegram: Bad Request: message to be replied not found (400)"):
			return d.SendMessage(c, toSend, Send)
		}

		var timeout int
		timeout, err = tools.ExtractPossibleTimeout(err)
		if err != nil {
			d.logger.Error(err)
			return nil, err
		}
		time.Sleep(time.Duration(timeout) * time.Second)
		m, err = b.Reply(message, toSend)
		if err != nil {
			d.logger.Error(err)
		}
	}

	return m, nil
}

func (d DistorterBot) SendMessageWithRepeater(c tb.Context, toSend interface{}) error {
	_, err := d.SendMessage(c, toSend, Reply)
	return err
}

func (d DistorterBot) ApplyShutdownMiddleware(h tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) error {
		d.graceWg.Add(1)
		err := h(c)
		d.graceWg.Done()
		return err
	}
}

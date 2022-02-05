package main

import (
	"os"
	"strings"
	"time"

	tb "github.com/graynk/telebot"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/tools"
)

const (
	NotEnoughRights = "The bot does not have enough rights to send media to your chat"
	NotSupported    = "Not supported yet, sorry"
)

func (d DistorterBot) HandleAnimationCommon(m *tb.Message) (*tb.Message, string, string, error) {
	progressMessage, err := d.SendMessageWithRepeater(m.Chat, "Downloading...")
	if err != nil {
		d.logger.Error(err)
		return nil, "", "", err
	}
	filename, err := tools.JustGetTheFile(d.b, m)
	if err != nil {
		d.logger.Error(err)
		return nil, "", "", err
	}
	animationOutput := filename + ".mp4"
	progressChan := make(chan string, 3)
	go distorters.DistortVideo(filename, animationOutput, progressChan)
	for report := range progressChan {
		progressMessage, _ = d.b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	_, err = os.Stat(animationOutput)
	return progressMessage, filename, animationOutput, err
}

func (d DistorterBot) HandleVideoCommon(m *tb.Message) (string, error) {
	progressMessage, filename, animationOutput, err := d.HandleAnimationCommon(m)
	if err != nil {
		if progressMessage != nil && progressMessage.Text != distorters.TooLong {
			d.DoneMessageWithRepeater(progressMessage, true)
		}
		return "", err
	}
	defer os.Remove(filename)
	defer os.Remove(animationOutput)
	soundOutput := filename + ".ogg"
	err = distorters.DistortSound(filename, soundOutput)
	if err != nil {
		soundOutput = ""
	} else {
		defer os.Remove(soundOutput)
	}
	output := filename + "Final.mp4"
	d.b.Edit(progressMessage, "Muxing frames with sound back together...")
	err = distorters.CollectAnimationAndSound(animationOutput, soundOutput, output)
	d.DoneMessageWithRepeater(progressMessage, err != nil)
	return output, err
}

func (d DistorterBot) dealWithStatusMessage(m *tb.Message, failed bool) error {
	var err error
	if failed {
		_, err = d.b.Edit(m, distorters.Failed)
	} else {
		err = d.b.Delete(m)
	}
	return err
}

func (d DistorterBot) DoneMessageWithRepeater(m *tb.Message, failed bool) {
	err := d.dealWithStatusMessage(m, failed)
	for err != nil {
		var timeout int
		timeout, err = tools.ExtractPossibleTimeout(err)
		if err != nil {
			return
		}
		time.Sleep(time.Duration(timeout) * time.Second)
		err = d.dealWithStatusMessage(m, failed)
	}
}

func (d DistorterBot) SendMessageWithRepeater(chat *tb.Chat, toSend interface{}) (*tb.Message, error) {
	m, err := d.b.Send(chat, toSend)
	for err != nil {
		if strings.Contains(err.Error(), "not enough rights to send") {
			d.b.Send(chat, NotEnoughRights)
		}
		var timeout int
		timeout, err = tools.ExtractPossibleTimeout(err)
		if err != nil {
			d.logger.Error(err)
			return nil, err
		}
		time.Sleep(time.Duration(timeout) * time.Second)
		m, err = d.b.Send(chat, toSend)
		if err != nil {
			d.logger.Error(err)
		}
	}

	return m, nil
}

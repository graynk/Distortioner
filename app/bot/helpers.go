package bot

import (
	"os"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/tools"
)

func (d distorterBot) HandleAnimationCommon(c tb.Context) (*tb.Message, string, string, error) {
	//m := c.Message()
	//b := c.Bot()
	//progressMessage, err := d.GetProgressMessage(c, "Downloading...")
	//if err != nil {
	//	d.logger.Error(err)
	//	return nil, "", "", err
	//}
	//filename, err := DownloadFile(b, JustGetTheMedia(m))
	//if err != nil {
	//	d.logger.Error(err)
	//	return nil, "", "", err
	//}
	//animationOutput := filename + ".mp4"
	//progressChan := make(chan string, 3)
	//go distorters.DistortVideo(filename, animationOutput, progressChan)
	//for report := range progressChan {
	//	if progressMessage == nil {
	//		continue
	//	}
	//	msg, err := b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	//	if err == nil && msg != nil {
	//		progressMessage = msg
	//	}
	//}
	//_, err = os.Stat(animationOutput)
	//return progressMessage, filename, animationOutput, err
	return nil, "", "", nil
}

func (d distorterBot) HandleVideoCommon(c tb.Context) (string, *tb.Message, error) {
	progressMessage, filename, animationOutput, err := d.HandleAnimationCommon(c)
	defer os.Remove(filename)
	if err != nil {
		if progressMessage != nil && progressMessage.Text != distorters.TooLong {
			d.DoneMessageWithRepeater(c.Bot(), progressMessage, true)
		}
		return "", progressMessage, err
	}
	defer os.Remove(animationOutput)
	soundOutput, err := distorters.DistortSound(filename)
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

func (d distorterBot) HandleVideoSticker(c tb.Context) (string, string, error) {
	//filename, err := tools.DownloadFile(c.Bot(), tools.JustGetTheMedia(c.Message()))
	//if err != nil {
	//	d.logger.Error(err)
	//	return "", "", err
	//}
	//animationOutput := filename + ".webm"
	//group := sync.WaitGroup{}
	//group.Add(1)
	//go distorters.DistortVideoSticker(filename, animationOutput, &group)
	//group.Wait()
	//_, err = os.Stat(animationOutput)
	//return filename, animationOutput, err
	return "", "", nil
}

func (d distorterBot) dealWithStatusMessage(b *tb.Bot, m *tb.Message, failed bool) error {
	if m == nil {
		return nil
	}
	var err error
	if failed {
		_, err = b.Edit(m, tools.Failed)
	} else {
		err = b.Delete(m)
	}
	return err
}

func (d distorterBot) DoneMessageWithRepeater(b *tb.Bot, m *tb.Message, failed bool) {
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

func (d distorterBot) GetProgressMessage(c tb.Context, toSend interface{}) (*tb.Message, error) {
	b := c.Bot()
	chat := c.Chat()
	m, err := b.Send(chat, toSend)
	for err != nil {
		if strings.Contains(err.Error(), "not enough rights to send") {
			b.Send(chat, tools.NotEnoughRights)
		}
		var timeout int
		timeout, err = tools.ExtractPossibleTimeout(err)
		if err != nil {
			d.logger.Error(err)
			return nil, err
		}
		time.Sleep(time.Duration(timeout) * time.Second)
		m, err = b.Send(chat, toSend)
		if err != nil {
			d.logger.Error(err)
		}
	}

	return m, nil
}

func (d distorterBot) SendMessageWithRepeater(c tb.Context, toSend interface{}) error {
	_, err := d.GetProgressMessage(c, toSend)
	return err
}

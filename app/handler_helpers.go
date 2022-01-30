package main

import (
	"log"
	"os"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/tools"
)

func HandleAnimationCommon(b *tb.Bot, m *tb.Message) (*tb.Message, string, string, error) {
	progressMessage, err := SendMessageWithRepeater(b, m.Chat, "Downloading...")
	if err != nil {
		log.Println(err)
		return nil, "", "", err
	}
	filename, err := tools.JustGetTheFile(b, m)
	if err != nil {
		return nil, "", "", err
	}
	animationOutput := filename + ".mp4"
	progressChan := make(chan string, 3)
	go distorters.DistortVideo(filename, animationOutput, progressChan)
	for report := range progressChan {
		progressMessage, _ = b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	_, err = os.Stat(animationOutput)
	return progressMessage, filename, animationOutput, err
}

func HandleVideoCommon(b *tb.Bot, m *tb.Message) (string, error) {
	progressMessage, filename, animationOutput, err := HandleAnimationCommon(b, m)
	if err != nil {
		if progressMessage != nil && progressMessage.Text != distorters.TooLong {
			DoneMessageWithRepeater(b, progressMessage, true)
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
	b.Edit(progressMessage, "Muxing frames with sound back together...")
	err = distorters.CollectAnimationAndSound(animationOutput, soundOutput, output)
	DoneMessageWithRepeater(b, progressMessage, err != nil)
	return output, err
}

func dealWithStatusMessage(b *tb.Bot, m *tb.Message, failed bool) error {
	var err error
	if failed {
		_, err = b.Edit(m, distorters.Failed)
	} else {
		err = b.Delete(m)
	}
	return err
}

func DoneMessageWithRepeater(b *tb.Bot, m *tb.Message, failed bool) {
	err := dealWithStatusMessage(b, m, failed)
	for err != nil {
		var timeout int
		timeout, err = tools.ExtractPossibleTimeout(err)
		if err != nil {
			return
		}
		log.Printf("sleeping for %d before finishing up\n", timeout)
		time.Sleep(time.Duration(timeout) * time.Second)
		err = dealWithStatusMessage(b, m, failed)
	}
}

func SendMessageWithRepeater(b *tb.Bot, chat *tb.Chat, toSend interface{}) (*tb.Message, error) {
	m, err := b.Send(chat, toSend)
	for err != nil {
		var timeout int
		timeout, err = tools.ExtractPossibleTimeout(err)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		log.Printf("sleeping for %d\n", timeout)
		time.Sleep(time.Duration(timeout) * time.Second)
		m, err = b.Send(chat, toSend)
		if err != nil {
			log.Println(err)
		}
	}

	return m, nil
}

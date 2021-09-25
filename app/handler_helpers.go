package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"os"
	"time"
)

func handleAnimationCommon(b *tb.Bot, m *tb.Message) (*tb.Message, string, string, error) {
	progressMessage, err := b.Send(m.Chat, "Downloading...")
	if err != nil {
		log.Println(err)
		return nil, "", "", err
	}
	filename, err := justGetTheFile(b, m)
	if err != nil {
		return nil, "", "", err
	}
	animationOutput := filename + ".mp4"
	progressChan := make(chan string, 3)
	go distortVideo(filename, animationOutput, progressChan)
	for report := range progressChan {
		b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	return progressMessage, filename, animationOutput, nil
}

func handleVideoCommon(b *tb.Bot, m *tb.Message) (string, error) {
	progressMessage, filename, animationOutput, err := handleAnimationCommon(b, m)
	if err != nil {
		return "", err
	}
	defer os.Remove(filename)
	defer os.Remove(animationOutput)
	soundOutput := filename + ".ogg"
	distortSound(filename, soundOutput)
	defer os.Remove(soundOutput)
	output := filename + "Final.mp4"
	b.Edit(progressMessage, "Muxing frames with sound back together...")
	collectAnimationAndSound(animationOutput, soundOutput, output)
	doneMessageRepeater(b, progressMessage)
	return output, nil
}

func doneMessageRepeater(b *tb.Bot, m *tb.Message) {
	done := "Done!"
	_, err := b.Edit(m, done)
	for err != nil {
		timeout, err := extractPossibleTimeout(err)
		if err != nil {
			return
		}
		time.Sleep(time.Duration(timeout))
		_, err = b.Edit(m, done)
	}
}

func sendMessageRepeater(b *tb.Bot, chat *tb.Chat, toSend interface{}) {
	_, err := b.Send(chat, toSend)
	for err != nil {
		timeout, err := extractPossibleTimeout(err)
		if err != nil {
			return
		}
		time.Sleep(time.Duration(timeout))
		_, err = b.Send(chat, toSend)
		if err != nil {
			log.Println(err)
		}
	}
}
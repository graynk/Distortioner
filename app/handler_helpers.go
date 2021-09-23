package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"os"
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
	go distortVideo(filename, animationOutput, progressChan, false)
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
	b.Edit(progressMessage, "Done!")
	return output, nil
}

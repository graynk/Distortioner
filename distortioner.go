package main

import (
	"log"
	"os"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func handlePhotoDistortion(b *tb.Bot, m *tb.Message) {
	filename := uniqueFileName(m.Photo.FileID, m.Unixtime)
	err := b.Download(m.Photo.MediaFile(), filename)
	if err != nil {
		log.Println(err)
		return
	}
	defer os.Remove(filename)
	distortImage(filename)
	// sure would be nice to have generics here
	distorted := &tb.Photo{File: tb.FromDisk(filename)}
	_, err = b.Send(m.Sender, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleStickerDistortion(b *tb.Bot, m *tb.Message) {
	filename := uniqueFileName(m.Sticker.FileID, m.Unixtime)
	err := b.Download(&m.Sticker.File, filename)
	if err != nil {
		log.Println(err)
		return
	}
	defer os.Remove(filename)
	distortImage(filename)
	distorted := &tb.Sticker{File: tb.FromDisk(filename)}
	_, err = b.Send(m.Sender, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleAnimationDistortion(b *tb.Bot, m *tb.Message) {
	filename := uniqueFileName(m.Animation.FileID, m.Unixtime)
	progressMessage, err := b.Send(m.Sender, "Downloading...")
	if err != nil {
		log.Println(err)
		return
	}
	err = b.Download(m.Animation.MediaFile(), filename)
	if err != nil {
		b.Edit(progressMessage, "Couldn't download animation")
		log.Println(err)
		return
	}
	defer os.Remove(filename)

	output := filename + ".mp4"
	progressChan := make(chan string, 3)
	go distortVideo(filename, output, progressChan)
	for report := range progressChan {
		b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	defer os.Remove(output)

	distorted := &tb.Animation{File: tb.FromDisk(output)}
	_, err = b.Send(m.Sender, distorted)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("DISTORTIONER_BOT_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/start", func(m *tb.Message) {
		b.Send(m.Sender, "Send me a picture, a sticker or a GIF and I'll distort it")
	})

	b.Handle(tb.OnAnimation, func(m *tb.Message) {
		handleAnimationDistortion(b, m)
	})

	b.Handle(tb.OnSticker, func(m *tb.Message) {
		handleStickerDistortion(b, m)
	})

	b.Handle(tb.OnPhoto, func(m *tb.Message) {
		handlePhotoDistortion(b, m)
	})

	b.Start()
}

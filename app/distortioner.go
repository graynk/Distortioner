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
		b.Send(m.Chat, "Failed to download photo")
		return
	}
	defer os.Remove(filename)
	distortImage(filename)
	// sure would be nice to have generics here
	distorted := &tb.Photo{File: tb.FromDisk(filename)}
	if m.Caption != "" {
		distorted.Caption = distortText(m.Caption)
	}
	_, err = b.Send(m.Chat, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleStickerDistortion(b *tb.Bot, m *tb.Message) {
	filename := uniqueFileName(m.Sticker.FileID, m.Unixtime)
	err := b.Download(&m.Sticker.File, filename)
	if err != nil {
		log.Println(err)
		b.Send(m.Chat, "Failed to download sticker")
		return
	}
	defer os.Remove(filename)
	distortImage(filename)
	distorted := &tb.Sticker{File: tb.FromDisk(filename)}
	_, err = b.Send(m.Chat, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleAnimationDistortion(b *tb.Bot, m *tb.Message) {
	if m.Animation.Duration > 30 {
		b.Send(m.Chat, "Senpai, it's too long..")
		return
	}
	filename := uniqueFileName(m.Animation.FileID, m.Unixtime)
	progressMessage, err := b.Send(m.Chat, "Downloading...")
	if err != nil {
		log.Println(err)
		return
	}
	err = b.Download(m.Animation.MediaFile(), filename)
	if err != nil {
		b.Edit(progressMessage, "Failed to download animation")
		log.Println(err)
		return
	}
	defer os.Remove(filename)

	output := filename + ".mp4"
	progressChan := make(chan string, 3)
	go distortVideo(filename, output, progressChan, true)
	for report := range progressChan {
		b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	defer os.Remove(output)

	distorted := &tb.Animation{File: tb.FromDisk(output)}
	if m.Caption != "" {
		distorted.Caption = distortText(m.Caption)
	}
	_, err = b.Send(m.Chat, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleVoiceDistortion(b *tb.Bot, m *tb.Message) {
	filename := uniqueFileName(m.Voice.FileID, m.Unixtime)
	err := b.Download(&m.Voice.File, filename)
	if err != nil {
		log.Println(err)
		b.Send(m.Chat, "Failed to download voice message")
		return
	}
	defer os.Remove(filename)
	output := filename + ".ogg"
	distortSound(filename, output)
	defer os.Remove(output)

	distorted := &tb.Voice{File: tb.FromDisk(output)}
	_, err = b.Send(m.Chat, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleVideoDistortion(b *tb.Bot, m *tb.Message) {
	if m.Video.Duration > 30 {
		b.Send(m.Chat, "Senpai, it's too long..")
		return
	}
	filename := uniqueFileName(m.Video.FileID, m.Unixtime)
	progressMessage, err := b.Send(m.Chat, "Downloading...")
	if err != nil {
		log.Println(err)
		return
	}
	err = b.Download(&m.Video.File, filename)
	if err != nil {
		log.Println(err)
		b.Send(m.Chat, "Failed to download video")
		return
	}
	defer os.Remove(filename)
	animationOutput := filename + ".mp4"
	soundOutput := filename + ".ogg"
	output := filename + "Final.mp4"
	progressChan := make(chan string, 3)
	go distortSound(filename, soundOutput) // no need to pass channel, it's always faster
	go distortVideo(filename, animationOutput, progressChan, false)
	for report := range progressChan {
		b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	defer os.Remove(animationOutput)
	defer os.Remove(soundOutput)
	b.Edit(progressMessage, "Muxing frames with sound back together...")
	collectAnimationAndSound(animationOutput, soundOutput, output)
	defer os.Remove(output)
	b.Edit(progressMessage, "Done!")

	distorted := &tb.Video{File: tb.FromDisk(output)}
	_, err = b.Send(m.Chat, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleVideoNoteDistortion(b *tb.Bot, m *tb.Message) {
	filename := uniqueFileName(m.VideoNote.FileID, m.Unixtime)
	progressMessage, err := b.Send(m.Chat, "Downloading...")
	if err != nil {
		log.Println(err)
		return
	}
	err = b.Download(&m.VideoNote.File, filename)
	if err != nil {
		log.Println(err)
		b.Send(m.Chat, "Failed to download video note")
		return
	}
	defer os.Remove(filename)
	animationOutput := filename + ".mp4"
	soundOutput := filename + ".ogg"
	output := filename + "Final.mp4"
	progressChan := make(chan string, 3)
	go distortSound(filename, soundOutput) // no need to pass channel, it's always faster
	go distortVideo(filename, animationOutput, progressChan, false)
	for report := range progressChan {
		b.Edit(progressMessage, report, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	defer os.Remove(animationOutput)
	defer os.Remove(soundOutput)
	b.Edit(progressMessage, "Muxing frames with sound back together...")
	collectAnimationAndSound(animationOutput, soundOutput, output)
	defer os.Remove(output)
	b.Edit(progressMessage, "Done!")

	distorted := &tb.VideoNote{File: tb.FromDisk(output)}
	_, err = b.Send(m.Chat, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleTextDistortion(b *tb.Bot, m *tb.Message) {
	b.Send(m.Chat, distortText(m.Text))
}

func handleReplyDistortion(b *tb.Bot, m *tb.Message) {
	if m.ReplyTo == nil {
		b.Send(m.Chat, "You need to reply with this command to the media you want distorted")
		return
	}
	original := m.ReplyTo
	if original.Animation != nil {
		handleAnimationDistortion(b, original)
	} else if original.Sticker != nil {
		handleStickerDistortion(b, original)
	} else if original.Photo != nil {
		handlePhotoDistortion(b, original)
	} else if original.Voice != nil {
		handleVoiceDistortion(b, original)
	} else if original.Video != nil {
		handleVideoDistortion(b, original)
	} else if original.VideoNote != nil {
		handleVideoNoteDistortion(b, original)
	} else if original.Text != "" {
		handleTextDistortion(b, original)
	}
}

func main() {
	// TODO: Attach middleware to reduce boilerplate
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("DISTORTIONER_BOT_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/start", func(m *tb.Message) {
		b.Send(m.Chat, "Send me a picture, a sticker, a voice message or a GIF and I'll distort it")
	})

	b.Handle("/distort", func(m *tb.Message) {
		handleReplyDistortion(b, m)
	})

	b.Handle(tb.OnAnimation, func(m *tb.Message) {
		handleAnimationDistortion(b, m)
	})

	b.Handle(tb.OnSticker, func(m *tb.Message) {
		if m.Sticker.Animated {
			// TODO: there might be a nice way to distort them too, just parse the data and move around stuff, I guess
			return
		}
		handleStickerDistortion(b, m)
	})

	b.Handle(tb.OnPhoto, func(m *tb.Message) {
		handlePhotoDistortion(b, m)
	})

	b.Handle(tb.OnVoice, func(m *tb.Message) {
		handleVoiceDistortion(b, m)
	})

	b.Handle(tb.OnVideo, func(m *tb.Message) {
		handleVideoDistortion(b, m)
	})

	b.Handle(tb.OnVideoNote, func(m *tb.Message) {
		handleVideoNoteDistortion(b, m)
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		if m.FromGroup() {
			return
		}
		handleTextDistortion(b, m)
	})

	b.Start()
}

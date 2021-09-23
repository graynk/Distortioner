package main

import (
	"log"
	"os"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func handleAnimationDistortion(b *tb.Bot, m *tb.Message) {
	if m.Animation.Duration > 30 {
		b.Send(m.Chat, "Senpai, it's too long..")
		return
	}

	_, filename, output, err := handleAnimationCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(filename)
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

func handlePhotoDistortion(b *tb.Bot, m *tb.Message) {
	filename, err := justGetTheFile(b, m)
	if err != nil {
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
	filename, err := justGetTheFile(b, m)
	if err != nil {
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

func handleTextDistortion(b *tb.Bot, m *tb.Message) {
	b.Send(m.Chat, distortText(m.Text))
}

func handleVideoDistortion(b *tb.Bot, m *tb.Message) {
	if m.Video.Duration > 30 {
		b.Send(m.Chat, "Senpai, it's too long..")
		return
	}
	output, err := handleVideoCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(output)

	distorted := &tb.Video{File: tb.FromDisk(output)}
	_, err = b.Send(m.Chat, distorted)
	if err != nil {
		log.Println(err)
	}
}

func handleVideoNoteDistortion(b *tb.Bot, m *tb.Message) {
	output, err := handleVideoCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(output)
	distorted := &tb.VideoNote{File: tb.FromDisk(output)}
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
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("DISTORTIONER_BOT_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/start", func(m *tb.Message) {
		b.Send(m.Chat, "Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
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

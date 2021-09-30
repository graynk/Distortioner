package main

import (
	"log"
	"os"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

const MaxSizeMb = 20_000_000

func handleAnimationDistortion(b *tb.Bot, m *tb.Message, rl *RateLimiter) {
	if m.Animation.Duration > 30 {
		b.Send(m.Chat, "Senpai, it's too long..")
		return
	} else if m.Animation.FileSize > MaxSizeMb {
		b.Send(m.Chat, "Senpai, it's too big..")
		return
	} else if rate, diff := rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > AllowedOverTime {
		b.Send(m.Chat, formatRateLimitResponse(diff))
		return
	}

	log.Printf("start processing animation")
	progressMessage, filename, output, err := handleAnimationCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(filename)
	defer os.Remove(output)
	doneMessageWithRepeater(b, progressMessage)

	distorted := &tb.Animation{File: tb.FromDisk(output)}
	if m.Caption != "" {
		distorted.Caption = distortText(m.Caption)
	}
	sendMessageWithRepeater(b, m.Chat, distorted)
}

func handlePhotoDistortion(b *tb.Bot, m *tb.Message) {
	log.Printf("start processing photo")
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
	sendMessageWithRepeater(b, m.Chat, distorted)
}

func handleStickerDistortion(b *tb.Bot, m *tb.Message) {
	if m.Sticker.Animated {
		// TODO: there might be a nice way to distort them too, just parse the data and move around stuff, I guess
		return
	}
	log.Printf("start processing sticker")
	filename, err := justGetTheFile(b, m)
	if err != nil {
		return
	}
	defer os.Remove(filename)
	distortImage(filename)
	distorted := &tb.Sticker{File: tb.FromDisk(filename)}
	sendMessageWithRepeater(b, m.Chat, distorted)
}

func handleTextDistortion(b *tb.Bot, m *tb.Message) {
	log.Printf("start processing text")
	sendMessageWithRepeater(b, m.Chat, distortText(m.Text))
}

func handleVideoDistortion(b *tb.Bot, m *tb.Message, rl *RateLimiter) {
	if m.Video.Duration > 30 {
		b.Send(m.Chat, "Senpai, it's too long..")
		return
	} else if m.Video.FileSize > MaxSizeMb {
		b.Send(m.Chat, "Senpai, it's too big..")
		return
	} else if rate, diff := rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > AllowedOverTime {
		b.Send(m.Chat, formatRateLimitResponse(diff))
		return
	}
	log.Printf("start processing video")
	output, err := handleVideoCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(output)

	distorted := &tb.Video{File: tb.FromDisk(output)}
	sendMessageWithRepeater(b, m.Chat, distorted)
}

func handleVideoNoteDistortion(b *tb.Bot, m *tb.Message, rl *RateLimiter) {
	// video notes are limited with 1 minute anyway
	if m.VideoNote.FileSize > MaxSizeMb {
		b.Send(m.Chat, "Senpai, it's too big..")
		return
	} else if rate, diff := rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > AllowedOverTime {
		b.Send(m.Chat, formatRateLimitResponse(diff))
		return
	}
	log.Printf("start processing video note")
	output, err := handleVideoCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(output)
	distorted := &tb.VideoNote{File: tb.FromDisk(output)}
	sendMessageWithRepeater(b, m.Chat, distorted)
}

func handleVoiceDistortion(b *tb.Bot, m *tb.Message) {
	if m.Voice.FileSize > MaxSizeMb {
		b.Send(m.Chat, "Senpai, it's too big..")
		return
	}
	log.Printf("start processing voice")
	filename, err := justGetTheFile(b, m)
	if err != nil {
		return
	}
	defer os.Remove(filename)
	output := filename + ".ogg"
	err = distortSound(filename, output)
	if err != nil {
		panic(err)
	}
	defer os.Remove(output)

	distorted := &tb.Voice{File: tb.FromDisk(output)}
	sendMessageWithRepeater(b, m.Chat, distorted)
}

func handleReplyDistortion(b *tb.Bot, m *tb.Message, rl *RateLimiter) {
	log.Printf("used /distort")
	if m.ReplyTo == nil {
		b.Send(m.Chat, "You need to reply with this command to the media you want distorted")
		return
	}
	original := m.ReplyTo
	if original.Animation != nil {
		handleAnimationDistortion(b, original, rl)
	} else if original.Sticker != nil {
		handleStickerDistortion(b, original)
	} else if original.Photo != nil {
		handlePhotoDistortion(b, original)
	} else if original.Voice != nil {
		handleVoiceDistortion(b, original)
	} else if original.Video != nil {
		handleVideoDistortion(b, original, rl)
	} else if original.VideoNote != nil {
		handleVideoNoteDistortion(b, original, rl)
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

	rl := NewRateLimiter()

	b.Handle("/start", func(m *tb.Message) {
		b.Send(m.Chat, "Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
	})

	b.Handle("/distort", func(m *tb.Message) {
		handleReplyDistortion(b, m, rl)
	})

	b.Handle(tb.OnAnimation, func(m *tb.Message) {
		handleAnimationDistortion(b, m, rl)
	})

	b.Handle(tb.OnSticker, func(m *tb.Message) {
		handleStickerDistortion(b, m)
	})

	b.Handle(tb.OnPhoto, func(m *tb.Message) {
		handlePhotoDistortion(b, m)
	})

	b.Handle(tb.OnVoice, func(m *tb.Message) {
		handleVoiceDistortion(b, m)
	})

	b.Handle(tb.OnVideo, func(m *tb.Message) {
		handleVideoDistortion(b, m, rl)
	})

	b.Handle(tb.OnVideoNote, func(m *tb.Message) {
		handleVideoNoteDistortion(b, m, rl)
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		if m.FromGroup() {
			return
		}
		handleTextDistortion(b, m)
	})

	b.Start()
}

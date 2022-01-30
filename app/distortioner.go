package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/stats"
	"github.com/graynk/distortioner/tools"
)

const (
	MaxSizeMb = 20_000_000
)

func handleAnimationDistortion(b *tb.Bot, m *tb.Message, rl *tools.RateLimiter) {
	if m.Animation.FileSize > MaxSizeMb {
		b.Send(m.Chat, distorters.TooBig)
		return
	} else if rate, diff := rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		b.Send(m.Chat, tools.FormatRateLimitResponse(diff))
		return
	}

	progressMessage, filename, output, err := HandleAnimationCommon(b, m)
	failed := err != nil && progressMessage.Text != distorters.TooLong
	DoneMessageWithRepeater(b, progressMessage, failed)
	if failed {
		return
	}
	defer os.Remove(filename)
	defer os.Remove(output)

	distorted := &tb.Animation{File: tb.FromDisk(output)}
	if m.Caption != "" {
		distorted.Caption = distorters.DistortText(m.Caption)
	}
	SendMessageWithRepeater(b, m.Chat, distorted)
}

func handlePhotoDistortion(b *tb.Bot, m *tb.Message) {
	filename, err := tools.JustGetTheFile(b, m)
	if err != nil {
		return
	}
	defer os.Remove(filename)
	err = distorters.DistortImage(filename)
	// sure would be nice to have generics here
	if err != nil {
		SendMessageWithRepeater(b, m.Chat, distorters.Failed)
		return
	}
	distorted := &tb.Photo{File: tb.FromDisk(filename)}
	if m.Caption != "" {
		distorted.Caption = distorters.DistortText(m.Caption)
	}
	SendMessageWithRepeater(b, m.Chat, distorted)
}

func handleStickerDistortion(b *tb.Bot, m *tb.Message) {
	if m.Sticker.Animated {
		// TODO: there might be a nice way to distort them too, just parse the data and move stuff around, I guess
		return
	}
	filename, err := tools.JustGetTheFile(b, m)
	if err != nil {
		return
	}
	defer os.Remove(filename)
	err = distorters.DistortImage(filename)
	if err != nil {
		SendMessageWithRepeater(b, m.Chat, distorters.Failed)
		return
	}
	distorted := &tb.Sticker{File: tb.FromDisk(filename)}
	SendMessageWithRepeater(b, m.Chat, distorted)
}

func handleTextDistortion(b *tb.Bot, m *tb.Message) {
	SendMessageWithRepeater(b, m.Chat, distorters.DistortText(m.Text))
}

func handleVideoDistortion(b *tb.Bot, m *tb.Message, rl *tools.RateLimiter) {
	if m.Video.FileSize > MaxSizeMb {
		b.Send(m.Chat, distorters.TooBig)
		return
	} else if rate, diff := rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		b.Send(m.Chat, tools.FormatRateLimitResponse(diff))
		return
	}
	output, err := HandleVideoCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(output)

	distorted := &tb.Video{File: tb.FromDisk(output)}
	SendMessageWithRepeater(b, m.Chat, distorted)
}

func handleVideoNoteDistortion(b *tb.Bot, m *tb.Message, rl *tools.RateLimiter) {
	// video notes are limited with 1 minute anyway
	if m.VideoNote.FileSize > MaxSizeMb {
		b.Send(m.Chat, distorters.TooBig)
		return
	} else if rate, diff := rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		b.Send(m.Chat, tools.FormatRateLimitResponse(diff))
		return
	}
	output, err := HandleVideoCommon(b, m)
	if err != nil {
		return
	}
	defer os.Remove(output)
	distorted := &tb.VideoNote{File: tb.FromDisk(output)}
	SendMessageWithRepeater(b, m.Chat, distorted)
}

func handleVoiceDistortion(b *tb.Bot, m *tb.Message) {
	if m.Voice.FileSize > MaxSizeMb {
		b.Send(m.Chat, distorters.TooBig)
		return
	}
	filename, err := tools.JustGetTheFile(b, m)
	if err != nil {
		return
	}
	defer os.Remove(filename)
	output := filename + ".ogg"
	err = distorters.DistortSound(filename, output)
	if err != nil {
		SendMessageWithRepeater(b, m.Chat, distorters.Failed)
		return
	}
	defer os.Remove(output)

	distorted := &tb.Voice{File: tb.FromDisk(output)}
	SendMessageWithRepeater(b, m.Chat, distorted)
}

func handleReplyDistortion(b *tb.Bot, m *tb.Message, rl *tools.RateLimiter) {
	if m.ReplyTo == nil {
		msg := "You need to reply with this command to the media you want distorted."
		if m.FromGroup() {
			msg += "\nYou might also need to make chat history visible for new members if your group is private."
		}
		b.Send(m.Chat, msg)
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

func handleStatRequest(b *tb.Bot, m *tb.Message, db *stats.DistortionerDB, period stats.Period, adminID int64) {
	if m.Sender.ID != adminID {
		return
	}
	stat, err := db.GetStat(period)
	if err != nil {
		log.Println(err)
		b.Send(m.Chat, err.Error())
		return
	}
	header := "Stats for the past %s"
	switch period {
	case stats.Daily:
		header = fmt.Sprintf(header, "24 hours")
	case stats.Weekly:
		header = fmt.Sprintf(header, "week")
	case stats.Monthly:
		header = fmt.Sprintf(header, "month")
	default:
		log.Printf("stats asked for a weird period: %s\n", period)
		return
	}
	message := fmt.Sprintf("*%s*\nDistorted %d messages in %d distinct chats, %d of which were group chats\n",
		header, stat.Interactions, stat.Chats, stat.Groups)
	details := fmt.Sprintf(`
*Breakdown by type*
_Stickers_: %d
_GIFs_: %d
_Videos_: %d
_Video notes_: %d
_Voice messages_: %d
_Photos_: %d
_Text messages_: %d
`,
		stat.Sticker, stat.Animation, stat.Video, stat.VideoNote, stat.Voice, stat.Photo, stat.Text)
	b.Send(m.Chat, message+details, tb.ModeMarkdown)
}

func main() {
	db := stats.InitDB()
	defer db.Close()

	adminID, err := strconv.ParseInt(os.Getenv("DISTORTIONER_ADMIN_ID"), 10, 64)
	if err != nil {
		adminID = -1
		log.Println("DISTORTIONER_ADMIN_ID variable is not set")
	}

	b, err := tb.NewBot(tb.Settings{
		Token: os.Getenv("DISTORTIONER_BOT_TOKEN"),
	})
	b.Poller = tb.NewMiddlewarePoller(&tb.LongPoller{Timeout: 10 * time.Second}, func(update *tb.Update) bool {
		if update.Message == nil {
			return false
		}
		m := update.Message
		isCommand := len(m.Entities) > 0 && m.Entities[0].Type == tb.EntityCommand
		text := update.Message.Text
		if m.FromGroup() && !(isCommand && strings.HasSuffix(text, b.Me.Username)) {
			return false
		}
		if text != "/daily" && text != "/weekly" && text != "/monthly" {
			go db.SaveStat(update.Message, isCommand)
		}
		return true
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	rl := tools.NewRateLimiter()

	b.Handle("/start", func(m *tb.Message) {
		b.Send(m.Chat, "Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
	})

	b.Handle("/distort", func(m *tb.Message) {
		handleReplyDistortion(b, m, rl)
	})

	b.Handle("/daily", func(m *tb.Message) {
		handleStatRequest(b, m, db, stats.Daily, adminID)
	})

	b.Handle("/weekly", func(m *tb.Message) {
		handleStatRequest(b, m, db, stats.Weekly, adminID)
	})

	b.Handle("/monthly", func(m *tb.Message) {
		handleStatRequest(b, m, db, stats.Monthly, adminID)
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
		handleTextDistortion(b, m)
	})

	b.Start()
}

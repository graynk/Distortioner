package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tb "github.com/graynk/telebot"
	"go.uber.org/zap"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/stats"
	"github.com/graynk/distortioner/tools"
)

const (
	MaxSizeMb = 20_000_000
)

type DistorterBot struct {
	b      *tb.Bot
	rl     *tools.RateLimiter
	logger *zap.SugaredLogger
}

func (d DistorterBot) handleAnimationDistortion(m *tb.Message) {
	if m.Animation.FileSize > MaxSizeMb {
		d.b.Send(m.Chat, distorters.TooBig)
		return
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		d.b.Send(m.Chat, tools.FormatRateLimitResponse(diff))
		return
	}

	progressMessage, filename, output, err := d.HandleAnimationCommon(m)
	failed := err != nil && progressMessage.Text != distorters.TooLong
	d.DoneMessageWithRepeater(progressMessage, failed)
	if failed {
		return
	}
	defer os.Remove(filename)
	defer os.Remove(output)

	distorted := &tb.Animation{File: tb.FromDisk(output)}
	if m.Caption != "" {
		distorted.Caption = distorters.DistortText(m.Caption)
	}
	d.SendMessageWithRepeater(m.Chat, distorted)
}

func (d DistorterBot) handlePhotoDistortion(m *tb.Message) {
	filename, err := tools.JustGetTheFile(d.b, m)
	if err != nil {
		d.logger.Error(err)
		return
	}
	defer os.Remove(filename)
	err = distorters.DistortImage(filename)
	// sure would be nice to have generics here
	if err != nil {
		d.SendMessageWithRepeater(m.Chat, distorters.Failed)
		return
	}
	distorted := &tb.Photo{File: tb.FromDisk(filename)}
	if m.Caption != "" {
		distorted.Caption = distorters.DistortText(m.Caption)
	}
	d.SendMessageWithRepeater(m.Chat, distorted)
}

func (d DistorterBot) handleStickerDistortion(m *tb.Message) {
	if m.Sticker.Animated {
		// TODO: there might be a nice way to distort them too, just parse the data and move stuff around, I guess
		d.SendMessageWithRepeater(m.Chat, NotSupported)
		return
	} else if m.Sticker.Video {
		d.SendMessageWithRepeater(m.Chat, NotSupported)
		return
	}
	filename, err := tools.JustGetTheFile(d.b, m)
	if err != nil {
		d.logger.Error(err)
		return
	}
	defer os.Remove(filename)
	err = distorters.DistortImage(filename)
	if err != nil {
		d.SendMessageWithRepeater(m.Chat, distorters.Failed)
		return
	}
	distorted := &tb.Sticker{File: tb.FromDisk(filename)}
	d.SendMessageWithRepeater(m.Chat, distorted)
}

func (d DistorterBot) handleTextDistortion(m *tb.Message) {
	d.SendMessageWithRepeater(m.Chat, distorters.DistortText(m.Text))
}

func (d DistorterBot) handleVideoDistortion(m *tb.Message) {
	if m.Video.FileSize > MaxSizeMb {
		d.b.Send(m.Chat, distorters.TooBig)
		return
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		d.b.Send(m.Chat, tools.FormatRateLimitResponse(diff))
		return
	}
	output, err := d.HandleVideoCommon(m)
	if err != nil {
		return
	}
	defer os.Remove(output)

	distorted := &tb.Video{File: tb.FromDisk(output)}
	d.SendMessageWithRepeater(m.Chat, distorted)
}

func (d DistorterBot) handleVideoNoteDistortion(m *tb.Message) {
	// video notes are limited with 1 minute anyway
	if m.VideoNote.FileSize > MaxSizeMb {
		d.b.Send(m.Chat, distorters.TooBig)
		return
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		d.b.Send(m.Chat, tools.FormatRateLimitResponse(diff))
		return
	}
	output, err := d.HandleVideoCommon(m)
	if err != nil {
		return
	}
	defer os.Remove(output)
	distorted := &tb.VideoNote{File: tb.FromDisk(output)}
	d.SendMessageWithRepeater(m.Chat, distorted)
}

func (d DistorterBot) handleVoiceDistortion(m *tb.Message) {
	if m.Voice.FileSize > MaxSizeMb {
		d.b.Send(m.Chat, distorters.TooBig)
		return
	}
	filename, err := tools.JustGetTheFile(d.b, m)
	if err != nil {
		d.logger.Error(err)
		return
	}
	defer os.Remove(filename)
	output := filename + ".ogg"
	err = distorters.DistortSound(filename, output)
	if err != nil {
		d.SendMessageWithRepeater(m.Chat, distorters.Failed)
		return
	}
	defer os.Remove(output)

	distorted := &tb.Voice{File: tb.FromDisk(output)}
	d.SendMessageWithRepeater(m.Chat, distorted)
}

func (d DistorterBot) handleReplyDistortion(m *tb.Message) {
	if m.ReplyTo == nil {
		msg := "You need to reply with this command to the media you want distorted."
		if m.FromGroup() {
			msg += "\nYou might also need to make chat history visible for new members if your group is private."
		}
		d.b.Send(m.Chat, msg)
		return
	}
	original := m.ReplyTo
	if original.Animation != nil {
		d.handleAnimationDistortion(original)
	} else if original.Sticker != nil {
		d.handleStickerDistortion(original)
	} else if original.Photo != nil {
		d.handlePhotoDistortion(original)
	} else if original.Voice != nil {
		d.handleVoiceDistortion(original)
	} else if original.Video != nil {
		d.handleVideoDistortion(original)
	} else if original.VideoNote != nil {
		d.handleVideoNoteDistortion(original)
	} else if original.Text != "" {
		d.handleTextDistortion(original)
	}
}

func (d DistorterBot) handleStatRequest(m *tb.Message, db *stats.DistortionerDB, period stats.Period, adminID int64) {
	if m.Sender.ID != adminID {
		return
	}
	stat, err := db.GetStat(period)
	if err != nil {
		d.logger.Error(err)
		d.b.Send(m.Chat, err.Error())
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
		d.logger.Warnf("stats asked for a weird period", zap.String("period", string(period)))
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
	d.b.Send(m.Chat, message+details, tb.ModeMarkdown)
}

func main() {
	lg, _ := zap.NewProduction()
	defer lg.Sync() // flushes buffer, if any
	logger := lg.Sugar()
	db := stats.InitDB(logger)
	defer db.Close()

	adminID, err := strconv.ParseInt(os.Getenv("DISTORTIONER_ADMIN_ID"), 10, 64)
	if err != nil {
		adminID = -1
		logger.Warn("DISTORTIONER_ADMIN_ID variable is not set")
	}

	b, err := tb.NewBot(tb.Settings{
		Token: os.Getenv("DISTORTIONER_BOT_TOKEN"),
	})
	d := DistorterBot{
		b:      b,
		rl:     tools.NewRateLimiter(),
		logger: logger,
	}
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
		if m.FromGroup() {
			chat, err := b.ChatByID(strconv.FormatInt(m.Chat.ID, 10))
			if err != nil {
				logger.Error("Failed to get chat", zap.Int64("chat_id", m.Chat.ID), zap.Error(err))
				return false
			}
			permissions := chat.Permissions
			if permissions != nil {
				if !permissions.CanSendMessages {
					logger.Warn("can't send anything at all", zap.Int64("chat_id", m.Chat.ID))
					return false
				} else if (!permissions.CanSendMedia && tools.IsMedia(m.ReplyTo)) || (!permissions.CanSendOther && tools.IsNonMediaMedia(m.ReplyTo)) {
					d.SendMessageWithRepeater(m.Chat, NotEnoughRights)
					return false
				}
			}
		}
		if text != "/daily" && text != "/weekly" && text != "/monthly" {
			go db.SaveStat(update.Message, isCommand)
		}
		return true
	})

	if err != nil {
		logger.Fatal(err)
		return
	}

	b.Handle("/start", func(m *tb.Message) {
		d.b.Send(m.Chat, "Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
	})

	b.Handle("/distort", func(m *tb.Message) {
		d.handleReplyDistortion(m)
	})

	b.Handle("/daily", func(m *tb.Message) {
		d.handleStatRequest(m, db, stats.Daily, adminID)
	})

	b.Handle("/weekly", func(m *tb.Message) {
		d.handleStatRequest(m, db, stats.Weekly, adminID)
	})

	b.Handle("/monthly", func(m *tb.Message) {
		d.handleStatRequest(m, db, stats.Monthly, adminID)
	})

	b.Handle(tb.OnAnimation, func(m *tb.Message) {
		d.handleAnimationDistortion(m)
	})

	b.Handle(tb.OnSticker, func(m *tb.Message) {
		d.handleStickerDistortion(m)
	})

	b.Handle(tb.OnPhoto, func(m *tb.Message) {
		d.handlePhotoDistortion(m)
	})

	b.Handle(tb.OnVoice, func(m *tb.Message) {
		d.handleVoiceDistortion(m)
	})

	b.Handle(tb.OnVideo, func(m *tb.Message) {
		d.handleVideoDistortion(m)
	})

	b.Handle(tb.OnVideoNote, func(m *tb.Message) {
		d.handleVideoNoteDistortion(m)
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		d.handleTextDistortion(m)
	})

	b.Start()
}

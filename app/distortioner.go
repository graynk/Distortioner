package main

import (
	"encoding/base32"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/stats"
	"github.com/graynk/distortioner/tools"
)

const (
	MaxSizeMb = 20_000_000
)

type DistorterBot struct {
	adminID int64
	rl      *tools.RateLimiter
	logger  *zap.SugaredLogger
}

func (d DistorterBot) handleAnimationDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if m.Animation.FileSize > MaxSizeMb {
		return c.Send(distorters.TooBig)
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		return c.Send(tools.FormatRateLimitResponse(diff))
	}

	progressMessage, filename, output, err := d.HandleAnimationCommon(c)
	failed := err != nil
	if failed {
		if progressMessage.Text != distorters.TooLong {
			d.DoneMessageWithRepeater(b, progressMessage, failed)
		}
		return err
	}
	defer os.Remove(filename)
	defer os.Remove(output)

	// not sure why, but now I'm forced to specify filename manually
	distorted := &tb.Animation{File: tb.FromDisk(output), FileName: output}
	if m.Caption != "" {
		distorted.Caption = distorters.DistortText(m.Caption)
	}
	_, err = d.SendMessageWithRepeater(c, distorted)
	d.DoneMessageWithRepeater(b, progressMessage, failed)
	return err
}

func (d DistorterBot) handlePhotoDistortion(c tb.Context) error {
	m := c.Message()
	filename, err := tools.JustGetTheFile(c.Bot(), m)
	if err != nil {
		d.logger.Error(err)
		return err
	}
	defer os.Remove(filename)
	err = distorters.DistortImage(filename)
	// sure would be nice to have generics here
	if err != nil {
		d.SendMessageWithRepeater(c, distorters.Failed)
		return err
	}
	distorted := &tb.Photo{File: tb.FromDisk(filename)}
	if m.Caption != "" {
		distorted.Caption = distorters.DistortText(m.Caption)
	}
	_, err = d.SendMessageWithRepeater(c, distorted)
	return err
}

func (d DistorterBot) handleRegularStickerDistortion(c tb.Context) error {
	m := c.Message()
	filename, err := tools.JustGetTheFile(c.Bot(), m)
	if err != nil {
		d.logger.Error(err)
		return err
	}
	defer os.Remove(filename)
	err = distorters.DistortImage(filename)
	if err != nil {
		d.SendMessageWithRepeater(c, distorters.Failed)
		return err
	}
	distorted := &tb.Sticker{File: tb.FromDisk(filename)}
	_, err = d.SendMessageWithRepeater(c, distorted)
	return err
}

func (d DistorterBot) handleVideoStickerDistortion(c tb.Context) error {
	filename, output, err := d.HandleVideoSticker(c)
	if err != nil {
		d.SendMessageWithRepeater(c, distorters.Failed)
		return err
	}
	webm := tb.FromDisk(output)
	uniquePart, _ := uuid.New().MarshalBinary()
	uniquePartStr := base32.NewEncoding(UuidAlphabet).WithPadding(base32.NoPadding).EncodeToString(uniquePart)

	b := c.Bot()
	botUsername := b.Me.Username
	name := fmt.Sprintf("%s_by_%s", uniquePartStr, botUsername)
	set := &tb.StickerSet{
		Name:   name,
		Title:  "bot api sucks",
		WebM:   &webm,
		Emojis: "🍆",
	}
	// an ugly workaround, can't find a way to avoid it
	err = b.CreateStickerSet(&tb.User{ID: d.adminID}, *set)
	if err != nil {
		d.logger.Error(err, zap.String("name", name))
		d.SendMessageWithRepeater(c, distorters.Failed)
		return err
	}
	set, err = b.StickerSet(name)
	if err != nil {
		d.logger.Error(err)
		d.SendMessageWithRepeater(c, distorters.Failed)
		return err
	}
	if len(set.Stickers) == 0 {
		d.logger.Error("empty stickers field", zap.String("stickerSet", set.Name))
		d.SendMessageWithRepeater(c, distorters.Failed)
		return errors.New("empty stickers field")
	}
	sticker := set.Stickers[0]
	_, err = d.SendMessageWithRepeater(c, &sticker)
	defer os.Remove(filename)
	defer os.Remove(output)
	b.DeleteSticker(sticker.FileID)
	return err
}

func (d DistorterBot) handleStickerDistortion(c tb.Context) error {
	m := c.Message()
	var err error
	switch {
	case m.Sticker.Animated:
		_, err = d.SendMessageWithRepeater(c, NotSupported)

	case m.Sticker.Video:
		err = d.handleVideoStickerDistortion(c)
	default:
		err = d.handleRegularStickerDistortion(c)
	}
	return err
}

func (d DistorterBot) handleTextDistortion(c tb.Context) error {
	_, err := d.SendMessageWithRepeater(c, distorters.DistortText(c.Text()))
	return err
}

func (d DistorterBot) handleVideoDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if m.Video.FileSize > MaxSizeMb {
		return c.Send(distorters.TooBig)
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		return c.Send(tools.FormatRateLimitResponse(diff))
	}
	output, progressMessage, err := d.HandleVideoCommon(c)
	failed := err != nil
	if failed {
		if progressMessage.Text != distorters.TooLong {
			d.DoneMessageWithRepeater(b, progressMessage, failed)
		}
		return err
	}
	defer os.Remove(output)

	distorted := &tb.Video{File: tb.FromDisk(output)}
	_, err = d.SendMessageWithRepeater(c, distorted)
	d.DoneMessageWithRepeater(b, progressMessage, failed)
	return err
}

func (d DistorterBot) handleVideoNoteDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	// video notes are limited with 1 minute anyway
	if m.VideoNote.FileSize > MaxSizeMb {
		return c.Send(m.Chat, distorters.TooBig)
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, m.Unixtime); rate > tools.AllowedOverTime {
		return c.Send(m.Chat, tools.FormatRateLimitResponse(diff))
	}
	output, progressMessage, err := d.HandleVideoCommon(c)
	failed := err != nil
	if failed {
		if progressMessage.Text != distorters.TooLong {
			d.DoneMessageWithRepeater(b, progressMessage, failed)
		}
		return err
	}
	defer os.Remove(output)
	distorted := &tb.VideoNote{File: tb.FromDisk(output)}
	_, err = d.SendMessageWithRepeater(c, distorted)
	d.DoneMessageWithRepeater(b, progressMessage, failed)
	return err
}

func (d DistorterBot) handleVoiceDistortion(c tb.Context) error {
	m := c.Message()
	if m.Voice.FileSize > MaxSizeMb {
		return c.Send(m.Chat, distorters.TooBig)
	}
	filename, err := tools.JustGetTheFile(c.Bot(), m)
	if err != nil {
		d.logger.Error(err)
		return err
	}
	defer os.Remove(filename)
	output := filename + ".ogg"
	err = distorters.DistortSound(filename, output)
	if err != nil {
		d.SendMessageWithRepeater(c, distorters.Failed)
		return err
	}
	defer os.Remove(output)

	distorted := &tb.Voice{File: tb.FromDisk(output)}
	_, err = d.SendMessageWithRepeater(c, distorted)
	return err
}

func (d DistorterBot) handleReplyDistortion(c tb.Context) error {
	m := c.Message()
	if m.ReplyTo == nil {
		msg := "You need to reply with this command to the media you want distorted."
		if m.FromGroup() {
			msg += "\nYou might also need to make chat history visible for new members if your group is private."
		}
		return c.Send(msg)
	}
	original := m.ReplyTo
	update := c.Update()
	update.Message = original
	tweakedContext := c.Bot().NewContext(update)
	switch {
	case original.Animation != nil:
		return d.handleAnimationDistortion(tweakedContext)
	case original.Sticker != nil:
		return d.handleStickerDistortion(tweakedContext)
	case original.Photo != nil:
		return d.handlePhotoDistortion(tweakedContext)
	case original.Voice != nil:
		return d.handleVoiceDistortion(tweakedContext)
	case original.Video != nil:
		return d.handleVideoDistortion(tweakedContext)
	case original.VideoNote != nil:
		return d.handleVideoNoteDistortion(tweakedContext)
	case original.Text != "":
		return d.handleTextDistortion(tweakedContext)
	}
	return nil
}

func (d DistorterBot) handleStatRequest(c tb.Context, db *stats.DistortionerDB, period stats.Period) error {
	m := c.Message()
	if m.Sender.ID != d.adminID {
		return nil
	}
	stat, err := db.GetStat(period)
	if err != nil {
		d.logger.Error(err)
		return c.Send(err.Error())
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
		return nil
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
	return c.Send(message+details, tb.ModeMarkdown)
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
		logger.Fatal("DISTORTIONER_ADMIN_ID variable is not set")
	}

	b, err := tb.NewBot(tb.Settings{
		Token: os.Getenv("DISTORTIONER_BOT_TOKEN"),
	})
	d := DistorterBot{
		adminID: adminID,
		rl:      tools.NewRateLimiter(),
		logger:  logger,
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
			chat, err := b.ChatByID(m.Chat.ID)
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
					b.Send(m.Chat, NotEnoughRights)
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

	b.Handle("/start", func(c tb.Context) error {
		return c.Send("Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
	})

	b.Handle("/daily", func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Daily)
	})

	b.Handle("/weekly", func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Weekly)
	})

	b.Handle("/monthly", func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Monthly)
	})

	b.Handle("/distort", d.handleReplyDistortion)
	b.Handle(tb.OnAnimation, d.handleAnimationDistortion)
	b.Handle(tb.OnSticker, d.handleStickerDistortion)
	b.Handle(tb.OnPhoto, d.handlePhotoDistortion)
	b.Handle(tb.OnVoice, d.handleVoiceDistortion)
	b.Handle(tb.OnVideo, d.handleVideoDistortion)
	b.Handle(tb.OnVideoNote, d.handleVideoNoteDistortion)
	b.Handle(tb.OnText, d.handleTextDistortion)

	b.Start()
}

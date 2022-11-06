package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/stats"
	"github.com/graynk/distortioner/tools"
)

const (
	MaxSizeMb = 20_000_000
)

type DistorterBot struct {
	adminID     int64
	rl          *tools.RateLimiter
	logger      *zap.SugaredLogger
	mu          *sync.Mutex
	graceWg     *sync.WaitGroup
	videoWorker *tools.VideoWorker
}

func (d DistorterBot) handleAnimationDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if m.Animation.FileSize > MaxSizeMb {
		return d.SendMessageWithRepeater(c, distorters.TooBig)
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
		return d.SendMessageWithRepeater(c, tools.FormatRateLimitResponse(diff))
	}

	//TODO: Jesus, just find the time to refactor all of this already
	d.videoWorker.Submit(m.Chat.ID, func() {
		progressMessage, filename, output, err := d.HandleAnimationCommon(c)
		failed := err != nil
		if failed {
			if progressMessage != nil && progressMessage.Text != distorters.TooLong {
				d.DoneMessageWithRepeater(b, progressMessage, failed)
			}
			d.logger.Error(err)
			return
		}
		defer os.Remove(filename)
		defer os.Remove(output)

		// not sure why, but now I'm forced to specify filename manually
		distorted := &tb.Animation{File: tb.FromDisk(output), FileName: output}
		if m.Caption != "" {
			distorted.Caption = distorters.DistortText(m.Caption)
		}
		err = d.SendMessageWithRepeater(c, distorted)
		d.DoneMessageWithRepeater(b, progressMessage, failed)
	})
	if d.videoWorker.IsBusy() {
		d.SendMessageWithRepeater(c, distorters.Queued)
	}
	return nil
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
	if err != nil {
		d.SendMessageWithRepeater(c, distorters.Failed)
		return err
	}
	distorted := &tb.Photo{File: tb.FromDisk(filename)}
	if m.Caption != "" {
		distorted.Caption = distorters.DistortText(m.Caption)
	}
	return d.SendMessageWithRepeater(c, distorted)
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
	return d.SendMessageWithRepeater(c, distorted)
}

func (d DistorterBot) handleVideoStickerDistortion(c tb.Context) error {
	return c.Reply("You can go vote for this suggestion, for .webm stickers handling to become somewhat tolerable https://bugs.telegram.org/c/14858")
}

func (d DistorterBot) handleStickerDistortion(c tb.Context) error {
	m := c.Message()
	var err error
	switch {
	case m.Sticker.Animated:
		err = d.SendMessageWithRepeater(c, NotSupported)
	case m.Sticker.Video:
		err = d.handleVideoStickerDistortion(c)
	default:
		err = d.handleRegularStickerDistortion(c)
	}
	return err
}

func (d DistorterBot) handleTextDistortion(c tb.Context) error {
	return d.SendMessageWithRepeater(c, distorters.DistortText(c.Text()))
}

func (d DistorterBot) handleVideoDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if m.Video.FileSize > MaxSizeMb {
		return d.SendMessageWithRepeater(c, distorters.TooBig)
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
		return d.SendMessageWithRepeater(c, tools.FormatRateLimitResponse(diff))
	}

	d.videoWorker.Submit(m.Chat.ID, func() {
		output, progressMessage, err := d.HandleVideoCommon(c)
		failed := err != nil
		if failed {
			if progressMessage != nil && progressMessage.Text != distorters.TooLong {
				d.DoneMessageWithRepeater(b, progressMessage, failed)
			}
			d.logger.Error(err)
			return
		}
		defer os.Remove(output)

		distorted := &tb.Video{File: tb.FromDisk(output)}
		err = d.SendMessageWithRepeater(c, distorted)
		d.DoneMessageWithRepeater(b, progressMessage, failed)
		if err != nil {
			d.logger.Error(err)
		}
	})
	if d.videoWorker.IsBusy() {
		d.SendMessageWithRepeater(c, distorters.Queued)
	}
	return nil
}

func (d DistorterBot) handleVideoNoteDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if m.VideoNote.FileSize > MaxSizeMb {
		return d.SendMessageWithRepeater(c, distorters.TooBig)
	} else if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
		return d.SendMessageWithRepeater(c, tools.FormatRateLimitResponse(diff))
	}

	d.videoWorker.Submit(m.Chat.ID, func() {
		output, progressMessage, err := d.HandleVideoCommon(c)
		failed := err != nil
		if failed {
			if progressMessage != nil && progressMessage.Text != distorters.TooLong {
				d.DoneMessageWithRepeater(b, progressMessage, failed)
			}
			d.logger.Error(err)
			return
		}
		defer os.Remove(output)
		distorted := &tb.VideoNote{File: tb.FromDisk(output)}
		err = d.SendMessageWithRepeater(c, distorted)
		d.DoneMessageWithRepeater(b, progressMessage, failed)
	})
	if d.videoWorker.IsBusy() {
		d.SendMessageWithRepeater(c, distorters.Queued)
	}
	return nil
}

func (d DistorterBot) handleVoiceDistortion(c tb.Context) error {
	m := c.Message()
	if m.Voice.FileSize > MaxSizeMb {
		return c.Reply(distorters.TooBig)
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
	return d.SendMessageWithRepeater(c, distorted)
}

func (d DistorterBot) handleReplyDistortion(c tb.Context) error {
	m := c.Message()
	if m.ReplyTo == nil {
		msg := "You need to reply with this command to the media you want distorted."
		if m.FromGroup() {
			msg += "\nYou might also need to make chat history visible for new members if your group is private."
		}
		return c.Reply(msg)
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
		return c.Reply(err.Error())
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
	return c.Reply(message+details, tb.ModeMarkdown)
}

func (d DistorterBot) handleQueueStats(c tb.Context) error {
	if c.Message().Sender.ID != d.adminID {
		return nil
	}
	length, users := d.videoWorker.QueueStats()
	return c.Reply(fmt.Sprintf("Currently in queue: %d requests from %d users", length, users))
}

func main() {
	lg, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
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
		adminID:     adminID,
		rl:          tools.NewRateLimiter(),
		logger:      logger,
		mu:          &sync.Mutex{},
		graceWg:     &sync.WaitGroup{},
		videoWorker: tools.NewVideoWorker(3),
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
		// throw away old messages
		if time.Now().Sub(m.Time()) > 2*time.Hour {
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
					b.Reply(m, NotEnoughRights)
					return false
				}
			}
		}
		if text != "/daily" && text != "/weekly" && text != "/monthly" && text != "/queue" {
			go db.SaveStat(update.Message, isCommand)
		}
		return true
	})

	if err != nil {
		logger.Fatal(err)
		return
	}

	b.Handle("/start", func(c tb.Context) error {
		return c.Reply("Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
	})

	b.Handle("/daily", d.ApplyShutdownMiddleware(func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Daily)
	}))

	b.Handle("/weekly", d.ApplyShutdownMiddleware(func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Weekly)
	}))

	b.Handle("/monthly", d.ApplyShutdownMiddleware(func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Monthly)
	}))

	b.Handle("/queue", d.handleQueueStats)

	b.Handle("/distort", d.ApplyShutdownMiddleware(d.handleReplyDistortion))
	b.Handle(tb.OnAnimation, d.ApplyShutdownMiddleware(d.handleAnimationDistortion))
	b.Handle(tb.OnSticker, d.ApplyShutdownMiddleware(d.handleStickerDistortion))
	b.Handle(tb.OnPhoto, d.ApplyShutdownMiddleware(d.handlePhotoDistortion))
	b.Handle(tb.OnVoice, d.ApplyShutdownMiddleware(d.handleVoiceDistortion))
	b.Handle(tb.OnVideo, d.ApplyShutdownMiddleware(d.handleVideoDistortion))
	b.Handle(tb.OnVideoNote, d.ApplyShutdownMiddleware(d.handleVideoNoteDistortion))
	b.Handle(tb.OnText, d.ApplyShutdownMiddleware(d.handleTextDistortion))
	b.Use(middleware.Recover())

	go func() {
		signChan := make(chan os.Signal, 1)
		signal.Notify(signChan, os.Interrupt, syscall.SIGTERM)
		sig := <-signChan

		logger.Info("shutdown: ", zap.String("signal", sig.String()))
		d.videoWorker.Shutdown()
		d.graceWg.Wait()
		b.Stop()
	}()

	b.Start()
}

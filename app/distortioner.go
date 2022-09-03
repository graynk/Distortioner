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

	"github.com/graynk/distortioner/distorters"
	"github.com/graynk/distortioner/stats"
	"github.com/graynk/distortioner/tools"
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
	if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
		return d.SendMessageWithRepeater(c, tools.FormatRateLimitResponse(diff))
	}

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

func (d DistorterBot) handleSimpleMediaDistortion(c tb.Context) error {
	m := c.Message()
	media := tools.JustGetTheMedia(m)
	filename, err := tools.JustGetTheFile(c.Bot(), media)

	if err != nil {
		return err
	}

	defer os.Remove(filename)

	var distorted tb.Sendable
	switch media.MediaType() {
	case tools.Photo:
		err = distorters.DistortImage(filename)
		photo := &tb.Photo{File: tb.FromDisk(filename)}
		if m.Caption != "" {
			photo.Caption = distorters.DistortText(m.Caption)
		}
		distorted = photo
	case tools.Sticker:
		if m.Sticker.Animated || m.Sticker.Video {
			return tools.NotSupportedErr
		}
		err = distorters.DistortImage(filename)
		distorted = &tb.Sticker{File: tb.FromDisk(filename)}
	case tools.Voice:
		var output string
		output, err = distorters.DistortSound(filename)
		defer os.Remove(output)
		distorted = &tb.Voice{File: tb.FromDisk(output)}
	}

	if err != nil {
		return err
	}

	return d.SendMessageWithRepeater(c, distorted)
}

func (d DistorterBot) handleTextDistortion(c tb.Context) error {
	return d.SendMessageWithRepeater(c, distorters.DistortText(c.Text()))
}

func (d DistorterBot) handleVideoDistortion(c tb.Context) error {
	m := c.Message()
	b := c.Bot()
	if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
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
	if rate, diff := d.rl.GetRateOverPeriod(m.Chat.ID, time.Now().Unix()); rate > tools.AllowedOverTime {
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
	originalMedia := tools.JustGetTheMedia(original)
	switch originalMedia.MediaType() {
	case tools.Animation:
		return d.handleAnimationDistortion(tweakedContext)
	case tools.Sticker:
		fallthrough
	case tools.Photo:
		fallthrough
	case tools.Voice:
		return d.handleSimpleMediaDistortion(tweakedContext)
	case tools.Video:
		return d.handleVideoDistortion(tweakedContext)
	case tools.VideoNote:
		return d.handleVideoNoteDistortion(tweakedContext)
	default:
		if original.Text != "" {
			return d.handleTextDistortion(tweakedContext)
		}
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

func (d DistorterBot) handleQueueStats(c tb.Context) error {
	if c.Message().Sender.ID != d.adminID {
		return nil
	}
	length, users := d.videoWorker.QueueStats()
	return c.Send(fmt.Sprintf("Currently in queue: %d requests from %d users", length, users))
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
					b.Send(m.Chat, tools.NotEnoughRights)
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

	commonGroup := b.Group()
	commonGroup.Use(d.ErrMiddleware, d.ShutdownMiddleware)

	commonGroup.Handle("/start", func(c tb.Context) error {
		return d.SendMessageWithRepeater(c, "Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
	})

	commonGroup.Handle("/daily", func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Daily)
	})

	commonGroup.Handle("/weekly", func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Weekly)
	})

	commonGroup.Handle("/monthly", func(c tb.Context) error {
		return d.handleStatRequest(c, db, stats.Monthly)
	})

	commonGroup.Handle("/queue", d.handleQueueStats)

	commonGroup.Handle("/distort", d.handleReplyDistortion)
	commonGroup.Handle(tb.OnAnimation, d.handleAnimationDistortion)
	commonGroup.Handle(tb.OnSticker, d.handleSimpleMediaDistortion)
	commonGroup.Handle(tb.OnPhoto, d.handleSimpleMediaDistortion)
	commonGroup.Handle(tb.OnVoice, d.handleSimpleMediaDistortion)
	commonGroup.Handle(tb.OnVideo, d.handleVideoDistortion)
	commonGroup.Handle(tb.OnVideoNote, d.handleVideoNoteDistortion)
	commonGroup.Handle(tb.OnText, d.handleTextDistortion)

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

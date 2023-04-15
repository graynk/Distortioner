package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"

	"github.com/graynk/distortioner/bot"
	"github.com/graynk/distortioner/stats"
	"github.com/graynk/distortioner/tools"
)

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
	d := bot.NewDistorterBot(adminID, logger)
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
	commonGroup.Use(middleware.Recover(), d.GetTheFileOrErrorOutMiddleware, d.ErrMiddleware, d.ShutdownMiddleware)

	commonGroup.Handle("/start", func(c tb.Context) error {
		return d.SendMessageWithRepeater(c, "Send me a picture, a sticker, a voice message, a video[note] or a GIF and I'll distort it")
	})

	commonGroup.Handle("/daily", func(c tb.Context) error {
		return d.HandleStatRequest(c, db, stats.Daily)
	})

	commonGroup.Handle("/weekly", func(c tb.Context) error {
		return d.HandleStatRequest(c, db, stats.Weekly)
	})

	commonGroup.Handle("/monthly", func(c tb.Context) error {
		return d.HandleStatRequest(c, db, stats.Monthly)
	})

	commonGroup.Handle("/queue", d.HandleQueueStats)

	commonGroup.Handle("/distort", d.HandleReplyDistortion)
	commonGroup.Handle(tb.OnAnimation, d.HandleAnimationDistortion)
	commonGroup.Handle(tb.OnSticker, d.HandleSimpleMediaDistortion)
	commonGroup.Handle(tb.OnPhoto, d.HandleSimpleMediaDistortion)
	commonGroup.Handle(tb.OnVoice, d.HandleSimpleMediaDistortion)
	commonGroup.Handle(tb.OnVideo, d.HandleVideoDistortion)
	commonGroup.Handle(tb.OnVideoNote, d.HandleVideoNoteDistortion)
	commonGroup.Handle(tb.OnText, d.HandleTextDistortion)

	go func() {
		signChan := make(chan os.Signal, 1)
		signal.Notify(signChan, os.Interrupt, syscall.SIGTERM)
		sig := <-signChan

		logger.Info("shutdown: ", zap.String("signal", sig.String()))
		d.Shutdown()
		b.Stop()
	}()

	b.Start()
}

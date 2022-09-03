package bot

import (
	"fmt"

	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"

	"github.com/graynk/distortioner/stats"
)

func (d distorterBot) HandleStatRequest(c tb.Context, db *stats.DistortionerDB, period stats.Period) error {
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

func (d distorterBot) HandleQueueStats(c tb.Context) error {
	if c.Message().Sender.ID != d.adminID {
		return nil
	}
	length, users := d.videoWorker.QueueStats()
	return c.Send(fmt.Sprintf("Currently in queue: %d requests from %d users", length, users))
}

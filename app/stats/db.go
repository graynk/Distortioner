package stats

import (
	"database/sql"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
)

type DistortionerDB struct {
	db     *sql.DB
	insert *sql.Stmt
	logger *zap.SugaredLogger
}

type Stat struct {
	Interactions int
	Chats        int
	Groups       int
	Sticker      int
	Animation    int
	Video        int
	VideoNote    int
	Voice        int
	Photo        int
	Text         int
}

type Period string

const (
	Daily   Period = "-1 day"
	Weekly  Period = "-7 days"
	Monthly Period = "-1 month"
)

const statQuery = `
	select
		   count(*) as interactions,
		   count(distinct(user_id)) as users,
		   count(distinct (case when is_group_chat = 1 then user_id end)) as groups,
		   count(case when type = 'sticker' then type end) as sticker,
		   count(case when type = 'animation' then type end) as animation,
		   count(case when type = 'video' then type end) as video,
		   count(case when type = 'videonote' then type end) as videonote,
		   count(case when type = 'voice' then type end) as voice,
		   count(case when type = 'photo' then type end) as photo,
		   count(case when type = 'text' then type end) as text
	from stats
	where date >= datetime('now', ?, 'localtime') and datetime('now','localtime');
`

func InitDB(logger *zap.SugaredLogger) *DistortionerDB {
	err := os.Mkdir("data", os.ModePerm)
	if err != nil && !os.IsExist(err) {
		logger.Fatal("Failed to create data directory for stat DB", err)
	}
	db, err := sql.Open("sqlite3", "file:data/distortioner.sqlite?cache=shared")
	if err != nil {
		logger.Fatal(err)
	}
	dist := DistortionerDB{
		db:     db,
		logger: logger,
	}
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`create table if not exists stats(id integer not null primary key, user_id integer, is_group_chat integer, date integer, type text);`)
	if err != nil {
		logger.Fatal(err)
	}
	_, err = db.Exec(`create index if not exists dateidx on stats(date asc);`)
	if err != nil {
		logger.Fatal(err)
	}
	insertStat, err := db.Prepare(`insert into stats(user_id, is_group_chat, date, type) values(?, ?, ?, ?);`)
	if err != nil {
		logger.Fatal(err)
	}
	dist.insert = insertStat
	return &dist
}

func (d *DistortionerDB) SaveStat(message *tb.Message, isCommand bool) {
	if message == nil {
		return
	}
	if isCommand {
		d.SaveStat(message.ReplyTo, false)
		return
	}
	messageType := "text"
	switch {
	case message.Animation != nil:
		messageType = "animation"
	case message.Video != nil:
		messageType = "video"
	case message.VideoNote != nil:
		messageType = "videonote"
	case message.Voice != nil:
		messageType = "voice"
	case message.Sticker != nil:
		messageType = "sticker"
	case message.Photo != nil:
		messageType = "photo"
	}
	_, err := d.insert.Exec(message.Chat.ID, message.FromGroup(), time.Now(), messageType)
	if err != nil {
		d.logger.Error(err)
	}
}

func (d *DistortionerDB) GetStat(period Period) (Stat, error) {
	row := d.db.QueryRow(statQuery, period)
	var stat Stat
	err := row.Scan(&stat.Interactions, &stat.Chats, &stat.Groups, &stat.Sticker, &stat.Animation, &stat.Video,
		&stat.VideoNote, &stat.Voice, &stat.Photo, &stat.Text)
	return stat, err
}

func (d *DistortionerDB) Close() {
	d.db.Close()
}

package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tb "gopkg.in/tucnak/telebot.v2"
)

type distortionerDB struct {
	db     *sql.DB
	insert *sql.Stmt
}

func initDB() distortionerDB {
	db, err := sql.Open("sqlite3", "file:distortioner.sqlite?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	dist := distortionerDB{
		db: db,
	}
	db.SetMaxOpenConns(1)

	sqlStmt := `
	create table if not exists stats(id integer not null primary key, user_id integer, is_group_chat integer, date integer, type text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := db.Prepare(`insert into stats(user_id, is_group_chat, date, type) values(?, ?, ?, ?);`)
	if err != nil {
		log.Fatal(err)
	}
	dist.insert = stmt
	return dist
}

func (d *distortionerDB) SaveStat(message *tb.Message) {
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
		log.Println(err)
	}
}

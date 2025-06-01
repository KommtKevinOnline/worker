package models

import (
	"time"

	"kommtkevinonline.de/database"
)

type Prediction struct {
	ID        *string   `db:"id"`
	ClipID    string    `db:"clip_id"`
	Source    string    `db:"source"`
	Date      time.Time `db:"date"`
	Topic     string    `db:"topic"`
	Type      string    `db:"event_type"`
	CreatedAt time.Time `db:"created_at"`
}

func (prediction Prediction) Save() error {
	db := database.GetConnection()

	_, err := db.NamedExec(`
		INSERT INTO
		predictions
		(clip_id, source, date, topic, event_type)
		VALUES (:clip_id, :source, :date, :topic, :event_type)
		ON CONFLICT (id) DO UPDATE SET clip_id = :clip_id, source = :source, date = :date, topic = :topic, event_type = :event_type`,
		prediction,
	)

	return err
}

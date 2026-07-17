package models

import (
	"time"

	"kommtkevinonline.de/database"
)

type Stream struct {
	ID        string     `db:"id"`
	StartedAt time.Time  `db:"started_at"`
	EndedAt   *time.Time `db:"ended_at"`
	Title     string     `db:"title"`
	Category  string     `db:"category"`
	CreatedAt time.Time  `db:"created_at"`
}

func (stream Stream) Save() error {
	db := database.GetConnection()

	_, err := db.NamedExec(`
		INSERT INTO streams (id, started_at, title, category)
		VALUES (:id, :started_at, :title, :category)
		ON CONFLICT (id) DO UPDATE SET
			started_at = EXCLUDED.started_at,
			title = EXCLUDED.title,
			category = EXCLUDED.category`,
		stream,
	)

	return err
}

// MarkLatestStreamEnded closes the most recent open stream row. Called on
// stream.offline, which carries no stream id.
func MarkLatestStreamEnded(endedAt time.Time) error {
	db := database.GetConnection()

	_, err := db.Exec(`
		UPDATE streams SET ended_at = $1
		WHERE id = (
			SELECT id FROM streams
			WHERE ended_at IS NULL
			ORDER BY started_at DESC
			LIMIT 1
		)`,
		endedAt,
	)

	return err
}

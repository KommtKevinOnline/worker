package models

import (
	"fmt"
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

// MedianStreamStart returns the median start time ("HH:MM", Europe/Berlin) of
// the streamer's recent streams, preferring exact starts from the streams
// table and falling back to VOD publish dates. Returns empty when there is
// not enough data.
func MedianStreamStart() (string, error) {
	db := database.GetConnection()

	var medianMinutes *float64
	err := db.Get(&medianMinutes, `
		WITH starts AS (
			SELECT started_at AS ts FROM streams
			UNION ALL
			SELECT date FROM vods WHERE date IS NOT NULL
			ORDER BY ts DESC
			LIMIT 30
		)
		SELECT percentile_cont(0.5) WITHIN GROUP (
			ORDER BY date_part('hour', ts AT TIME ZONE 'Europe/Berlin') * 60
				+ date_part('minute', ts AT TIME ZONE 'Europe/Berlin')
		)
		FROM starts`,
	)

	if err != nil || medianMinutes == nil {
		return "", err
	}

	total := int(*medianMinutes)

	return fmt.Sprintf("%02d:%02d", total/60%24, total%60), nil
}

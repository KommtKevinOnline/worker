package models

import (
	"time"

	"kommtkevinonline.de/database"
)

type Prediction struct {
	ID         *string   `db:"id"`
	ClipID     string    `db:"clip_id"`
	Source     string    `db:"source"`
	Date       time.Time `db:"date"`
	Day        string    `db:"day"`
	Topic      string    `db:"topic"`
	Type       string    `db:"event_type"`
	Confidence *float64  `db:"confidence"`
	Quote      string    `db:"quote"`
	QuoteStart *float64  `db:"quote_start"`
	CreatedAt  time.Time `db:"created_at"`
}

// Save upserts the prediction keyed on its calendar day (Europe/Berlin), so a
// newer announcement replaces an older one for the same day. Rows edited by
// hand (source = 'manual') are never overwritten by automated sources.
// Returns false when the manual-wins guard skipped the write.
func (prediction Prediction) Save() (bool, error) {
	db := database.GetConnection()

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return false, err
	}

	prediction.Day = prediction.Date.In(loc).Format("2006-01-02")

	res, err := db.NamedExec(`
		INSERT INTO predictions
		(clip_id, source, date, day, topic, event_type, confidence, quote, quote_start)
		VALUES (:clip_id, :source, :date, :day, :topic, :event_type, :confidence, :quote, :quote_start)
		ON CONFLICT (day) DO UPDATE SET
			clip_id = EXCLUDED.clip_id,
			source = EXCLUDED.source,
			date = EXCLUDED.date,
			topic = EXCLUDED.topic,
			event_type = EXCLUDED.event_type,
			confidence = EXCLUDED.confidence,
			quote = EXCLUDED.quote,
			quote_start = EXCLUDED.quote_start,
			created_at = current_timestamp
		WHERE predictions.source IS DISTINCT FROM 'manual'`,
		prediction,
	)

	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

// UpcomingPredictions returns predictions from the given day (inclusive) onwards.
func UpcomingPredictions(from time.Time, limit int) ([]Prediction, error) {
	db := database.GetConnection()

	predictions := []Prediction{}
	// Explicit columns: the table still carries a legacy "type" column that has
	// no struct destination, so SELECT * breaks sqlx scanning. day::text keeps
	// the YYYY-MM-DD form (lib/pq would otherwise render the date column as a
	// full timestamp), and the COALESCEs guard against NULLs in pre-migration
	// rows.
	err := db.Select(&predictions, `
		SELECT id, COALESCE(clip_id, '') AS clip_id, COALESCE(source, '') AS source,
			date, day::text AS day, COALESCE(topic, '') AS topic,
			COALESCE(event_type, '') AS event_type, confidence,
			COALESCE(quote, '') AS quote, quote_start, created_at
		FROM predictions
		WHERE day >= $1
		ORDER BY day ASC
		LIMIT $2`,
		from.Format("2006-01-02"), limit,
	)

	return predictions, err
}

// ClearPredictionDay removes an automated prediction for a calendar day.
// Manual rows are protected, same as in Save.
func ClearPredictionDay(day string) (int64, error) {
	db := database.GetConnection()

	res, err := db.Exec(`
		DELETE FROM predictions
		WHERE day = $1 AND source IS DISTINCT FROM 'manual'`,
		day,
	)

	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

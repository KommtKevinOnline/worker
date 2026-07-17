package models

import (
	"time"

	"kommtkevinonline.de/database"
)

// PredictionRun is the audit trail of one LLM prediction attempt: what went
// in, which operations the model performed, and what came back. Enables
// replaying past inputs against improved prompts and debugging bad output.
type PredictionRun struct {
	ID         *int      `db:"id"`
	Source     string    `db:"source"`
	ClipID     string    `db:"clip_id"`
	Model      string    `db:"model"`
	Input      string    `db:"input"`
	Operations string    `db:"operations"`
	Response   string    `db:"response"`
	Error      string    `db:"error"`
	CreatedAt  time.Time `db:"created_at"`
}

func (run PredictionRun) Save() error {
	db := database.GetConnection()

	_, err := db.NamedExec(`
		INSERT INTO prediction_runs (source, clip_id, model, input, operations, response, error)
		VALUES (:source, :clip_id, :model, :input, :operations, :response, :error)`,
		run,
	)

	return err
}

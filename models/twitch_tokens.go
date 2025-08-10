package models

import (
	"time"

	"kommtkevinonline.de/database"
)

type TwitchToken struct {
	ID           string    `db:"id"`
	AccessToken  string    `db:"access_token" json:"access_token"`
	RefreshToken string    `db:"refresh_token" json:"refresh_token"`
	ExpiresIn    int       `db:"expires_in" json:"expires_in"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

func (token TwitchToken) Save() {
	db := database.GetConnection()

	token.ID = "1"

	db.NamedExec(`
	INSERT INTO twitch_tokens (id,access_token, refresh_token, expires_in) VALUES (:id, :access_token, :refresh_token, :expires_in)
	ON CONFLICT (id) DO UPDATE SET access_token = :access_token, refresh_token = :refresh_token, expires_in = :expires_in
	`, token)
}

func (token TwitchToken) Get() (TwitchToken, error) {
	db := database.GetConnection()

	selectedToken := TwitchToken{}

	err := db.Get(&selectedToken, "SELECT * FROM twitch_tokens LIMIT 1")
	if err != nil {
		return TwitchToken{}, err
	}

	return selectedToken, nil
}

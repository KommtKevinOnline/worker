package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Adeithe/go-twitch/api"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func GetConnection() *sql.DB {
	connectionString := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_DATABASE"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"))

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func GetDownloadedVods() ([]string, error) {
	db := GetConnection()

	var vodIds []string

	rows, err := db.Query("SELECT vodid FROM vods")

	if err != nil {
		return vodIds, err
	}

	for rows.Next() {
		var vodId string

		if err := rows.Scan(&vodId); err != nil {
			return vodIds, err
		}

		vodIds = append(vodIds, vodId)
	}

	if err = rows.Err(); err != nil {
		return vodIds, err
	}

	return vodIds, nil
}

func Persist(transcript string, vod api.Video, upcoming []string, duration time.Duration) {
	db := GetConnection()

	// TODO: Write upcoming streams to its own table together with the vodId in which it was found
	sqlStatement := `INSERT INTO vods (transcript, vodid, title, date, url, thumbnail, view_count, online_intend_date, duration) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := db.Exec(sqlStatement, transcript, vod.ID, vod.Title, vod.PublishedAt, vod.URL, vod.ThumbnailURL, vod.ViewCount, strings.Join(upcoming, ","), duration.Seconds())

	if err != nil {
		log.Fatalf("Error executing SQL statement: %v", err)
	}
}

func GetLatestVod() (string, time.Time, error) {
	db := sqlx.NewDb(GetConnection(), "postgres")

	sqlStatement := `SELECT vodId, online_intend_date FROM vods ORDER BY date DESC LIMIT 1`
	row := db.QueryRow(sqlStatement)

	var vodId string
	var onlineIntendDate time.Time

	if err := row.Scan(&vodId, &onlineIntendDate); err != nil {
		return "", time.Time{}, err
	}

	return vodId, onlineIntendDate, nil
}

func SaveTwitchLoginData(accessToken, refreshToken string, expiresIn int) error {
	db := GetConnection()

	sqlStatement := `INSERT INTO twitch_login_data (access_token, refresh_token, expires_in) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET access_token = $1, refresh_token = $2, expires_in = $3`
	_, err := db.Exec(sqlStatement, accessToken, refreshToken, expiresIn)

	if err != nil {
		return err
	}

	return nil
}

func GetTwitchLoginData() (string, string, int, error) {
	db := sqlx.NewDb(GetConnection(), "postgres")
	var accessToken, refreshToken string
	var expiresIn int
	query := `SELECT access_token, refresh_token, expires_in FROM twitch_login_data ORDER BY id DESC LIMIT 1`
	row := db.QueryRowx(query)
	if err := row.Scan(&accessToken, &refreshToken, &expiresIn); err != nil {
		return "", "", 0, err
	}
	return accessToken, refreshToken, expiresIn, nil
}

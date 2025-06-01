package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Adeithe/go-twitch/api"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var connection *sqlx.DB

func GetConnection() *sqlx.DB {
	if connection != nil {
		return connection
	}

	connectionString := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_DATABASE"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"))

	connection, err := sqlx.Connect("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	return connection
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
	db := GetConnection()

	sqlStatement := `SELECT vodId, online_intend_date FROM vods ORDER BY date DESC LIMIT 1`
	row := db.QueryRow(sqlStatement)

	var vodId string
	var onlineIntendDate time.Time

	if err := row.Scan(&vodId, &onlineIntendDate); err != nil {
		return "", time.Time{}, err
	}

	return vodId, onlineIntendDate, nil
}

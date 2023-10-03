package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Adeithe/go-twitch/api"
	_ "github.com/lib/pq"
)

func getConnection() *sql.DB {
	connectionString := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_DATABASE"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"))

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	return db;
}

func getDownloadedVods() ([]string, error) {
	db := getConnection()

	var vodIds []string

	rows, err := db.Query("SELECT vodid FROM vods")

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

func persist(transcript string, vod api.Video, upcoming Upcoming) {
	db := getConnection()

	sqlStatement := `INSERT INTO vods (transcript, vodid, title, date, url, thumbnail, view_count, online_intend_date, online_intend, duration) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := db.Exec(sqlStatement, transcript, vod.ID, vod.Title, vod.PublishedAt, vod.URL, vod.ThumbnailURL, vod.ViewCount, upcoming.Date, upcoming.OnlineIntend, vod.Duration.AsDuration().String())

	if err != nil {
		log.Fatalf("Error executing SQL statement: %v", err)
	}
}
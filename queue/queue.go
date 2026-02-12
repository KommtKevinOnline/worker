package queue

import (
	"container/list"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Adeithe/go-twitch/api"
	"kommtkevinonline.de/ai"
	"kommtkevinonline.de/converter"
	"kommtkevinonline.de/models"
	videoDownloader "kommtkevinonline.de/video-downloader"
)

var queue list.List = list.List{}

func Process() {
	var next *list.Element

	for queueItem := queue.Front(); queueItem != nil; queueItem = next {
		video := queueItem.Value.(api.Video)

		var duration time.Duration = video.Duration.AsDuration()

		if duration < time.Minute*5 {
			duration = 0
		} else {
			duration = duration - time.Minute*5
		}

		vod, err := videoDownloader.DownloadVod(duration, video.URL)

		if err != nil {
			fmt.Printf("Error downloading vod: %s", err)
			return
		}

		vodAudio, err := converter.Convert(vod)

		if err != nil {
			fmt.Printf("Error converting vod: %s", err)
			return
		}

		transcription, err := ai.Transcribe(vodAudio)

		if err != nil {
			fmt.Printf("Error transcribing vod: %s", err)
			return
		}

		predictionRes, err := ai.Predict(transcription.Text, video)

		if err != nil {
			fmt.Printf("Error predicting vod: %s", err)
			return
		}

		for _, prediction := range predictionRes.Predictions {
			parsedDate, parseErr := time.Parse(time.RFC3339, prediction.DateTime)
			if parseErr != nil {
				fmt.Printf("Error parsing date: %s", parseErr)
				continue
			}
			predictionModel := models.Prediction{
				ClipID: video.ID,
				Source: "twitch",
				Date:   parsedDate,
				Type:   prediction.EventType,
				Topic:  prediction.Topic,
			}

			err = predictionModel.Save()

			if err != nil {
				fmt.Printf("Error saving prediction: %s", err)
			}
		}

		fillOffdayGaps(video, predictionRes.Predictions)

		fmt.Printf("Vod \"%s\" processed successfully.\n", video.ID)

		next = queueItem.Next()
		queue.Remove(queueItem)
	}
}

func AddToQueue(vod *api.Video) {
	queue.PushBack(*vod)
}

// fillOffdayGaps inserts offday predictions for each calendar day between the
// VOD's published date and the earliest "live" prediction that has no prediction yet.
func fillOffdayGaps(video api.Video, predictions []ai.PredictionStructuredResponse) {
	loc := video.PublishedAt.Location()

	// Find the earliest "live" prediction date.
	var earliestLive *time.Time
	// Track which calendar days already have a prediction from ChatGPT.
	coveredDays := make(map[string]bool)

	for _, p := range predictions {
		parsed, err := time.Parse(time.RFC3339, p.DateTime)
		if err != nil {
			continue
		}
		dayKey := parsed.In(loc).Format("2006-01-02")
		coveredDays[dayKey] = true

		if p.EventType == "live" {
			t := parsed.In(loc)
			if earliestLive == nil || t.Before(*earliestLive) {
				earliestLive = &t
			}
		}
	}

	// If there is no live prediction, there is no meaningful range to fill.
	if earliestLive == nil {
		return
	}

	// Parse DEFAULT_STREAM_TIME (expected format "HH:MM") for the offday time component.
	defaultHour, defaultMin := 18, 0 // fallback
	if dst := os.Getenv("DEFAULT_STREAM_TIME"); dst != "" {
		parts := strings.SplitN(dst, ":", 2)
		if len(parts) == 2 {
			if h, err := strconv.Atoi(parts[0]); err == nil {
				defaultHour = h
			}
			if m, err := strconv.Atoi(parts[1]); err == nil {
				defaultMin = m
			}
		}
	}

	// Start date: the calendar day after the VOD was published.
	vodDay := time.Date(
		video.PublishedAt.Year(), video.PublishedAt.Month(), video.PublishedAt.Day(),
		0, 0, 0, 0, loc,
	)
	startDay := vodDay.AddDate(0, 0, 1)

	// End date: the calendar day of the earliest live prediction (exclusive).
	endDay := time.Date(
		earliestLive.Year(), earliestLive.Month(), earliestLive.Day(),
		0, 0, 0, 0, loc,
	)

	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		dayKey := day.Format("2006-01-02")
		if coveredDays[dayKey] {
			continue
		}

		offdayDate := time.Date(day.Year(), day.Month(), day.Day(), defaultHour, defaultMin, 0, 0, loc)
		predictionModel := models.Prediction{
			ClipID: video.ID,
			Source: "twitch",
			Date:   offdayDate,
			Type:   "offday",
			Topic:  "",
		}

		if err := predictionModel.Save(); err != nil {
			fmt.Printf("Error saving offday prediction for %s: %s\n", dayKey, err)
		}
	}
}

package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Adeithe/go-twitch/api"
	openai "github.com/sashabaranov/go-openai"
	"kommtkevinonline.de/ai"
	"kommtkevinonline.de/converter"
	"kommtkevinonline.de/database"
	"kommtkevinonline.de/models"
	videoDownloader "kommtkevinonline.de/video-downloader"
)

const maxAttempts = 3

type item struct {
	video    api.Video
	attempts int
}

var (
	mu        sync.Mutex
	items     []item
	processMu sync.Mutex
)

func AddToQueue(vod *api.Video) {
	mu.Lock()
	defer mu.Unlock()

	for _, existing := range items {
		if existing.video.ID == vod.ID {
			return
		}
	}

	items = append(items, item{video: *vod})
}

func drain() []item {
	mu.Lock()
	defer mu.Unlock()

	drained := items
	items = nil

	return drained
}

func requeue(failed []item) {
	mu.Lock()
	defer mu.Unlock()

	items = append(items, failed...)
}

// Process works through the queue once. Failed items are kept for the next
// run (up to maxAttempts) so one broken VOD never blocks the others.
func Process() {
	if !processMu.TryLock() {
		log.Println("queue: processing already running, skipping")
		return
	}
	defer processMu.Unlock()

	loc, err := time.LoadLocation(os.Getenv("TZ"))
	if err != nil {
		loc, _ = time.LoadLocation("Europe/Berlin")
	}

	var failed []item

	for _, queueItem := range drain() {
		if err := processVideo(queueItem.video, loc); err != nil {
			queueItem.attempts++
			log.Printf("queue: processing vod %q failed (attempt %d/%d): %v", queueItem.video.ID, queueItem.attempts, maxAttempts, err)

			if queueItem.attempts < maxAttempts {
				failed = append(failed, queueItem)
			} else {
				log.Printf("queue: dropping vod %q after %d attempts", queueItem.video.ID, maxAttempts)
			}

			continue
		}

		log.Printf("queue: vod %q processed successfully", queueItem.video.ID)
	}

	requeue(failed)
}

func processVideo(video api.Video, loc *time.Location) error {
	var startOffset time.Duration

	if d := video.Duration.AsDuration(); d >= 5*time.Minute {
		startOffset = d - 5*time.Minute
	}

	vod, err := videoDownloader.DownloadVod(startOffset, video.URL)
	if err != nil {
		return err
	}

	vodAudio, err := converter.Convert(vod)
	if err != nil {
		return err
	}

	// Keep the raw bytes so retries don't read from a drained buffer
	audioBytes := vodAudio.Bytes()

	transcription, err := retry(3, func() (openai.AudioResponse, error) {
		return ai.Transcribe(bytes.NewReader(audioBytes))
	})
	if err != nil {
		return err
	}

	input := ai.Input{
		Text:              annotateTranscript(transcription),
		Source:            "twitch",
		ClipID:            video.ID,
		ReferenceTime:     video.PublishedAt,
		TailOffsetSeconds: startOffset.Seconds(),
		HasSegments:       len(transcription.Segments) > 0,
	}

	outcome, err := ai.RunPrediction(context.Background(), ai.NewOpenAIClient(), ai.NewDBStore(), input, loc)
	if err != nil {
		return err
	}

	log.Printf("queue: vod %q (model %s) saved %d predictions, cleared %d days", video.ID, outcome.ModelUsed, len(outcome.Saved), len(outcome.ClearedDays))

	fillOffdayGaps(video, outcome.Saved, loc)

	transcriptJSON, err := json.Marshal(transcription)
	if err != nil {
		log.Printf("queue: error marshaling transcript for vod %q: %v", video.ID, err)
		transcriptJSON = []byte("")
	}

	// Persisting the VOD marks it processed; without this row it would be
	// re-downloaded and re-transcribed on every stream.offline event.
	if err := database.Persist(string(transcriptJSON), video, nil, startOffset); err != nil {
		return err
	}

	return nil
}

// annotateTranscript prefixes each segment with its [Ns] offset so the model
// can reference where an announcement was made (quote_start_seconds).
func annotateTranscript(transcription openai.AudioResponse) string {
	if len(transcription.Segments) == 0 {
		return transcription.Text
	}

	var b strings.Builder
	for _, segment := range transcription.Segments {
		fmt.Fprintf(&b, "[%ds] %s\n", int(segment.Start), strings.TrimSpace(segment.Text))
	}

	return b.String()
}

func retry[T any](attempts int, fn func() (T, error)) (T, error) {
	var res T
	var err error

	for i := 0; i < attempts; i++ {
		res, err = fn()
		if err == nil {
			return res, nil
		}

		log.Printf("queue: attempt %d/%d failed: %v", i+1, attempts, err)

		if i < attempts-1 {
			time.Sleep(time.Duration(i+1) * 5 * time.Second)
		}
	}

	return res, err
}

// fillOffdayGaps inserts offday predictions for each calendar day between the
// VOD's published date and the earliest "live" prediction that has no prediction yet.
func fillOffdayGaps(video api.Video, predictions []models.Prediction, loc *time.Location) {

	// Find the earliest "live" prediction date.
	var earliestLive *time.Time
	// Track which calendar days already have a prediction from the model.
	coveredDays := make(map[string]bool)

	for _, p := range predictions {
		t := p.Date.In(loc)
		coveredDays[t.Format("2006-01-02")] = true

		if p.Type == "live" {
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

	for _, offdayDate := range gapDays(video.PublishedAt, *earliestLive, coveredDays, defaultHour, defaultMin, loc) {
		dayKey := offdayDate.Format("2006-01-02")
		predictionModel := models.Prediction{
			ClipID: video.ID,
			Source: "twitch",
			Date:   offdayDate,
			Type:   "offday",
			Topic:  "",
		}

		if _, err := predictionModel.Save(); err != nil {
			log.Printf("queue: error saving offday prediction for %s: %v", dayKey, err)
		}
	}
}

// gapDays returns the offday timestamps for every uncovered calendar day
// strictly between the VOD's publish day and the earliest live prediction day.
func gapDays(published time.Time, earliestLive time.Time, coveredDays map[string]bool, hour, min int, loc *time.Location) []time.Time {
	published = published.In(loc)
	earliestLive = earliestLive.In(loc)

	startDay := time.Date(published.Year(), published.Month(), published.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	endDay := time.Date(earliestLive.Year(), earliestLive.Month(), earliestLive.Day(), 0, 0, 0, 0, loc)

	var days []time.Time

	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		if coveredDays[day.Format("2006-01-02")] {
			continue
		}

		days = append(days, time.Date(day.Year(), day.Month(), day.Day(), hour, min, 0, 0, loc))
	}

	return days
}

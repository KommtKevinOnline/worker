package queue

import (
	"container/list"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Adeithe/go-twitch/api"
	"niki2k1.dev/m/chatgpt"
	"niki2k1.dev/m/converter"
	"niki2k1.dev/m/downloader"
	"niki2k1.dev/m/postgres"
	"niki2k1.dev/m/transcriber"
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

		vod, err := downloader.DownloadVod(duration, video.URL)

		if err != nil {
			panic(err)
		}

		vodAudio, err := converter.Convert(vod)

		if err != nil {
			panic(err)
		}

		transcription, err := transcriber.Transcribe(vodAudio)

		if err != nil {
			panic(err)
		}

		upcoming, err := chatgpt.Classify(transcription.Text, video)

		if err != nil {
			panic(err)
		}

		transcriptionJson, err := json.Marshal(transcription)

		if err != nil {
			fmt.Println(err)
			return
		}

		postgres.Persist(string(transcriptionJson), video, upcoming.Dates, duration)
		fmt.Printf("Vod \"%s\" processed successfully.\n", video.ID)

		next = queueItem.Next()
		queue.Remove(queueItem)
	}
}

func AddToQueue(vod *api.Video) {
	queue.PushBack(*vod)
}

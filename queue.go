package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Adeithe/go-twitch/api"
)

var queue list.List = list.List{}

func process() {
	var next *list.Element

	for queueItem := queue.Front(); queueItem != nil; queueItem = next {
		video := queueItem.Value.(api.Video)

		var duration time.Duration = video.Duration.AsDuration()

		if (duration < time.Minute * 5) {
			duration = 0
		} else {
			duration = duration - time.Minute * 5
		}

		vod, err := downloadVod(duration, video.URL)

		if err != nil {
			panic(err)
		}

		transcription, err := transcribe(vod)
		if err != nil {
			panic(err)
		}

		transcriptionJson, err := json.Marshal(transcription)
	
    if err != nil {
      panic(err)
    }

		upcoming, err := classify(transcription.Text, video)

		if err != nil {
      panic(err)
    }

		persist(string(transcriptionJson), video, upcoming, duration)
		fmt.Printf("Vod \"%s\" processed successfully.\n", video.ID)

		next = queueItem.Next()
    queue.Remove(queueItem)
	}
}

func addToQueue(vod *api.Video) {
	queue.PushBack(*vod)
}
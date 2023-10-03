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
	for queueItem := queue.Front(); queueItem != nil; queueItem = queueItem.Next() {
		video := queueItem.Value.(api.Video)

		vod, err := downloadVod(video.Duration.AsDuration() - time.Minute * 2, video.URL)

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

		upcoming, err := classify(transcription.Text)

		if err != nil {
      panic(err)
    }

		persist(string(transcriptionJson), video, upcoming)
		fmt.Printf("Vod \"%s\" processed successfully.\n", video.ID)

		queue.Remove(queueItem)
	}
}

func addToQueue(vod *api.Video) {
	queue.PushBack(*vod)
}
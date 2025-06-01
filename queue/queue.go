package queue

import (
	"container/list"
	"fmt"
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
			fmt.Println(err)
			return
		}

		vodAudio, err := converter.Convert(vod)

		if err != nil {
			fmt.Println(err)
			return
		}

		transcription, err := ai.Transcribe(vodAudio)

		if err != nil {
			fmt.Println(err)
			return
		}

		predictions, err := ai.Predict(transcription.Text, video)

		if err != nil {
			fmt.Println(err)
			return
		}

		for _, prediction := range predictions {
			predictionModel := models.Prediction{
				ClipID: video.ID,
				Source: "twitch",
				Date:   prediction.Date,
				Type:   prediction.EventType,
				Topic:  prediction.Topic,
			}

			err = predictionModel.Save()

			if err != nil {
				fmt.Println(err)
			}
		}

		fmt.Printf("Vod \"%s\" processed successfully.\n", video.ID)

		next = queueItem.Next()
		queue.Remove(queueItem)
	}
}

func AddToQueue(vod *api.Video) {
	queue.PushBack(*vod)
}

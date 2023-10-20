package main

import (
	"fmt"

	"github.com/Adeithe/go-twitch/api"
	"github.com/joho/godotenv"
	"github.com/samber/lo"
)

func main() {
	godotenv.Load()
	
	isOnline, err := isStreamerLive()

	if err == nil && isOnline {
		fmt.Println("Streamer is online; skip")
	} else if !isOnline {
		fmt.Println("Streamer is offline")

		alreadyDownloaded, err := getDownloadedVods()

		if err != nil {
			panic(err)
		}

		videos, err := getLatestVideos()

		if err != nil {
			panic(err)
		}

		lo.ForEach[api.Video](*videos, func(video api.Video, index int) {
			if lo.Contains(alreadyDownloaded, video.ID) {
				return
			}
			
			fmt.Printf("New Vod \"%s\" found.\n", video.ID)
			addToQueue(&video)
		})

		process()
	} else {
		panic(err)
	}
}
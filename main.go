package main

import (
	"fmt"

	"github.com/Adeithe/go-twitch/api"
	"github.com/joho/godotenv"
	"github.com/robfig/cron"
	"github.com/samber/lo"
)

func run() {
	isOnline, err := isStreamerLive()

	if isOnline {
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
			
			fmt.Println(fmt.Sprintf("New Vod \"%s\" found.", video.ID))
			addToQueue(&video)
		})

		process()
	} else {
		panic(err)
	}
}

func main() {
	godotenv.Load()

	c := cron.New()
	c.AddFunc("0 0 * * * *", run)
	c.Start()

	run()

	for true {}
}
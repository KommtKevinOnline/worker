package twitch

import (
	"fmt"
	"os"

	"log"
	"strings"

	"github.com/Adeithe/go-twitch/api"
	eventsub "github.com/joeyak/go-twitch-eventsub/v3"
	"github.com/samber/lo"
	"kommtkevinonline.de/database"
	"kommtkevinonline.de/models"
	"kommtkevinonline.de/queue"
)

func onStreamerOffline() {
	alreadyDownloaded, err := database.GetDownloadedVods()

	if err != nil {
		panic(err)
	}

	videos, err := GetLatestVideos()

	if err != nil {
		panic(err)
	}

	lo.ForEach(*videos, func(video api.Video, index int) {
		if lo.Contains(alreadyDownloaded, video.ID) {
			return
		}

		fmt.Printf("New Vod \"%s\" found.\n", video.ID)
		queue.AddToQueue(&video)
	})

	queue.Process()
}

func RegisterEventSub() *eventsub.Client {
	client := eventsub.NewClient()

	client.OnError(func(err error) {
		fmt.Printf("ERROR: %v\n", err)
	})

	client.OnWelcome(func(message eventsub.WelcomeMessage) {
		token, err := models.TwitchToken.Get(models.TwitchToken{})
		if err != nil {
			log.Printf("Failed to get twitch login data: %v", err)
			return
		}

		_, err = eventsub.SubscribeEvent(eventsub.SubscribeRequest{
			SessionID:   message.Payload.Session.ID,
			ClientID:    os.Getenv("TWITCH_CLIENT_ID"),
			AccessToken: token.AccessToken,
			Event:       eventsub.SubStreamOffline,
			Condition: map[string]string{
				"broadcaster_user_id": os.Getenv("TWITCH_STREAMER_ID"),
			},
		})

		if err != nil {
			if strings.Contains(err.Error(), "401") {
				RefreshToken()

				// retry subscribing
				RegisterEventSub()
			} else {
				fmt.Printf("ERROR subscribing: %v\n", err)
				return
			}
		}

		fmt.Printf("Subscribed to stream.offline event\n")
	})

	client.OnEventStreamOffline(func(event eventsub.EventStreamOffline) {
		fmt.Printf("Received stream.offline event: %+v\n", event)
		onStreamerOffline()
	})

	if err := client.Connect(); err != nil {
		fmt.Printf("Could not connect client: %v\n", err)
	}

	return client
}

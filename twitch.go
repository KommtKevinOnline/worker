package main

import (
	"context"
	"os"

	"github.com/Adeithe/go-twitch"
	"github.com/Adeithe/go-twitch/api"
	"golang.org/x/oauth2/clientcredentials"
)

func getConfig() clientcredentials.Config {
	return clientcredentials.Config {
		ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		TokenURL: 	 "https://id.twitch.tv/oauth2/token",
	}
}

func getClient() *api.Client {
	conf := getConfig()

	return twitch.API(conf.ClientID, api.WithClientSecret(conf.ClientSecret))
}

func getBearer() (string, error) {
	conf := getConfig()

	bearer, err := conf.Token(context.Background())

	if err != nil {
		return "", err
	}

	return bearer.AccessToken, err
}

func isStreamerLive() (bool, error) {
	client := getClient()

	call := client.Streams.List().UserID([]string{os.Getenv("TWITCH_STREAMER_ID")}).Type("live").First(1)

	bearer, err := getBearer()

	if err != nil {
		return false, err
	}

	streams, err := call.Do(context.Background(), api.WithBearerToken(bearer))
		
	return len(streams.Data) == 1, err
}

func getLatestVideos() (*[]api.Video, error) {
	client := getClient()

	bearer, err := getBearer()

	if err != nil {
		return nil, err
	}

	videoCall := client.Videos.List().UserID(os.Getenv("TWITCH_STREAMER_ID")).Type("archive").First(20)

	videos, err := videoCall.Do(context.Background(), api.WithBearerToken(bearer))

	if err != nil {
		panic(err)
	}

	return &videos.Data, nil
}
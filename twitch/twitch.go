package twitch

import (
	"context"
	"fmt"
	"os"

	"github.com/Adeithe/go-twitch"
	"github.com/Adeithe/go-twitch/api"
	"golang.org/x/oauth2/clientcredentials"
)

func getConfig() clientcredentials.Config {
	return clientcredentials.Config{
		ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		TokenURL:     "https://id.twitch.tv/oauth2/token",
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

func IsStreamerLive() (bool, error) {
	client := getClient()

	call := client.Streams.List().UserID([]string{os.Getenv("TWITCH_STREAMER_ID")}).Type("live").First(1)

	bearer, err := getBearer()

	if err != nil {
		return false, err
	}

	streams, err := call.Do(context.Background(), api.WithBearerToken(bearer))

	return len(streams.Data) == 1, err
}

func GetLatestVideos(limit int) (*[]api.Video, error) {
	client := getClient()

	bearer, err := getBearer()

	if err != nil {
		return nil, err
	}

	videoCall := client.Videos.List().UserID(os.Getenv("TWITCH_STREAMER_ID")).Type("archive").First(limit)

	videos, err := videoCall.Do(context.Background(), api.WithBearerToken(bearer))

	if err != nil {
		return nil, err
	}

	return &videos.Data, nil
}

func GetVideoById(id string) (*api.Video, error) {
	client := getClient()

	bearer, err := getBearer()

	if err != nil {
		return nil, err
	}

	video, err := client.Videos.List().ID([]string{id}).Do(context.Background(), api.WithBearerToken(bearer))

	if err != nil {
		return nil, err
	}

	if len(video.Data) == 0 {
		return nil, fmt.Errorf("no video found for id %s", id)
	}

	return &video.Data[0], nil
}

func GetCurrentStream() (*api.Stream, error) {
	client := getClient()

	bearer, err := getBearer()

	if err != nil {
		return nil, err
	}

	streams, err := client.Streams.List().UserID([]string{os.Getenv("TWITCH_STREAMER_ID")}).First(1).Do(context.Background(), api.WithBearerToken(bearer))

	if err != nil {
		return nil, err
	}

	if len(streams.Data) == 0 {
		return nil, nil
	}

	return &streams.Data[0], nil
}

package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	twitchdl "github.com/jybp/twitch-downloader"
	"github.com/samber/lo"
)

func downloadVod(start time.Duration, url string) (io.ReadCloser, error) {
	qualities, err := twitchdl.Qualities(context.Background(), &http.Client{}, os.Getenv("TWITCH_GQL_CLIENT_ID"), url)

	if err != nil {
		return nil, err
	}

	var resolution string

	if (lo.Contains[string](qualities, "Audio Only")) {
		resolution = "Audio Only"
	} else {
		resolution = qualities[0]
	}

	return twitchdl.Download(context.Background(), &http.Client{}, os.Getenv("TWITCH_GQL_CLIENT_ID"), url, resolution, start, 0)
}
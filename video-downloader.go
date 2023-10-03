package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	twitchdl "github.com/jybp/twitch-downloader"
)

func downloadVod(start time.Duration, url string) (io.ReadCloser, error) {
	return twitchdl.Download(context.Background(), &http.Client{}, os.Getenv("TWITCH_GQL_CLIENT_ID"), url, "Audio Only", start, 0)
}
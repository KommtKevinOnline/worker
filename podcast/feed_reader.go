package podcast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mmcdole/gofeed"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"kommtkevinonline.de/ai"
)

func GetLatestEpisode() {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(os.Getenv("PODCAST_FEED_URL"))
	fmt.Println(feed.Items[len(feed.Items)-1].Title)

	url := feed.Items[len(feed.Items)-1].Enclosures[0].URL

	fmt.Println(url)

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	inputStream, err := Convert(resp.Body)

	if err != nil {
		fmt.Println(err)
	}

	audio, err := ai.Transcribe(inputStream)

	// TODO: check max content-length 26214400

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(json.Marshal(audio))
}

func Convert(inputData io.ReadCloser) (*bytes.Buffer, error) {
	// Read the input data
	inputDataBytes, err := io.ReadAll(inputData)

	if err != nil {
		return nil, fmt.Errorf("failed to read input data: %w", err)
	}

	// Create a buffer to hold the input and output data
	inputBuffer := bytes.NewReader(inputDataBytes)
	outputBuffer := &bytes.Buffer{}

	fs, err := os.Create("output.ogg")
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer fs.Close()

	// Set up the FFmpeg process
	err = ffmpeg.Input("pipe:0").
		Output("pipe:1", ffmpeg.KwArgs{
			"f":   "ogg",
			"b:a": "12k",
			"ac":  "1",
			"ar":  "16000",
		}).
		WithInput(inputBuffer).
		WithOutput(outputBuffer).
		WithOutput(fs).
		Run()

	if err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	// Retrieve the output data as WAV
	return outputBuffer, nil
}

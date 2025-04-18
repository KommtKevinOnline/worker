package main

import (
	"context"
	"io"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

type TranscribeResponse struct {
	Text     string    `json:"text"`
	Segments []Segment `json:"segments"`
	Language string    `json:"language"`
}

func transcribe(file io.Reader) (openai.AudioResponse, error) {
	client := openai.NewClient(os.Getenv(("OPENAI_CHATGPT_TOKEN")))

	ctx := context.Background()

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   file,
		FilePath: "audio.webm",
		Format:   openai.AudioResponseFormatVerboseJSON,
		Language: "german",
		TimestampGranularities: []openai.TranscriptionTimestampGranularity{
			openai.TranscriptionTimestampGranularitySegment,
		},
	}

	resp, err := client.CreateTranscription(ctx, req)

	return resp, err
}

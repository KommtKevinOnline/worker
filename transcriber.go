package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Segment struct {
	ID               int       `json:"id"`
	Seek             int       `json:"seek"`
	Start            float64   `json:"start"`
	End              float64   `json:"end"`
	Text             string    `json:"text"`
	Tokens           []int     `json:"tokens"`
	Temperature      float64   `json:"temperature"`
	AvgLogprob       float64   `json:"avg_logprob"`
	CompressionRatio float64   `json:"compression_ratio"`
	NoSpeechProb     float64   `json:"no_speech_prob"`
}

type TranscribeResponse struct {
	Text     string    `json:"text"`
	Segments []Segment `json:"segments"`
	Language string    `json:"language"`
}

func transcribe(file io.Reader) (TranscribeResponse, error) {
	apiURL := fmt.Sprintf("%s/transcribe", os.Getenv("WHISPER_BASE_URL"))

	// Create a new HTTP request
	req, err := http.NewRequest("POST", apiURL, file)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("WHISPER_API_KEY")))

	if err != nil {
		return TranscribeResponse{}, err
	}

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return TranscribeResponse{}, err
	}

	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return TranscribeResponse{}, err
	}

	var data TranscribeResponse

	err = json.Unmarshal([]byte(responseBody), &data)
	if err != nil {
		fmt.Println("Error:", err)
		return TranscribeResponse{}, err
	}

	return data, nil
}
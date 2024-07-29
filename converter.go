package main

import (
	"bytes"
	"io"
	"log"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func convert(inputData io.ReadCloser) (*bytes.Buffer, error) {
	// Read the input data
	inputDataBytes, err := io.ReadAll(inputData)

	if err != nil {
		log.Fatalf("failed to read input data: %v", err)
		return nil, err
	}

	// Create a buffer to hold the input and output data
	inputBuffer := bytes.NewReader(inputDataBytes)
	outputBuffer := &bytes.Buffer{}

	// Set up the FFmpeg process
	err = ffmpeg.Input("pipe:0").
		Output("pipe:1", ffmpeg.KwArgs{"f": "webm"}).
		WithInput(inputBuffer).
		WithOutput(outputBuffer).
		Run()

	if err != nil {
		log.Fatalf("%v", err)
		log.Fatalf("ffmpeg conversion failed: %v", err)
	}

	// Retrieve the output data as WAV
	return outputBuffer, nil
}
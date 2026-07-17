package converter

import (
	"bytes"
	"fmt"
	"io"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func Convert(inputData io.ReadCloser) (*bytes.Buffer, error) {
	defer inputData.Close()

	inputDataBytes, err := io.ReadAll(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to read input data: %w", err)
	}

	inputBuffer := bytes.NewReader(inputDataBytes)
	outputBuffer := &bytes.Buffer{}

	err = ffmpeg.Input("pipe:0").
		Output("pipe:1", ffmpeg.KwArgs{"f": "webm"}).
		WithInput(inputBuffer).
		WithOutput(outputBuffer).
		Run()

	if err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	return outputBuffer, nil
}

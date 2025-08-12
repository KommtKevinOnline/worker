package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Adeithe/go-twitch/api"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Upcoming struct {
	Dates      []string `json:"dates"`
	ViltOnline bool     `json:"vilt_online"`
}

func classify(transcription string, video api.Video) (Upcoming, error) {
	client := openai.NewClient(os.Getenv("OPENAI_CHATGPT_TOKEN"))

	functionDefinitions := openai.FunctionDefinition{
		Name: "has_online_intend",
		Description: "Evaluate if the streamer will stream again. Set vilt_online=true if uncertain or no clear data.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"dates": {
					Type: jsonschema.Array,
					Items: &jsonschema.Definition{
						Type: jsonschema.String,
					},
					Description: "Array of planned stream dates in RFC3339 format",
				},
				"vilt_online": {
					Type: jsonschema.Boolean,
					Description: "True if streamer is uncertain or no data about streaming",
				},
			},
		},
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     openai.GPT4oMini,
			Functions: []openai.FunctionDefinition{functionDefinitions},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You evaluate if a streamer will stream again. Return vilt_online=true if uncertain or no clear statement. Return empty array only if clearly no plans.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(
						"Transcription from %s: %s",
						video.PublishedAt.Format(time.RFC3339),
						transcription,
					),
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return Upcoming{}, err
	}

	if len(resp.Choices) == 0 {
		fmt.Println("No choices found")
		return Upcoming{}, nil
	}

	// Debugging: Save raw response
	if f, err := os.Create("chatgpt.json"); err == nil {
		defer f.Close()
		json.NewEncoder(f).Encode(resp)
	}

	var data Upcoming
	if err := json.Unmarshal(
		[]byte(resp.Choices[0].Message.FunctionCall.Arguments),
		&data,
	); err != nil {
		fmt.Printf("Unmarshal error: %v\nRaw: %s\n", 
			err,
			resp.Choices[0].Message.FunctionCall.Arguments,
		)
		return Upcoming{}, err
	}

	return data, nil
}
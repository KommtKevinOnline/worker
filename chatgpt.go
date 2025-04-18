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
	Dates []string `json:"dates"`
}

func classify(transcription string, video api.Video) (Upcoming, error) {
	client := openai.NewClient(os.Getenv(("OPENAI_CHATGPT_TOKEN")))

	functionDefinitions := openai.FunctionDefinition{
		Name:        "has_online_intend",
		Description: "Evaluate the text that had been said by a twitch streamer at the end of his livestream. Try to find out based on that text, if the streamer is going to stream again the following days.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"dates": {
					Type: jsonschema.Array,
					Items: &jsonschema.Definition{
						Type: jsonschema.String,
					},
					Description: "The array of dates he plan's to stream again in RFC3339 format.",
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
					Content: "You are an assistant that trys to evaluate if and when a twitch livestreamer will stream again based on a text that had been said during a past livestream. If you are unsure about the time, default to 16:30. If he doesn't say something if he is planning to stream or not, default to an empty array.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(
						"The date of the transcription is is %s. text: %s",
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

	f, err := os.Create("chatgpt.json")
	if err != nil {
		fmt.Println(err)
		f.Close()
		return Upcoming{}, err
	}

	respJson, err := json.Marshal(resp)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return Upcoming{}, err
	}

	f.Write(respJson)

	f.Close()

	var data Upcoming

	err = json.Unmarshal([]byte(resp.Choices[0].Message.FunctionCall.Arguments), &data)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Response:", resp.Choices[0].Message.FunctionCall.Arguments)
		return Upcoming{}, err
	}

	return data, nil
}

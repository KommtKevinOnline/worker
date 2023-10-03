package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Adeithe/go-twitch/api"
	openai "github.com/sashabaranov/go-openai"
)


type Upcoming struct {
	Date *time.Time	`json:"date,omitempty"`
}

func classify(transcription string, video api.Video) ([]Upcoming, error) {
	client := openai.NewClient(os.Getenv(("OPENAI_CHATGPT_TOKEN")))

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(
				"Evaluate when the streamer called \"Papaplatte\" that has said the following will be livestreaming again (including opcoming streams). The date of the transcription is is %s. Output the result only as json (important: not with any other text) with the following schema: [{ \"date\": \"date in RFC3339 Format\" }]. If he plans multiple streams, add more objects to the array. If he is not planning to stream just output this: [{\"date\": null}]\n text: %s",
						video.PublishedAt.Format(time.RFC3339),
						transcription,
					),
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return []Upcoming{}, err
	}

	var data []Upcoming

	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &data)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Response:", resp.Choices[0].Message.Content)
		return []Upcoming{}, err
	}

	return data, nil
}
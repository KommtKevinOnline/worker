package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	openai "github.com/sashabaranov/go-openai"
)


type Upcoming struct {
	Date time.Time
	OnlineIntend bool
}

func classify(transcription string) (Upcoming, error) {
	client := openai.NewClient(os.Getenv(("OPENAI_CHATGPT_TOKEN")))

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(
						"Evaluate when the streamer called \"Papaplatte\" that has said the following will be livestreaming again (including opcoming streams). Todays date is %s. Output the result only as json (not with any other text) with the keys \"date\", \"online_intend\" (as boolean) and it should always in an array, If he plans multiple streams output them. If he is not planning to stream just output the json array with an object with date: null and online_inted: \"false\" \n text: %s",
						time.Now().Format(time.RFC3339),
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

	var data Upcoming

	fmt.Println(resp.Choices[0].Message.Content)

	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &data)
	if err != nil {
		fmt.Println("Error:", err)
		return Upcoming{}, err
	}

	return data, nil
}
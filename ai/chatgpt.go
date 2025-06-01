package ai

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Adeithe/go-twitch/api"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type PredictionStructuredResponse struct {
	EventType string    `json:"event_type" required:"true" description:"The Type of the Prediction. Possible values are: live, offday"`
	Date      time.Time `json:"date" required:"true" description:"The date of the prediction. Format: RFC3339"`
	Topic     string    `json:"topic" required:"true" description:"The topic of the prediction. This is a short description of the topic of the next livestream."`
}

func Predict(transcription string, video api.Video) ([]PredictionStructuredResponse, error) {
	client := openai.NewClient(os.Getenv(("OPENAI_CHATGPT_TOKEN")))

	var result []PredictionStructuredResponse
	schema, err := jsonschema.GenerateSchemaForType(result)

	if err != nil {
		return []PredictionStructuredResponse{}, err
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
					Name:   "prediction_structured_response",
					Schema: schema,
					Strict: true,
				},
			},
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf(`
						You are an assistant that trys to evaluate if and when a twitch livestreamer will stream again based on different inputs.
						Possbile Inputs are:
						- a transcript of the past livestream.
						- a whatsapp message.

						The EventTypes explained:
						- live: The streamer will stream again at the specified date.
						- offday: The streamer will definetly not stream again at the specified date.

						If the streamer did not specifically announced a time, default to %s.
						`, os.Getenv("DEFAULT_STREAM_TIME")),
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
		fmt.Printf("CreateChatCompletion error: %v", err)
	}

	err = schema.Unmarshal(resp.Choices[0].Message.Content, &result)
	if err != nil {
		fmt.Printf("Unmarshal schema error: %v", err)
	}

	return result, err
}

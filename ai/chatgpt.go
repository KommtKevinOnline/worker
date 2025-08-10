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
	EventType string `json:"event_type" description:"The Type of the Prediction. Possible values are: live, offday"`
	DateTime  string `json:"date" description:"The datetime of the prediction in RFC3339 (e.g. 2006-01-02T15:04:05Z07:00)"`
	Topic     string `json:"topic" description:"The topic of the prediction. This is a short description of the topic of the next livestream."`
}

type PredictionResponse struct {
	Predictions []PredictionStructuredResponse `json:"predictions" description:"The predictions for the next livestream."`
}

func Predict(transcription string, video api.Video) (PredictionResponse, error) {
	client := openai.NewClient(os.Getenv(("OPENAI_CHATGPT_TOKEN")))

	var result PredictionResponse
	schema, err := jsonschema.GenerateSchemaForType(result)

	if err != nil {
		return PredictionResponse{}, err
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
						The topic should be a short description of the topic of the next livestream and in German.
						There should only be one prediction per date.
						
						Possbile Inputs are:
						- a transcript of the past livestream.
						- a whatsapp message.


						Include the EventType. EventTypes explained:
						- live: The streamer will stream again at the specified date.
						- offday: The streamer will definetly not stream again at the specified date.

						You must include the time in the date. If the streamer did not specifically announced a time, default to %s.
						`, os.Getenv("DEFAULT_STREAM_TIME")),
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(
						"Please evaluate the following inputs and return the predictions. The date of the transcription is is %s. text: %s",
						video.PublishedAt.Format(time.RFC3339),
						transcription,
					),
				},
			},
		},
	)

	if err != nil {
		return PredictionResponse{}, fmt.Errorf("CreateChatCompletion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return PredictionResponse{}, fmt.Errorf("empty response from ChatCompletion")
	}

	fmt.Println(resp.Choices[0].Message.Content)

	err = schema.Unmarshal(resp.Choices[0].Message.Content, &result)
	if err != nil {
		return PredictionResponse{}, fmt.Errorf("unmarshal schema error: %w", err)
	}

	return result, err
}

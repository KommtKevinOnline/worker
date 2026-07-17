package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"kommtkevinonline.de/models"
)

const (
	maxTurns = 8
	// Predictions may only target days inside this window around the input's
	// reference time; anything else is rejected as model error.
	maxDaysAhead = 14
	maxDaysBack  = 1
)

// Store abstracts the database operations the prediction tools perform, so
// the engine can be tested without a database.
type Store interface {
	UpcomingPredictions(from time.Time, limit int) ([]models.Prediction, error)
	SavePrediction(prediction models.Prediction) (bool, error)
	ClearDay(day string) (int64, error)
	SaveRun(run models.PredictionRun) error
}

// Completer abstracts the OpenAI client for testing.
type Completer interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

type Input struct {
	Text          string
	Source        string // "twitch" | "whatsapp"
	ClipID        string
	ReferenceTime time.Time
	// Where the transcribed tail starts within the full VOD, in seconds.
	// Used to convert segment-relative quote timestamps to absolute ones.
	TailOffsetSeconds float64
	HasSegments       bool
}

type Outcome struct {
	Saved       []models.Prediction
	ClearedDays []string
	FinalText   string
	ModelUsed   string
}

type dbStore struct{}

func (dbStore) UpcomingPredictions(from time.Time, limit int) ([]models.Prediction, error) {
	return models.UpcomingPredictions(from, limit)
}

func (dbStore) SavePrediction(prediction models.Prediction) (bool, error) {
	return prediction.Save()
}

func (dbStore) ClearDay(day string) (int64, error) {
	return models.ClearPredictionDay(day)
}

func (dbStore) SaveRun(run models.PredictionRun) error {
	return run.Save()
}

func NewDBStore() Store {
	return dbStore{}
}

func NewOpenAIClient() Completer {
	return openai.NewClient(os.Getenv("OPENAI_CHATGPT_TOKEN"))
}

func primaryModel() string {
	if m := os.Getenv("OPENAI_MODEL"); m != "" {
		return m
	}
	return openai.GPT5Nano
}

func escalationModel() string {
	if m := os.Getenv("OPENAI_ESCALATION_MODEL"); m != "" {
		return m
	}
	return openai.GPT5Mini
}

// RunPrediction lets the model read the input and maintain the prediction
// table through validated tools. On hard failure of the primary model it
// escalates once to a stronger model. Every attempt is persisted as a
// prediction_runs row.
func RunPrediction(ctx context.Context, completer Completer, store Store, input Input, loc *time.Location) (Outcome, error) {
	attemptModels := []string{primaryModel()}
	if esc := escalationModel(); esc != "" && esc != attemptModels[0] {
		attemptModels = append(attemptModels, esc)
	}

	var lastErr error

	for _, model := range attemptModels {
		outcome, opsLog, err := runLoop(ctx, completer, store, input, loc, model)

		run := models.PredictionRun{
			Source:     input.Source,
			ClipID:     input.ClipID,
			Model:      model,
			Input:      input.Text,
			Operations: opsLog,
			Response:   outcome.FinalText,
		}

		if err != nil {
			run.Error = err.Error()
		}

		if saveErr := store.SaveRun(run); saveErr != nil {
			log.Printf("ai: error saving prediction run: %v", saveErr)
		}

		if err == nil {
			outcome.ModelUsed = model
			return outcome, nil
		}

		lastErr = err
		log.Printf("ai: model %s failed, escalating if possible: %v", model, err)
	}

	return Outcome{}, lastErr
}

func runLoop(ctx context.Context, completer Completer, store Store, input Input, loc *time.Location, model string) (Outcome, string, error) {
	outcome := Outcome{}

	upcoming, err := store.UpcomingPredictions(input.ReferenceTime.In(loc), 20)
	if err != nil {
		return outcome, "", fmt.Errorf("loading upcoming predictions: %w", err)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt(input, loc, upcoming),
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt(input, loc),
		},
	}

	var ops []map[string]any

	for turn := 0; turn < maxTurns; turn++ {
		resp, err := completer.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
			Tools:    toolDefinitions(),
		})

		if err != nil {
			return outcome, marshalOps(ops), fmt.Errorf("chat completion: %w", err)
		}

		if len(resp.Choices) == 0 {
			return outcome, marshalOps(ops), fmt.Errorf("empty response from model")
		}

		message := resp.Choices[0].Message
		messages = append(messages, message)

		if len(message.ToolCalls) == 0 {
			outcome.FinalText = message.Content
			return outcome, marshalOps(ops), nil
		}

		for _, call := range message.ToolCalls {
			result := executeTool(store, input, loc, call, &outcome)

			ops = append(ops, map[string]any{
				"tool":   call.Function.Name,
				"args":   json.RawMessage(call.Function.Arguments),
				"result": result,
			})

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	return outcome, marshalOps(ops), fmt.Errorf("model did not finish within %d turns", maxTurns)
}

func marshalOps(ops []map[string]any) string {
	if len(ops) == 0 {
		return ""
	}

	data, err := json.Marshal(ops)
	if err != nil {
		return fmt.Sprintf("marshal error: %v", err)
	}

	return string(data)
}

type upsertArgs struct {
	Day               string  `json:"day"`
	Time              string  `json:"time"`
	EventType         string  `json:"event_type"`
	Topic             string  `json:"topic"`
	Confidence        float64 `json:"confidence"`
	Quote             string  `json:"quote"`
	QuoteStartSeconds float64 `json:"quote_start_seconds"`
}

type clearArgs struct {
	Day string `json:"day"`
}

func toolDefinitions() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_upcoming_predictions",
				Description: "Returns the currently stored predictions from today onwards, including their source. Rows with source 'manual' cannot be changed.",
				Parameters: jsonschema.Definition{
					Type:       jsonschema.Object,
					Properties: map[string]jsonschema.Definition{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "upsert_prediction",
				Description: "Creates or replaces THE prediction for one calendar day (there is exactly one per day). Use event_type 'live' when a stream is announced, 'offday' when the streamer explicitly said there will be no stream that day.",
				Parameters: jsonschema.Definition{
					Type:     jsonschema.Object,
					Required: []string{"day", "time", "event_type", "confidence"},
					Properties: map[string]jsonschema.Definition{
						"day": {
							Type:        jsonschema.String,
							Description: "Calendar day in Europe/Berlin, format YYYY-MM-DD",
						},
						"time": {
							Type:        jsonschema.String,
							Description: "Announced start time in Europe/Berlin, format HH:MM (24h). Use the default stream time when no time was announced.",
						},
						"event_type": {
							Type: jsonschema.String,
							Enum: []string{"live", "offday"},
						},
						"topic": {
							Type:        jsonschema.String,
							Description: "Short German description of the planned stream content, empty if unknown",
						},
						"confidence": {
							Type:        jsonschema.Number,
							Description: "How certain the announcement is, 0.0-1.0. Firm announcement with time: >=0.8. Vague ('vielleicht', 'mal schauen'): <=0.5.",
						},
						"quote": {
							Type:        jsonschema.String,
							Description: "Verbatim quote from the input that this prediction is based on",
						},
						"quote_start_seconds": {
							Type:        jsonschema.Number,
							Description: "The [Ns] timestamp of the segment containing the quote, if the input has segment markers",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "clear_day",
				Description: "Removes the stored prediction for a day, e.g. when a previously announced stream was cancelled without a clear replacement. Prefer upsert_prediction with 'offday' when the streamer explicitly said there is no stream.",
				Parameters: jsonschema.Definition{
					Type:     jsonschema.Object,
					Required: []string{"day"},
					Properties: map[string]jsonschema.Definition{
						"day": {
							Type:        jsonschema.String,
							Description: "Calendar day in Europe/Berlin, format YYYY-MM-DD",
						},
					},
				},
			},
		},
	}
}

func executeTool(store Store, input Input, loc *time.Location, call openai.ToolCall, outcome *Outcome) string {
	switch call.Function.Name {
	case "get_upcoming_predictions":
		upcoming, err := store.UpcomingPredictions(input.ReferenceTime.In(loc), 20)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return formatPredictions(upcoming, loc)

	case "upsert_prediction":
		var args upsertArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		return executeUpsert(store, input, loc, args, outcome)

	case "clear_day":
		var args clearArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if err := validateDay(args.Day, input.ReferenceTime, loc); err != nil {
			return fmt.Sprintf("error: %v", err)
		}

		removed, err := store.ClearDay(args.Day)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		if removed == 0 {
			return "no automated prediction stored for that day (manual rows cannot be removed)"
		}

		outcome.ClearedDays = append(outcome.ClearedDays, args.Day)
		return "cleared"

	default:
		return fmt.Sprintf("error: unknown tool %q", call.Function.Name)
	}
}

func executeUpsert(store Store, input Input, loc *time.Location, args upsertArgs, outcome *Outcome) string {
	if err := validateDay(args.Day, input.ReferenceTime, loc); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	if args.EventType != "live" && args.EventType != "offday" {
		return "error: event_type must be 'live' or 'offday'"
	}

	timePart := args.Time
	if timePart == "" {
		timePart = defaultStreamTime()
	}

	date, err := time.ParseInLocation("2006-01-02 15:04", args.Day+" "+timePart, loc)
	if err != nil {
		return fmt.Sprintf("error: invalid time %q, expected HH:MM", args.Time)
	}

	confidence := min(max(args.Confidence, 0), 1)

	prediction := models.Prediction{
		ClipID:     input.ClipID,
		Source:     input.Source,
		Date:       date,
		Topic:      strings.TrimSpace(args.Topic),
		Type:       args.EventType,
		Confidence: &confidence,
		Quote:      strings.TrimSpace(args.Quote),
	}

	if input.HasSegments && args.QuoteStartSeconds > 0 {
		absolute := input.TailOffsetSeconds + args.QuoteStartSeconds
		prediction.QuoteStart = &absolute
	}

	saved, err := store.SavePrediction(prediction)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	if !saved {
		return "skipped: that day has a manual prediction which automated sources cannot overwrite"
	}

	outcome.Saved = append(outcome.Saved, prediction)
	return "saved"
}

func validateDay(day string, reference time.Time, loc *time.Location) error {
	parsed, err := time.ParseInLocation("2006-01-02", day, loc)
	if err != nil {
		return fmt.Errorf("invalid day %q, expected YYYY-MM-DD", day)
	}

	refDay := time.Date(reference.In(loc).Year(), reference.In(loc).Month(), reference.In(loc).Day(), 0, 0, 0, 0, loc)
	diff := int(parsed.Sub(refDay).Hours() / 24)

	if diff < -maxDaysBack || diff > maxDaysAhead {
		return fmt.Errorf("day %s is outside the allowed window (%d days back to %d days ahead of %s)", day, maxDaysBack, maxDaysAhead, refDay.Format("2006-01-02"))
	}

	return nil
}

func defaultStreamTime() string {
	if dst := os.Getenv("DEFAULT_STREAM_TIME"); dst != "" {
		return dst
	}
	return "15:00"
}

func formatPredictions(predictions []models.Prediction, loc *time.Location) string {
	if len(predictions) == 0 {
		return "no predictions stored"
	}

	var b strings.Builder
	for _, p := range predictions {
		fmt.Fprintf(&b, "- %s: %s at %s (source: %s", p.Day, p.Type, p.Date.In(loc).Format("15:04"), p.Source)
		if p.Topic != "" {
			fmt.Fprintf(&b, ", topic: %s", p.Topic)
		}
		b.WriteString(")\n")
	}

	return b.String()
}

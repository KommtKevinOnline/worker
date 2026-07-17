package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"kommtkevinonline.de/models"
)

type fakeStore struct {
	saved      []models.Prediction
	cleared    []string
	runs       []models.PredictionRun
	manualDays map[string]bool
}

func (s *fakeStore) UpcomingPredictions(from time.Time, limit int) ([]models.Prediction, error) {
	return nil, nil
}

func (s *fakeStore) SavePrediction(p models.Prediction) (bool, error) {
	day := p.Date.Format("2006-01-02")
	if s.manualDays[day] {
		return false, nil
	}
	s.saved = append(s.saved, p)
	return true, nil
}

func (s *fakeStore) ClearDay(day string) (int64, error) {
	if s.manualDays[day] {
		return 0, nil
	}
	s.cleared = append(s.cleared, day)
	return 1, nil
}

func (s *fakeStore) SaveRun(run models.PredictionRun) error {
	s.runs = append(s.runs, run)
	return nil
}

type fakeCompleter struct {
	responses []openai.ChatCompletionResponse
	errs      []error
	calls     int
	models    []string
}

func (c *fakeCompleter) CreateChatCompletion(_ context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	i := c.calls
	c.calls++
	c.models = append(c.models, req.Model)

	if i < len(c.errs) && c.errs[i] != nil {
		return openai.ChatCompletionResponse{}, c.errs[i]
	}

	if i >= len(c.responses) {
		return openai.ChatCompletionResponse{}, errors.New("no scripted response")
	}

	return c.responses[i], nil
}

func toolCallResponse(name, args string) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant,
				ToolCalls: []openai.ToolCall{{
					ID:       "call_1",
					Type:     openai.ToolTypeFunction,
					Function: openai.FunctionCall{Name: name, Arguments: args},
				}},
			},
		}},
	}
}

func textResponse(text string) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: text,
			},
		}},
	}
}

func testInput(t *testing.T) (Input, *time.Location) {
	t.Helper()

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}

	return Input{
		Text:              "[10s] wir sehen uns morgen um 3",
		Source:            "twitch",
		ClipID:            "v123",
		ReferenceTime:     time.Date(2026, 7, 14, 22, 0, 0, 0, loc),
		TailOffsetSeconds: 20000,
		HasSegments:       true,
	}, loc
}

func TestRunPredictionSavesValidUpsert(t *testing.T) {
	input, loc := testInput(t)
	store := &fakeStore{}
	completer := &fakeCompleter{
		responses: []openai.ChatCompletionResponse{
			toolCallResponse("upsert_prediction", `{"day":"2026-07-15","time":"15:00","event_type":"live","topic":"Minecraft","confidence":0.9,"quote":"wir sehen uns morgen um 3","quote_start_seconds":10}`),
			textResponse("done"),
		},
	}

	outcome, err := RunPrediction(context.Background(), completer, store, input, loc)
	if err != nil {
		t.Fatal(err)
	}

	if len(store.saved) != 1 {
		t.Fatalf("expected 1 saved prediction, got %d", len(store.saved))
	}

	p := store.saved[0]
	if p.Type != "live" || p.Source != "twitch" || p.ClipID != "v123" {
		t.Errorf("unexpected prediction: %+v", p)
	}
	if p.Date.In(loc).Format("2006-01-02 15:04") != "2026-07-15 15:00" {
		t.Errorf("unexpected date: %v", p.Date.In(loc))
	}
	if p.Confidence == nil || *p.Confidence != 0.9 {
		t.Errorf("unexpected confidence: %v", p.Confidence)
	}
	if p.QuoteStart == nil || *p.QuoteStart != 20010 {
		t.Errorf("expected absolute quote start 20010, got %v", p.QuoteStart)
	}

	if len(store.runs) != 1 || store.runs[0].Error != "" {
		t.Errorf("expected 1 clean run row, got %+v", store.runs)
	}
	if len(outcome.Saved) != 1 {
		t.Errorf("expected outcome to contain saved prediction")
	}
}

func TestRunPredictionRejectsDayOutsideWindow(t *testing.T) {
	input, loc := testInput(t)
	store := &fakeStore{}
	completer := &fakeCompleter{
		responses: []openai.ChatCompletionResponse{
			toolCallResponse("upsert_prediction", `{"day":"2026-09-01","time":"15:00","event_type":"live","confidence":0.9}`),
			textResponse("done"),
		},
	}

	_, err := RunPrediction(context.Background(), completer, store, input, loc)
	if err != nil {
		t.Fatal(err)
	}

	if len(store.saved) != 0 {
		t.Fatalf("out-of-window prediction must not be saved, got %+v", store.saved)
	}
}

func TestRunPredictionRespectsManualRows(t *testing.T) {
	input, loc := testInput(t)
	store := &fakeStore{manualDays: map[string]bool{"2026-07-15": true}}
	completer := &fakeCompleter{
		responses: []openai.ChatCompletionResponse{
			toolCallResponse("upsert_prediction", `{"day":"2026-07-15","time":"15:00","event_type":"live","confidence":0.9}`),
			textResponse("done"),
		},
	}

	outcome, err := RunPrediction(context.Background(), completer, store, input, loc)
	if err != nil {
		t.Fatal(err)
	}

	if len(store.saved) != 0 || len(outcome.Saved) != 0 {
		t.Fatalf("manual day must not be overwritten")
	}
}

func TestRunPredictionClearDay(t *testing.T) {
	input, loc := testInput(t)
	store := &fakeStore{}
	completer := &fakeCompleter{
		responses: []openai.ChatCompletionResponse{
			toolCallResponse("clear_day", `{"day":"2026-07-15"}`),
			textResponse("done"),
		},
	}

	outcome, err := RunPrediction(context.Background(), completer, store, input, loc)
	if err != nil {
		t.Fatal(err)
	}

	if len(store.cleared) != 1 || store.cleared[0] != "2026-07-15" {
		t.Fatalf("expected day cleared, got %v", store.cleared)
	}
	if len(outcome.ClearedDays) != 1 {
		t.Errorf("expected cleared day in outcome")
	}
}

func TestRunPredictionEscalatesOnFailure(t *testing.T) {
	input, loc := testInput(t)
	store := &fakeStore{}
	completer := &fakeCompleter{
		errs: []error{errors.New("boom"), nil},
		responses: []openai.ChatCompletionResponse{
			{}, // consumed by the error slot
			textResponse("Keine Ansage enthalten."),
		},
	}

	outcome, err := RunPrediction(context.Background(), completer, store, input, loc)
	if err != nil {
		t.Fatal(err)
	}

	if completer.calls != 2 {
		t.Fatalf("expected escalation call, got %d calls", completer.calls)
	}
	if completer.models[0] == completer.models[1] {
		t.Errorf("expected different models, got %v", completer.models)
	}
	if outcome.ModelUsed != completer.models[1] {
		t.Errorf("expected outcome model %q, got %q", completer.models[1], outcome.ModelUsed)
	}

	// one failed run + one clean run persisted
	if len(store.runs) != 2 || store.runs[0].Error == "" || store.runs[1].Error != "" {
		t.Errorf("unexpected run rows: %+v", store.runs)
	}
}

func TestRunPredictionNoToolsMeansNoWrites(t *testing.T) {
	input, loc := testInput(t)
	store := &fakeStore{}
	completer := &fakeCompleter{
		responses: []openai.ChatCompletionResponse{
			textResponse("Keine Ansage zum nächsten Stream enthalten."),
		},
	}

	outcome, err := RunPrediction(context.Background(), completer, store, input, loc)
	if err != nil {
		t.Fatal(err)
	}

	if len(store.saved) != 0 || len(store.cleared) != 0 {
		t.Fatalf("expected no writes")
	}
	if outcome.FinalText == "" {
		t.Errorf("expected final text")
	}
}

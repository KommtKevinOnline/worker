package ai

import (
	"encoding/json"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func responseFromSegments(t *testing.T, texts ...string) openai.AudioResponse {
	t.Helper()

	segments := make([]map[string]any, len(texts))
	for i, text := range texts {
		segments[i] = map[string]any{"id": i, "text": text}
	}

	raw, err := json.Marshal(map[string]any{"segments": segments})
	if err != nil {
		t.Fatal(err)
	}

	var resp openai.AudioResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}

	return resp
}

func TestCleanTranscriptionDropsArtifactSegments(t *testing.T) {
	resp := responseFromSegments(t,
		" Wir sehen uns morgen um 15 Uhr.",
		" Untertitel im Auftrag des ZDF für funk, 2017",
		" Untertitel der Amara.org-Community",
		" Copyright WDR 2021",
		" Macht's gut!",
	)

	cleaned := CleanTranscription(resp)

	if len(cleaned.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(cleaned.Segments))
	}
	if cleaned.Text != "Wir sehen uns morgen um 15 Uhr. Macht's gut!" {
		t.Errorf("unexpected text: %q", cleaned.Text)
	}
}

func TestCleanTranscriptionCollapsesRepetition(t *testing.T) {
	resp := responseFromSegments(t, " Oh, oh, oh, oh, oh, oh, oh, oh, oh danke")

	cleaned := CleanTranscription(resp)

	if got := cleaned.Segments[0].Text; got != " oh danke" {
		t.Errorf("unexpected collapsed text: %q", got)
	}
}

func TestCleanTranscriptionAllArtifactsMeansEmpty(t *testing.T) {
	resp := responseFromSegments(t, " Untertitel der Amara.org-Community")

	cleaned := CleanTranscription(resp)

	if len(cleaned.Segments) != 0 || cleaned.Text != "" {
		t.Errorf("expected empty result, got %+v", cleaned)
	}
}

func TestCleanTranscriptionKeepsNormalSpeech(t *testing.T) {
	resp := responseFromSegments(t, " In dem Sinne, Freunde, bis morgen!")

	cleaned := CleanTranscription(resp)

	if len(cleaned.Segments) != 1 || cleaned.Text != "In dem Sinne, Freunde, bis morgen!" {
		t.Errorf("normal speech must survive, got %+v", cleaned)
	}
}

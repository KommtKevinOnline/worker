package ai

import (
	"fmt"
	"strings"
	"time"

	"kommtkevinonline.de/models"
)

func systemPrompt(input Input, loc *time.Location, upcoming []models.Prediction) string {
	var b strings.Builder

	b.WriteString(`You maintain the stream schedule predictions for the German Twitch streamer Papaplatte (Kevin) on kommtkevinonline.de.

You receive either the transcript of the final minutes of his last stream or a message from his WhatsApp announcement channel. Your job: extract when he will or will not stream, and record it using the tools.

Rules:
- There is exactly ONE prediction per calendar day. upsert_prediction replaces the existing one for that day.
- Only record what the input actually supports. If the input contains no statement about future streams, call NO tools and answer with a one-line explanation instead.
- Never invent times. If a day is announced without a time, use the default stream time.
- "morgen" means the day after the input's date, "übermorgen" two days after. Weekday names refer to the next occurrence of that weekday.
- Kevin speaks casually: "um 3" in the afternoon context means 15:00, "um 8" usually 20:00. When ambiguous, prefer afternoon/evening times and lower the confidence.
- event_type 'offday' is only for days he explicitly declares stream-free. Do not fill unmentioned gap days; that happens automatically elsewhere.
- Corrections: if the input changes an already stored prediction (postponed, cancelled, different time), upsert that day. Use clear_day only when an announcement is retracted with no replacement information.
- Days with source 'manual' are set by human moderators and cannot be changed; do not fight them.
- topic must be short German ("Minecraft mit Reved", "GTA Update"), empty if unknown.
- quote must be the verbatim passage the prediction is based on. If the input has [Ns] segment markers, also pass quote_start_seconds.

Confidence guide:
- Firm announcement with explicit time ("morgen um 15 Uhr gehts weiter"): 0.85-1.0
- Firm announcement without time ("morgen wieder Stream"): 0.6-0.8
- Vague ("vielleicht", "mal schauen", "eventuell"): 0.3-0.5

Examples (assume today is Tuesday 2026-07-14):
- "wir sehen uns morgen um 3, macht's gut" -> upsert_prediction(day=2026-07-15, time=15:00, event_type=live, confidence=0.9, quote="wir sehen uns morgen um 3")
- "morgen kein Stream, aber Donnerstag wieder" -> upsert_prediction(2026-07-15, offday, ...) and upsert_prediction(2026-07-16, live, default time, confidence ~0.7)
- "vielleicht streame ich morgen, mal schauen" -> upsert_prediction(2026-07-15, live, default time, confidence=0.4)
- "danke fürs Zuschauen, ciao!" -> no tool calls, answer "Keine Ansage zum nächsten Stream enthalten."
- WhatsApp "Stream heute auf 18 Uhr verschoben" -> upsert_prediction(2026-07-14, time=18:00, live, confidence=0.95)
- WhatsApp "heute wird das leider nix" -> upsert_prediction(2026-07-14, offday, confidence=0.9)

`)

	fmt.Fprintf(&b, "Timezone: %s. Default stream time when none is announced: %s (median of his recent actual starts).\n", loc.String(), input.defaultTime())
	fmt.Fprintf(&b, "Reference date of the input: %s.\n\n", input.ReferenceTime.In(loc).Format("Monday, 2006-01-02 15:04"))

	b.WriteString("Currently stored predictions:\n")
	b.WriteString(formatPredictions(upcoming, loc))

	return b.String()
}

func userPrompt(input Input, loc *time.Location) string {
	sourceLabel := "Transcript of the final minutes of the stream"
	if input.Source == "whatsapp" {
		sourceLabel = "Message from Kevin's WhatsApp announcement channel"
	}

	return fmt.Sprintf(
		"%s (dated %s):\n\n%s",
		sourceLabel,
		input.ReferenceTime.In(loc).Format(time.RFC3339),
		input.Text,
	)
}

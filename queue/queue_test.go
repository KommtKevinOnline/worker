package queue

import (
	"testing"
	"time"
)

func mustLoc(t *testing.T) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}

	return loc
}

func TestGapDaysFillsUncoveredDays(t *testing.T) {
	loc := mustLoc(t)

	published := time.Date(2026, 7, 10, 23, 30, 0, 0, loc)
	earliestLive := time.Date(2026, 7, 14, 18, 0, 0, 0, loc)

	days := gapDays(published, earliestLive, map[string]bool{}, 18, 0, loc)

	if len(days) != 3 {
		t.Fatalf("expected 3 gap days, got %d: %v", len(days), days)
	}

	want := []string{"2026-07-11", "2026-07-12", "2026-07-13"}
	for i, day := range days {
		if got := day.Format("2006-01-02"); got != want[i] {
			t.Errorf("day %d: expected %s, got %s", i, want[i], got)
		}
		if day.Hour() != 18 || day.Minute() != 0 {
			t.Errorf("day %d: expected 18:00, got %02d:%02d", i, day.Hour(), day.Minute())
		}
	}
}

func TestGapDaysSkipsCoveredDays(t *testing.T) {
	loc := mustLoc(t)

	published := time.Date(2026, 7, 10, 22, 0, 0, 0, loc)
	earliestLive := time.Date(2026, 7, 13, 18, 0, 0, 0, loc)

	days := gapDays(published, earliestLive, map[string]bool{"2026-07-11": true}, 18, 0, loc)

	if len(days) != 1 {
		t.Fatalf("expected 1 gap day, got %d: %v", len(days), days)
	}

	if got := days[0].Format("2006-01-02"); got != "2026-07-12" {
		t.Errorf("expected 2026-07-12, got %s", got)
	}
}

func TestGapDaysNextDayLiveHasNoGaps(t *testing.T) {
	loc := mustLoc(t)

	published := time.Date(2026, 7, 10, 22, 0, 0, 0, loc)
	earliestLive := time.Date(2026, 7, 11, 18, 0, 0, 0, loc)

	if days := gapDays(published, earliestLive, map[string]bool{}, 18, 0, loc); len(days) != 0 {
		t.Fatalf("expected no gap days, got %v", days)
	}
}

func TestGapDaysUsesBerlinDayBoundaries(t *testing.T) {
	loc := mustLoc(t)

	// 23:30 UTC on the 10th is already the 11th in Berlin (CEST)
	published := time.Date(2026, 7, 10, 23, 30, 0, 0, time.UTC)
	earliestLive := time.Date(2026, 7, 13, 18, 0, 0, 0, loc)

	days := gapDays(published, earliestLive, map[string]bool{}, 18, 0, loc)

	if len(days) != 1 {
		t.Fatalf("expected 1 gap day, got %d: %v", len(days), days)
	}

	if got := days[0].Format("2006-01-02"); got != "2026-07-12" {
		t.Errorf("expected 2026-07-12, got %s", got)
	}
}

package domain

import (
	"testing"
	"time"
)

func TestTemplateStatusDuration(t *testing.T) {
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
	dur := 60
	tpl := Template{Text: "Focus", Emoji: ":dart:", DurationInMinutes: &dur}

	got := tpl.Status(now, nil)
	if got.Text != "Focus" || got.Emoji != ":dart:" {
		t.Fatalf("unexpected status: %+v", got)
	}
	if want := now.Add(time.Hour); !got.Expiration.Equal(want) {
		t.Errorf("expiration got %v, want %v", got.Expiration, want)
	}
}

func TestTemplateStatusCustomDurationOverrides(t *testing.T) {
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
	dur := 60
	tpl := Template{Text: "x", Emoji: ":x:", DurationInMinutes: &dur}
	custom := 30 * time.Minute

	got := tpl.Status(now, &custom)
	if want := now.Add(custom); !got.Expiration.Equal(want) {
		t.Errorf("expiration got %v, want %v", got.Expiration, want)
	}
}

func TestTemplateStatusUntilTimeRollsToNextDay(t *testing.T) {
	// 23:00 now, until 09:00 should be tomorrow.
	now := time.Date(2026, time.January, 15, 23, 0, 0, 0, time.UTC)
	tpl := Template{Text: "x", Emoji: ":x:", UntilTime: "09:00"}

	got := tpl.Status(now, nil)
	want := time.Date(2026, time.January, 16, 9, 0, 0, 0, time.UTC)
	if !got.Expiration.Equal(want) {
		t.Errorf("expiration got %v, want %v", got.Expiration, want)
	}
}

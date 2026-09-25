package domain

import (
	"testing"
	"time"
)

func TestClearModeExpiration(t *testing.T) {
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		mode   ClearMode
		custom time.Duration
		want   time.Time
	}{
		{ClearNever, 0, time.Time{}},
		{Clear30Minutes, 0, now.Add(30 * time.Minute)},
		{Clear1Hour, 0, now.Add(time.Hour)},
		{Clear4Hours, 0, now.Add(4 * time.Hour)},
		{ClearToday, 0, time.Date(2026, time.January, 15, 23, 59, 59, 0, time.UTC)},
		{ClearCustom, 90 * time.Minute, now.Add(90 * time.Minute)},
		{ClearCustom, 0, time.Time{}},
	}
	for _, tc := range cases {
		if got := tc.mode.Expiration(now, tc.custom); !got.Equal(tc.want) {
			t.Errorf("%v: got %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestEndOfWeek(t *testing.T) {
	// Thursday 2026-01-15 -> Sunday 2026-01-18.
	thursday := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
	want := time.Date(2026, time.January, 18, 23, 59, 59, 0, time.UTC)
	if got := ClearThisWeek.Expiration(thursday, 0); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStatusExpired(t *testing.T) {
	now := time.Now()
	if (Status{Expiration: now.Add(-time.Minute)}).Expired(now) != true {
		t.Error("past expiration should be expired")
	}
	if (Status{Expiration: now.Add(time.Minute)}).Expired(now) != false {
		t.Error("future expiration should not be expired")
	}
	if (Status{}).Expired(now) != false {
		t.Error("zero expiration should never expire")
	}
}

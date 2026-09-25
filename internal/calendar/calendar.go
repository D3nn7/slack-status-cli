// Package calendar fetches and parses ICS calendar feeds and derives the
// events relevant for temporary status overrides.
package calendar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	ical "github.com/emersion/go-ical"
)

const userAgent = "slack-status-cli"

// Fetcher downloads and parses ICS feeds.
type Fetcher struct {
	http *http.Client
}

// NewFetcher returns a Fetcher with a sensible request timeout.
func NewFetcher() *Fetcher {
	return &Fetcher{http: &http.Client{Timeout: 15 * time.Second}}
}

// Fetch downloads the ICS document at url and parses all events.
func (f *Fetcher) Fetch(ctx context.Context, url string) ([]domain.Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ICS request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ICS fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ICS server: HTTP %d", resp.StatusCode)
	}
	return Parse(resp.Body)
}

// Parse reads an ICS document and returns its events.
func Parse(r io.Reader) ([]domain.Event, error) {
	cal, err := ical.NewDecoder(r).Decode()
	if err != nil {
		return nil, fmt.Errorf("ICS parse: %w", err)
	}

	var events []domain.Event
	for _, child := range cal.Children {
		if child.Name != ical.CompEvent {
			continue
		}
		ev, err := parseEvent(child)
		if err != nil {
			continue // skip malformed entries
		}
		events = append(events, ev)
	}
	return events, nil
}

func parseEvent(comp *ical.Component) (domain.Event, error) {
	startProp := comp.Props.Get(ical.PropDateTimeStart)
	endProp := comp.Props.Get(ical.PropDateTimeEnd)
	if startProp == nil || endProp == nil {
		return domain.Event{}, errors.New("DTSTART/DTEND missing")
	}

	start, err := parseDateTime(startProp)
	if err != nil {
		return domain.Event{}, err
	}
	end, err := parseDateTime(endProp)
	if err != nil {
		return domain.Event{}, err
	}

	var id, subject string
	if p := comp.Props.Get(ical.PropUID); p != nil {
		id = p.Value
	}
	if p := comp.Props.Get(ical.PropSummary); p != nil {
		subject = p.Value
	}

	allDay := startProp.Params.Get(ical.ParamValue) == "DATE" ||
		len(strings.ReplaceAll(startProp.Value, "-", "")) == 8

	return domain.Event{
		ID:      id,
		Subject: subject,
		Start:   start.UTC(),
		End:     end.UTC(),
		AllDay:  allDay,
	}, nil
}

func parseDateTime(prop *ical.Prop) (time.Time, error) {
	loc := ResolveTimezone(prop.Params.Get(ical.ParamTimezoneID))
	if t, err := prop.DateTime(loc); err == nil {
		return t, nil
	}
	return parseDateTimeRaw(prop.Value)
}

func parseDateTimeRaw(s string) (time.Time, error) {
	type attempt struct {
		layout string
		loc    *time.Location
	}
	attempts := []attempt{
		{"20060102T150405Z", time.UTC},
		{"20060102T150405", time.Local},
		{"20060102", time.Local},
		{"2006-01-02T15:04:05Z07:00", time.UTC},
		{"2006-01-02T15:04:05Z", time.UTC},
		{"2006-01-02T15:04:05", time.Local},
		{"2006-01-02", time.Local},
	}
	for _, a := range attempts {
		if t, err := time.ParseInLocation(a.layout, s, a.loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format: %q", s)
}

// MarkMeetings flags events whose subject looks like a meeting.
func MarkMeetings(events []domain.Event, extraKeywords []string) []domain.Event {
	for i := range events {
		events[i].Meeting = domain.IsMeetingSubject(events[i].Subject, extraKeywords)
	}
	return events
}

// Active returns the timed events running at now, excluding all-day entries.
func Active(events []domain.Event, now time.Time) []domain.Event {
	var active []domain.Event
	for _, ev := range events {
		if ev.AllDay {
			continue
		}
		if !ev.Start.After(now) && ev.End.After(now) {
			active = append(active, ev)
		}
	}
	return active
}

// Earliest returns the event with the earliest start time.
func Earliest(events []domain.Event) domain.Event {
	earliest := events[0]
	for _, ev := range events[1:] {
		if ev.Start.Before(earliest.Start) {
			earliest = ev
		}
	}
	return earliest
}

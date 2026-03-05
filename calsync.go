package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slack-go/slack"
)

// ── Debug logger ─────────────────────────────────────────────────────────────

var calDebugLog *os.File

func initCalDebugLog(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	calDebugLog = f
	logCal("=== Cal-Sync Debug Log gestartet ===")
}

func logCal(format string, args ...interface{}) {
	if calDebugLog == nil {
		return
	}
	fmt.Fprintf(calDebugLog, "[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

// ── Windows → IANA timezone map (häufigste Exchange-Namen) ───────────────────

var windowsTZMap = map[string]string{
	"W. Europe Standard Time":       "Europe/Berlin",
	"Central Europe Standard Time":  "Europe/Budapest",
	"Romance Standard Time":         "Europe/Paris",
	"GMT Standard Time":             "Europe/London",
	"Eastern Standard Time":         "America/New_York",
	"Central Standard Time":         "America/Chicago",
	"Mountain Standard Time":        "America/Denver",
	"Pacific Standard Time":         "America/Los_Angeles",
	"UTC":                           "UTC",
	"Greenwich Standard Time":       "Atlantic/Reykjavik",
	"Russian Standard Time":         "Europe/Moscow",
	"China Standard Time":           "Asia/Shanghai",
	"Tokyo Standard Time":           "Asia/Tokyo",
	"AUS Eastern Standard Time":     "Australia/Sydney",
	"E. Europe Standard Time":       "Europe/Nicosia",
	"Turkey Standard Time":          "Europe/Istanbul",
	"Israel Standard Time":          "Asia/Jerusalem",
	"Arab Standard Time":            "Asia/Riyadh",
	"India Standard Time":           "Asia/Calcutta",
	"SE Asia Standard Time":         "Asia/Bangkok",
	"Korea Standard Time":           "Asia/Seoul",
	"New Zealand Standard Time":     "Pacific/Auckland",
	"Central America Standard Time": "America/Guatemala",
	"SA Eastern Standard Time":      "America/Cayenne",
	"E. South America Standard Time": "America/Sao_Paulo",
}

func resolveTimezone(tzid string) *time.Location {
	if tzid == "" {
		return time.Local
	}
	// IANA-Name direkt versuchen
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc
	}
	// Windows-Name nachschlagen
	if iana, ok := windowsTZMap[tzid]; ok {
		if loc, err := time.LoadLocation(iana); err == nil {
			logCal("TZID %q → IANA %q", tzid, iana)
			return loc
		}
	}
	logCal("WARNUNG: Unbekannte TZID %q – verwende Local", tzid)
	return time.Local
}

// ── Polling ───────────────────────────────────────────────────────────────────

func startCalSyncTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return calSyncTickMsg{}
	})
}

func pollCalendarCmd(cfg calSyncConfig) tea.Cmd {
	return func() tea.Msg {
		now := time.Now()
		logCal("Poll gestartet, now=%s", now.Format("15:04:05"))
		events, err := fetchICSEvents(cfg.ICSUrl)
		if err != nil {
			logCal("FEHLER beim Fetch: %v", err)
			return calSyncErrMsg{Err: err, IsFatal: false}
		}
		logCal("Fetch OK: %d Events total", len(events))
		active := filterActiveEvents(events, now)
		logCal("Aktive Events (jetzt laufend): %d", len(active))
		for _, ev := range active {
			logCal("  → %q  start=%s end=%s", ev.Subject, ev.StartTime.Local().Format("15:04"), ev.EndTime.Local().Format("15:04"))
		}
		return calEventsMsg{Events: events, FetchedAt: now}
	}
}

// ── ICS Fetch + Parse ─────────────────────────────────────────────────────────

func fetchICSEvents(icsURL string) ([]calEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, icsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ICS request: %w", err)
	}
	req.Header.Set("User-Agent", "slack-status-cli/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ICS fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ICS server: HTTP %d", resp.StatusCode)
	}

	cal, err := ical.NewDecoder(resp.Body).Decode()
	if err != nil {
		return nil, fmt.Errorf("ICS parse: %w", err)
	}

	var events []calEvent
	skipped := 0
	for _, child := range cal.Children {
		if child.Name != ical.CompEvent {
			continue
		}
		ev, err := parseICSEvent(child)
		if err != nil {
			logCal("Event übersprungen: %v", err)
			skipped++
			continue
		}
		events = append(events, ev)
	}
	if skipped > 0 {
		logCal("WARNUNG: %d Events konnten nicht geparst werden", skipped)
	}
	return events, nil
}

func parseICSEvent(comp *ical.Component) (calEvent, error) {
	startProp := comp.Props.Get(ical.PropDateTimeStart)
	endProp := comp.Props.Get(ical.PropDateTimeEnd)
	if startProp == nil {
		return calEvent{}, errors.New("DTSTART fehlt")
	}
	if endProp == nil {
		// DTEND fehlt → versuche DURATION zu berechnen
		return calEvent{}, errors.New("DTEND fehlt (DURATION nicht unterstützt)")
	}

	startTime, err := parsePropDateTime(startProp)
	if err != nil {
		return calEvent{}, fmt.Errorf("DTSTART %q: %w", startProp.Value, err)
	}
	endTime, err := parsePropDateTime(endProp)
	if err != nil {
		return calEvent{}, fmt.Errorf("DTEND %q: %w", endProp.Value, err)
	}

	id := ""
	if p := comp.Props.Get(ical.PropUID); p != nil {
		id = p.Value
	}
	subject := ""
	if p := comp.Props.Get(ical.PropSummary); p != nil {
		subject = p.Value
	}

	// All-day: DTSTART hat VALUE=DATE (nur Datum, keine Uhrzeit)
	isAllDay := startProp.Params.Get(ical.ParamValue) == "DATE" || len(strings.ReplaceAll(startProp.Value, "-", "")) == 8

	logCal("Event: %q  start=%s end=%s allday=%v id=%s",
		subject,
		startTime.Local().Format("2006-01-02 15:04"),
		endTime.Local().Format("2006-01-02 15:04"),
		isAllDay,
		id[:min(8, len(id))],
	)

	return calEvent{
		ID:        id,
		Subject:   subject,
		StartTime: startTime.UTC(),
		EndTime:   endTime.UTC(),
		IsAllDay:  isAllDay,
	}, nil
}

// parsePropDateTime parst eine ICS-Datumseigenschaft robust:
// 1. Versucht go-ical's DateTime() mit TZID-Auflösung
// 2. Fallback: manuelles Parsen bekannter Formate
func parsePropDateTime(prop *ical.Prop) (time.Time, error) {
	tzid := prop.Params.Get(ical.ParamTimezoneID)
	loc := resolveTimezone(tzid)

	// Versuche go-ical-Parsing mit aufgelöstem Location
	t, err := prop.DateTime(loc)
	if err == nil {
		return t, nil
	}
	logCal("go-ical DateTime() fehlgeschlagen (TZID=%q value=%q): %v – versuche manuelles Parsen", tzid, prop.Value, err)

	return parseDateTimeRaw(prop.Value)
}

// parseDateTimeRaw versucht alle bekannten ICS-Datumformate.
func parseDateTimeRaw(s string) (time.Time, error) {
	type attempt struct {
		layout string
		loc    *time.Location
	}
	attempts := []attempt{
		{"20060102T150405Z", time.UTC},
		{"20060102T150405", time.Local},
		{"20060102", time.Local}, // DATE-only
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
	return time.Time{}, fmt.Errorf("unbekanntes Datumsformat: %q", s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── Filter + Priorität ────────────────────────────────────────────────────────

func filterActiveEvents(events []calEvent, now time.Time) []calEvent {
	var active []calEvent
	for _, ev := range events {
		if ev.IsAllDay {
			continue
		}
		if !ev.StartTime.After(now) && ev.EndTime.After(now) {
			active = append(active, ev)
		}
	}
	return active
}

func earliestStartEvent(events []calEvent) calEvent {
	earliest := events[0]
	for _, ev := range events[1:] {
		if ev.StartTime.Before(earliest.StartTime) {
			earliest = ev
		}
	}
	return earliest
}

// ── Slack-Status Cmds ─────────────────────────────────────────────────────────

func saveCurrentStatusCmd(client *slack.Client, statePath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		profile, err := client.GetUserProfileContext(ctx, &slack.GetUserProfileParameters{})
		if err != nil {
			return calSyncErrMsg{Err: fmt.Errorf("status sichern: %w", err), IsFatal: false}
		}

		snap := savedStatus{
			Text:           profile.StatusText,
			Emoji:          profile.StatusEmoji,
			ExpirationUnix: int64(profile.StatusExpiration),
			SavedAt:        time.Now().Unix(),
		}
		logCal("Aktuellen Status gesichert: text=%q emoji=%q exp=%d", snap.Text, snap.Emoji, snap.ExpirationUnix)

		data, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			return calSyncErrMsg{Err: fmt.Errorf("status sichern: marshal: %w", err), IsFatal: false}
		}
		if err := os.WriteFile(statePath, data, 0o644); err != nil {
			return calSyncErrMsg{Err: fmt.Errorf("status sichern: schreiben: %w", err), IsFatal: false}
		}
		return calStatusSavedMsg{Snapshot: snap}
	}
}

func restorePreviousStatusCmd(client *slack.Client, statePath string) tea.Cmd {
	return func() tea.Msg {
		snap, err := loadSavedStatus(statePath)
		if err != nil {
			return calSyncErrMsg{Err: fmt.Errorf("status wiederherstellen: %w", err), IsFatal: false}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.SetUserCustomStatusContext(ctx, snap.Text, snap.Emoji, snap.ExpirationUnix); err != nil {
			return calSyncErrMsg{Err: fmt.Errorf("status wiederherstellen: %w", err), IsFatal: false}
		}

		_ = os.Remove(statePath)
		logCal("Vorherigen Status wiederhergestellt: text=%q emoji=%q", snap.Text, snap.Emoji)
		return calStatusRestoredMsg{PreviousText: snap.Text}
	}
}

func setMeetingStatusCmd(client *slack.Client, cfg calSyncConfig, event calEvent) tea.Cmd {
	return func() tea.Msg {
		text := cfg.DefaultText
		if cfg.UseEventTitle && event.Subject != "" {
			text = event.Subject
		}
		emoji := cfg.DefaultEmoji
		expiration := event.EndTime.Unix()

		logCal("Setze Meeting-Status: text=%q emoji=%q bis=%s", text, emoji, event.EndTime.Local().Format("15:04"))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.SetUserCustomStatusContext(ctx, text, emoji, expiration); err != nil {
			return calSyncErrMsg{Err: fmt.Errorf("meeting-status setzen: %w", err), IsFatal: false}
		}
		return calStatusSetMsg{
			EventID:     event.ID,
			EventEnd:    event.EndTime,
			StatusText:  text,
			StatusEmoji: emoji,
		}
	}
}

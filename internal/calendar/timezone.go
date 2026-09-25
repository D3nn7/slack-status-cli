package calendar

import "time"

// windowsTZMap maps common Exchange/Windows timezone names to IANA names, as
// ICS feeds from Outlook often use the Windows identifiers.
var windowsTZMap = map[string]string{
	"W. Europe Standard Time":        "Europe/Berlin",
	"Central Europe Standard Time":   "Europe/Budapest",
	"Romance Standard Time":          "Europe/Paris",
	"GMT Standard Time":              "Europe/London",
	"Eastern Standard Time":          "America/New_York",
	"Central Standard Time":          "America/Chicago",
	"Mountain Standard Time":         "America/Denver",
	"Pacific Standard Time":          "America/Los_Angeles",
	"UTC":                            "UTC",
	"Greenwich Standard Time":        "Atlantic/Reykjavik",
	"Russian Standard Time":          "Europe/Moscow",
	"China Standard Time":            "Asia/Shanghai",
	"Tokyo Standard Time":            "Asia/Tokyo",
	"AUS Eastern Standard Time":      "Australia/Sydney",
	"E. Europe Standard Time":        "Europe/Nicosia",
	"Turkey Standard Time":           "Europe/Istanbul",
	"Israel Standard Time":           "Asia/Jerusalem",
	"Arab Standard Time":             "Asia/Riyadh",
	"India Standard Time":            "Asia/Calcutta",
	"SE Asia Standard Time":          "Asia/Bangkok",
	"Korea Standard Time":            "Asia/Seoul",
	"New Zealand Standard Time":      "Pacific/Auckland",
	"Central America Standard Time":  "America/Guatemala",
	"SA Eastern Standard Time":       "America/Cayenne",
	"E. South America Standard Time": "America/Sao_Paulo",
}

// ResolveTimezone turns a TZID from an ICS property into a *time.Location,
// falling back to the local timezone for unknown values.
func ResolveTimezone(tzid string) *time.Location {
	if tzid == "" {
		return time.Local
	}
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc
	}
	if iana, ok := windowsTZMap[tzid]; ok {
		if loc, err := time.LoadLocation(iana); err == nil {
			return loc
		}
	}
	return time.Local
}

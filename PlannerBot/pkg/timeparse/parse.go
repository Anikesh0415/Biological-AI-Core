package timeparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parse converts a natural language time expression into an absolute time.Time in UTC.
// It uses `now` as the reference point for relative expressions.
//
// Supported patterns:
//   - "7pm", "7:00pm", "19:00" → today at that time (or tomorrow if already past)
//   - "tomorrow at 5pm" → next day at 5pm
//   - "today at 6pm" → today at 6pm
//   - "in 15 minutes", "in 2 hours" → relative offset from now
//   - "Monday at 9am" → next Monday at 9am
//   - "next Monday at 10:30am" → next Monday at 10:30am
//   - "October 24th at 4pm" → that date at 4pm
//   - "day after tomorrow at 9am" → 2 days from now at 9am
func Parse(expr string, now time.Time) (time.Time, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return time.Time{}, fmt.Errorf("timeparse: empty expression")
	}

	lower := strings.ToLower(expr)

	// Pattern 1: Relative offsets ("in X minutes/hours")
	if t, ok := parseRelative(lower, now); ok {
		return t, nil
	}

	// Pattern 2: "day after tomorrow at ..."
	if t, ok := parseDayAfterTomorrow(lower, now); ok {
		return t, nil
	}

	// Pattern 3: "tomorrow at ..."
	if t, ok := parseTomorrow(lower, now); ok {
		return t, nil
	}

	// Pattern 4: "today at ..."
	if t, ok := parseToday(lower, now); ok {
		return t, nil
	}

	// Pattern 5: Day names ("Monday at 9am", "next Thursday at 2pm")
	if t, ok := parseDayName(lower, now); ok {
		return t, nil
	}

	// Pattern 6: Month + day ("October 24th at 4pm", "November 15 at 8:30am")
	if t, ok := parseMonthDay(lower, now); ok {
		return t, nil
	}

	// Pattern 7: Bare clock time ("7pm", "7:30pm", "19:00", "8pm tonight")
	if t, ok := parseBareTime(lower, now); ok {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("timeparse: could not parse %q", expr)
}

// --- Relative Offsets ---

var relativeRegex = regexp.MustCompile(`(?:in\s+)?(\d+)\s*(minute|minutes|min|mins|hour|hours|hr|hrs|second|seconds|sec|secs|day|days)`)

func parseRelative(expr string, now time.Time) (time.Time, bool) {
	m := relativeRegex.FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, false
	}

	amount, _ := strconv.Atoi(m[1])
	unit := m[2]

	switch {
	case strings.HasPrefix(unit, "minute") || strings.HasPrefix(unit, "min"):
		return now.Add(time.Duration(amount) * time.Minute), true
	case strings.HasPrefix(unit, "hour") || strings.HasPrefix(unit, "hr"):
		return now.Add(time.Duration(amount) * time.Hour), true
	case strings.HasPrefix(unit, "second") || strings.HasPrefix(unit, "sec"):
		return now.Add(time.Duration(amount) * time.Second), true
	case strings.HasPrefix(unit, "day"):
		return now.Add(time.Duration(amount) * 24 * time.Hour), true
	}

	return time.Time{}, false
}

// --- Day After Tomorrow ---

func parseDayAfterTomorrow(expr string, now time.Time) (time.Time, bool) {
	if !strings.Contains(expr, "day after tomorrow") {
		return time.Time{}, false
	}

	dayAfter := now.AddDate(0, 0, 2)
	clockTime := extractClockTime(expr)
	if clockTime == nil {
		return time.Time{}, false
	}

	return combineDateAndTime(dayAfter, *clockTime), true
}

// --- Tomorrow ---

func parseTomorrow(expr string, now time.Time) (time.Time, bool) {
	if !strings.Contains(expr, "tomorrow") {
		return time.Time{}, false
	}

	tomorrow := now.AddDate(0, 0, 1)
	clockTime := extractClockTime(expr)
	if clockTime == nil {
		// Default to 9:00 AM if no time specified
		return combineDateAndTime(tomorrow, clockInfo{hour: 9, minute: 0}), true
	}

	return combineDateAndTime(tomorrow, *clockTime), true
}

// --- Today ---

func parseToday(expr string, now time.Time) (time.Time, bool) {
	if !strings.Contains(expr, "today") {
		return time.Time{}, false
	}

	clockTime := extractClockTime(expr)
	if clockTime == nil {
		return time.Time{}, false
	}

	return combineDateAndTime(now, *clockTime), true
}

// --- Day Names ---

var dayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"thursdays": time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

var dayNameRegex = regexp.MustCompile(`(?:next\s+|on\s+)?(sunday|monday|tuesday|wednesday|thursday|thursdays|friday|saturday)`)

func parseDayName(expr string, now time.Time) (time.Time, bool) {
	m := dayNameRegex.FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, false
	}

	targetDay, ok := dayNames[m[1]]
	if !ok {
		return time.Time{}, false
	}

	// Calculate next occurrence of that day
	daysUntil := int(targetDay) - int(now.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7
	}
	targetDate := now.AddDate(0, 0, daysUntil)

	clockTime := extractClockTime(expr)
	if clockTime == nil {
		return combineDateAndTime(targetDate, clockInfo{hour: 9, minute: 0}), true
	}

	return combineDateAndTime(targetDate, *clockTime), true
}

// --- Month + Day ---

var months = map[string]time.Month{
	"january": time.January, "february": time.February, "march": time.March,
	"april": time.April, "may": time.May, "june": time.June,
	"july": time.July, "august": time.August, "september": time.September,
	"october": time.October, "november": time.November, "december": time.December,
}

var monthDayRegex = regexp.MustCompile(`(january|february|march|april|may|june|july|august|september|october|november|december)\s+(\d{1,2})(?:st|nd|rd|th)?`)

func parseMonthDay(expr string, now time.Time) (time.Time, bool) {
	m := monthDayRegex.FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, false
	}

	month, ok := months[m[1]]
	if !ok {
		return time.Time{}, false
	}

	day, _ := strconv.Atoi(m[2])
	year := now.Year()

	// If the date is in the past, use next year
	targetDate := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	if targetDate.Before(now) {
		targetDate = targetDate.AddDate(1, 0, 0)
	}

	clockTime := extractClockTime(expr)
	if clockTime == nil {
		return combineDateAndTime(targetDate, clockInfo{hour: 9, minute: 0}), true
	}

	return combineDateAndTime(targetDate, *clockTime), true
}

// --- Bare Clock Time ---

func parseBareTime(expr string, now time.Time) (time.Time, bool) {
	clockTime := extractClockTime(expr)
	if clockTime == nil {
		return time.Time{}, false
	}

	result := combineDateAndTime(now, *clockTime)

	// If the time is already past, schedule for tomorrow
	if result.Before(now) {
		result = result.AddDate(0, 0, 1)
	}

	return result, true
}

// --- Clock Time Extraction Helpers ---

type clockInfo struct {
	hour   int
	minute int
}

// extractClockTime finds a clock time pattern in the string.
// Supported: "7pm", "7:30pm", "7:00 AM", "19:00", "10:30am", "12:00pm"
var clockRegex12h = regexp.MustCompile(`(\d{1,2})(?::(\d{2}))?\s*(am|pm|AM|PM)`)
var clockRegex24h = regexp.MustCompile(`(\d{1,2}):(\d{2})(?:\s|$)`)

func extractClockTime(expr string) *clockInfo {
	// Try 12-hour format first (more common in natural language)
	if m := clockRegex12h.FindStringSubmatch(expr); m != nil {
		hour, _ := strconv.Atoi(m[1])
		minute := 0
		if m[2] != "" {
			minute, _ = strconv.Atoi(m[2])
		}

		ampm := strings.ToLower(m[3])
		if ampm == "pm" && hour != 12 {
			hour += 12
		} else if ampm == "am" && hour == 12 {
			hour = 0
		}

		return &clockInfo{hour: hour, minute: minute}
	}

	// Try 24-hour format
	if m := clockRegex24h.FindStringSubmatch(expr); m != nil {
		hour, _ := strconv.Atoi(m[1])
		minute, _ := strconv.Atoi(m[2])
		if hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59 {
			return &clockInfo{hour: hour, minute: minute}
		}
	}

	return nil
}

// combineDateAndTime creates a time.Time using the date portion from `date` and the time from `clock`.
func combineDateAndTime(date time.Time, clock clockInfo) time.Time {
	return time.Date(
		date.Year(), date.Month(), date.Day(),
		clock.hour, clock.minute, 0, 0,
		date.Location(),
	)
}

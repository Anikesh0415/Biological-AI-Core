package timeparse

import (
	"testing"
	"time"
)

// ref is a fixed reference time: Wednesday, October 15, 2025 at 2:30 PM UTC
var ref = time.Date(2025, time.October, 15, 14, 30, 0, 0, time.UTC)

func TestParseEmpty(t *testing.T) {
	_, err := Parse("", ref)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseRelativeMinutes(t *testing.T) {
	tests := []struct {
		input   string
		minutes int
	}{
		{"in 15 minutes", 15},
		{"in 30 mins", 30},
		{"in 1 minute", 1},
		{"in 5 min", 5},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input, ref)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			expected := ref.Add(time.Duration(tt.minutes) * time.Minute)
			if !got.Equal(expected) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, expected)
			}
		})
	}
}

func TestParseRelativeHours(t *testing.T) {
	tests := []struct {
		input string
		hours int
	}{
		{"in 2 hours", 2},
		{"in 1 hour", 1},
		{"in 3 hrs", 3},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input, ref)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			expected := ref.Add(time.Duration(tt.hours) * time.Hour)
			if !got.Equal(expected) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, expected)
			}
		})
	}
}

func TestParseBareTime(t *testing.T) {
	// ref is 2:30 PM, so "7pm" should be today at 7pm (still in the future)
	got, err := Parse("7pm", ref)
	if err != nil {
		t.Fatalf("Parse('7pm') error: %v", err)
	}
	expected := time.Date(2025, time.October, 15, 19, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("Parse('7pm') = %v, want %v", got, expected)
	}
}

func TestParseBareTimePast(t *testing.T) {
	// ref is 2:30 PM, so "7am" should be TOMORROW at 7am (already past today)
	got, err := Parse("7am", ref)
	if err != nil {
		t.Fatalf("Parse('7am') error: %v", err)
	}
	expected := time.Date(2025, time.October, 16, 7, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("Parse('7am') = %v, want %v", got, expected)
	}
}

func TestParseBareTimeWithMinutes(t *testing.T) {
	got, err := Parse("7:30pm", ref)
	if err != nil {
		t.Fatalf("Parse('7:30pm') error: %v", err)
	}
	expected := time.Date(2025, time.October, 15, 19, 30, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("Parse('7:30pm') = %v, want %v", got, expected)
	}
}

func TestParseTomorrow(t *testing.T) {
	got, err := Parse("tomorrow at 5pm", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 16, 17, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseTomorrowNoTime(t *testing.T) {
	got, err := Parse("tomorrow", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// Should default to 9:00 AM
	expected := time.Date(2025, time.October, 16, 9, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseToday(t *testing.T) {
	got, err := Parse("today at 6pm", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 15, 18, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseDayAfterTomorrow(t *testing.T) {
	got, err := Parse("day after tomorrow at 9am", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 17, 9, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseDayName(t *testing.T) {
	// ref is Wednesday Oct 15, so "Monday" should be Oct 20
	got, err := Parse("Monday at 9am", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 20, 9, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseNextDayName(t *testing.T) {
	got, err := Parse("next Friday at 2pm", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// Friday is 2 days after Wednesday
	expected := time.Date(2025, time.October, 17, 14, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseMonthDay(t *testing.T) {
	got, err := Parse("October 24th at 4pm", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 24, 16, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseMonthDayFuture(t *testing.T) {
	got, err := Parse("November 15 at 8:30am", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.November, 15, 8, 30, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseNoon(t *testing.T) {
	got, err := Parse("12:00pm", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// 12:00pm is noon, which is past 2:30pm so should be tomorrow
	expected := time.Date(2025, time.October, 16, 12, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseMidnight(t *testing.T) {
	got, err := Parse("12:00am", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// 12:00am (midnight) is past for today, so should be tomorrow at midnight
	expected := time.Date(2025, time.October, 16, 0, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParse24HourFormat(t *testing.T) {
	got, err := Parse("19:00", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 15, 19, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseUnrecognized(t *testing.T) {
	_, err := Parse("sometime next year maybe", ref)
	if err == nil {
		t.Fatal("expected error for unrecognized expression")
	}
}

func TestParse8pmTonight(t *testing.T) {
	got, err := Parse("8pm tonight", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 15, 20, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseFridayBy5pm(t *testing.T) {
	got, err := Parse("Friday 5pm", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := time.Date(2025, time.October, 17, 17, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseThursdaysAt2pm(t *testing.T) {
	got, err := Parse("Thursdays at 2pm", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// Thursday is the next day after Wednesday
	expected := time.Date(2025, time.October, 16, 14, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestParseIn45Mins(t *testing.T) {
	got, err := Parse("in 45 mins", ref)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	expected := ref.Add(45 * time.Minute)
	if !got.Equal(expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

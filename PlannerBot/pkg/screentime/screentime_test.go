package screentime

import (
	"testing"
	"time"
)

func TestParseScreentimeText_iOSFormat(t *testing.T) {
	rawiOS := `
Screen Time
Daily Average: 6h 15m
Pickups: 92
Notifications: 140

Instagram
2h 30m

VS Code
1h 45m

YouTube
1h 10m

Slack
30m

WhatsApp
20m
`
	report := ParseScreentimeText(rawiOS)
	if report.TotalTime != 6*time.Hour+15*time.Minute {
		t.Errorf("expected total time 6h15m, got %v (%s)", report.TotalTime, report.TotalTimeStr)
	}

	if len(report.Apps) != 5 {
		t.Fatalf("expected 5 parsed apps, got %d", len(report.Apps))
	}

	if report.Pickups != 92 {
		t.Errorf("expected 92 pickups, got %d", report.Pickups)
	}
	if report.Notifications != 140 {
		t.Errorf("expected 140 notifications, got %d", report.Notifications)
	}

	// Instagram should be #1
	if report.Apps[0].App != "Instagram" {
		t.Errorf("expected top app Instagram, got %s", report.Apps[0].App)
	}
	if !report.Apps[0].IsDopamineTrap {
		t.Errorf("expected Instagram to be classified as dopamine trap")
	}

	if report.DopamineTrapScore <= 0 {
		t.Errorf("expected positive dopamine trap score, got %d", report.DopamineTrapScore)
	}

	if report.AnnualLostHours <= 0 {
		t.Errorf("expected projected annual lost hours > 0, got %d", report.AnnualLostHours)
	}
}

func TestParseScreentimeText_AndroidFormat(t *testing.T) {
	rawAndroid := `
Digital Wellbeing
Today: 5h 20m

TikTok - 3h 10m
Notion - 1h 15m
Reddit - 40m
Chrome - 15m
`
	report := ParseScreentimeText(rawAndroid)
	if report.TotalTime != 5*time.Hour+20*time.Minute {
		t.Errorf("expected total time 5h20m, got %v (%s)", report.TotalTime, report.TotalTimeStr)
	}

	if len(report.Apps) < 3 {
		t.Errorf("expected at least 3 apps parsed, got %d", len(report.Apps))
	}

	if report.Apps[0].App != "TikTok" {
		t.Errorf("expected top app TikTok, got %s", report.Apps[0].App)
	}
	if report.Apps[0].Category != CategoryDopamineSink {
		t.Errorf("expected TikTok to be Dopamine Trap, got %s", report.Apps[0].Category)
	}
}

func TestParseScreentimeText_SumFallback(t *testing.T) {
	rawNoTotal := `
YouTube 1h 20m
VSCode 2h 00m
Instagram 45m
`
	report := ParseScreentimeText(rawNoTotal)
	expectedTotal := 1*time.Hour + 20*time.Minute + 2*time.Hour + 45*time.Minute
	if report.TotalTime != expectedTotal {
		t.Errorf("expected total sum %v, got %v", expectedTotal, report.TotalTime)
	}

	// VSCode should be sorted to top (2h)
	if report.Apps[0].App != "VSCode" {
		t.Errorf("expected VSCode as top app, got %s", report.Apps[0].App)
	}
	if report.Apps[0].Category != CategoryProductive {
		t.Errorf("expected VSCode to be Productive, got %s", report.Apps[0].Category)
	}
}

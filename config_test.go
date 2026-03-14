package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseCSVList(t *testing.T) {
	repos, err := parseCSVList("cli/cli, cli/cli , openai/openai-go")
	if err != nil {
		t.Fatalf("parseCSVList returned error: %v", err)
	}
	want := []string{"cli/cli", "openai/openai-go"}
	if !reflect.DeepEqual(repos, want) {
		t.Fatalf("repos mismatch\nwant: %#v\ngot:  %#v", want, repos)
	}
}

func TestMonthsInRange(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC)
	months := monthsInRange(start, end)
	if len(months) != 3 {
		t.Fatalf("expected 3 months, got %d", len(months))
	}
	if months[0].Label != "2025-01" || months[2].Label != "2025-03" {
		t.Fatalf("unexpected months: %#v", months)
	}
}

func TestClipMonth(t *testing.T) {
	cfg := Config{
		TimeFrom: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		TimeTo:   time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC),
	}
	month := MonthRange{Label: "2025-02", Start: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)}
	start, end := clipMonth(month, cfg)
	if got, want := start.Format(time.DateOnly), "2025-02-01"; got != want {
		t.Fatalf("start mismatch: want %s got %s", want, got)
	}
	if got, want := end.Format(time.DateOnly), "2025-02-20"; got != want {
		t.Fatalf("end mismatch: want %s got %s", want, got)
	}
}

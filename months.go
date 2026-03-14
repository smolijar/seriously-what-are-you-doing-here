package main

import (
	"fmt"
	"time"
)

type MonthRange struct {
	Label string
	Start time.Time
	End   time.Time
}

func parseMonth(value string) (MonthRange, error) {
	start, err := time.ParseInLocation("2006-01", value, time.UTC)
	if err != nil {
		return MonthRange{}, fmt.Errorf("invalid month %q, expected YYYY-MM", value)
	}
	return MonthRange{Label: start.Format("2006-01"), Start: start, End: start.AddDate(0, 1, 0)}, nil
}

func monthsInRange(start, end time.Time) []MonthRange {
	if !start.Before(end) {
		return nil
	}
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	months := make([]MonthRange, 0)
	for current.Before(end) {
		next := current.AddDate(0, 1, 0)
		months = append(months, MonthRange{Label: current.Format("2006-01"), Start: current, End: next})
		current = next
	}
	return months
}

func clipMonth(month MonthRange, cfg Config) (time.Time, time.Time) {
	start := month.Start
	if cfg.TimeFrom.After(start) {
		start = cfg.TimeFrom
	}
	end := month.End
	if cfg.TimeTo.Before(end) {
		end = cfg.TimeTo
	}
	return start, end
}

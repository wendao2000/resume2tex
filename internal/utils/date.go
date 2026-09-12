package utils

import (
	"fmt"
	"strings"
	"time"
)

// ParseDate returns the inclusive interval represented by a year, month, or day.
func ParseDate(value string) (earliest time.Time, latest time.Time, err error) {
	var layout string
	switch len(value) {
	case len("2006"):
		layout = "2006"
	case len("2006-01"):
		layout = "2006-01"
	case len("2006-01-02"):
		layout = "2006-01-02"
	}
	invalid := func() (time.Time, time.Time, error) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date %q: expected YYYY, YYYY-MM, or YYYY-MM-DD with a valid calendar date", value)
	}
	if layout == "" {
		return invalid()
	}
	earliest, err = time.Parse(layout, value)
	if err != nil || earliest.Year() < 1 || earliest.Format(layout) != value {
		return invalid()
	}
	switch layout {
	case "2006":
		latest = earliest.AddDate(1, 0, 0)
	case "2006-01":
		latest = earliest.AddDate(0, 1, 0)
	default:
		latest = earliest.AddDate(0, 0, 1)
	}
	return earliest, latest.Add(-time.Nanosecond), nil
}

// FormatDate turns a year, month, or day into display text with the same precision.
// Blank input produces an empty string.
func FormatDate(value string) (string, error) {
	return formatDate(value, len("2006-01-02"))
}

func formatDate(value string, precision int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	date, _, err := ParseDate(value)
	if err != nil {
		return "", err
	}
	switch min(len(value), precision) {
	case 4:
		return date.Format("2006"), nil
	case 7:
		return date.Format("January 2006"), nil
	default:
		return date.Format("January 02, 2006"), nil
	}
}

// DateRange formats start and end dates, using Present for a missing end only when
// ongoing is true and a start date is present.
func DateRange(start, end string, ongoing bool) (string, error) {
	return formatDateRange(start, end, ongoing, len("2006-01-02"))
}

// MonthYearRange formats a range with at most month and year precision.
// Year-only dates remain year-only, and ongoing ranges follow DateRange rules.
func MonthYearRange(start, end string, ongoing bool) (string, error) {
	return formatDateRange(start, end, ongoing, len("2006-01"))
}

// YearRange formats a range as years, with the same ongoing rules as DateRange.
func YearRange(start, end string, ongoing bool) (string, error) {
	return formatDateRange(start, end, ongoing, len("2006"))
}

func formatDateRange(start, end string, ongoing bool, precision int) (string, error) {
	first, err := formatDate(start, precision)
	if err != nil {
		return "", err
	}
	last, err := formatDate(end, precision)
	if err != nil {
		return "", err
	}
	if first == "" {
		return last, nil
	}
	if last == "" {
		if !ongoing {
			return first, nil
		}
		last = "Present"
	}
	return first + " - " + last, nil
}

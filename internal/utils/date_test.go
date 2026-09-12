package utils

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		value string
		first string
		last  string
	}{
		{"2024", "2024-01-01", "2024-12-31"},
		{"2024-02", "2024-02-01", "2024-02-29"},
		{"2023-02", "2023-02-01", "2023-02-28"},
		{"2024-02-29", "2024-02-29", "2024-02-29"},
		{"0001", "0001-01-01", "0001-12-31"},
		{"9999", "9999-01-01", "9999-12-31"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			first, last, err := ParseDate(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			if got := first.Format(time.DateOnly); got != tt.first {
				t.Errorf("first = %s, want %s", got, tt.first)
			}
			if got := last.Format(time.DateOnly); got != tt.last {
				t.Errorf("last = %s, want %s", got, tt.last)
			}
			if first.Hour() != 0 || last.Hour() != 23 || last.Nanosecond() != 999999999 {
				t.Errorf("expected inclusive full-day bounds, got %s to %s", first, last)
			}
		})
	}
	for _, value := range []string{"", "0000", "202", "20240", "2024-1", "2024-00", "2024-13", "2024-1-01", "2024-01-1", "2024-01-00", "2024-02-30", "2023-02-29", "2024-04-31", "2024-01-01T00:00:00Z", "Present", " 2024"} {
		t.Run("invalid "+value, func(t *testing.T) {
			if _, _, err := ParseDate(value); err == nil {
				t.Errorf("invalid date %q accepted", value)
			}
		})
	}
}

func TestRangePrecision(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		end       string
		ongoing   bool
		source    string
		monthYear string
		year      string
	}{
		{"full dates", "2020-02-29", "2024-04-30", false, "February 29, 2020 - April 30, 2024", "February 2020 - April 2024", "2020 - 2024"},
		{"month precision", "2020-02", "2024-04", false, "February 2020 - April 2024", "February 2020 - April 2024", "2020 - 2024"},
		{"year precision", "2020", "2024", false, "2020 - 2024", "2020 - 2024", "2020 - 2024"},
		{"mixed precision", "2020", "2024-04-30", false, "2020 - April 30, 2024", "2020 - April 2024", "2020 - 2024"},
		{"ongoing", "2020-02-29", "", true, "February 29, 2020 - Present", "February 2020 - Present", "2020 - Present"},
		{"ongoing year", "2020", "", true, "2020 - Present", "2020 - Present", "2020 - Present"},
		{"unknown end", "2020-02-29", "", false, "February 29, 2020", "February 2020", "2020"},
		{"unknown start", "", "2024-04-30", true, "April 30, 2024", "April 2024", "2024"},
		{"explicit end", "2020-02-29", "2024-04-30", true, "February 29, 2020 - April 30, 2024", "February 2020 - April 2024", "2020 - 2024"},
		{"unknown dates", "", "", true, "", "", ""},
		{"whitespace", " 2020-02-29 ", "  ", true, "February 29, 2020 - Present", "February 2020 - Present", "2020 - Present"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, formatter := range []struct {
				name string
				fn   func(string, string, bool) (string, error)
				want string
			}{
				{"source", DateRange, tt.source},
				{"month and year", MonthYearRange, tt.monthYear},
				{"year", YearRange, tt.year},
			} {
				t.Run(formatter.name, func(t *testing.T) {
					got, err := formatter.fn(tt.start, tt.end, tt.ongoing)
					if err != nil {
						t.Fatal(err)
					}
					if got != formatter.want {
						t.Errorf("got %q, want %q", got, formatter.want)
					}
				})
			}
		})
	}
}

func TestRangePrecisionRejectsInvalidDates(t *testing.T) {
	for name, formatter := range map[string]func(string, string, bool) (string, error){
		"source": DateRange, "month and year": MonthYearRange, "year": YearRange,
	} {
		t.Run(name, func(t *testing.T) {
			for _, invalid := range []string{"2023-02-29", "2024-04-31", "2024-13", "0000", "Present"} {
				if _, err := formatter(invalid, "2024", false); err == nil {
					t.Errorf("invalid start %q accepted", invalid)
				}
				if _, err := formatter("2020", invalid, true); err == nil {
					t.Errorf("invalid end %q accepted", invalid)
				}
			}
		})
	}
}

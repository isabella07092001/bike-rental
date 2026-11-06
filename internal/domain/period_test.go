package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)

// at returns a time on 2026-11-07 (the day after testNow) at hh:mm.
func at(hh, mm int) time.Time {
	return time.Date(2026, 11, 7, hh, mm, 0, 0, time.UTC)
}

func TestPeriodValidate(t *testing.T) {
	start := at(10, 0)
	tests := []struct {
		name    string
		period  Period
		wantErr string // "" means valid
	}{
		{"30 min is the minimum", Period{start, start.Add(30 * time.Minute)}, ""},
		{"29 min is too short", Period{start, start.Add(29 * time.Minute)}, "at least 30 minutes"},
		{"29 min 59 s is too short", Period{start, start.Add(30*time.Minute - time.Second)}, "at least 30 minutes"},
		{"48 h is the maximum", Period{start, start.Add(48 * time.Hour)}, ""},
		{"48 h 1 min is too long", Period{start, start.Add(48*time.Hour + time.Minute)}, "at most 48 hours"},
		{"end before start", Period{start, start.Add(-time.Hour)}, "end must be after start"},
		{"end equals start", Period{start, start}, "end must be after start"},
		{"missing start", Period{time.Time{}, start}, "required"},
		{"missing end", Period{start, time.Time{}}, "required"},
		{"start in the past", Period{testNow.Add(-time.Hour), testNow.Add(time.Hour)}, "in the future"},
		{"start exactly now", Period{testNow, testNow.Add(time.Hour)}, "in the future"},
		{"start one second from now", Period{testNow.Add(time.Second), testNow.Add(time.Hour)}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.period.Validate(testNow)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want ErrInvalid containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestPeriodBilledHours(t *testing.T) {
	start := at(10, 0)
	tests := []struct {
		duration time.Duration
		want     int64
	}{
		{30 * time.Minute, 1},
		{59 * time.Minute, 1},
		{60 * time.Minute, 1},
		{61 * time.Minute, 2},
		{time.Hour + time.Second, 2},
		{90 * time.Minute, 2},
		{150 * time.Minute, 3},
		{48 * time.Hour, 48},
		{0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			if got := (Period{start, start.Add(tt.duration)}).BilledHours(); got != tt.want {
				t.Errorf("BilledHours() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPeriodConflictsWith(t *testing.T) {
	existing := Period{at(10, 0), at(12, 0)}
	tests := []struct {
		name string
		p    Period
		want bool
	}{
		{"exactly 30 min after", Period{at(12, 30), at(14, 0)}, false},
		{"29 min after", Period{at(12, 29), at(14, 0)}, true},
		{"directly after", Period{at(12, 0), at(14, 0)}, true},
		{"exactly 30 min before", Period{at(8, 0), at(9, 30)}, false},
		{"29 min before", Period{at(8, 0), at(9, 31)}, true},
		{"overlapping", Period{at(11, 0), at(13, 0)}, true},
		{"inside", Period{at(10, 30), at(11, 0)}, true},
		{"surrounding", Period{at(9, 0), at(13, 0)}, true},
		{"far away", Period{at(16, 0), at(18, 0)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.ConflictsWith(existing); got != tt.want {
				t.Errorf("ConflictsWith() = %v, want %v", got, tt.want)
			}
			if got := existing.ConflictsWith(tt.p); got != tt.want {
				t.Errorf("ConflictsWith() is not symmetric: %v", got)
			}
		})
	}
}

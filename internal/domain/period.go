package domain

import (
	"fmt"
	"time"
)

// Rental rules from the stakeholders (see specs/reservation-management).
const (
	MinRentalDuration = 30 * time.Minute
	MaxRentalDuration = 48 * time.Hour
	// BookingBuffer is the minimum gap between two reservations of the same bike.
	BookingBuffer = 30 * time.Minute
)

// Period is a rental time span [Start, End).
type Period struct {
	Start time.Time
	End   time.Time
}

// Duration returns End - Start.
func (p Period) Duration() time.Duration {
	return p.End.Sub(p.Start)
}

// Validate checks the rental rules: end after start, 30 min ≤ duration ≤ 48 h,
// and start in the future. Errors wrap ErrInvalid.
func (p Period) Validate(now time.Time) error {
	switch {
	case p.Start.IsZero() || p.End.IsZero():
		return fmt.Errorf("%w: start and end are required", ErrInvalid)
	case !p.End.After(p.Start):
		return fmt.Errorf("%w: end must be after start", ErrInvalid)
	case p.Duration() < MinRentalDuration:
		return fmt.Errorf("%w: a rental must last at least 30 minutes", ErrInvalid)
	case p.Duration() > MaxRentalDuration:
		return fmt.Errorf("%w: a rental must last at most 48 hours", ErrInvalid)
	case !p.Start.After(now):
		return fmt.Errorf("%w: start must be in the future", ErrInvalid)
	}
	return nil
}

// BilledHours returns the duration rounded up to full hours ("per started hour").
func (p Period) BilledHours() int64 {
	d := p.Duration()
	if d <= 0 {
		return 0
	}
	hours := int64(d / time.Hour)
	if d%time.Hour != 0 {
		hours++
	}
	return hours
}

// ConflictsWith reports whether the two periods are closer than BookingBuffer
// (overlapping periods always conflict). A gap of exactly BookingBuffer is allowed.
func (p Period) ConflictsWith(o Period) bool {
	return p.Start.Before(o.End.Add(BookingBuffer)) && o.Start.Before(p.End.Add(BookingBuffer))
}

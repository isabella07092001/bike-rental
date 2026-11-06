package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	cityBike     = ReservedBike{BikeID: "b-1", Category: CategoryCity, HourlyPriceCents: 600}
	childrenBike = ReservedBike{BikeID: "b-2", Category: CategoryChildren, HourlyPriceCents: 400}
)

func newTestReservation(t *testing.T, bikes ...ReservedBike) Reservation {
	t.Helper()
	r, err := NewReservation("p-1", "L01X00T47", bikes, Period{at(10, 0), at(12, 30)}, testNow)
	if err != nil {
		t.Fatalf("NewReservation: %v", err)
	}
	return r
}

func TestNewReservationCalculatesPrice(t *testing.T) {
	r := newTestReservation(t, cityBike, childrenBike)

	if r.Status != ReservationBooked || !r.CreatedAt.Equal(testNow) {
		t.Errorf("status/createdAt = %q/%v", r.Status, r.CreatedAt)
	}
	// 2 h 30 min → 3 started hours × (6 € + 4 €) = 30 €
	if r.BilledHours != 3 || r.TotalPriceCents != 3000 {
		t.Errorf("billed %d h, %d cents; want 3 h, 3000 cents", r.BilledHours, r.TotalPriceCents)
	}
	if r.IDNumber != "L01X00T47" {
		t.Errorf("IDNumber = %q", r.IDNumber)
	}
}

func TestCalculatePrice(t *testing.T) {
	tests := []struct {
		hours int64
		bikes []ReservedBike
		want  int64
	}{
		{1, []ReservedBike{cityBike}, 600},
		{2, []ReservedBike{cityBike}, 1200},
		{1, []ReservedBike{cityBike, childrenBike}, 1000},
		{48, []ReservedBike{cityBike, childrenBike}, 48000},
		{3, nil, 0},
	}
	for _, tt := range tests {
		if got := CalculatePrice(tt.hours, tt.bikes); got != tt.want {
			t.Errorf("CalculatePrice(%d, %d bikes) = %d, want %d", tt.hours, len(tt.bikes), got, tt.want)
		}
	}
}

func TestNewReservationValidation(t *testing.T) {
	valid := Period{at(10, 0), at(12, 0)}
	tests := []struct {
		name     string
		personID string
		idNumber string
		bikes    []ReservedBike
		period   Period
		wantErr  string
	}{
		{"no bikes", "p-1", "X", nil, valid, "at least one bike"},
		{"duplicate bike", "p-1", "X", []ReservedBike{cityBike, cityBike}, valid, "selected twice"},
		{"missing ID number", "p-1", "", []ReservedBike{cityBike}, valid, "identification number"},
		{"missing person", "", "X", []ReservedBike{cityBike}, valid, "person"},
		{"too short", "p-1", "X", []ReservedBike{cityBike}, Period{at(10, 0), at(10, 29)}, "at least 30 minutes"},
		{"in the past", "p-1", "X", []ReservedBike{cityBike}, Period{testNow.Add(-2 * time.Hour), testNow.Add(-time.Hour)}, "in the future"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReservation(tt.personID, tt.idNumber, tt.bikes, tt.period, testNow)
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want ErrInvalid containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCancelDeadline(t *testing.T) {
	r := newTestReservation(t, cityBike) // starts at(10,0) → deadline 24 h earlier
	deadline := r.Start.Add(-24 * time.Hour)

	tests := []struct {
		name    string
		now     time.Time
		wantErr bool
	}{
		{"25 h before start", r.Start.Add(-25 * time.Hour), false},
		{"exactly 24 h before start", deadline, false},
		{"24 h minus 1 s before start", deadline.Add(time.Second), true},
		{"1 h before start", r.Start.Add(-time.Hour), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := r
			err := c.Cancel(tt.now)
			if tt.wantErr {
				if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "24 hours") {
					t.Fatalf("Cancel() = %v, want ErrConflict about 24 hours", err)
				}
				if c.Status != ReservationBooked {
					t.Error("status changed despite error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Cancel() = %v", err)
			}
			if c.Status != ReservationCancelled || !c.CancelledAt.Equal(tt.now) {
				t.Errorf("after Cancel: status %q, cancelledAt %v", c.Status, c.CancelledAt)
			}
		})
	}
}

func TestCancelTwice(t *testing.T) {
	r := newTestReservation(t, cityBike)
	if err := r.Cancel(testNow); err != nil {
		t.Fatal(err)
	}
	if err := r.Cancel(testNow); !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "already cancelled") {
		t.Fatalf("second Cancel() = %v", err)
	}
}

func TestReschedule(t *testing.T) {
	r := newTestReservation(t, cityBike)
	later := testNow.Add(time.Hour)

	// Change period and add a bike: 1 h 1 min → 2 h × 10 €.
	newPeriod := Period{at(14, 0), at(15, 1)}
	if err := r.Reschedule([]ReservedBike{cityBike, childrenBike}, newPeriod, later); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if !r.Start.Equal(newPeriod.Start) || len(r.Bikes) != 2 || r.BilledHours != 2 || r.TotalPriceCents != 2000 {
		t.Errorf("after Reschedule: %+v", r)
	}
	if !r.UpdatedAt.Equal(later) {
		t.Errorf("UpdatedAt = %v, want %v", r.UpdatedAt, later)
	}

	// Invalid new booking is rejected and leaves the reservation unchanged.
	before := r
	if err := r.Reschedule(nil, newPeriod, later); !errors.Is(err, ErrInvalid) {
		t.Errorf("Reschedule without bikes: err = %v", err)
	}
	if r.TotalPriceCents != before.TotalPriceCents || len(r.Bikes) != 2 {
		t.Error("failed Reschedule modified the reservation")
	}

	// After the deadline of the current start nothing can be changed.
	if err := r.Reschedule([]ReservedBike{cityBike}, Period{at(16, 0), at(17, 0)}, r.Start.Add(-23*time.Hour)); !errors.Is(err, ErrConflict) {
		t.Errorf("Reschedule after deadline: err = %v, want ErrConflict", err)
	}
}

func TestPhaseAndIsOpen(t *testing.T) {
	r := newTestReservation(t, cityBike)
	tests := []struct {
		name     string
		now      time.Time
		want     ReservationPhase
		wantOpen bool
	}{
		{"before start", r.Start.Add(-time.Second), PhaseUpcoming, true},
		{"at start", r.Start, PhaseActive, true},
		{"just before end", r.End.Add(-time.Second), PhaseActive, true},
		{"at end", r.End, PhaseCompleted, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Phase(tt.now); got != tt.want {
				t.Errorf("Phase() = %q, want %q", got, tt.want)
			}
			if got := r.IsOpen(tt.now); got != tt.wantOpen {
				t.Errorf("IsOpen() = %v, want %v", got, tt.wantOpen)
			}
		})
	}

	_ = r.Cancel(testNow)
	if r.Phase(testNow) != PhaseCancelled || r.IsOpen(testNow) {
		t.Errorf("cancelled: phase %q, open %v", r.Phase(testNow), r.IsOpen(testNow))
	}
}

func TestReservationHelpers(t *testing.T) {
	r := newTestReservation(t, cityBike, childrenBike)
	if !r.HasBike("b-2") || r.HasBike("b-9") {
		t.Error("HasBike wrong")
	}
	if got := strings.Join(r.BikeIDs(), ","); got != "b-1,b-2" {
		t.Errorf("BikeIDs() = %q", got)
	}
	if got := ReservedBikeFrom(Bike{ID: "b-7", Category: CategoryRacing}); got != (ReservedBike{"b-7", CategoryRacing, 1000}) {
		t.Errorf("ReservedBikeFrom = %+v", got)
	}
}

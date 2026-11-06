package domain

import (
	"fmt"
	"slices"
	"time"
)

// ChangeDeadline is how long before the start a reservation can still be cancelled or edited.
const ChangeDeadline = 24 * time.Hour

// ReservationStatus is the stored state of a reservation.
type ReservationStatus string

// Stored reservation states. Upcoming/active/completed are derived from time, see Phase.
const (
	ReservationBooked    ReservationStatus = "booked"
	ReservationCancelled ReservationStatus = "cancelled"
)

// ReservationPhase is the state of a reservation as shown to users, derived from time.
type ReservationPhase string

// Reservation phases.
const (
	PhaseUpcoming  ReservationPhase = "upcoming"
	PhaseActive    ReservationPhase = "active"
	PhaseCompleted ReservationPhase = "completed"
	PhaseCancelled ReservationPhase = "cancelled"
)

// Valid reports whether p is one of the known phases.
func (p ReservationPhase) Valid() bool {
	switch p {
	case PhaseUpcoming, PhaseActive, PhaseCompleted, PhaseCancelled:
		return true
	}
	return false
}

// ReservedBike is a bike in a reservation, with its category and price at booking time.
type ReservedBike struct {
	BikeID           string   `json:"bikeId"`
	Category         Category `json:"category"`
	HourlyPriceCents int64    `json:"hourlyPriceCents"`
}

// ReservedBikeFrom snapshots the bike's current category and hourly price.
func ReservedBikeFrom(b Bike) ReservedBike {
	return ReservedBike{BikeID: b.ID, Category: b.Category, HourlyPriceCents: b.Category.HourlyPrice()}
}

// Reservation books one or more bikes for one person and one period.
type Reservation struct {
	ID              string            `json:"id"`
	PersonID        string            `json:"personId"`
	IDNumber        string            `json:"idNumber"` // snapshot at booking time (Business Rule 1)
	Bikes           []ReservedBike    `json:"bikes"`
	Start           time.Time         `json:"start"`
	End             time.Time         `json:"end"`
	BilledHours     int64             `json:"billedHours"`
	TotalPriceCents int64             `json:"totalPriceCents"`
	Status          ReservationStatus `json:"status"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	CancelledAt     time.Time         `json:"cancelledAt,omitzero"`
}

// NewReservation creates a booked reservation after checking period and bikes.
// The person checks (active, ID copy) are done by the caller, who knows the person store.
func NewReservation(personID, idNumber string, bikes []ReservedBike, period Period, now time.Time) (Reservation, error) {
	if personID == "" || idNumber == "" {
		return Reservation{}, fmt.Errorf("%w: person and identification number are required", ErrInvalid)
	}
	r := Reservation{
		PersonID:  personID,
		IDNumber:  idNumber,
		Status:    ReservationBooked,
		CreatedAt: now,
	}
	if err := r.setBooking(bikes, period, now); err != nil {
		return Reservation{}, err
	}
	return r, nil
}

// Period returns the reservation's time span.
func (r Reservation) Period() Period {
	return Period{Start: r.Start, End: r.End}
}

// ChangeDeadline returns the last moment the reservation can be cancelled or edited.
func (r Reservation) ChangeDeadline() time.Time {
	return r.Start.Add(-ChangeDeadline)
}

// CanChange reports whether the reservation can still be cancelled or edited:
// it is booked and now is at most 24 hours before the start.
func (r Reservation) CanChange(now time.Time) bool {
	return r.Status == ReservationBooked && !now.After(r.ChangeDeadline())
}

// Cancel cancels the reservation if the 24-hour deadline has not passed.
func (r *Reservation) Cancel(now time.Time) error {
	if err := r.checkChangeable(now, "cancelled"); err != nil {
		return err
	}
	r.Status = ReservationCancelled
	r.CancelledAt = now
	r.UpdatedAt = now
	return nil
}

// CheckEditable returns an ErrConflict if the reservation can no longer be changed
// (cancelled, or less than 24 hours before its start).
func (r Reservation) CheckEditable(now time.Time) error {
	return r.checkChangeable(now, "changed")
}

// Reschedule changes period and bikes if the 24-hour deadline (of the current start)
// has not passed. The new booking must pass all rules again; the price is recalculated.
func (r *Reservation) Reschedule(bikes []ReservedBike, period Period, now time.Time) error {
	if err := r.CheckEditable(now); err != nil {
		return err
	}
	return r.setBooking(bikes, period, now)
}

// Phase derives the user-facing state from the stored status and the current time.
func (r Reservation) Phase(now time.Time) ReservationPhase {
	switch {
	case r.Status == ReservationCancelled:
		return PhaseCancelled
	case now.Before(r.Start):
		return PhaseUpcoming
	case now.Before(r.End):
		return PhaseActive
	default:
		return PhaseCompleted
	}
}

// IsOpen reports whether the reservation is booked and not yet over (upcoming or active).
func (r Reservation) IsOpen(now time.Time) bool {
	return r.Status == ReservationBooked && now.Before(r.End)
}

// HasBike reports whether the bike is part of the reservation.
func (r Reservation) HasBike(bikeID string) bool {
	return slices.ContainsFunc(r.Bikes, func(b ReservedBike) bool { return b.BikeID == bikeID })
}

// BikeIDs returns the IDs of all reserved bikes.
func (r Reservation) BikeIDs() []string {
	ids := make([]string, len(r.Bikes))
	for i, b := range r.Bikes {
		ids[i] = b.BikeID
	}
	return ids
}

// CalculatePrice returns billedHours × the sum of the bikes' hourly prices, in cents.
func CalculatePrice(billedHours int64, bikes []ReservedBike) int64 {
	var perHour int64
	for _, b := range bikes {
		perHour += b.HourlyPriceCents
	}
	return billedHours * perHour
}

func (r Reservation) checkChangeable(now time.Time, action string) error {
	if r.Status == ReservationCancelled {
		return fmt.Errorf("%w: reservation is already cancelled", ErrConflict)
	}
	if !r.CanChange(now) {
		return fmt.Errorf("%w: a reservation can only be %s until 24 hours before the start (deadline was %s)",
			ErrConflict, action, r.ChangeDeadline().Local().Format("02.01.2006 15:04"))
	}
	return nil
}

// setBooking validates and applies bikes and period and recalculates the price.
func (r *Reservation) setBooking(bikes []ReservedBike, period Period, now time.Time) error {
	if err := validateBikes(bikes); err != nil {
		return err
	}
	if err := period.Validate(now); err != nil {
		return err
	}
	r.Bikes = slices.Clone(bikes)
	r.Start = period.Start
	r.End = period.End
	r.BilledHours = period.BilledHours()
	r.TotalPriceCents = CalculatePrice(r.BilledHours, bikes)
	r.UpdatedAt = now
	return nil
}

func validateBikes(bikes []ReservedBike) error {
	if len(bikes) == 0 {
		return fmt.Errorf("%w: select at least one bike", ErrInvalid)
	}
	seen := make(map[string]bool, len(bikes))
	for _, b := range bikes {
		if seen[b.BikeID] {
			return fmt.Errorf("%w: bike %q is selected twice", ErrInvalid, b.BikeID)
		}
		seen[b.BikeID] = true
	}
	return nil
}

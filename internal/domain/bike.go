package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Status describes whether a bike can be rented.
type Status string

// Possible bike states.
const (
	StatusAvailable   Status = "available"
	StatusMaintenance Status = "maintenance"
	StatusRetired     Status = "retired"
)

// Valid reports whether s is one of the known states.
func (s Status) Valid() bool {
	switch s {
	case StatusAvailable, StatusMaintenance, StatusRetired:
		return true
	}
	return false
}

// Field length limits, counted in characters.
const (
	MaxNameLength        = 100
	MaxDescriptionLength = 500
)

// Bike is a single rentable bike of the shop's fleet.
type Bike struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    Category `json:"category"`
	Status      Status   `json:"status"`
	Description string   `json:"description"`
}

// Normalize trims text fields and sets the default status for new bikes.
func (b *Bike) Normalize() {
	b.Name = strings.TrimSpace(b.Name)
	b.Description = strings.TrimSpace(b.Description)
	if b.Status == "" {
		b.Status = StatusAvailable
	}
}

// Validate checks the bike's fields and returns an error wrapping ErrInvalid
// for the first rule that is violated. Call Normalize first.
func (b Bike) Validate() error {
	switch {
	case b.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalid)
	case utf8.RuneCountInString(b.Name) > MaxNameLength:
		return fmt.Errorf("%w: name must be at most %d characters", ErrInvalid, MaxNameLength)
	case !b.Category.Valid():
		return fmt.Errorf("%w: unknown category %q", ErrInvalid, b.Category)
	case !b.Status.Valid():
		return fmt.Errorf("%w: unknown status %q", ErrInvalid, b.Status)
	case utf8.RuneCountInString(b.Description) > MaxDescriptionLength:
		return fmt.Errorf("%w: description must be at most %d characters", ErrInvalid, MaxDescriptionLength)
	}
	return nil
}

// Retire takes the bike permanently out of the fleet. Bikes are never hard-deleted.
func (b *Bike) Retire() {
	b.Status = StatusRetired
}

// IsRetired reports whether the bike has been taken out of the fleet.
func (b Bike) IsRetired() bool {
	return b.Status == StatusRetired
}

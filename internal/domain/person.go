package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Field length limits for persons, counted in characters.
const (
	MaxPersonNameLength = 100
	MaxIDNumberLength   = 30
	MaxPhoneLength      = 30
	minPhoneDigits      = 5
)

// Person is a customer who rents bikes. Every person has an ID copy (stored separately).
type Person struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	IDNumber  string `json:"idNumber"`
	Phone     string `json:"phone"`
	Archived  bool   `json:"archived"`
}

// Normalize trims all text fields and upper-cases the ID number,
// so "ab 123" and "AB 123" count as the same document.
func (p *Person) Normalize() {
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.LastName = strings.TrimSpace(p.LastName)
	p.IDNumber = strings.ToUpper(strings.TrimSpace(p.IDNumber))
	p.Phone = strings.TrimSpace(p.Phone)
}

// Validate checks the person's fields and returns an error wrapping ErrInvalid
// for the first rule that is violated. Call Normalize first.
func (p Person) Validate() error {
	if err := checkRequired("first name", p.FirstName, MaxPersonNameLength); err != nil {
		return err
	}
	if err := checkRequired("last name", p.LastName, MaxPersonNameLength); err != nil {
		return err
	}
	if err := checkRequired("ID number", p.IDNumber, MaxIDNumberLength); err != nil {
		return err
	}
	if err := checkRequired("phone", p.Phone, MaxPhoneLength); err != nil {
		return err
	}
	return validatePhone(p.Phone)
}

// FullName returns "First Last".
func (p Person) FullName() string {
	return p.FirstName + " " + p.LastName
}

// Archive hides the person from the active list. Persons are never deleted.
func (p *Person) Archive() {
	p.Archived = true
}

// Reactivate makes an archived person active again.
func (p *Person) Reactivate() {
	p.Archived = false
}

func checkRequired(field, value string, maxLen int) error {
	if value == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalid, field)
	}
	if utf8.RuneCountInString(value) > maxLen {
		return fmt.Errorf("%w: %s must be at most %d characters", ErrInvalid, field, maxLen)
	}
	return nil
}

// validatePhone allows digits, spaces and + - / ( ) and needs a minimum number of digits.
func validatePhone(phone string) error {
	digits := 0
	for _, r := range phone {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case strings.ContainsRune(" +-/()", r):
		default:
			return fmt.Errorf("%w: phone may only contain digits, spaces and + - / ( )", ErrInvalid)
		}
	}
	if digits < minPhoneDigits {
		return fmt.Errorf("%w: phone must contain at least %d digits", ErrInvalid, minPhoneDigits)
	}
	return nil
}

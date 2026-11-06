package domain

import "errors"

// Sentinel errors. Wrap them with fmt.Errorf("%w: ...", ErrX) to add detail;
// callers check them with errors.Is.
var (
	ErrInvalid  = errors.New("invalid input")
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrTooLarge = errors.New("too large")
)

// Package domain holds the core structs (Bike, Person, Reservation) and their methods.
// All business rules of the bike rental shop live here, independent of HTTP and storage.
package domain

// Category is the type of a bike. It determines the hourly rental price.
type Category string

// The four bike categories offered by the shop.
const (
	CategoryChildren Category = "children"
	CategoryCity     Category = "city"
	CategoryMountain Category = "mountain"
	CategoryRacing   Category = "racing"
)

// categoryInfo holds the display label and hourly price (in cents) of a category.
// Prices are placeholders until the stakeholders confirm them (see specs/bike-management).
var categoryInfo = map[Category]struct {
	label       string
	hourlyPrice int64
}{
	CategoryChildren: {"Children's bike", 400},
	CategoryCity:     {"City bike", 600},
	CategoryMountain: {"Mountain bike", 900},
	CategoryRacing:   {"Racing bike", 1000},
}

// AllCategories returns all categories in a stable display order.
func AllCategories() []Category {
	return []Category{CategoryChildren, CategoryCity, CategoryMountain, CategoryRacing}
}

// Valid reports whether c is one of the known categories.
func (c Category) Valid() bool {
	_, ok := categoryInfo[c]
	return ok
}

// Label returns the human-readable name of the category, or "" if it is unknown.
func (c Category) Label() string {
	return categoryInfo[c].label
}

// HourlyPrice returns the rental price per hour in cents, or 0 if the category is unknown.
func (c Category) HourlyPrice() int64 {
	return categoryInfo[c].hourlyPrice
}

// Package seed fills empty stores with demo data for presentations: bikes in every state,
// customers with a sample ID copy, and reservations around "now" (active, completed,
// upcoming, cancelled). It is only used when the server is started with -demo.
package seed

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"time"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

type demoBike struct {
	name        string
	category    domain.Category
	status      domain.Status
	description string
}

var demoBikes = []demoBike{
	{"City-01", domain.CategoryCity, domain.StatusAvailable, "Red Gazelle, 7 gears, basket"},
	{"City-02", domain.CategoryCity, domain.StatusAvailable, "Mint green, step-through frame"},
	{"City-03", domain.CategoryCity, domain.StatusMaintenance, "Brakes being checked"},
	{"Kids-01", domain.CategoryChildren, domain.StatusAvailable, "20 inch, pink, with bell"},
	{"Kids-02", domain.CategoryChildren, domain.StatusAvailable, "16 inch, training wheels"},
	{"MTB-01", domain.CategoryMountain, domain.StatusAvailable, "Cube, full suspension"},
	{"MTB-02", domain.CategoryMountain, domain.StatusAvailable, "Trek, hardtail"},
	{"Race-01", domain.CategoryRacing, domain.StatusAvailable, "Carbon, 22 gears"},
	{"Race-02", domain.CategoryRacing, domain.StatusAvailable, "Aluminium, size M"},
	{"City-00", domain.CategoryCity, domain.StatusRetired, "Old frame, replaced in spring"},
}

var demoPersons = []domain.Person{
	{FirstName: "Clara", LastName: "Becker", IDNumber: "T22000129", Phone: "+49 170 1112223"},
	{FirstName: "David", LastName: "Schulz", IDNumber: "L01X00T47", Phone: "0711 4455667"},
	{FirstName: "Emma", LastName: "Wagner", IDNumber: "C4F2K9P1M", Phone: "+49 152 9876543"},
	{FirstName: "Finn", LastName: "Hoffmann", IDNumber: "F7H3J8D2X", Phone: "0621 334455", Archived: true},
}

// Load adds the demo data to the stores. The stores should be empty; otherwise name or
// ID-number conflicts are returned as errors.
func Load(bikes store.BikeStore, persons store.PersonStore, reservations store.ReservationStore, now time.Time) error {
	bikeIDs := map[string]string{}
	for _, d := range demoBikes {
		b, err := bikes.Create(domain.Bike{Name: d.name, Category: d.category, Status: d.status, Description: d.description})
		if err != nil {
			return fmt.Errorf("seed bike %s: %w", d.name, err)
		}
		bikeIDs[d.name] = b.ID
	}

	idCopy, err := sampleIDCard()
	if err != nil {
		return fmt.Errorf("seed ID card: %w", err)
	}
	personIDs := map[string]domain.Person{}
	for _, p := range demoPersons {
		doc, err := domain.NewIDDocument(fmt.Sprintf("id-%s.png", p.LastName), idCopy, now.AddDate(0, 0, -14))
		if err != nil {
			return fmt.Errorf("seed ID copy for %s: %w", p.FullName(), err)
		}
		e, err := persons.Create(p, doc)
		if err != nil {
			return fmt.Errorf("seed person %s: %w", p.FullName(), err)
		}
		personIDs[p.FirstName] = e.Person
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day := func(offset, hour int) time.Time {
		return today.AddDate(0, 0, offset).Add(time.Duration(hour) * time.Hour)
	}
	hourNow := now.Truncate(time.Hour)

	plans := []struct {
		person    string
		bikes     []string
		start     time.Time
		end       time.Time
		cancelled bool
	}{
		{"Clara", []string{"City-01", "Kids-01"}, hourNow.Add(-time.Hour), hourNow.Add(2 * time.Hour), false}, // active now
		{"David", []string{"MTB-01"}, day(-1, 10), day(-1, 14), false},                                        // completed yesterday
		{"Emma", []string{"Race-01"}, day(2, 9), day(2, 13), false},                                           // upcoming, still changeable
		{"David", []string{"City-01", "City-02"}, day(3, 10), day(3, 18), false},                              // upcoming, two bikes
		{"Clara", []string{"MTB-02"}, day(5, 8), day(7, 8), false},                                            // upcoming, 48 hours
		{"Emma", []string{"Kids-02"}, day(4, 14), day(4, 16), true},                                           // cancelled
		{"Clara", []string{"Race-02"}, day(10, 9), day(10, 17), false},                                        // beyond the 7-day dashboard window
	}
	for _, p := range plans {
		person := personIDs[p.person]
		reserved := make([]domain.ReservedBike, len(p.bikes))
		for i, name := range p.bikes {
			b, err := bikes.Get(bikeIDs[name])
			if err != nil {
				return fmt.Errorf("seed reservation: %w", err)
			}
			reserved[i] = domain.ReservedBikeFrom(b)
		}
		// Bookings were made a week before their start, so past and active ones are valid too.
		bookedAt := p.start.AddDate(0, 0, -7)
		r, err := domain.NewReservation(person.ID, person.IDNumber, reserved, domain.Period{Start: p.start, End: p.end}, bookedAt)
		if err != nil {
			return fmt.Errorf("seed reservation for %s: %w", person.FullName(), err)
		}
		if p.cancelled {
			if err := r.Cancel(bookedAt.Add(time.Hour)); err != nil {
				return fmt.Errorf("seed cancellation: %w", err)
			}
		}
		if _, err := reservations.Create(r); err != nil {
			return fmt.Errorf("seed reservation for %s: %w", person.FullName(), err)
		}
	}
	return nil
}

// sampleIDCard draws a simple placeholder ID card (no real data) as PNG.
func sampleIDCard() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 200))
	fill := func(x0, y0, x1, y1 int, c color.RGBA) {
		draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	fill(0, 0, 320, 200, color.RGBA{0xfc, 0xe4, 0xec, 0xff})   // card
	fill(0, 0, 320, 36, color.RGBA{0xc2, 0x18, 0x5b, 0xff})    // header band
	fill(20, 56, 100, 160, color.RGBA{0xd9, 0xc7, 0xcf, 0xff}) // photo
	for i, w := range []int{180, 140, 160, 110} {              // text lines
		y := 62 + i*26
		fill(120, y, 120+w, y+10, color.RGBA{0x8f, 0x72, 0x80, 0xff})
	}
	fill(20, 176, 300, 186, color.RGBA{0xf4, 0x8f, 0xb1, 0xff}) // machine-readable zone

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

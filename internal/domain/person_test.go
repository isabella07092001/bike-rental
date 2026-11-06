package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestPersonNormalize(t *testing.T) {
	p := Person{FirstName: " Alice ", LastName: "\tMeyer", IDNumber: " l01x00t47 ", Phone: " +49 170 1234567 "}
	p.Normalize()

	want := Person{FirstName: "Alice", LastName: "Meyer", IDNumber: "L01X00T47", Phone: "+49 170 1234567"}
	if p != want {
		t.Errorf("Normalize() = %+v, want %+v", p, want)
	}
}

func TestPersonValidate(t *testing.T) {
	valid := Person{FirstName: "Alice", LastName: "Meyer", IDNumber: "L01X00T47", Phone: "+49 (170) 123-4567"}

	tests := []struct {
		name    string
		modify  func(*Person)
		wantErr string // substring; "" means valid
	}{
		{"valid", func(*Person) {}, ""},
		{"empty first name", func(p *Person) { p.FirstName = "" }, "first name is required"},
		{"empty last name", func(p *Person) { p.LastName = "" }, "last name is required"},
		{"empty ID number", func(p *Person) { p.IDNumber = "" }, "ID number is required"},
		{"empty phone", func(p *Person) { p.Phone = "" }, "phone is required"},
		{"first name at limit", func(p *Person) { p.FirstName = strings.Repeat("ä", MaxPersonNameLength) }, ""},
		{"first name too long", func(p *Person) { p.FirstName = strings.Repeat("a", MaxPersonNameLength+1) }, "first name must be at most"},
		{"last name too long", func(p *Person) { p.LastName = strings.Repeat("a", MaxPersonNameLength+1) }, "last name must be at most"},
		{"ID number at limit", func(p *Person) { p.IDNumber = strings.Repeat("A", MaxIDNumberLength) }, ""},
		{"ID number too long", func(p *Person) { p.IDNumber = strings.Repeat("A", MaxIDNumberLength+1) }, "ID number must be at most"},
		{"phone too long", func(p *Person) { p.Phone = strings.Repeat("1", MaxPhoneLength+1) }, "phone must be at most"},
		{"phone with letters", func(p *Person) { p.Phone = "0170 CALL ME" }, "phone may only contain"},
		{"phone with 5 digits", func(p *Person) { p.Phone = "12 345" }, ""},
		{"phone with 4 digits", func(p *Person) { p.Phone = "+1 (234)" }, "at least 5 digits"},
		{"phone with slash", func(p *Person) { p.Phone = "0711/123456" }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid
			tt.modify(&p)
			err := p.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tt.wantErr)
			}
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("error %v does not wrap ErrInvalid", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestPersonArchiveReactivate(t *testing.T) {
	p := Person{FirstName: "Alice", LastName: "Meyer"}
	if p.FullName() != "Alice Meyer" {
		t.Errorf("FullName() = %q", p.FullName())
	}
	p.Archive()
	if !p.Archived {
		t.Error("Archive() did not set Archived")
	}
	p.Reactivate()
	if p.Archived {
		t.Error("Reactivate() did not clear Archived")
	}
}

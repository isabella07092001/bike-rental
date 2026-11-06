package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestBikeNormalize(t *testing.T) {
	b := Bike{Name: "  City-01 ", Description: "\tred  ", Category: CategoryCity}
	b.Normalize()

	if b.Name != "City-01" {
		t.Errorf("Name = %q, want %q", b.Name, "City-01")
	}
	if b.Description != "red" {
		t.Errorf("Description = %q, want %q", b.Description, "red")
	}
	if b.Status != StatusAvailable {
		t.Errorf("Status = %q, want default %q", b.Status, StatusAvailable)
	}

	m := Bike{Status: StatusMaintenance}
	m.Normalize()
	if m.Status != StatusMaintenance {
		t.Errorf("Normalize overwrote explicit status: got %q", m.Status)
	}
}

func TestBikeValidate(t *testing.T) {
	valid := Bike{Name: "City-01", Category: CategoryCity, Status: StatusAvailable}

	tests := []struct {
		name    string
		modify  func(*Bike)
		wantErr string // substring; "" means valid
	}{
		{"valid", func(*Bike) {}, ""},
		{"valid maintenance", func(b *Bike) { b.Status = StatusMaintenance }, ""},
		{"valid retired", func(b *Bike) { b.Status = StatusRetired }, ""},
		{"empty name", func(b *Bike) { b.Name = "" }, "name is required"},
		{"name at limit", func(b *Bike) { b.Name = strings.Repeat("a", MaxNameLength) }, ""},
		{"name too long", func(b *Bike) { b.Name = strings.Repeat("a", MaxNameLength+1) }, "name must be at most"},
		{"name limit counts characters not bytes", func(b *Bike) { b.Name = strings.Repeat("ü", MaxNameLength) }, ""},
		{"missing category", func(b *Bike) { b.Category = "" }, "unknown category"},
		{"unknown category", func(b *Bike) { b.Category = "tandem" }, "unknown category"},
		{"unknown status", func(b *Bike) { b.Status = "rented" }, "unknown status"},
		{"description at limit", func(b *Bike) { b.Description = strings.Repeat("x", MaxDescriptionLength) }, ""},
		{"description too long", func(b *Bike) { b.Description = strings.Repeat("x", MaxDescriptionLength+1) }, "description must be at most"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := valid
			tt.modify(&b)
			err := b.Validate()

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

func TestBikeRetire(t *testing.T) {
	b := Bike{Name: "City-01", Category: CategoryCity, Status: StatusAvailable}
	if b.IsRetired() {
		t.Fatal("new bike reports IsRetired() = true")
	}
	b.Retire()
	if !b.IsRetired() || b.Status != StatusRetired {
		t.Errorf("after Retire(): Status = %q, IsRetired() = %v", b.Status, b.IsRetired())
	}
}

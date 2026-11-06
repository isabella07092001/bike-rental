package domain

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

// Minimal file signatures that http.DetectContentType recognises.
var (
	testPDF  = []byte("%PDF-1.4\n%test\n")
	testPNG  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	testJPEG = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
)

func TestNewIDDocument(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		data     []byte
		wantType string
		wantErr  error
	}{
		{"pdf", testPDF, ContentTypePDF, nil},
		{"png", testPNG, ContentTypePNG, nil},
		{"jpeg", testJPEG, ContentTypeJPEG, nil},
		{"empty", nil, "", ErrInvalid},
		{"plain text", []byte("hello, I am not a PDF"), "", ErrInvalid},
		{"gif not allowed", []byte("GIF89a\x01\x00\x01\x00"), "", ErrInvalid},
		{"at size limit", append(bytes.Clone(testPDF), make([]byte, MaxIDDocumentSize-len(testPDF))...), ContentTypePDF, nil},
		{"over size limit", append(bytes.Clone(testPDF), make([]byte, MaxIDDocumentSize-len(testPDF)+1)...), "", ErrTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewIDDocument("scan.pdf", tt.data, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if doc.ContentType != tt.wantType {
				t.Errorf("ContentType = %q, want %q", doc.ContentType, tt.wantType)
			}
			if !doc.UploadedAt.Equal(now) {
				t.Errorf("UploadedAt = %v, want %v", doc.UploadedAt, now)
			}
		})
	}
}

func TestNewIDDocumentIgnoresClaimedExtension(t *testing.T) {
	// A text file renamed to .pdf must still be rejected.
	if _, err := NewIDDocument("passport.pdf", []byte("just text"), time.Now()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestIDDocumentFileName(t *testing.T) {
	tests := map[string]string{
		"scan.pdf":                      "scan.pdf",
		`C:\Users\alice\Desktop\id.png`: "id.png",
		"/home/bob/id.jpg":              "id.jpg",
		"  ":                            "id-document",
		"":                              "id-document",
	}
	for in, want := range tests {
		doc, err := NewIDDocument(in, testPNG, time.Now())
		if err != nil {
			t.Fatalf("NewIDDocument(%q): %v", in, err)
		}
		if doc.FileName != want {
			t.Errorf("FileName for %q = %q, want %q", in, doc.FileName, want)
		}
	}
}

func TestIDDocumentMeta(t *testing.T) {
	now := time.Now()
	doc, _ := NewIDDocument("scan.pdf", testPDF, now)
	meta := doc.Meta()
	want := IDDocumentMeta{FileName: "scan.pdf", ContentType: ContentTypePDF, Size: len(testPDF), UploadedAt: now}
	if meta != want {
		t.Errorf("Meta() = %+v, want %+v", meta, want)
	}
}

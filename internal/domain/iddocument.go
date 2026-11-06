package domain

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

// MaxIDDocumentSize is the largest accepted ID copy in bytes (5 MB).
const MaxIDDocumentSize = 5 << 20

// Content types accepted for ID copies. They are detected from the file bytes,
// never taken from the client.
const (
	ContentTypePDF  = "application/pdf"
	ContentTypeJPEG = "image/jpeg"
	ContentTypePNG  = "image/png"
)

// IDDocument is the uploaded copy of a person's identification document.
// It is sensitive personal data: kept in memory only, never logged.
type IDDocument struct {
	FileName    string
	ContentType string
	Data        []byte
	UploadedAt  time.Time
}

// IDDocumentMeta describes an ID copy without its content. Only this is sent in person JSON.
type IDDocumentMeta struct {
	FileName    string    `json:"fileName"`
	ContentType string    `json:"contentType"`
	Size        int       `json:"size"`
	UploadedAt  time.Time `json:"uploadedAt"`
}

// NewIDDocument checks an uploaded file and returns it as an IDDocument.
// It fails with ErrInvalid for empty files or unsupported types and with ErrTooLarge
// for files over MaxIDDocumentSize.
func NewIDDocument(fileName string, data []byte, uploadedAt time.Time) (IDDocument, error) {
	if len(data) == 0 {
		return IDDocument{}, fmt.Errorf("%w: ID copy is empty", ErrInvalid)
	}
	if len(data) > MaxIDDocumentSize {
		return IDDocument{}, fmt.Errorf("%w: ID copy must be at most %d MB", ErrTooLarge, MaxIDDocumentSize>>20)
	}
	contentType := detectContentType(data)
	switch contentType {
	case ContentTypePDF, ContentTypeJPEG, ContentTypePNG:
	default:
		return IDDocument{}, fmt.Errorf("%w: ID copy must be a PDF, JPG or PNG file", ErrInvalid)
	}
	return IDDocument{
		FileName:    cleanFileName(fileName),
		ContentType: contentType,
		Data:        data,
		UploadedAt:  uploadedAt,
	}, nil
}

// Meta returns the document's metadata without its content.
func (d IDDocument) Meta() IDDocumentMeta {
	return IDDocumentMeta{
		FileName:    d.FileName,
		ContentType: d.ContentType,
		Size:        len(d.Data),
		UploadedAt:  d.UploadedAt,
	}
}

// detectContentType sniffs the type from the first bytes and drops parameters like "; charset=…".
func detectContentType(data []byte) string {
	ct := http.DetectContentType(data)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.TrimSpace(ct)
}

// cleanFileName keeps only the base name of an uploaded file (browsers may send full
// Windows paths) and falls back to a default name.
func cleanFileName(name string) string {
	name = path.Base(strings.ReplaceAll(strings.TrimSpace(name), `\`, "/"))
	if name == "." || name == "/" || name == "" {
		return "id-document"
	}
	return name
}

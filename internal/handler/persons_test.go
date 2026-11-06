package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bike-rental/internal/domain"
)

var testPDF = []byte("%PDF-1.4\n%test document\n")

type upload struct {
	fields   map[string]string
	fileName string
	file     []byte // nil means: no file part
}

// sendUpload sends a multipart request and decodes the JSON response into out (if non-nil).
func sendUpload(t *testing.T, router http.Handler, method, path string, u upload, out any) int {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range u.fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if u.file != nil {
		fw, err := mw.CreateFormFile(idDocumentField, u.fileName)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(u.file)
	}
	mw.Close()

	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if out != nil {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decode response: %v", method, path, err)
		}
	}
	return rec.Code
}

func personFields(first, last, idNumber string) map[string]string {
	return map[string]string{"firstName": first, "lastName": last, "idNumber": idNumber, "phone": "+49 170 1234567"}
}

func createPerson(t *testing.T, router http.Handler, first, last, idNumber string) personResponse {
	t.Helper()
	var p personResponse
	code := sendUpload(t, router, http.MethodPost, "/api/persons",
		upload{fields: personFields(first, last, idNumber), fileName: "id.pdf", file: testPDF}, &p)
	if code != http.StatusCreated {
		t.Fatalf("create %s %s: status = %d", first, last, code)
	}
	return p
}

func TestPersonLifecycle(t *testing.T) {
	router := newTestRouter()

	created := createPerson(t, router, " Alice ", "Meyer", "l01x00t47")
	if created.ID == "" || created.FirstName != "Alice" || created.IDNumber != "L01X00T47" || created.Archived {
		t.Fatalf("create: got %+v", created)
	}
	if created.IDDocument.FileName != "id.pdf" || created.IDDocument.ContentType != domain.ContentTypePDF ||
		created.IDDocument.Size != len(testPDF) || created.IDDocument.UploadedAt.IsZero() {
		t.Errorf("create: document meta = %+v", created.IDDocument)
	}

	var got personResponse
	if code := do(t, router, http.MethodGet, "/api/persons/"+created.ID, "", &got); code != http.StatusOK || got.ID != created.ID {
		t.Fatalf("get: status = %d, person = %+v", code, got)
	}

	var updated personResponse
	code := do(t, router, http.MethodPut, "/api/persons/"+created.ID,
		`{"firstName":"Alice","lastName":"Müller","idNumber":"L01X00T47","phone":"0711 123456"}`, &updated)
	if code != http.StatusOK || updated.LastName != "Müller" || updated.IDDocument != created.IDDocument {
		t.Fatalf("update: status = %d, person = %+v", code, updated)
	}

	var archived personResponse
	if code := do(t, router, http.MethodPost, "/api/persons/"+created.ID+"/archive", "", &archived); code != http.StatusOK || !archived.Archived {
		t.Fatalf("archive: status = %d, person = %+v", code, archived)
	}
	// Editing an archived person keeps it archived.
	do(t, router, http.MethodPut, "/api/persons/"+created.ID,
		`{"firstName":"Alice","lastName":"Müller","idNumber":"L01X00T47","phone":"0711 123456"}`, &updated)
	if !updated.Archived {
		t.Error("update cleared the archived flag")
	}

	var reactivated personResponse
	if code := do(t, router, http.MethodPost, "/api/persons/"+created.ID+"/reactivate", "", &reactivated); code != http.StatusOK || reactivated.Archived {
		t.Fatalf("reactivate: status = %d, person = %+v", code, reactivated)
	}
}

func TestPersonDocumentDownloadAndReplace(t *testing.T) {
	router := newTestRouter()
	p := createPerson(t, router, "Alice", "Meyer", "A1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/persons/"+p.ID+"/id-document", nil))
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), testPDF) {
		t.Fatalf("download: status = %d, body = %q", rec.Code, rec.Body.String())
	}
	for header, want := range map[string]string{
		"Content-Type":           domain.ContentTypePDF,
		"Content-Disposition":    `inline; filename=id.pdf`,
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "no-store",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	var replaced personResponse
	code := sendUpload(t, router, http.MethodPut, "/api/persons/"+p.ID+"/id-document",
		upload{fileName: "photo.png", file: png}, &replaced)
	if code != http.StatusOK || replaced.IDDocument.ContentType != domain.ContentTypePNG || replaced.IDDocument.FileName != "photo.png" {
		t.Fatalf("replace: status = %d, person = %+v", code, replaced)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/persons/"+p.ID+"/id-document", nil))
	if !bytes.Equal(rec.Body.Bytes(), png) {
		t.Errorf("download after replace returned old content")
	}
}

func TestPersonJSONNeverContainsDocumentBytes(t *testing.T) {
	router := newTestRouter()
	p := createPerson(t, router, "Alice", "Meyer", "A1")

	for _, path := range []string{"/api/persons", "/api/persons/" + p.ID} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if strings.Contains(rec.Body.String(), "test document") || strings.Contains(rec.Body.String(), "data") {
			t.Errorf("%s leaks document content: %s", path, rec.Body.String())
		}
	}
}

func TestPersonListFilters(t *testing.T) {
	router := newTestRouter()
	createPerson(t, router, "Alice", "Meyer", "AAA111")
	createPerson(t, router, "Bob", "Schmidt", "BBB222")
	carol := createPerson(t, router, "Carol", "Meyer", "CCC333")
	do(t, router, http.MethodPost, "/api/persons/"+carol.ID+"/archive", "", nil)

	tests := []struct {
		query string
		want  string
	}{
		{"", "Alice,Bob"},
		{"?archived=true", "Carol"},
		{"?archived=false", "Alice,Bob"},
		{"?q=meyer", "Alice"},
		{"?q=MEYER+alice", "Alice"},
		{"?q=bbb2", "Bob"},
		{"?q=170", "Alice,Bob"},
		{"?q=meyer&archived=true", "Carol"},
		{"?q=nobody", ""},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			var persons []personResponse
			if code := do(t, router, http.MethodGet, "/api/persons"+tt.query, "", &persons); code != http.StatusOK {
				t.Fatalf("status = %d", code)
			}
			names := make([]string, 0, len(persons))
			for _, p := range persons {
				names = append(names, p.FirstName)
			}
			if got := strings.Join(names, ","); got != tt.want {
				t.Errorf("names = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPersonErrors(t *testing.T) {
	router := newTestRouter()
	existing := createPerson(t, router, "Alice", "Meyer", "TAKEN1")

	big := append(bytes.Clone(testPDF), make([]byte, domain.MaxIDDocumentSize)...)

	uploadTests := []struct {
		name       string
		method     string
		path       string
		u          upload
		wantStatus int
		wantError  string
	}{
		{"missing ID copy", http.MethodPost, "/api/persons", upload{fields: personFields("Bob", "S", "B1")}, 400, "ID copy is required"},
		{"empty ID copy", http.MethodPost, "/api/persons", upload{fields: personFields("Bob", "S", "B1"), fileName: "x.pdf", file: []byte{}}, 400, "empty"},
		{"text renamed to pdf", http.MethodPost, "/api/persons", upload{fields: personFields("Bob", "S", "B1"), fileName: "x.pdf", file: []byte("plain text")}, 400, "PDF, JPG or PNG"},
		{"too large", http.MethodPost, "/api/persons", upload{fields: personFields("Bob", "S", "B1"), fileName: "x.pdf", file: big}, 413, "at most 5 MB"},
		{"missing last name", http.MethodPost, "/api/persons", upload{fields: personFields("Bob", "", "B1"), fileName: "x.pdf", file: testPDF}, 400, "last name is required"},
		{"invalid phone", http.MethodPost, "/api/persons", upload{fields: map[string]string{"firstName": "Bob", "lastName": "S", "idNumber": "B1", "phone": "call me"}, fileName: "x.pdf", file: testPDF}, 400, "phone"},
		{"duplicate ID number", http.MethodPost, "/api/persons", upload{fields: personFields("Eve", "X", "taken1"), fileName: "x.pdf", file: testPDF}, 409, "already exists"},
		{"replace unknown", http.MethodPut, "/api/persons/p-99/id-document", upload{fileName: "x.pdf", file: testPDF}, 404, "not found"},
		{"replace without file", http.MethodPut, "/api/persons/" + existing.ID + "/id-document", upload{}, 400, "ID copy is required"},
	}
	for _, tt := range uploadTests {
		t.Run(tt.name, func(t *testing.T) {
			var resp map[string]string
			code := sendUpload(t, router, tt.method, tt.path, tt.u, &resp)
			if code != tt.wantStatus {
				t.Errorf("status = %d, want %d", code, tt.wantStatus)
			}
			if !strings.Contains(resp["error"], tt.wantError) {
				t.Errorf("error = %q, want it to contain %q", resp["error"], tt.wantError)
			}
		})
	}

	jsonTests := []struct {
		name, method, path, body string
		wantStatus               int
		wantError                string
	}{
		{"create as JSON", http.MethodPost, "/api/persons", `{"firstName":"Bob"}`, 400, "multipart"},
		{"get unknown", http.MethodGet, "/api/persons/p-99", "", 404, "not found"},
		{"download unknown", http.MethodGet, "/api/persons/p-99/id-document", "", 404, "not found"},
		{"update unknown", http.MethodPut, "/api/persons/p-99", `{}`, 404, "not found"},
		{"update invalid JSON", http.MethodPut, "/api/persons/" + existing.ID, `{`, 400, "not valid JSON"},
		{"update empty first name", http.MethodPut, "/api/persons/" + existing.ID, `{"firstName":"","lastName":"M","idNumber":"TAKEN1","phone":"12345"}`, 400, "first name is required"},
		{"archive unknown", http.MethodPost, "/api/persons/p-99/archive", "", 404, "not found"},
		{"reactivate unknown", http.MethodPost, "/api/persons/p-99/reactivate", "", 404, "not found"},
		{"invalid archived filter", http.MethodGet, "/api/persons?archived=maybe", "", 400, "archived"},
	}
	for _, tt := range jsonTests {
		t.Run(tt.name, func(t *testing.T) {
			var resp map[string]string
			code := do(t, router, tt.method, tt.path, tt.body, &resp)
			if code != tt.wantStatus {
				t.Errorf("status = %d, want %d", code, tt.wantStatus)
			}
			if !strings.Contains(resp["error"], tt.wantError) {
				t.Errorf("error = %q, want it to contain %q", resp["error"], tt.wantError)
			}
		})
	}
}

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// do sends a request to router and decodes the JSON response into out (if non-nil).
func do(t *testing.T, router http.Handler, method, path, body string, out any) int {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if out != nil {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decode response: %v", method, path, err)
		}
	}
	return rec.Code
}

func newTestRouter() http.Handler {
	return NewRouter("", store.NewMemoryStore())
}

func TestBikeLifecycle(t *testing.T) {
	router := newTestRouter()

	var created domain.Bike
	code := do(t, router, http.MethodPost, "/api/bikes",
		`{"name":"  City-01 ","category":"city","description":"red"}`, &created)
	if code != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", code)
	}
	if created.ID == "" || created.Name != "City-01" || created.Status != domain.StatusAvailable {
		t.Fatalf("create: got %+v", created)
	}

	var got domain.Bike
	if code := do(t, router, http.MethodGet, "/api/bikes/"+created.ID, "", &got); code != http.StatusOK || got != created {
		t.Fatalf("get: status = %d, bike = %+v", code, got)
	}

	var updated domain.Bike
	code = do(t, router, http.MethodPut, "/api/bikes/"+created.ID,
		`{"name":"City-01","category":"mountain","status":"maintenance","description":""}`, &updated)
	if code != http.StatusOK || updated.Category != domain.CategoryMountain || updated.Status != domain.StatusMaintenance {
		t.Fatalf("update: status = %d, bike = %+v", code, updated)
	}

	var retired domain.Bike
	code = do(t, router, http.MethodPost, "/api/bikes/"+created.ID+"/retire", "", &retired)
	if code != http.StatusOK || retired.Status != domain.StatusRetired {
		t.Fatalf("retire: status = %d, bike = %+v", code, retired)
	}

	// Reactivate via PUT.
	code = do(t, router, http.MethodPut, "/api/bikes/"+created.ID,
		`{"name":"City-01","category":"mountain","status":"available"}`, &updated)
	if code != http.StatusOK || updated.Status != domain.StatusAvailable {
		t.Fatalf("reactivate: status = %d, bike = %+v", code, updated)
	}
}

func TestListFilters(t *testing.T) {
	router := newTestRouter()
	for _, body := range []string{
		`{"name":"A","category":"city"}`,
		`{"name":"B","category":"racing","status":"maintenance"}`,
		`{"name":"C","category":"city","status":"retired"}`,
	} {
		if code := do(t, router, http.MethodPost, "/api/bikes", body, nil); code != http.StatusCreated {
			t.Fatalf("create %s: status = %d", body, code)
		}
	}

	tests := []struct {
		query string
		want  string
	}{
		{"", "A,B"},
		{"?category=city", "A"},
		{"?status=maintenance", "B"},
		{"?status=retired", "C"},
		{"?category=racing&status=retired", ""},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			var bikes []domain.Bike
			if code := do(t, router, http.MethodGet, "/api/bikes"+tt.query, "", &bikes); code != http.StatusOK {
				t.Fatalf("status = %d", code)
			}
			names := make([]string, 0, len(bikes))
			for _, b := range bikes {
				names = append(names, b.Name)
			}
			if got := strings.Join(names, ","); got != tt.want {
				t.Errorf("names = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListEmptyIsArray(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bikes", nil))
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body = %q, want []", body)
	}
}

func TestBikeErrors(t *testing.T) {
	router := newTestRouter()
	do(t, router, http.MethodPost, "/api/bikes", `{"name":"Taken","category":"city"}`, nil)

	tests := []struct {
		name, method, path, body string
		wantStatus               int
		wantError                string
	}{
		{"empty name", http.MethodPost, "/api/bikes", `{"name":" ","category":"city"}`, 400, "name is required"},
		{"unknown category", http.MethodPost, "/api/bikes", `{"name":"X","category":"tandem"}`, 400, "unknown category"},
		{"unknown status", http.MethodPost, "/api/bikes", `{"name":"X","category":"city","status":"rented"}`, 400, "unknown status"},
		{"too long description", http.MethodPost, "/api/bikes", `{"name":"X","category":"city","description":"` + strings.Repeat("x", 501) + `"}`, 400, "description"},
		{"invalid json", http.MethodPost, "/api/bikes", `{`, 400, "not valid JSON"},
		{"duplicate name", http.MethodPost, "/api/bikes", `{"name":"taken","category":"racing"}`, 409, "already exists"},
		{"invalid list filter", http.MethodGet, "/api/bikes?category=tandem", "", 400, "unknown category"},
		{"get unknown", http.MethodGet, "/api/bikes/b-99", "", 404, "not found"},
		{"update unknown", http.MethodPut, "/api/bikes/b-99", `{"name":"X","category":"city"}`, 404, "not found"},
		{"retire unknown", http.MethodPost, "/api/bikes/b-99/retire", "", 404, "not found"},
	}
	for _, tt := range tests {
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

func TestListCategories(t *testing.T) {
	var cats []categoryResponse
	if code := do(t, newTestRouter(), http.MethodGet, "/api/categories", "", &cats); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if len(cats) != 4 {
		t.Fatalf("len = %d, want 4", len(cats))
	}
	if cats[1].Value != domain.CategoryCity || cats[1].Label != "City bike" || cats[1].HourlyPriceCents != 600 {
		t.Errorf("cats[1] = %+v", cats[1])
	}
}

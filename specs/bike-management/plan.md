# Phase 1 – Bike Management: Plan

Requirements: [requirements.md](requirements.md). Each task group ends with passing tests
(`go test -race ./...`) and can be committed on its own.

## 1. Domain: Category

1.1 Define `Category` type with the four constants (`children`, `city`, `mountain`, `racing`)
1.2 Add `Valid()`, `Label()` and `HourlyPrice()` (cents) methods; placeholder prices 4/6/9/10 € in one place
1.3 Add `AllCategories()` for listing
1.4 Table-driven tests for valid/invalid values and prices

## 2. Domain: Bike

2.1 Define `Status` type (`available`, `maintenance`, `retired`) with `Valid()`
2.2 Define `Bike` struct (`ID`, `Name`, `Category`, `Status`, `Description`) with JSON tags
2.3 Add `Bike.Normalize()` (trim text, default status `available`)
2.4 Add `Bike.Validate()` (name required, ≤ 100 chars; description ≤ 500 chars; valid category and status)
2.5 Add `Bike.Retire()` (sets status `retired`)
2.6 Define domain errors (`ErrNotFound`, `ErrInvalid`, `ErrConflict`) for wrapping
2.7 Table-driven tests for `Normalize()`, `Validate()` and `Retire()`

## 3. Store

3.1 Define `BikeStore` interface (`List`, `Get`, `Create`, `Update`)
3.2 Implement `MemoryStore` with `map[string]domain.Bike` and `sync.RWMutex`
3.3 Generate IDs in the store (incrementing counter, e.g. `b-1`, `b-2`)
3.4 Enforce unique bike names (case-insensitive, including retired bikes) → `ErrConflict`
3.5 Return copies, never pointers into the map; list sorted by name
3.6 Tests: CRUD, not-found, duplicate name, concurrent access with `-race`

## 4. API Handlers

4.1 Add `BikeHandler` struct holding the store; register routes in `NewRouter`
4.2 Implement `GET /api/bikes` (hides retired; filters `?category=`, `?status=`) and `GET /api/bikes/{id}`
4.3 Implement `POST /api/bikes` and `PUT /api/bikes/{id}` (decode JSON, normalize, validate)
4.4 Implement `POST /api/bikes/{id}/retire`
4.5 Implement `GET /api/categories`
4.6 Map domain errors to status codes (`400`, `404`, `409`) with `{"error": "..."}`
4.7 Wire the store in `main.go`
4.8 `httptest` tests for happy paths and each error status

## 5. Frontend: Bikes Page

5.1 Small `api.js` helper for `fetch` with JSON and error handling
5.2 Bike table (name, category, hourly price, status, description, actions)
5.3 Toggle "show retired bikes"
5.4 Create/edit form (name, category dropdown from `/api/categories`, status, description)
5.5 Retire with confirmation dialog
5.6 Show API validation errors in the form

## 6. Wrap-up

6.1 Manual check in the browser: create, edit, retire and reactivate one bike per category
6.2 `gofmt`, `go vet`, `go test -race ./...`
6.3 Tick off Phase 1 in [roadmap.md](../roadmap.md) and the acceptance criteria in requirements.md
6.4 Commit

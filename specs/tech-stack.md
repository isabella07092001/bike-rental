# Tech Stack

Prototype project. Guiding principle: **quickly understandable and easy to change –
simple and clear beats elegant and complex.**

## Stack

| Layer | Choice | Notes |
|-------|--------|-------|
| Backend | **Go** (standard library `net/http`) | "Object-oriented" style: structs + methods, interfaces only where useful |
| Data storage | **In-memory store** | Maps guarded by `sync.RWMutex`; no Postgres/SQLite, no ORM |
| File storage (ID copies) | In-memory (`[]byte` + content type) | Lost on restart, acceptable for a prototype |
| Frontend | **Plain HTML + vanilla JavaScript** | Minimal JS, talks to the backend only via the JSON API |
| Version control | Local Git, GitHub if needed | Short, imperative commit messages |

A Flutter frontend is an option for later (see roadmap); it would use the same Go API and keep
state management simple (Provider or Riverpod only if needed).

## Project Structure

```
bike-rental/
├── main.go                  # entry point: flags (-demo), stores, HTTP server
├── internal/
│   ├── domain/              # core structs + methods: business rules, independent of HTTP
│   │   ├── category.go      # Category, hourly price
│   │   ├── bike.go          # Bike, Status
│   │   ├── person.go        # Person
│   │   ├── iddocument.go    # uploaded ID copy (type, size)
│   │   ├── period.go        # rental period: duration, billing, 30-min gap
│   │   ├── reservation.go   # Reservation: price, 24 h deadline, phase
│   │   └── errors.go        # ErrInvalid, ErrNotFound, ErrConflict, ErrTooLarge
│   ├── store/               # thread-safe in-memory stores behind interfaces
│   ├── booking/             # rules across stores: who may book which bike when
│   ├── dashboard/           # dashboard figures
│   ├── seed/                # demo data for -demo
│   └── handler/             # thin HTTP handlers (JSON), router, static files
├── web/                     # frontend: index.html, one script per page, style.css, logo.svg
└── specs/                   # stakeholder input, mission, tech stack, roadmap, feature specs
```

## Architecture Rules

- **Domain first** – all business rules live in `internal/domain` as methods on structs,
  independent of HTTP. Rules that need several stores live in `internal/booking`.
- **Methods instead of free functions** where it makes sense.
- **Handlers are thin** – parse request, call domain/booking/store, write JSON response.
- **Interfaces only where really needed** – the store interfaces, so the in-memory
  implementation can be swapped for a database later.
- **No dependency injection framework** – dependencies are passed explicitly (`handler.Deps`).
- **Thread safety** – the in-memory stores must be safe for concurrent access.
- **No globals** – except a single store instance if necessary.
- **Errors** – wrap with `fmt.Errorf("%w: ...", err)`, never swallow. Validation errors map to
  `400`, missing entities to `404`, rule conflicts to `409`, too-large uploads to `413`.
- **Frontend ↔ backend** – the frontend calls the Go JSON endpoints only; it never relies on
  internal data structures.
- **No new dependencies** without asking first.
- **No secrets in code**, not even in the prototype.
- **Generated files** are never edited by hand.

## API Style

- JSON over HTTP, resource-oriented, served by the same Go server as the frontend (`web/`):
  - `/api/bikes`, `/api/bikes/{id}`, `/api/bikes/{id}/retire`, `/api/bikes/available`, `/api/categories`
  - `/api/persons`, `/api/persons/{id}`, `/api/persons/{id}/id-document`, `/api/persons/{id}/archive`, `/api/persons/{id}/reactivate`
  - `/api/reservations`, `/api/reservations/{id}`, `/api/reservations/{id}/cancel`, `/api/reservations/quote`
  - `/api/dashboard`, `/api/health`
- Nothing is hard-deleted: bikes are retired, persons archived, reservations cancelled.
- Times are exchanged as RFC 3339 strings; money as integer cents.
- Errors as `{"error": "<message>"}` with a user-facing message.

## Commands

| Task | Command |
|------|---------|
| Run | `go run .` – with demo data: `go run . -demo` |
| Tests | `go test ./...` (race detector: `go test -race ./...`, needs cgo + gcc) |
| Format | `gofmt -w .` / `go fmt ./...` |
| Vet | `go vet ./...` |
| Lint (if set up) | `golangci-lint run` |
| Frontend | served by the Go server at <http://localhost:8080> |
| Frontend (Flutter, if added) | `flutter run` |

## Testing

- Table-driven unit tests for every business rule in `internal/domain`, including boundaries.
- Store tests including concurrent access.
- Booking and dashboard tests with an injected clock.
- Handler tests with `net/http/httptest` for happy paths and every error status.

## Working Rules

- Prototype focus: rather simple and clear than elegant and complex.
- When something is unclear: **ask instead of guessing**.
- Commit messages: short and imperative.

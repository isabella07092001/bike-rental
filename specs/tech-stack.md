# Tech Stack

Source: [TECH-STACK.md](../TECH-STACK.md). Guiding principle: **prototype focus — simple and clear
beats elegant and complex.**

## Stack

| Layer | Choice | Notes |
|-------|--------|-------|
| Backend | **Go** (standard library `net/http`) | "Object-oriented" style: structs + methods, interfaces only where useful |
| Data storage | **In-memory store** | Maps guarded by `sync.Mutex`/`sync.RWMutex`; no Postgres/SQLite, no ORM |
| File storage (ID copies) | In-memory (`[]byte` + content type) | Lost on restart, acceptable for a prototype |
| Frontend | **Plain HTML + vanilla JavaScript** | Minimal JS, talks to the backend only via the JSON API |
| Version control | Local Git, GitHub if needed | Short, imperative commit messages |

No new dependencies without asking first. No DI framework.

## Project Structure

```
bike-rental/
├── main.go                  # or cmd/server/main.go – entry point, HTTP server, routing
├── internal/
│   ├── domain/              # core structs + methods (business logic & rules)
│   │   ├── bike.go          # Bike, Category, hourly price
│   │   ├── person.go        # Person, identification data
│   │   └── reservation.go   # Reservation, duration/price/cancellation rules
│   ├── store/               # thread-safe in-memory store (Store interface + MemoryStore)
│   └── handler/             # HTTP handlers (JSON), call into domain + store
├── web/                     # static frontend (index.html, *.js, *.css)
└── specs/                   # mission, tech stack, roadmap, feature specs
```

## Architecture Rules

- **Domain first** – all business rules (duration limits, price calculation, cancellation
  deadline, ID requirement) live in `internal/domain` as methods on structs, independent of HTTP.
- **Handlers are thin** – parse request, call domain/store, write JSON response.
- **Store interface** – `internal/store` exposes an interface so the in-memory implementation
  can be swapped later. This is the only interface introduced up front.
- **Thread safety** – the in-memory store must be safe for concurrent access.
- **No globals** – except a single store instance if necessary.
- **Errors** – wrap with `fmt.Errorf("...: %w", err)`, never swallow. Domain validation errors
  map to HTTP `400`/`409`, missing entities to `404`.
- **Frontend ↔ backend** – the frontend calls the Go JSON endpoints only; it never relies on
  internal data structures.
- **No secrets in code**, not even in the prototype.
- **Generated files** are never edited by hand.

## API Style

- JSON over HTTP, resource-oriented, e.g.:
  - `GET/POST /api/bikes`, `GET/PUT/DELETE /api/bikes/{id}`
  - `GET/POST /api/persons`, `GET/PUT/DELETE /api/persons/{id}`
  - `POST /api/persons/{id}/id-document` (multipart upload), `GET /api/persons/{id}/id-document`
  - `GET/POST /api/reservations`, `GET /api/reservations/{id}`, `POST /api/reservations/{id}/cancel`
  - `GET /api/dashboard`
- Static frontend served by the same Go server (e.g. from `web/`).
- Times are exchanged as RFC 3339 strings; money as integer cents.

## Commands

| Task | Command |
|------|---------|
| Run backend | `go run .` (or `go run ./cmd/server`) |
| Tests | `go test ./...` |
| Format | `gofmt -w .` / `go fmt ./...` |
| Lint (if set up) | `golangci-lint run` |
| Frontend | served by the Go server; or open the files / use a small static server |

## Testing

- Unit tests for every business rule in `internal/domain` (table-driven tests).
- Store tests including concurrent access (`go test -race ./...`).
- Handler tests with `net/http/httptest` for the main happy and error paths.

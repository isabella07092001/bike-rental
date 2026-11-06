A bike rental management system for a bike rental shop owned by Alice and Bob.

Functional Requirements
- Dashboard
- Bike Management
- Person Management
- Reservation Management

Rules
- every reservation must contain the identification number of the person renting the bikes
and a copy of the identification document must be uploaded
- bikes can be rented on an hourly basis (minimum rental duration is 30 minutes, maximal
rental duration is two days), rental prices are calculated per hour
- rental prices depend on the bike category: children's bike, city bike, mountain bike, racing bike
- a reservation can only be cancelled up until 24 hours before the rental start

---

# Power Bike – Running the Prototype

Specs: [mission](specs/mission.md) · [tech stack](specs/tech-stack.md) · [roadmap](specs/roadmap.md)
· feature specs in [`specs/`](specs).

## Requirements

- Go 1.27.3 or newer (see `go.mod`). No other dependencies.
- A browser. The frontend is plain HTML/JS and is served by the Go server.

## Start

```bash
go run .
```

Open <http://localhost:8080>. The app starts on the dashboard.

To start with **demo data** (10 bikes, 4 customers with sample ID copies, reservations that are
active, completed, upcoming and cancelled around the current time):

```bash
go run . -demo
```

Use another port with the `PORT` environment variable, e.g. `PORT=9090 go run .`.

All data is kept **in memory only** and is lost when the server stops.

## Test

```bash
go test ./...
```

```bash
gofmt -l . && go vet ./...
```

The race detector (`go test -race ./...`) additionally needs cgo and a C compiler (gcc).

## Project Structure

| Path | Content |
|------|---------|
| `main.go` | entry point: flags, stores, HTTP server |
| `internal/domain` | business rules: bikes, categories and prices, persons, ID copies, rental periods, reservations |
| `internal/store` | thread-safe in-memory stores behind interfaces |
| `internal/booking` | booking rules across stores: who may book which bike when, price quote, availability |
| `internal/dashboard` | figures for the dashboard |
| `internal/handler` | JSON API and static file serving |
| `internal/seed` | demo data for `-demo` |
| `web/` | frontend: `index.html`, one script per page, `style.css`, `logo.svg` |

## Business Rules (summary)

- A reservation needs a registered, active person with ID number and uploaded ID copy (PDF/JPG/PNG, ≤ 5 MB).
- Rentals last 30 minutes to 48 hours and are billed **per started hour**.
- Hourly prices depend on the category (placeholders: children 4 €, city 6 €, mountain 9 €, racing 10 €).
- A bike needs at least 30 minutes between two rentals.
- Reservations can be changed or cancelled until 24 hours before the start.
- Bikes are retired and persons archived instead of deleted; both are blocked while they have open reservations.

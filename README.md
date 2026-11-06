# Power Bike

Bike rental management prototype for **Power Bike**, the bike rental shop of Alice and Bob.

Specs: [stakeholder input](specs/stakeholder-input.md) · [mission](specs/mission.md) · [tech stack](specs/tech-stack.md)
· [roadmap](specs/roadmap.md)
· feature specs in [`specs/`](specs).

## Getting started

See **[GETTING-STARTED.md](GETTING-STARTED.md)**: start the server with `go run . -demo`, then
double-click **`open-app.html`** to open the app in your browser.

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

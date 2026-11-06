# Roadmap

Small, incremental phases. Each phase ends with a runnable application, passing tests
(`go test ./...`) and a commit. Business rules from [mission.md](mission.md) are implemented in
the domain layer and tested before the UI is built on top.

## Phase 0 – Project Skeleton

- [x] `go mod init`, `main.go` with HTTP server and routing
- [x] Folder structure: `internal/domain`, `internal/store`, `internal/handler`, `web/`
- [x] Serve a static `web/index.html` with navigation (Dashboard, Bikes, Persons, Reservations)
- [x] Health endpoint `GET /api/health`
- [x] Initialize Git, add `.gitignore`

**Done when:** `go run .` serves the empty frontend shell.

## Phase 1 – Bike Management

- [x] Domain: `Bike` (ID, name, category, status, description), `Category` enum
      (children's, city, mountain, racing) with hourly price per category
- [x] In-memory store: `Store` interface + thread-safe `MemoryStore` for bikes
- [x] API: create, read, update, retire for `/api/bikes`
- [x] Frontend: bike list, create/edit form, retire (no hard delete)
- [x] Tests: domain validation, store (incl. `-race`), handlers (`-race` pending: no C compiler)

**Done when:** bikes of all four categories can be managed in the browser.

## Phase 2 – Person Management

- [x] Domain: `Person` (ID, first/last name, identification number, phone, archived)
- [x] Store + API for `/api/persons`: create (with ID copy), read, update, archive, reactivate
- [x] ID copy (PDF/JPG/PNG, ≤ 5 MB) required at creation; replace & view (`multipart/form-data`, in memory)
- [x] Frontend: person list with search, create form with upload, edit, archive (no hard delete)
- [x] Tests: validation (identification number required), upload handling (type, size, missing file)

**Done when:** a person with identification number and uploaded ID copy can be created.

## Phase 3 – Reservation Management

- [x] Domain: `Reservation` (ID, person, ID number snapshot, one or more bikes, start, end, status, total price)
- [x] Rule: person must have identification number **and** uploaded ID document
- [x] Rule: duration between 30 minutes and 48 hours
- [x] Rule: price = started hours (rounded up) × hourly category price (summed over all bikes)
- [x] Rule: a bike cannot be double-booked; at least 30 minutes between two rentals
- [x] Rule: cancellation and editing only allowed ≥ 24 hours before rental start
- [x] Hooks: retiring a bike / archiving a person with open reservations is rejected
- [x] API: create, list, get, edit, cancel reservations; price quote; available bikes
- [x] Frontend: reservation list, create/edit form with live price preview, cancel button
      (disabled when the deadline has passed)
- [x] Tests: table-driven tests for every rule incl. boundary values
      (29/30 min, 48 h/48 h 1 min, 60/61 min, 29/30 min gap, exactly 24 h before start)

**Done when:** all stakeholder rules are enforced by the backend and covered by tests.

## Phase 4 – Dashboard

- [x] API: `GET /api/dashboard` with aggregated figures
- [x] Fleet now: total, available, rented, in maintenance – overall and per category (retired excluded)
- [x] Upcoming reservations of the next 7 days (no revenue, no warnings)
- [x] Frontend: dashboard as start page, reloaded when opened

**Done when:** Alice and Bob see the current state of the shop at a glance.

## Phase 5 – Polish (optional)

- [x] Branding: shop name **Power Bike**, pink/rose design (colour tokens in `web/style.css`),
      logo `web/logo.svg` (pink bicycle with a woman rider with a ponytail), also used as favicon
- [x] Seed data for demos: `go run . -demo` (bikes in all states, customers with sample ID card,
      reservations active / completed / upcoming / cancelled around the current time)
- [x] Consistent error messages: one style for all API errors ("Name is required" instead of
      "invalid input: name is required"), clear message when the server is unreachable, same error box everywhere
- [x] Search/filters: bikes (name, description, category), persons (name, ID number, phone),
      reservations (person, ID number, bike; phase)
- [x] README section on how to run and test the prototype

## Later / Out of Scope for the Prototype

- Persistent database (swap `MemoryStore` behind the `Store` interface)
- Authentication and user roles
- Customer self-service booking, payment, invoicing
- Flutter frontend variant

## Open Questions to Resolve

See [mission.md → Open Questions](mission.md#open-questions). Billing, multi-bike reservations
and the buffer between rentals were decided in the Phase 3 spec; concrete prices are still open.

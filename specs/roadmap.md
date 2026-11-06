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

- [ ] Domain: `Bike` (ID, name/label, category, status), `Category` enum
      (children's, city, mountain, racing) with hourly price per category
- [ ] In-memory store: `Store` interface + thread-safe `MemoryStore` for bikes
- [ ] API: CRUD for `/api/bikes`
- [ ] Frontend: bike list, create/edit form, delete
- [ ] Tests: domain validation, store (incl. `-race`), handlers

**Done when:** bikes of all four categories can be managed in the browser.

## Phase 2 – Person Management

- [ ] Domain: `Person` (ID, name, contact data, identification number)
- [ ] Store + API: CRUD for `/api/persons`
- [ ] ID document upload & download (`multipart/form-data`, stored in memory)
- [ ] Frontend: person list, create/edit form, ID document upload
- [ ] Tests: validation (identification number required), upload handling

**Done when:** a person with identification number and uploaded ID copy can be created.

## Phase 3 – Reservation Management

- [ ] Domain: `Reservation` (ID, person, bikes, start, end, status, total price)
- [ ] Rule: person must have identification number **and** uploaded ID document
- [ ] Rule: duration between 30 minutes and 48 hours
- [ ] Rule: price = hours × hourly category price (summed over all bikes)
- [ ] Rule: a bike cannot be double-booked for overlapping periods
- [ ] Rule: cancellation only allowed ≥ 24 hours before rental start
- [ ] API: create, list, get, cancel reservations
- [ ] Frontend: reservation list, create form with live price preview, cancel button
      (disabled when the deadline has passed)
- [ ] Tests: table-driven tests for every rule incl. boundary values
      (29/30 min, 48 h/48 h 1 min, exactly 24 h before start)

**Done when:** all stakeholder rules are enforced by the backend and covered by tests.

## Phase 4 – Dashboard

- [ ] API: `GET /api/dashboard` with aggregated figures
- [ ] Figures: bikes per category and status (available / rented), active reservations,
      upcoming reservations (e.g. next 24 h), today's expected revenue
- [ ] Frontend: dashboard as start page

**Done when:** Alice and Bob see the current state of the shop at a glance.

## Phase 5 – Polish (optional)

- [ ] Seed data for demos (a few bikes, persons, reservations)
- [ ] Consistent error messages in the UI
- [ ] Simple filtering/search in lists
- [ ] README section on how to run and test the prototype

## Later / Out of Scope for the Prototype

- Persistent database (swap `MemoryStore` behind the `Store` interface)
- Authentication and user roles
- Customer self-service booking, payment, invoicing
- Flutter frontend variant

## Open Questions to Resolve

See [mission.md → Open Questions](mission.md#open-questions). Pricing granularity and
multi-bike reservations should be clarified **before Phase 3**.

# Phase 3 – Reservation Management: Plan

Requirements: [requirements.md](requirements.md). Each task group ends with passing tests
(`go test ./...`) and can be committed on its own.

## 1. Domain: Period and Price

1.1 Define `MinRentalDuration` (30 min), `MaxRentalDuration` (48 h), `BookingBuffer` (30 min)
1.2 Add `Period{Start, End}` with `Validate(now)` (end after start, min/max, start in future)
1.3 Add `Period.BilledHours()` (round up to full hours)
1.4 Add `Period.ConflictsWith(other)` including the 30-minute buffer
1.5 Add `CalculatePrice(billedHours, bikes)` (sum of hourly prices × hours)
1.6 Table-driven tests incl. boundaries: 29/30 min, 48 h / 48 h + 1 min, 60/61 min, 29/30 min gap

## 2. Domain: Reservation

2.1 Define `Reservation`, `ReservedBike` (bike ID, category, hourly price snapshot) and `ReservationStatus`
2.2 Add `CanChange(now)` (booked and `now ≤ Start − 24 h`)
2.3 Add `Cancel(now)` and `Reschedule(period, bikes, now)` using `CanChange`
2.4 Add `Phase(now)` → upcoming / active / completed / cancelled
2.5 Add `IsOpen(now)` (booked and not completed) for the retire/archive hooks
2.6 Tests: deadline boundary (exactly 24 h, 24 h − 1 s), phases, cancel/edit after deadline, already cancelled

## 3. Store

3.1 Define `ReservationStore` interface (`List`, `Get`, `Create`, `Update`)
3.2 Implement in-memory store with `sync.RWMutex`, IDs `r-1`, `r-2`, …
3.3 `Create`/`Update` check bike conflicts **inside the lock** (excluding the reservation itself on update)
3.4 Return copies (incl. bike slices); list sorted by start
3.5 Tests incl. concurrent booking of the same bike (exactly one succeeds)

## 4. Booking Service

4.1 Add `Booking` struct in `internal/booking` that combines person, bike and reservation stores and a clock
4.2 `Book(personID, bikeIDs, period)`: check person (exists, active, ID number + copy) and bikes
    (exist, no duplicates, `available`), snapshot ID number and prices, create
4.3 `Change(id, bikeIDs, period)`: deadline, re-check bikes and period, recalculate price
4.4 `Cancel(id)`
4.5 `Quote(bikeIDs, period, excludeID)` and `AvailableBikes(period, excludeID)`
4.6 `HasOpenReservations(bikeID / personID)` for the hooks
4.7 Tests with an injected clock

## 5. API Handlers

5.1 `ReservationHandler`; register routes; wire everything in `main.go`
5.2 `GET /api/reservations` (filters) and `GET /api/reservations/{id}` (person/bike names, phase, `canChange`)
5.3 `POST /api/reservations` and `PUT /api/reservations/{id}`
5.4 `POST /api/reservations/{id}/cancel`
5.5 `POST /api/reservations/quote` and `GET /api/bikes/available`
5.6 Hooks: retire bike / archive person → `409` if open reservations exist
5.7 `httptest` tests for each rule and status code

## 6. Frontend: Reservations Page

6.1 Reservation table with phase badges, phase filter, edit/cancel buttons (disabled with hint after deadline)
6.2 Create form: active-person select, start/end inputs, available-bike checkboxes reloaded on period change
6.3 Live price preview via `/api/reservations/quote`
6.4 Edit mode: person fixed, own bikes selectable, save via `PUT`
6.5 Error display; show retire/archive conflicts on the Bikes/Persons pages

## 7. Wrap-up

7.1 Manual check in the browser: book several bikes, double-book attempt, gap < 30 min, too short/long,
    edit and cancel before/after deadline, retire booked bike, archive person with booking
7.2 `gofmt`, `go vet`, `go test ./...`
7.3 Tick off Phase 3 in [roadmap.md](../roadmap.md), the acceptance criteria and answered questions in mission.md
7.4 Commit

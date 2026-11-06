# Phase 4 – Dashboard: Plan

Requirements: [requirements.md](requirements.md). Each task group ends with passing tests
(`go test ./...`) and can be committed on its own.

## 1. Aggregation

1.1 Add package `internal/dashboard` with a `Builder` (bike, person, reservation stores + clock)
1.2 Define the response structs (`Summary`, `Fleet`, `CategoryFleet`, `Upcoming`, `UpcomingReservation`)
1.3 Determine the bikes rented now from booked reservations in phase `active`
1.4 Fleet figures: each non-retired bike in exactly one state (rented now > maintenance > available),
    overall and for all four categories in display order
1.5 Upcoming: booked reservations with `now < start ≤ now + 7 days`, sorted by start,
    with person name and bike names

## 2. Tests

2.1 Fixture with bikes in all states (incl. retired), persons and reservations around a fixed "now"
2.2 Fleet counts add up; retired excluded; rented bike in maintenance counts as rented
2.3 Rented-now boundaries: exactly at start (rented), exactly at end (not rented), cancelled (not rented)
2.4 Upcoming boundaries: start = now (not upcoming, active), start = now + 7 days (included),
    start = now + 7 days + 1 s (excluded); cancelled excluded; sort order
2.5 Empty stores: all zeros, four categories, empty list

## 3. API

3.1 `GET /api/dashboard` handler; register in `NewRouter`
3.2 `httptest` test for status, shape and a few values

## 4. Frontend

4.1 Figure tiles (total, available now, rented now, maintenance)
4.2 Fleet table per category
4.3 Upcoming list (next 7 days) with empty state and link to the Reservations page
4.4 "Generated at" time; reload whenever `#dashboard` is opened; dashboard as start page

## 5. Wrap-up

5.1 Manual check in the browser with test data (active rental, maintenance bike, retired bike,
    reservation in 3 days and in 8 days, cancelled reservation)
5.2 `gofmt`, `go vet`, `go test ./...`
5.3 Tick off Phase 4 in [roadmap.md](../roadmap.md) and the acceptance criteria
5.4 Commit

# Phase 3 – Reservation Management: Requirements

Status: **Implemented** (2026-11-06)

## Context

Reservations connect persons (Phase 2) with bikes (Phase 1) for a time period. This is the
phase where most stakeholder rules from [mission.md](../mission.md) are enforced:

1. **Identification required** – the reservation contains the person's identification number,
   and an ID copy must be uploaded. Since Phase 2 every person has an ID copy, so the check
   is: the person exists, is not archived, and has an ID number and ID copy.
2. **Hourly rentals** – minimum 30 minutes, maximum 2 days (48 hours), price per hour.
3. **Category-based pricing** – hourly price from the bike's category (placeholders 4/6/9/10 €).
4. **Cancellation deadline** – cancellation only up until 24 hours before the rental start.

Phase 1 and 2 left two hooks for this phase: retiring a bike and archiving a person must be
blocked while they have upcoming or active reservations.

## Scope

### In scope

- Domain model for `Reservation` with all business rules
- Availability check: a bike cannot be double-booked, and needs a 30-minute gap between rentals
- Price calculation per started hour from the bike categories
- Reservations with one or more bikes
- Editing period and bikes until 24 hours before the start
- Thread-safe in-memory store for reservations
- API to create, list, view, edit and cancel reservations, plus a price/availability quote
- Blocking retire/archive of bikes/persons with upcoming or active reservations
- "Reservations" page: list, create/edit form with live price preview, cancel
- Tests for all rules including boundary values

### Out of scope

- Changing the person of a reservation (cancel and rebook instead)
- Check-out/return handling, late fees, damage (phase is derived from time only)
- Opening hours (any start/end time is allowed)
- Payments, invoices, notifications
- Dashboard figures (Phase 4)
- Persistence across restarts

## Domain Model

### Reservation

| Field | Type | Rules |
|-------|------|-------|
| `ID` | string | assigned by the store (`r-1`, `r-2`, …) |
| `PersonID` | string | must reference an active (not archived) person; cannot be changed |
| `IDNumber` | string | snapshot of the person's ID number at booking time (Business Rule 1) |
| `Bikes` | list of `{bikeId, category, hourlyPriceCents}` | ≥ 1 bike, no duplicates; each bike exists and has status `available`; category and price are snapshots |
| `Start`, `End` | time | `End − Start` between 30 min and 48 h (both inclusive); `Start` in the future |
| `BilledHours` | int | duration rounded **up** to full hours |
| `TotalPriceCents` | int | `BilledHours × Σ hourly prices of all bikes`, stored as snapshot |
| `Status` | `booked` \| `cancelled` | |
| `CreatedAt`, `UpdatedAt`, `CancelledAt` | time | set by the server |

Derived for display (not stored): **phase** = `upcoming` (before start), `active`
(start ≤ now < end), `completed` (after end) or `cancelled`.

### Price examples (per started hour)

| Duration | Billed hours | 1 city bike (6 €/h) | city + children (6 + 4 €/h) |
|----------|--------------|---------------------|-----------------------------|
| 30 min | 1 | 6 € | 10 € |
| 1 h | 1 | 6 € | 10 € |
| 1 h 1 min | 2 | 12 € | 20 € |
| 2 h 30 min | 3 | 18 € | 30 € |
| 48 h | 48 | 288 € | 480 € |

### Availability with 30-minute buffer

Two reservations of the same bike conflict unless there are **at least 30 minutes** between
them. A new period `[start, end)` conflicts with a booked period `[S, E)` if
`start < E + 30 min` and `S < end + 30 min`. Cancelled reservations never block.

| Existing | New | Result |
|----------|-----|--------|
| 10:00–12:00 | 12:30–14:00 | ✅ exactly 30 min gap |
| 10:00–12:00 | 12:29–14:00 | ❌ gap too small |
| 10:00–12:00 | 08:00–09:30 | ✅ exactly 30 min gap |
| 10:00–12:00 | 11:00–13:00 | ❌ overlap |

### Rules

| Rule | Status code |
|------|-------------|
| Person must exist (`404`), be active and have ID number + ID copy (`400`) | `400` / `404` |
| At least one bike, no duplicates; every bike exists (`404`) and is `available` (`400`) | `400` / `404` |
| No conflict with another `booked` reservation of the same bike incl. 30-min buffer | `409` |
| 30 min ≤ duration ≤ 48 h; end after start | `400` |
| Start must be in the future | `400` |
| Cancel only if status is `booked` and `now ≤ Start − 24 h` | `409` |
| Edit only if status is `booked` and `now ≤ Start − 24 h` (deadline of the **current** start); the new period and bikes must pass all booking rules again; price is recalculated | `409` / `400` |

## API

| Method | Path | Success | Errors |
|--------|------|---------|--------|
| `GET` | `/api/reservations` | `200` list sorted by start; filters `?phase=`, `?personId=`, `?bikeId=` | `400` |
| `POST` | `/api/reservations` | `201` created reservation — body `{personId, bikeIds, start, end}` | `400`, `404`, `409` |
| `GET` | `/api/reservations/{id}` | `200` reservation incl. person name, bike names, phase, `canChange` | `404` |
| `PUT` | `/api/reservations/{id}` | `200` updated reservation — body `{bikeIds, start, end}` | `400`, `404`, `409` |
| `POST` | `/api/reservations/{id}/cancel` | `200` cancelled reservation | `404`, `409` |
| `POST` | `/api/reservations/quote` | `200` `{billedHours, totalPriceCents, conflicts}` for `{bikeIds, start, end[, reservationId]}` | `400` |
| `GET` | `/api/bikes/available?start=&end=[&excludeReservation=]` | `200` bookable bikes free in the period | `400` |

Times are RFC 3339 strings; money is integer cents. Error shape: `{"error": "<message>"}`.

## Frontend

- Reservation list: period, person, bikes, price, phase badge, actions
  (edit / cancel, disabled with a hint once the 24 h deadline has passed); filter by phase
- Create form: person (select from active persons), start and end (date + time), checkboxes of
  bikes available in the chosen period (reloaded when the period changes), live price preview
- Edit form: same form with person fixed; the reservation's own bikes stay selectable
- Error messages from the API next to the form

## Decisions

| # | Decision | Reason |
|---|----------|--------|
| D1 | Billing per **started hour** (round up) | Chosen by product owner; simple and common for rentals |
| D2 | One reservation can contain **several bikes**; price is the sum | Chosen by product owner; family/group rentals |
| D3 | **30-minute buffer** between two reservations of the same bike | Chosen by product owner; time to check and clean the bike |
| D4 | Reservations can be **edited until 24 h before start** (period and bikes, not the person) | Chosen by product owner; same deadline as cancellation; changing the person would break the ID snapshot |
| D5 | Price, categories and ID number are snapshots stored with the reservation | Later price/category/person changes do not alter existing bookings |
| D6 | Only bikes with status `available` can be booked | Maintenance/retired bikes must not be rented |
| D7 | Stored status `booked`/`cancelled`; phase derived from time | No check-out handling in the prototype |
| D8 | Cancelled reservations are kept, never deleted | History; consistent with retiring/archiving |
| D9 | Retiring a bike / archiving a person with upcoming or active booked reservations → `409` | Hooks announced in Phase 1 and 2 |
| D10 | Setting a booked bike to `maintenance` stays allowed | Repairs happen; the shop handles the conflict manually (shown in Phase 4) |
| D11 | Availability check and insert happen atomically in the store | No double booking under concurrent requests |
| D12 | The server clock is injectable (`now func() time.Time`) | Deterministic tests for deadlines |
| D13 | UI uses the browser's local time (the shop's PC, Europe/Berlin); API uses RFC 3339 | Unambiguous API, familiar UI |
| D14 | No opening hours | Not requested by stakeholders; can be added later |

## Acceptance Criteria

- [x] A reservation with one or more bikes can be created, listed, viewed, edited and cancelled in the browser.
- [x] Every business rule is enforced by the backend and covered by tests, including boundaries
      (29/30 min, 48 h / 48 h + 1 min, 60/61 min billing, exactly 24 h before start, 29/30 min gap).
- [x] A bike cannot be double-booked; only available bikes are offered.
- [x] Price preview and stored price match "per started hour".
- [x] Editing and cancelling are rejected after the 24 h deadline.
- [x] Retiring a bike / archiving a person with upcoming reservations is rejected with `409`.
- [x] `go test ./...` passes, including a concurrent booking test (exactly one of two competing bookings succeeds).

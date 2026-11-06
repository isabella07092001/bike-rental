# Phase 4 – Dashboard: Requirements

Status: **Implemented** (2026-11-06)

## Context

Alice and Bob want to see the state of the shop at a glance when they open the application
(see [mission.md](../mission.md), Core Capability 1). Phases 1–3 provide all the data: bikes
with category and status, and reservations with period, bikes and phase.

The dashboard focuses on two questions:

1. **What does the fleet look like right now?** – how many bikes are available, rented or in
   maintenance, overall and per category.
2. **What is coming up?** – the booked reservations of the next 7 days.

The dashboard is **read-only**: it aggregates existing data and links to the pages where
something can be changed. It becomes the start page of the application.

## Scope

### In scope

- One aggregated API endpoint `GET /api/dashboard`
- Fleet status right now: total, available now, rented now, in maintenance – overall and per category
- Upcoming reservations: booked reservations starting within the next 7 days
- Dashboard page as start page (`#dashboard`), reloaded when it is opened
- Tests for the aggregation with a fixed clock

### Out of scope

- Revenue figures (decided: not on the dashboard)
- Today's pick-up/return schedule as a separate section
- Warnings (e.g. a booked bike in maintenance, Phase 3 decision D10) – possible follow-up
- Automatic refresh (decided: only when opened)
- Editing anything on the dashboard; charts, exports, statistics
- Persistence across restarts

## Data Definitions

All figures are calculated by the server at request time (`now`).

| Term | Definition |
|------|------------|
| **Fleet** | all bikes except `retired` |
| **Rented now** | bike in the fleet that is part of a booked reservation with `start ≤ now < end` (phase `active`) |
| **In maintenance** | bike with status `maintenance` that is not rented now |
| **Available now** | bike with status `available` that is not rented now |
| **Upcoming (next 7 days)** | booked reservation with `now < start ≤ now + 7 days`, sorted by start |

Every fleet bike is counted in exactly one of *rented now*, *in maintenance* or
*available now*, so `total = availableNow + rentedNow + maintenance`. A bike that is rented
now but was set to maintenance counts as *rented now* (it is physically out).

## API

`GET /api/dashboard` → `200`

```json
{
  "generatedAt": "2026-11-06T12:00:00+01:00",
  "fleet": {
    "total": 5, "availableNow": 3, "rentedNow": 1, "maintenance": 1,
    "byCategory": [
      { "category": "children", "label": "Children's bike", "total": 1, "availableNow": 1, "rentedNow": 0, "maintenance": 0 },
      { "category": "city",     "label": "City bike",       "total": 2, "availableNow": 1, "rentedNow": 1, "maintenance": 0 }
    ]
  },
  "upcoming": {
    "days": 7,
    "count": 2,
    "reservations": [
      { "id": "r-2", "start": "...", "end": "...", "personName": "Alice Meyer",
        "bikes": ["City-01", "Kids-01"], "totalPriceCents": 2000 }
    ]
  }
}
```

- `byCategory` always contains all four categories in display order, also with zero counts.

## Frontend

- Figure tiles: fleet total, available now, rented now, in maintenance
- Table per category with the same four figures
- List "Upcoming – next 7 days": date/time, person, bikes, price; empty state text;
  link to the Reservations page
- Shows when the data was generated; reloaded each time the dashboard is opened
- Start page: the app opens on `#dashboard`

## Decisions

| # | Decision | Reason |
|---|----------|--------|
| D1 | Sections: fleet status now + upcoming reservations | Chosen by product owner |
| D2 | No revenue on the dashboard | Chosen by product owner |
| D3 | Upcoming window: next 7 days | Chosen by product owner; weekly planning view |
| D4 | Refresh only when the dashboard is opened (no polling) | Chosen by product owner; simplest |
| D5 | One endpoint returns all dashboard data | One request per view; aggregation in Go and testable |
| D6 | Aggregation in `internal/dashboard` with an injected clock | Deterministic tests for "now" and the 7-day window |
| D7 | Every fleet bike counts in exactly one state; "rented now" wins over "maintenance" | Figures add up; a rented bike is physically out of the shop |
| D8 | Retired bikes and cancelled reservations are excluded | They are history, not current state |
| D9 | The dashboard is read-only | Changes stay on the existing pages with their rules |

## Acceptance Criteria

- [x] The dashboard is the start page and shows fleet status and upcoming reservations.
- [x] `total = availableNow + rentedNow + maintenance`, overall and per category; retired bikes are excluded.
- [x] A rental counts as "rented now" exactly from its start until (excluding) its end.
- [x] Upcoming lists booked reservations with `now < start ≤ now + 7 days` (boundaries tested); cancelled ones are excluded.
- [x] The dashboard reloads its data whenever it is opened.
- [x] `go test ./...` passes.

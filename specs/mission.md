# Mission

## Purpose

Build a **bike rental management system** for the bike rental shop owned by **Alice and Bob**.
The system replaces manual bookkeeping with a simple web application that lets the owners
manage their bikes, their customers and their reservations in one place.

This is a **prototype**: the priority is to be quickly understandable and easy to change,
not to be production-ready.

## Users

| User | Goal |
|------|------|
| Shop owners / staff (Alice, Bob) | See the state of the shop at a glance, maintain the bike fleet, register customers, create and manage reservations |

Customers do not use the system directly in the prototype; staff enter reservations on their behalf.

## Core Capabilities

1. **Dashboard** – overview of the shop: bikes available / rented, upcoming and active reservations.
2. **Bike Management** – create, view, edit and remove bikes; each bike belongs to a category.
3. **Person Management** – create, view, edit and remove customers (persons who rent bikes), including their identification data.
4. **Reservation Management** – create, view and cancel reservations of one or more bikes for a person and a time period, with automatic price calculation.

## Business Rules

These rules come from the stakeholders and are non-negotiable. They belong in the domain layer
and must be covered by tests.

1. **Identification required** – every reservation must contain the identification number of the
   person renting the bikes, and a copy of the identification document must be uploaded.
2. **Hourly rentals** – bikes are rented on an hourly basis.
   - Minimum rental duration: **30 minutes**
   - Maximum rental duration: **2 days (48 hours)**
   - Rental prices are calculated **per hour**.
3. **Category-based pricing** – the hourly price depends on the bike category:
   - Children's bike
   - City bike
   - Mountain bike
   - Racing bike
4. **Cancellation deadline** – a reservation can only be cancelled up until **24 hours before
   the rental start**.

## Success Criteria

- Alice and Bob can perform all four core capabilities through the browser.
- Every business rule is enforced by the backend (not only the UI) and covered by a unit test.
- A new developer can understand the codebase and run it within a few minutes.

## Non-Goals (for the prototype)

- Online payment, invoicing or accounting
- Customer self-service / public booking portal
- User accounts, authentication and role management
- Persistent storage (data lives in memory and is lost on restart)
- Multi-shop / multi-location support

## Open Questions

To be clarified with the stakeholders before or during implementation (ask rather than guess):

- What are the concrete hourly prices per category?
- How are partial hours billed (e.g. 90 minutes → 1.5 h or rounded up to 2 h)? Is a 30-minute rental billed as half an hour or a full hour?
- Can one reservation contain several bikes (e.g. a family rental)? *(Assumption: yes.)*
- ~~Which file types / sizes are allowed for the ID document upload?~~ → PDF, JPG, PNG up to 5 MB (see person-management spec)
- May a bike be reserved back-to-back, or is a buffer between rentals needed?
- Are there opening hours that restrict rental start and end times?

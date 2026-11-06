# Stakeholder Input

> Original input from the stakeholders (Alice and Bob), kept unchanged.
> Derived specs: [mission.md](mission.md), [roadmap.md](roadmap.md) and the feature specs in this folder.

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

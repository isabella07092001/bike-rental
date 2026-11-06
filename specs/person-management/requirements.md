# Phase 2 – Person Management: Requirements

Status: **Implemented** (2026-11-06)

## Context

Every reservation must contain the identification number of the person renting the bikes,
and a copy of the identification document must be uploaded (see [mission.md](../mission.md),
Business Rule 1). Phase 2 lets Alice and Bob register these persons. Because the ID copy is
required **when a person is created**, every active person in the system already satisfies
Business Rule 1 when Phase 3 creates a reservation.

This phase builds on Phase 1: domain errors (`ErrInvalid`, `ErrNotFound`, `ErrConflict`),
the store pattern (`BikeStore` + `MemoryStore`), the JSON helpers in `internal/handler`, the
`api.js` helper in the frontend, and the "retire instead of delete" approach for bikes.

ID copies are **personal, sensitive data**. Even in the prototype they are only kept in memory,
never written to disk or logged.

## Scope

### In scope

- Domain model for `Person` and the uploaded `IDDocument`
- Thread-safe in-memory store for persons and their ID documents
- API to create a person **together with** the ID copy, edit, archive and reactivate persons,
  replace and view the ID copy
- "Persons" page in the frontend: list, search, create form with upload, edit, replace copy,
  view copy, archive
- Tests for domain, store and handlers

### Out of scope

- Reservations; blocking archiving of persons with open reservations (Phase 3)
- Email, address, date of birth
- Deleting persons or ID copies (data protection deletion is a later topic)
- Customer self-registration, login, consent management
- Scanning/OCR or verifying the ID document
- Persistence across restarts

## Domain Model

### Person

| Field | Type | Rules |
|-------|------|-------|
| `ID` | string | assigned by the store (`p-1`, `p-2`, …), not editable |
| `FirstName` | string | required, trimmed, max. 100 chars |
| `LastName` | string | required, trimmed, max. 100 chars |
| `IDNumber` | string | required, trimmed, upper-cased, max. 30 chars, unique (also among archived persons) |
| `Phone` | string | required, trimmed, max. 30 chars, only digits, spaces and `+ - / ( )`, at least 5 digits |
| `Archived` | bool | `false` for new persons; changed only via archive/reactivate |
| `IDDocument` | metadata | read-only: file name, content type, size, upload time – never the bytes |

### IDDocument

| Field | Type | Rules |
|-------|------|-------|
| `FileName` | string | original file name (base name only), used for download |
| `ContentType` | string | detected from the bytes: `application/pdf`, `image/jpeg` or `image/png` |
| `Data` | `[]byte` | not empty, max. **5 MB**; never part of person JSON |
| `UploadedAt` | time | set by the server |

A person **always** has exactly one ID document. It can be replaced, but not removed.

## API

| Method | Path | Success | Errors |
|--------|------|---------|--------|
| `GET` | `/api/persons` | `200` active persons; `?archived=true` lists archived ones; `?q=` searches name / ID number / phone | `400` invalid filter |
| `POST` | `/api/persons` | `201` created person — **multipart**: fields `firstName`, `lastName`, `idNumber`, `phone` + file `idDocument` | `400` validation / missing file / wrong type, `409` duplicate ID number, `413` file too large |
| `GET` | `/api/persons/{id}` | `200` person (also when archived) | `404` |
| `PUT` | `/api/persons/{id}` | `200` updated person (JSON: `firstName`, `lastName`, `idNumber`, `phone`) | `400`, `404`, `409` |
| `PUT` | `/api/persons/{id}/id-document` | `200` person with new document metadata (multipart file `idDocument`) | `400`, `404`, `413` |
| `GET` | `/api/persons/{id}/id-document` | `200` the file (`Content-Disposition: inline`) | `404` |
| `POST` | `/api/persons/{id}/archive` | `200` person with `archived: true` | `404` |
| `POST` | `/api/persons/{id}/reactivate` | `200` person with `archived: false` | `404` |

Error responses use the shape `{"error": "<message>"}`.

## Frontend

- Table of persons: name, ID number, phone, ID copy (link "view"), actions (edit, archive / reactivate)
- Search field (name / ID number / phone) and toggle "show archived persons"
- Create form including the ID copy file input (required)
- Edit form for the text fields; separate "replace ID copy" action
- Archive asks for confirmation
- Validation errors from the API are shown next to the form

## Decisions

| # | Decision | Reason |
|---|----------|--------|
| D1 | Person fields: first name, last name, ID number, phone | Chosen by product owner; email/address/birth date not needed now |
| D2 | Phone is required | It is the only contact field; the shop must be able to reach a renter (e.g. late return) |
| D3 | ID copy is uploaded **at creation**; a person cannot exist without one | Chosen by product owner; guarantees Business Rule 1 for every person |
| D4 | Create is one multipart request (fields + file) | Atomic: no person without ID copy, even if the browser fails halfway |
| D5 | Allowed files: PDF, JPG, PNG, max. 5 MB | Chosen by product owner; covers scans and phone photos |
| D6 | Content type is detected from the bytes (`http.DetectContentType`), not trusted from the client | Prevents disguised uploads |
| D7 | Persons are archived, never deleted; the ID copy is kept | Chosen by product owner; consistent with retiring bikes, reservations keep their person |
| D8 | ID number is unique (trimmed, case-insensitive), including archived persons | One record per real person; an archived person is reactivated instead of re-created |
| D9 | ID documents live in memory only, never on disk or in logs | Sensitive personal data |
| D10 | Person JSON contains document metadata only; bytes via a separate endpoint | Keeps lists small and avoids leaking ID copies |
| D11 | Reuse store, error and JSON patterns from Phase 1; standard library only | Consistency; no new dependencies |

## Acceptance Criteria

- [x] A person can be created with an ID copy, listed, searched and edited in the browser.
- [x] Creating a person without an ID copy is rejected with `400`.
- [x] The ID copy can be viewed and replaced.
- [x] Missing/invalid fields are rejected with `400`; a duplicate ID number with `409`.
- [x] A wrong file type (e.g. `.txt` renamed to `.pdf`) is rejected with `400`, a file over 5 MB with `413`.
- [x] Archived persons are hidden by default, visible with the toggle, and can be reactivated.
- [x] Person JSON never contains the document bytes.
- [x] `go test ./...` passes, including a concurrent store test.

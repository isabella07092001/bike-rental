# Phase 2 – Person Management: Plan

Requirements: [requirements.md](requirements.md). Each task group ends with passing tests
(`go test ./...`) and can be committed on its own.

## 1. Domain: IDDocument

1.1 Define `IDDocument` struct (`FileName`, `ContentType`, `Data`, `UploadedAt`)
1.2 Define allowed content types (PDF, JPEG, PNG) and `MaxIDDocumentSize` (5 MB) as constants
1.3 Add `NewIDDocument(fileName, data, now)`: base file name, detect content type from bytes, validate
1.4 Add `IDDocument.Meta()` returning metadata without bytes (file name, type, size, upload time)
1.5 Tests: allowed types, disguised file, empty file, size boundary (5 MB / 5 MB + 1 byte)

## 2. Domain: Person

2.1 Define `Person` struct (`ID`, `FirstName`, `LastName`, `IDNumber`, `Phone`, `Archived`) with JSON tags
2.2 Add `Person.Normalize()` (trim fields, upper-case ID number)
2.3 Add `Person.Validate()` (required fields, length limits, phone format)
2.4 Add `Archive()`, `Reactivate()` and `FullName()`
2.5 Table-driven tests for `Normalize()`, `Validate()`, `Archive()`/`Reactivate()`

## 3. Store

3.1 Define `PersonStore` interface (`List`, `Get`, `Create(person, doc)`, `Update`, `SetIDDocument`, `GetIDDocument`)
3.2 Implement in-memory `PersonMemoryStore` with `sync.RWMutex`, IDs `p-1`, `p-2`, …
3.3 `Create` stores person and document together (atomic)
3.4 Enforce unique ID number (case-insensitive, including archived) → `ErrConflict`
3.5 Return copies (incl. copied byte slices); list sorted by last name, first name
3.6 Tests: create with document, CRUD, not-found, duplicate ID number, document replace/get, concurrent access

## 4. API Handlers

4.1 Add `PersonHandler` struct; register routes in `NewRouter`; wire the store in `main.go`
4.2 Define person response (person fields + document metadata)
4.3 Implement `GET /api/persons` (`?archived=true`, `?q=` search) and `GET /api/persons/{id}`
4.4 Implement `POST /api/persons` (multipart: fields + `idDocument`, size limit → `413`)
4.5 Implement `PUT /api/persons/{id}` (JSON, text fields only)
4.6 Implement `PUT /api/persons/{id}/id-document` (replace copy)
4.7 Implement `GET /api/persons/{id}/id-document` (`Content-Type`, `Content-Disposition: inline`, `X-Content-Type-Options: nosniff`)
4.8 Implement `POST /api/persons/{id}/archive` and `/reactivate`
4.9 `httptest` tests for happy paths and each error status, incl. missing file, wrong type, too large

## 5. Frontend: Persons Page

5.1 Person table (name, ID number, phone, ID copy link, actions)
5.2 Search field and "show archived persons" toggle
5.3 Create form with required file input (sent as `FormData`)
5.4 Edit form for text fields; "replace ID copy" action
5.5 Archive with confirmation; reactivate for archived persons
5.6 Extend `api.js` for `FormData` bodies; show API errors in the form

## 6. Wrap-up

6.1 Manual check in the browser: create with copy, edit, search, view and replace copy, archive, reactivate, wrong file type
6.2 `gofmt`, `go vet`, `go test ./...`
6.3 Tick off Phase 2 in [roadmap.md](../roadmap.md) and the acceptance criteria in requirements.md
6.4 Commit

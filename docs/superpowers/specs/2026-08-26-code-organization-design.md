# Chirpy Go Code Organization Design

## Goal

Reorganize the handwritten Go code into a small, idiomatic same-package layout without changing application behavior.

## Structure

- `main.go` keeps environment loading, database setup, route registration, and server startup.
- `handlers.go` contains `apiConfig`, the health, metrics, reset, and user handlers, middleware, and shared JSON error handling.
- `chirps.go` contains chirp request and response types, the chirp creation handler, and profanity cleaning.
- Generated sqlc code remains under `internal/database` and is not edited manually.

All three handwritten files remain in `package main`. No new packages, interfaces, dependencies, factories, or configuration are introduced.

## Behavior Preservation

The refactor preserves:

- Every registered route and HTTP method.
- Existing HTTP status codes, response fields, response messages, and content types.
- Existing request decoding, UUID and length validation, profanity replacement, and database calls.
- Existing environment loading, database initialization, server address, static file serving, and metrics behavior.
- Existing error and logging behavior, including currently ignored return values.

Private identifiers may be renamed for spelling or clarity when that does not alter behavior.

## Verification

No tests will be added or run, per user direction. Verification consists of:

- `gofmt` on handwritten Go files.
- `sqlc compile` to validate schema and queries.
- `go build ./...` with writable Go caches.

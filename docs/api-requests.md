# Chirpy API requests in Yaak

Generated from the current `main.go` handlers and SQL on 2026-10-04. This documents current behavior, including handler limitations. No project application code was changed and no HTTP requests were sent during generation.

## Open and restore

Workspace: **Chirpy Go (Boot.dev)** (`wk_PLukoQxY7E`). Select its **Local** environment (`ev_HrQgnAQCMJ`). Requests already exist in Yaak's default local database. The desktop app uses the same database.

Backup: `docs/chirpy.yaak.json`. Keep it with the project to avoid rebuilding after another machine refresh. Restore using the desktop import action or `yaak import docs/chirpy.yaak.json`. Do not re-import into this installation unless you intend to create another copy.

## Local setup and usage

1. Start the project server from the `chirpy-go` directory with its existing PostgreSQL database and `.env` configuration. The server requires `DB_URL`; `PLATFORM=dev` enables destructive reset. Yaak does not start the server.
2. Select **Local**, with `base_url=http://localhost:8080`. No authentication is currently implemented.
3. Send **Health check**. This verifies HTTP availability only.
4. Set `email` to a unique test email, then send **Create user**. Copy the returned `id` into `user_id`.
5. Set `chirp_body`, then send **Create chirp**. Copy the returned `id` into `chirp_id`.
6. Send **List chirps** or **Get chirp by ID**.

`user_id` and `chirp_id` deliberately start empty until real records are created. No credentials or values from `.env` are included in the backup. The Go GET routes also accept HEAD through standard ServeMux behavior; separate HEAD requests are not included. The `/app/` route is a prefix, represented by one GET request.

**Do not send the whole workspace.** It includes a clearly labeled admin reset request that deletes all users and chirps. Send that request only when you deliberately want to clear your development data.

## Endpoints

### GET /api/healthz

Returns 200 with text/plain; charset=utf-8. Current body: "OK" (the text associated with the 200 status code). Does not check database connectivity. Source: main.go healthHandler.

Yaak request: `rq_tv9e5Y4KS9`.

### POST /api/users

Creates a user from JSON {"email":"..."}. Email has a UNIQUE database constraint; use a fresh email for repeated creation. Success: 201 JSON with id, created_at, updated_at, email. Copy id into the Local user_id variable. Malformed JSON: 400 plain text Invalid JSON. Current handler logs database errors but returns an empty default 200; do not interpret that as successful creation. No email-format validation is implemented. Source: main.go createUser; sql/queries/users.sql; sql/schema/001_users.sql.

Yaak request: `rq_UQdvhCNtsP`.

### POST /api/chirps

Requires an existing user_id UUID. Send JSON with body and user_id. Set Local user_id from Create user before sending. Body length must be at most 140 bytes, as checked by Go len(string). Empty body is currently accepted. Space-separated words kerfuffle, sharbert and fornax are replaced by **** case-insensitively. Success: 201 JSON with id, created_at, updated_at, body, user_id. Copy id into Local chirp_id. Invalid JSON/UUID or zero UUID: 400 {"error":"Invalid request"}; oversized body: 400 {"error":"Chirp is too long"}; database errors, including nonexistent user: 500 {"error":"something went wrong"}. Source: main.go createChirp and profaneHandler; sql/queries/chirps.sql.

Yaak request: `rq_oBA2wRJTur`.

### GET /api/chirps

Returns 200 JSON array ordered by created_at ascending. Each item has id, created_at, updated_at, body, user_id. Empty result is []. Database errors: 500 {"error":"something went wrong"}. No pagination or filters are implemented. Source: main.go getChirps; sql/queries/chirps.sql.

Yaak request: `rq_DhoHtqz7Ud`.

### GET /api/chirps/{id}

Route: GET /api/chirps/{id}. Set Local chirp_id to a UUID returned by Create chirp. Success: 200 JSON with id, created_at, updated_at, body, user_id. Current handler omits assignment of body, so the returned body is an empty string. Invalid UUID: 400 {"error":"Invalid chirp ID"}. Missing record or database failure currently logs an error and returns an empty default 200, rather than 404/500. Source: main.go getChirp; sql/queries/chirps.sql.

Yaak request: `rq_Sebd93m8br`.

### GET /app/

Representative request for the /app/ file server prefix. Serves index.html from the server working directory, normally 200 text/html. Every request through this prefix increments the visit counter, including asset and missing-file requests. /app/assets/logo.png is another valid example. Missing files return 404. The prefix is registered without a method restriction; this collection uses GET for normal browsing. Source: main.go route registration and middlewareMetricsInc.

Yaak request: `rq_5JCA6PackT`.

### GET /admin/metrics

Returns 200 text/html; charset=utf-8 with the current /app/ request count. The counter is held in memory and resets on server restart. This endpoint does not itself increment the counter. Source: main.go metricsHandler.

Yaak request: `rq_hWUxfu3rmi`.

### POST /admin/reset

DESTRUCTIVE: deletes ALL users, which cascades to ALL chirps via the foreign key, and clears the visit counter. Only allowed when server PLATFORM=dev; otherwise returns 403 with an empty body. On success returns 200 text/plain; charset=utf-8 with Hits: <previous count>. No request body or authentication. Current handler ignores database deletion errors, so 200 does not guarantee deletion succeeded. Never include this request in an indiscriminate workspace/folder send. Source: main.go resetHandler; sql/queries/users.sql; sql/schema/20260826010121_chirps.sql.

Yaak request: `rq_8p2pe44aiV`.

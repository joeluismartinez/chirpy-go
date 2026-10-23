# Chirpy

A small Twitter-style JSON API written in Go (Boot.dev course project). It uses Postgres for storage, **goose** for migrations, and **sqlc** to generate type-safe Go code from SQL.

This README covers the setup and day-to-day commands, so coming back to the project is about Go, not about remembering tooling. For the HTTP endpoints, see [docs/api-requests.md](docs/api-requests.md).

---

## TL;DR: get it running

From the repo root:

```bash
brew services start postgresql@18   # make sure Postgres is up (no-op if already running)
goose up                            # apply any pending migrations
go build -o chirpy . && ./chirpy    # build and run on http://localhost:8080
```

Check that it works with `curl localhost:8080/api/healthz`. Stop the server with `Ctrl+C`.

### Cheat sheet

| I want to…                         | Command                                              |
| ---------------------------------- | ---------------------------------------------------- |
| Build and run                      | `go build -o chirpy . && ./chirpy`                   |
| Run without keeping a binary       | `go run .`                                           |
| Open a SQL shell                   | `psql chirpy`                                        |
| See migration status               | `goose status`                                       |
| Apply migrations                   | `goose up`                                           |
| Undo the last migration            | `goose down`                                         |
| Create a new migration             | `goose create add_something sql`                     |
| Regenerate Go code from SQL        | `sqlc generate`                                      |
| Start / stop Postgres              | `brew services start postgresql@18` / `stop`         |
| Wipe users and chirps (dev only)   | `curl -X POST localhost:8080/admin/reset`            |

Run everything from the **repo root**. Both the app and goose read `.env` from the current directory.

---

## 1. Tools

| Tool       | What it does                                   | Install                       | Check             |
| ---------- | ---------------------------------------------- | ----------------------------- | ----------------- |
| Go         | Compiler and toolchain                         | `brew install go`             | `go version`      |
| PostgreSQL | The database (Homebrew `postgresql@18`)        | `brew install postgresql@18`  | `psql --version`  |
| goose      | Runs the SQL migrations in `sql/schema/`       | `brew install goose`          | `goose -version`  |
| sqlc       | Generates Go code from the SQL in `sql/`       | `brew install sqlc`           | `sqlc version`    |

> **sqlc is not currently installed on this machine.** You only need it when you add or change SQL queries. Install it with `brew install sqlc` before then. The existing generated code was produced with sqlc v1.31.1.

---

## 2. Postgres (local, via Homebrew)

Postgres runs as a background service under your macOS user. Locally there is **no password**: you connect as your macOS username (`joeluismartinez`).

### Start, stop, check

```bash
brew services list                      # is postgresql@18 "started"?
brew services start postgresql@18
brew services stop postgresql@18
brew services restart postgresql@18
```

### Connect with psql

```bash
psql chirpy                             # shorthand: local socket, your user, "chirpy" database
psql "postgres://joeluismartinez@localhost:5432/chirpy?sslmode=disable"   # same thing, as a URL
```

Useful commands inside `psql`:

| Command                  | Does                                  |
| ------------------------ | ------------------------------------- |
| `\l`                     | List databases                        |
| `\c chirpy`              | Connect to the `chirpy` database      |
| `\dt`                    | List tables                           |
| `\d users`               | Describe a table (columns, indexes)   |
| `SELECT * FROM chirps;`  | Regular SQL (don't forget the `;`)    |
| `\x`                     | Toggle expanded output for wide rows  |
| `\q`                     | Quit                                  |

You can also run a one-off query without opening the shell: `psql chirpy -c 'SELECT count(*) FROM users;'`

### Creating the database from scratch (new machine)

```bash
brew install postgresql@18
brew services start postgresql@18
createdb chirpy
cp .env.example .env                    # then replace YOUR_POSTGRES_USER with `whoami`
goose up
```

---

## 3. Environment variables (`.env`)

`.env` is git-ignored. `.env.example` is the template to copy from.

```dotenv
DB_URL=postgres://joeluismartinez@localhost:5432/chirpy?sslmode=disable
PLATFORM=dev

# goose reads these from .env automatically
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=${DB_URL}
GOOSE_MIGRATION_DIR=sql/schema
```

- `DB_URL`: the app's connection string, read in `main()` via `godotenv.Load()` and `os.Getenv("DB_URL")`.
- `PLATFORM=dev`: enables `POST /admin/reset`, which **deletes all users** (chirps go too, via `ON DELETE CASCADE`). Any other value makes it return 403.
- `GOOSE_*`: lets you type plain `goose up` instead of `goose -dir sql/schema postgres "<url>" up`. goose loads `.env` from the current directory by default, and `${DB_URL}` is expanded.

The URL format is `postgres://USER[:PASSWORD]@HOST:PORT/DATABASE?sslmode=disable`. Local Homebrew Postgres has no SSL, hence `sslmode=disable`.

---

## 4. Migrations with goose

Migrations live in `sql/schema/`. Each file has an **Up** section (apply the change) and a **Down** section (undo it):

```sql
-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE users;
```

goose records what it has applied in a `goose_db_version` table in the same database.

### Commands

```bash
goose status                     # which migrations are applied / pending
goose up                         # apply all pending migrations
goose down                       # roll back the most recent one
goose redo                       # down + up on the latest (handy while iterating on a migration)
goose reset                      # roll back EVERYTHING (drops all tables, destroys data)
goose create add_refresh_tokens sql    # new timestamped file in sql/schema/
```

> **File naming:** the first migration is `001_users.sql` (sequential) and the second is `20260826010121_chirps.sql` (timestamp, goose's default for `create`). goose orders by the number, so this works fine. Keep using `goose create`, or add `-s` (`goose -s create name sql`) to get sequential names. Just don't renumber files that are already applied.

### Without the `.env` shortcut

If `GOOSE_*` isn't in `.env` (or you're in a different directory), this is the full form:

```bash
goose -dir sql/schema postgres "postgres://joeluismartinez@localhost:5432/chirpy?sslmode=disable" up
```

---

## 5. Wiring data models: goose → sqlc → Go

There are three layers. You write the first two by hand, and sqlc writes the third:

```
sql/schema/*.sql    ── goose ──▶  tables in Postgres         (the shape of the data)
sql/queries/*.sql   ── sqlc  ──▶  internal/database/*.go     (Go structs + methods)
main.go handlers    ── call  ──▶  cfg.dbQueries.CreateUser(...)
```

sqlc reads **both** `sql/schema` (to learn the table types) and `sql/queries` (to know what functions to write). See `sqlc.yaml`. It never touches the database itself, so goose and sqlc are independent: goose changes the real DB, and sqlc only generates code.

### Recipe: adding a new table and queries

Say you're adding refresh tokens.

**1. Write the migration.** Run `goose create refresh_tokens sql`, then fill in the file:

```sql
-- +goose Up
CREATE TABLE refresh_tokens (
    token TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE refresh_tokens;
```

**2. Apply it:** run `goose up`, then `goose status` to confirm.

**3. Write queries** in a new `sql/queries/refresh_tokens.sql`. The `-- name:` comment is required. It sets the Go method name and the return shape:

```sql
-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1;
```

| Annotation | Go method returns          | Use for                       |
| ---------- | -------------------------- | ----------------------------- |
| `:one`     | `(Model, error)`           | `RETURNING *`, single-row get |
| `:many`    | `([]Model, error)`         | lists                         |
| `:exec`    | `error`                    | deletes / updates you don't read back |

**4. Generate the Go code:** run `sqlc generate`. This rewrites `internal/database/` (never edit those files by hand). You'll get:

- a `RefreshToken` struct in `models.go`
- `CreateRefreshToken(ctx, CreateRefreshTokenParams{...})` and `GetRefreshToken(ctx, token)` in `refresh_tokens.sql.go`. With more than one `$N` parameter, sqlc generates a `...Params` struct.

**5. Use it in a handler**, the same way `createChirp` does:

```go
tok, err := cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
    Token:     token,
    UserID:    user.ID,
    ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
})
```

**6. Build and run** to make sure it compiles: `go build -o chirpy . && ./chirpy`

### Changing an existing table

Don't edit a migration that's already been applied, because goose won't re-run it. Create a **new** migration (`ALTER TABLE ... ADD COLUMN ...` in Up, and `ALTER TABLE ... DROP COLUMN ...` in Down), run `goose up`, then `sqlc generate`.

While a migration is brand-new and only on your machine, it's fine to edit it and run `goose redo`.

---

## 6. Building and running

```bash
go build -o chirpy . && ./chirpy    # compile to ./chirpy, then run it if the build succeeded
go run .                            # compile to a temp location and run (nothing left behind)
```

- The server listens on **:8080**.
- **Run it from the repo root.** It loads `.env` from the current directory, and `/app/` serves files from `.` (so `index.html` and `assets/` need to be there).
- `&&` means "only run the second command if the first succeeded", so a failed build won't start an old binary.

Handy Go commands while learning:

```bash
go vet ./...        # catches common mistakes (e.g. Printf args with no %s)
gofmt -w .          # format all files
go mod tidy         # sync go.mod/go.sum after adding or removing imports
```

---

## 7. Troubleshooting

| Symptom | Likely cause / fix |
| --- | --- |
| `Error loading .env file` | You're not in the repo root, or `.env` doesn't exist (`cp .env.example .env`). |
| `connection refused` on 5432 | Postgres isn't running: `brew services start postgresql@18`. |
| `database "chirpy" does not exist` | Run `createdb chirpy`, then `goose up`. |
| `relation "xyz" does not exist` | Pending migration. Run `goose status`, then `goose up`. |
| `bind: address already in use` | An old server is still on 8080. Run `lsof -ti :8080 \| xargs kill`. |
| `sqlc: command not found` | Run `brew install sqlc`. |
| goose says `no migrations` / wrong DB | Not in the repo root, or the `GOOSE_*` lines are missing from `.env`. |
| sqlc errors about unknown column | The column isn't in any `sql/schema` file yet. Add a migration first. |

## Project layout

```
main.go                 handlers, routing, server startup
sql/schema/             goose migrations (source of truth for tables)
sql/queries/            SQL queries sqlc turns into Go
internal/database/      sqlc-generated code (DO NOT EDIT)
sqlc.yaml               sqlc config
.env / .env.example     local config (DB_URL, PLATFORM, GOOSE_*)
docs/api-requests.md    endpoint docs + Yaak request collection
index.html, assets/     static files served at /app/
```

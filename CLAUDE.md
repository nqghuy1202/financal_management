# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A personal finance tracker. Users record income and expenses; the system
aggregates them into a financial picture for a month or a year. **The
reports are the product** — recording is only the input side. When a design
choice trades off between easier data entry and correct aggregation,
correct aggregation wins.

There is deliberately **no transfer feature**. Moving money between two of
your own sources changes neither total income nor total expense, so it
changes nothing in the summary, while forcing every reporting query to
remember to exclude it. `transactions.type` is only `income` or `expense`.

Planning documents: `docs/ROADMAP.md` (phases, decisions), `docs/ERD.md`
(schema and the reasoning behind each choice). Both record *why*, and are
worth reading before changing the data model.

## Commands

```bash
make help          # every target with a description
make up            # start postgres, redis, kafka, kafka-ui, mailhog
make run           # run the API (needs `make up` first)
make watch         # run with live reload via air
make check         # fmt + vet + lint + test — run before committing
```

Testing:

```bash
make test                              # unit tests, no docker needed
go test ./internal/services/ -run Summary -v   # a single test or group
make itest                             # integration tests (needs docker)
```

`make test-race` exists but **cannot run on a machine without a C
compiler** — `-race` requires cgo. CI runs it on Linux; locally use
`make test`.

Database:

```bash
make tools           # install goose and sqlc (once)
make migrate-up      # apply pending migrations
make sqlc            # regenerate query code after editing SQL
make sqlc-verify     # check generated code matches the SQL
```

Frontend (`web/`, Next.js 16 + React 19 + Tailwind v4):

```bash
cd web && npm run dev
```

## Architecture

One HTTP binary today (`cmd/api`). Kafka workers are planned but not built.

### Request path

```
routers → controller → service → repo → sqlc → PostgreSQL
```

Each layer only knows the one below it.

- **controller** parses and validates HTTP input, converts to/from response
  structs. Never contains business rules.
- **service** holds the rules and **must not import gin**. A Kafka consumer
  should be able to call the same service without an HTTP request.
- **repo** only reads and writes. It translates driver errors (`pgx.ErrNoRows`,
  unique-violation SQLSTATE) into package-level sentinels like
  `repo.ErrUserNotFound`, so no other layer imports a database driver.

### Wiring

`internal/initialize/wire.go` is the only place that knows which
implementation satisfies which interface. `routers.Deps` declares what the
route layer needs — **declared in `routers`, not `initialize`**, so
`initialize` can import `routers` without a cycle.

`global` holds only `Config` and `Logger`. The connection pool and Redis
client are returned by `InitPostgres`/`InitRedis` and passed down
explicitly, so repositories are testable against a throwaway database.

### Errors

Services return `*response.AppError` carrying a business code from
`internal/pkg/response/code.go`, which maps each code to an HTTP status.
Handlers report failures with `_ = c.Error(err)` and return; the
`ErrorHandler` middleware renders the response and logs the wrapped cause.
`response.AsAppError` funnels any unrecognised error into a 500 so internal
detail never reaches a client.

### Auth tokens

Two tokens with different jobs, and the split is deliberate.

The **access token** is a JWT valid for 15 minutes, returned in the
response body. A JWT cannot be revoked, so it is kept short-lived; the
frontend holds it in memory and never writes it to disk.

The **refresh token** is a random string in Redis, returned only as an
**HttpOnly cookie** set by the Go API. It is deliberately absent from
`sessionResponse` — returning it in the body too would let JavaScript read
it and defeat the cookie. `refresh` and `logout` read it from the cookie
and take no request body.

The API sets the cookie directly rather than proxying through Next.js:
`localhost:3000` and `localhost:8080` are the same site (site is the
domain, ports do not count), so `SameSite=Lax` works with no extra layer.
The cookie's `Path` is scoped to `/api/v1/auth`, so it is not attached to
ordinary API calls. Startup rejects `sameSite=none` without `secure`,
because browsers drop such a cookie silently and login would fail in
production with nothing to trace.

`ConsumeRefresh` uses Redis `GETDEL`, so a refresh token works exactly
once and two concurrent requests cannot both redeem it.

### Middleware order

Set in `internal/initialize/router.go`. `RequireAuth` must run **before**
the per-user rate limit, since keying a limit by account requires knowing
the account. Rate limits are scoped (`ip`, `user`, `login`) and counted
under separate Redis keys, so exhausting the login budget leaves browsing
intact.

## Conventions that matter

**Money.** `NUMERIC(19,4)` in Postgres, `model.Money` in Go — an amount
bound to its currency. It exposes no method taking a `float64`, and adding
two currencies returns an error. Amounts cross the API as **JSON strings**;
JavaScript parses JSON numbers as `float64` and would round large amounts
before the frontend saw them.

**Ownership is enforced in SQL, not Go.** Every query filters on
`user_id` in its own WHERE clause. A handler cannot forget a check that is
not written in Go. Another user's row returns 404, never 403 — a
distinguishable "forbidden" lets ids be probed for existence.

**Reports aggregate on `occurred_at`, never `created_at`.** Entering
yesterday's lunch today must report against yesterday.

**Timezone.** `occurred_at` is stored in UTC but users think in local time.
`model.AppTimezone` is applied both when grouping by month in SQL and when
parsing date-only query parameters. Getting this wrong shifts every
transaction between midnight and 07:00 on a period's first day into the
previous period, and the totals still look plausible.

**Half-open periods `[from, to)`.** Adjacent periods never double-count a
boundary transaction. When deriving the month range for a chart, use the
month of the last instant *inside* the period, not the month of `to`.

**Cursor pagination, not OFFSET.** OFFSET makes the database read and
discard skipped rows, and inserts between requests shift later pages. The
cursor is the `(occurred_at, id)` pair the sort uses, base64url-encoded —
a raw RFC3339 timestamp contains `+07:00`, and `+` in a query string
decodes to a space.

**Soft delete on financial data**, and foreign keys use `RESTRICT`, never
`CASCADE`.

## sqlc pitfalls

`sqlc.yaml` documents these inline; they are silent failures, not errors.

- With an explicit `import`, `type` must be the bare type name and
  `package` the package — writing `type: "uuid.UUID"` yields
  `uuid.uuid.UUID`, which does not compile.
- Two overrides of one database type (nullable and not) must declare their
  import **identically**, or sqlc emits the import twice.
- Catalog prefixes are inconsistent: `timestamptz` works bare,
  `pg_catalog.numeric` and `pg_catalog.timestamp` need the prefix. A wrong
  `db_type` is ignored without warning.

**After every `sqlc generate`, open `internal/repo/sqlc/models.go` and
check the types.**

## Migrations

Migrations land with the feature that needs them, rather than creating
tables ahead of use. The 16 default categories are seeded by a migration as
system rows with a null `user_id`, so registration stays a single write and
renaming a category is one edit.

Unreleased migrations on this branch are rewritten in place rather than
patched by a follow-up drop; `goose down-to` first, then edit, then
`goose up`.

## Verification

Unit tests here have missed a whole class of bug — URL encoding, timezone
boundaries, wrong error wording — because the tests shared the same wrong
assumption as the code. **Finish each module with a smoke test over real
HTTP** against the running stack, asserting concrete numbers, with data
placed at boundaries (a transaction at 00:30 on the first of the month,
Vietnamese text, emoji). Clean up the test rows afterwards.

## Environment notes

- A native PostgreSQL may already hold port 5432, so the container
  publishes **5433**. `POSTGRES_PORT` in `.env` and `postgres.port` in
  `config/local.yaml` must agree.
- Docker Desktop needs to be started manually and takes a few minutes
  before the daemon accepts connections.
- Git Bash mangles Vietnamese text and emoji in command arguments. When
  testing the API with accented data, write JSON to a file and use
  `curl --data-binary @file`, or drive it from a Python script.
- Windows locks a running executable, so stop the process before
  `go build -o bin/api.exe` or the build fails quietly.

## Frontend

`web/AGENTS.md` is generated by `next dev` and warns that **Next.js 16
differs from older versions**; its docs are in `node_modules/next/dist/docs/`.
Read the relevant guide before writing frontend code rather than relying on
recalled Next.js conventions.

## Language

Code comments, commit bodies and user-facing strings follow the existing
files: comments and API messages in Vietnamese, commit messages in English.
Comments explain **why**, not what.

---
title: 'Story 1.2: Khai báo Lương cho chu kỳ hiện tại'
type: 'feature'
created: '2026-09-12'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
context: []
baseline_commit: 'a8ed02f9e578987f13eca13b8c26ee452111b523'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The app has no concept of a budget cycle or declared income yet — both are prerequisites for Safe-to-spend (later stories in this epic). Nothing lets a user record "this is my income for the current cycle," and nothing computes cycle boundaries.

**Approach:** Add the `incomes` table, a pure `CycleWindow`/`PreviousCycles` function pair in `cycle.go` (hardcoded `cycleStartDay = 1` for now — Story 1.4 wires the real user setting into the same function later, per architecture), an `IncomeRepo`, and a single `POST /incomes` upsert endpoint (mirrors `BudgetRepo.Upsert`/`POST /budgets`). **Backend-only this story** — per user decision, no frontend UI ships yet; the real income UI arrives with the `safe-to-spend-hero`/cycle-update-sheet in Story 1.5. `PUT /cycle-settings` (the eventual shared endpoint per architecture) is not built now — Story 1.4 absorbs this story's income-write into it later.

## Boundaries & Constraints

**Always:** Follow the existing `XxxRepo{db dbtx}`/`NewXxxRepo`/`ok()`/`fail()` conventions exactly (see `repo_budget.go`/`budgets.go` as the template). Scope every query by `user_id`. New table via `Migrate()`'s existing `CREATE TABLE IF NOT EXISTS` slice — never alter the four existing tables. `CycleWindow`'s day-of-month math must be correct for arbitrary `cycleStartDay` (1-31) even though this story only calls it with `1` — Story 1.4 will swap in the real setting without touching the math itself. Error codes use the next free domain block after budgets' `3x` (`40040`/`40041` bind/validation, `50040` upsert failure) to avoid collisions.

**Never:** Do not build `PUT /cycle-settings`, `user_settings`, `fixed_costs`, or any frontend UI (no income form, no Dashboard changes) — explicitly out of scope per user decision. Do not touch the dead learning-scaffold tree.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| First declaration this cycle | No income row for current `cycle_start_date` | `POST /incomes {amount}` inserts a new row, returns it | N/A |
| Re-declare same cycle | Income row already exists for current `cycle_start_date` | 2nd `POST /incomes` with a different amount updates in place (no 2nd row) | ON DUPLICATE KEY UPDATE |
| Invalid amount | `amount <= 0` or missing | `fail` 400 (`40041`), no row written | Field validation before repo call |
| No carry-over | Income declared in cycle N, then `asOf` moves into cycle N+1 | `IncomeRepo.Get` for cycle N+1's `cycle_start_date` returns not-found (`sql.ErrNoRows`) | Distinct row per `(user_id, cycle_start_date)` |
| CycleWindow day-of-month clamp | `cycleStartDay=31`, `asOf` in February | `start`/`end` clamp to the last day of the short month, not month+1 overflow | Pure function, no I/O |
| CycleWindow boundary | `asOf` exactly on `cycleStartDay` | Counts as the start of the new cycle (`start` inclusive, `end` exclusive) | `>= start && < end` |

</frozen-after-approval>

## Code Map

- `internal/api/db.go` `Migrate()` -- append one more `CREATE TABLE IF NOT EXISTS incomes (id CHAR(36) PK, user_id CHAR(36), cycle_start_date DATE, amount BIGINT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, UNIQUE uq_income(user_id, cycle_start_date), FK fk_income_user->users ON DELETE CASCADE) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` statement to the existing slice, matching the four existing tables' exact style.
- `internal/api/cycle.go` (new) -- `CycleWindow(cycleStartDay int, asOf time.Time) (start, end time.Time)` and `PreviousCycles(cycleStartDay int, asOf time.Time, n int) []struct{ Start, End time.Time }` (or a named `CycleWindow`-shaped slice element) — pure, no DB access. This is the one source of truth every later story/epic imports.
- `internal/api/repo_income.go` (new) -- `IncomeRepo{db dbtx}` + `NewIncomeRepo`. `Upsert(ctx, userID string, cycleStartDate time.Time, amount int64) (Income, error)` mirroring `BudgetRepo.Upsert`'s INSERT-ON-DUPLICATE-then-re-SELECT shape (reuse the same known, pre-existing re-select-fallback pattern — do not "fix" it here, out of scope). `Get(ctx, userID string, cycleStartDate time.Time) (Income, error)` returning `sql.ErrNoRows` when not declared.
- `internal/api/incomes.go` (new) -- `Income` struct (`ID`, `CycleStartDate` using the existing `dateLayout` const from `repo_transaction.go`, `Amount int64`), `UpsertIncome(c *gin.Context)` handler: bind JSON `{amount int64}`, validate `amount > 0`, compute `cycleStartDate = CycleWindow(1, time.Now()).start`, call `h.incomes.Upsert`, `ok(c, income)`.
- `internal/api/api.go` -- add `incomes *IncomeRepo` field to `Handler`, wire in `NewHandler`.
- `internal/api/router.go` -- add `auth.POST("/incomes", h.UpsertIncome)` alongside the existing budgets/transactions group.
- `internal/api/repo_budget.go`, `budgets.go` -- reference pattern only, no changes.

## Tasks & Acceptance

**Execution:**
- [x] `internal/api/db.go` -- add `incomes` table to `Migrate()` -- new persistence for declared income
- [x] `internal/api/cycle.go` -- implement `CycleWindow`/`PreviousCycles` with correct day-of-month clamping -- single source of truth for cycle boundaries
- [x] `internal/api/cycle_test.go` -- cover the I/O matrix's CycleWindow scenarios (clamp, boundary, plus a plain day-1 case and a mid-month case) -- pins the pure-function math before other stories depend on it
- [x] `internal/api/repo_income.go` -- `IncomeRepo` with `Upsert`/`Get` -- data-access layer
- [x] `internal/api/repo_income_test.go` -- cover insert, update-in-place, and not-found `Get` (sqlmock) -- pins repo behavior
- [x] `internal/api/incomes.go` -- `UpsertIncome` handler + `Income` model -- exposes the feature
- [x] `internal/api/incomes_test.go` -- cover happy path, invalid-amount rejection, and the no-carry-over scenario (via repo `Get`) -- pins handler contract
- [x] `internal/api/api.go`, `router.go` -- wire `IncomeRepo` into `Handler` and mount `POST /incomes`
- [x] repo root -- run `go build ./...`, `go vet ./...`, `go test ./internal/api/...` -- confirm zero regressions

**Acceptance Criteria:**
- Given no income declared for the current cycle, when `IncomeRepo.Get` is called for that cycle, then it returns `sql.ErrNoRows`.
- Given a valid `POST /incomes` request, when it succeeds, then a subsequent `Get` for the same cycle returns the saved amount, and a 2nd `POST` with a different amount updates the same row (no duplicate).
- Given `amount <= 0` or missing, when `POST /incomes` is called, then it returns 400 (`40041`) and writes nothing.

## Implementation Notes

- `internal/api/cycle.go` (new): `CycleWindow(cycleStartDay int, asOf time.Time) (start, end time.Time)` computes the effective clamped start-of-month day for `asOf`'s own month first (`clampDay`), then compares `asOf`'s actual day against that *clamped* value — not the raw `cycleStartDay` — to decide whether the current cycle began this month or last month. This makes the "asOf exactly on the clamped boundary" case (e.g. `cycleStartDay=31`, `asOf=2026-02-28`) correctly count as the start of a new cycle, with the following cycle's end resolving against the unclamped day once a longer month (March, 31 days) makes it valid again (`end=2026-03-31`). `PreviousCycles` walks backward by re-deriving each prior window from one day before the current window's start, so it inherits `CycleWindow`'s clamping correctness for free.
- `internal/api/repo_income.go` (new): `IncomeRepo.Upsert` mirrors `BudgetRepo.Upsert`'s INSERT-ON-DUPLICATE-then-re-SELECT shape exactly, including its known re-select-fallback behavior (swallows a non-`ErrNoRows` re-SELECT failure and returns the just-inserted values) — left as-is per the spec's explicit "do not fix here" instruction. `Get` returns `sql.ErrNoRows` verbatim when nothing is declared for a cycle.
- `internal/api/incomes.go` (new): `UpsertIncome` binds `{amount int64}`, rejects `amount <= 0` (`40041`) or unparseable JSON (`40040`) before touching the repo, computes `cycleStartDate` via `CycleWindow(1, time.Now())` (hardcoded `cycleStartDay=1`, per spec — Story 1.4 will thread the real setting through this same call), and returns the canonical row via `ok()`.
- Error codes used: `40040` (bad JSON), `40041` (invalid amount), `50040` (upsert failure) — confirmed free (no collision) by grepping all existing `400xx`/`500xx` literals in `internal/api` before assigning them.
- Wired `IncomeRepo` into `Handler`/`NewHandler` (`internal/api/api.go`) and mounted `auth.POST("/incomes", h.UpsertIncome)` in `router.go`, alongside the existing budgets/transactions group.
- All I/O & Edge-Case Matrix scenarios are covered and pass: first declaration (`TestIncomeRepo_Upsert_Insert`, `TestUpsertIncome_HappyPath`), re-declare/update-in-place (`TestIncomeRepo_Upsert_UpdateInPlace`, `TestUpsertIncome_Idempotency`), invalid amount (`TestUpsertIncome_InvalidAmount_Zero/Negative/Missing`, `TestUpsertIncome_MalformedJSON`), no carry-over (`TestIncomeRepo_Get_NotFound`, `TestNoCarryOver`), CycleWindow clamp (`TestCycleWindow_ClampShortMonth`, `TestCycleWindow_BoundaryInclusive_ClampedDay`), and CycleWindow boundary (`TestCycleWindow_BoundaryInclusive`), plus a plain day-1 case and a mid-month case (`TestCycleWindow_Day1PlainCase`, `TestCycleWindow_MidMonth`).
- No changes made to `repo_budget.go`, `budgets.go`, the four pre-existing `Migrate()` tables, `user_settings`/`fixed_costs`/`PUT /cycle-settings`, or any frontend code — all explicitly out of scope per the spec.
- `go build ./...`, `go vet ./...`, `go test ./internal/api/... -v` (64 tests total, all passing) and `gofmt -l internal/api cmd/web` (no output) all confirm zero regressions.

## Spec Change Log

- **2026-09-13 (code review round 2):** All 5 `patch` items from round 1 below were already resolved in this diff's own shipped code (the Review Triage Log had gone stale relative to its own commit — same class of self-inconsistency later confirmed in Story 1.1's round 2). Round 2 found no new actionable gap: `incomes.go`/`UpsertIncome`/`incomes_test.go` no longer exist at all (Story 1.4 retired `POST /incomes` in favor of `PUT /cycle-settings`, see `cycle_settings.go`), and the one gap that did still apply at review time (`IncomeRepo.Upsert`'s initial `INSERT` failing, distinct from the re-SELECT failure) is now covered at the handler level by Story 1.4's `TestUpdateCycleSettings_PartialFailureRollsBack`. No code changes made; this document updated for accuracy only.

## Review Triage Log

- **medium / patch — MOOT (2026-09-13)** — `incomes_test.go`'s handler tests used `sqlmock.AnyArg()` for `cycle_start_date` and never asserted the response's `cycleStartDate`. Already false as filed: `incomes_test.go`, as shipped in this same diff, passes the concrete `cycleStr` value as the mock argument on both the INSERT and re-SELECT, and does assert `cycleStartDate` in the response body. Superseded regardless: `incomes.go`/`incomes_test.go` were deleted by Story 1.4 (`PUT /cycle-settings` replaced `POST /incomes`); the equivalent coverage now lives in `cycle_settings_test.go`.
- **low / patch — RESOLVED (already true in this diff)** — `PreviousCycles` (cycle.go) claimed to panic for `n < 0` with no guard. Already false as filed: the shipped `cycle.go` in this same diff opens with `if n <= 0 { return nil }`.
- **low / patch — RESOLVED (already true in this diff)** — `TestPreviousCycles` claimed to lack coverage for the backward walk through a clamped short-month boundary. Already false as filed: this same diff's `cycle_test.go` includes `TestPreviousCycles_ClampShortMonthBoundary`, covering exactly that.
- **low / patch — RESOLVED (already true in this diff)** — `repo_income_test.go` claimed to lack a test for `IncomeRepo.Upsert`'s re-SELECT genuine-error fallback. Already false as filed: this same diff's `repo_income_test.go` includes `TestIncomeRepo_Upsert_ReSelectErrorFallsBack`, covering exactly that.
- **low / patch — RESOLVED (already true in this diff)** — `Income` struct claimed to live in `incomes.go`, breaking convention. Already false as filed: this same diff adds `Income` to `api.go` alongside `User`/`Category`/`Transaction`/`Budget`, matching convention; only the unexported `incomeInput` lives in `incomes.go`.
- **low / reject** — `UpsertIncome` accepts unbounded `amount` (only `<= 0` is rejected). Verified: `transactions.go`/`budgets.go` apply the exact same "only reject `<= 0`" rule with no upper bound, so this matches existing codebase convention rather than deviating from it; adding a ceiling here alone would be a new, inconsistent invariant unscoped by any planning artifact. [blind-hunter]
- **low / reject** — `CycleWindow(1, time.Now())` uses the server process's local timezone with no per-user timezone concept. Verified: no PRD/architecture requirement establishes per-user timezones anywhere in this project (self-hosted, personal-use v1); fixing this would require new scope (a timezone setting), not a direct correction. [blind-hunter]
- **defer** — `IncomeRepo.Upsert`'s INSERT-then-re-SELECT is non-atomic (no transaction/lock), so a concurrent upsert to the same `(user_id, cycle_start_date)` row between the two steps can return another request's amount. Verified: this is the identical non-atomic two-step pattern already deferred for `BudgetRepo.Upsert` in Story 1.1's review, here deliberately mirrored onto a new table per this story's own spec direction, not a new defect invented by this diff. [edge-case-hunter]

### Review Findings (2026-09-13, round 2)

No code patches — nothing actionable survived verification (see Spec Change Log above). All 5 round-1 "patch" items were already resolved in the code this same diff shipped, and the one gap found this round (`IncomeRepo.Upsert`'s initial `ExecContext` failure path, untested at the time of this story's diff) is now covered by Story 1.4's `TestUpdateCycleSettings_PartialFailureRollsBack` at the handler level. `incomes.go`/`incomes_test.go` (the standalone `POST /incomes` handler this story added) no longer exist — retired by Story 1.4's `PUT /cycle-settings` merge — so findings specific to that handler are moot.

**Rejected:**
- `false` — a commit-message claim of a dedicated "start/end argument-swap regression test": no test is specifically named for this, though every `CycleWindow` test already asserts `start` and `end` independently against distinct expected values, which inherently catches a swap. Not spec- or code-actionable (concerns commit-message wording, not shippable content).

## Verification

**Commands:**
- `go build ./...` -- expected: succeeds, no errors
- `go vet ./...` -- expected: succeeds, no warnings
- `go test ./internal/api/... -v` -- expected: all tests pass, including new `cycle_test.go`/`repo_income_test.go`/`incomes_test.go`
- `gofmt -l internal/api cmd/web` -- expected: no output

---
title: 'Story 1.4: Cấu hình ngày bắt đầu chu kỳ ngân sách'
type: 'feature'
created: '2026-09-12'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
context: []
baseline_commit: 'eb22fd5493acb66870c47d81de256b3e1e985a1d'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `cycle_start_day` exists in `user_settings` (Story 1.3) but nothing lets a user change it, and `CycleWindow`'s only caller still hardcodes `1` (Story 1.2's known, documented gap). Income (Story 1.2) and settings-write (Story 1.3) also still live behind two separate endpoints that were always meant to converge.

**Approach:** Build the shared `PUT /cycle-settings` — the single endpoint epics.md and the architecture doc both specify: one request, one `h.withTx`, writing income + `savings_goal` + `cycle_start_day` together, matching the cycle-update-sheet's single "Save" action. This **replaces and retires** `POST /incomes` (Story 1.2) and `PUT /settings/savings-goal` (Story 1.3) — both were explicitly built as narrow, absorbed-later placeholders (see their own specs' Intent sections); their repo layers (`IncomeRepo`, `SettingsRepo`) stay and get called from the new handler, but the standalone HTTP handlers, routes, and their tests are removed as dead/superseded code, not left running alongside the new endpoint. The request supplies the new `cycle_start_day` directly (PUT full-replace semantics, no separate read-before-write), and that new value is what `CycleWindow` uses to compute the income's `cycle_start_date` in the same save — giving immediate effect, per epics.md. **[ASSUMPTION, backend-only]**: continuing the Story 1.2/1.3 precedent, no frontend ships yet.

## Boundaries & Constraints

**Always:** One `h.withTx` covers both the income upsert and the settings upsert — either both succeed or neither does, matching the single "Save" action's atomicity. Build repos over the transaction inside the closure (`NewIncomeRepo(tx)`, `NewSettingsRepo(tx)`), mirroring `auth.go`'s `RegisterUser`. `SettingsRepo.UpsertSavingsGoal` becomes `SettingsRepo.Upsert(ctx, userID string, savingsGoal int64, cycleStartDay int) (Settings, error)` — extend it in place (its only caller is being retired anyway) rather than adding a second method. Prior `incomes` rows tied to the old `cycle_start_date` become unreferenced history, not deleted (per architecture — no effective-dating in v1). New error codes use block `7x` (`40070` bind, `40071` invalid income, `40072` invalid savings goal, `40073` invalid `cycle_start_day`, `50070` transaction failure) — verify no collision with existing `4x`/`5x`/`6x` blocks (the `6x` settings block becomes free once retired, but use a fresh block anyway for log clarity).

**Never:** Do not touch `CycleWindow`/`PreviousCycles`'s internal math (Story 1.2 already made it correct for arbitrary `cycleStartDay`) — only change what value gets passed into it. Do not add any frontend UI or touch `fixed_costs`/`GET /cycle/summary` (Story 1.5's job). Do not add data migration for rows written via the retired endpoints — this is pre-launch, no real user data exists yet.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Full save, day 1 default | `PUT /cycle-settings {income, savingsGoal, cycleStartDay: 1}` | Income upserted keyed by `CycleWindow(1, now).start`; settings upserted with both columns; one response with both canonical rows | Single `withTx` |
| Change cycle_start_day mid-save | `cycleStartDay: 15` (previously 1) | The income in this same request is keyed by `CycleWindow(15, now).start` — the NEW day, not the old one (immediate effect) | Settings write happens before the `CycleWindow` call inside the same closure/logic, using the request's value directly |
| Invalid income | `income <= 0` | `fail` 400 (`40071`), nothing written (transaction never begins, or rolls back) | Validate before `withTx` |
| Invalid cycle_start_day | `cycleStartDay` outside 1-31 | `fail` 400 (`40073`), nothing written | Validate before `withTx` |
| Old endpoints retired | `POST /incomes` or `PUT /settings/savings-goal` | 404 (route no longer registered) | Router no longer mounts them |
| Partial failure rolls back | One write inside `withTx` fails (e.g. simulated DB error on the settings upsert) | Both writes roll back — income is NOT left updated while settings failed | `withTx`'s existing rollback-on-error behavior (already tested generically in `auth_test.go`) |

</frozen-after-approval>

## Code Map

- `internal/api/cycle_settings.go` (new) -- `cycleSettingsInput{Income, SavingsGoal int64, CycleStartDay int}`, `CycleSettingsResult{Income Income, Settings Settings}` model (in `api.go` per convention), handler `UpdateCycleSettings`: validate all three fields, then `h.withTx(ctx, func(tx *sql.Tx) error { ... })` calling `NewIncomeRepo(tx).Upsert(...)` and `NewSettingsRepo(tx).Upsert(...)`, computing `cycleStart` via `CycleWindow(in.CycleStartDay, time.Now())` using the request's own value.
- `internal/api/repo_settings.go` -- rename/extend `UpsertSavingsGoal` to `Upsert(ctx, userID string, savingsGoal int64, cycleStartDay int) (Settings, error)`, writing both columns in the `INSERT`/`ON DUPLICATE KEY UPDATE`.
- `internal/api/repo_settings_test.go` -- update existing tests for the new signature; add a case proving `cycle_start_day` is actually persisted and updated in place.
- `internal/api/incomes.go`, `incomes_test.go` -- **delete**: `UpsertIncome`/`incomeInput` and their tests are superseded by `cycle_settings.go`. Keep `Income` model (moves to/stays in `api.go`) and `internal/api/repo_income.go`/`repo_income_test.go` (still used internally).
- `internal/api/settings.go`, `settings_test.go` -- **delete**: `UpdateSavingsGoal`/`settingsInput` and their tests are superseded. Keep `Settings` model in `api.go`.
- `internal/api/router.go` -- remove `POST /incomes` and `PUT /settings/savings-goal`; add `PUT /cycle-settings`.
- `internal/api/auth.go` -- reference pattern only for the `withTx`-with-two-repos shape (`RegisterUser`'s category-seeding block).

## Tasks & Acceptance

**Execution:**
- [ ] `internal/api/repo_settings.go` + `repo_settings_test.go` -- extend `Upsert` to take `cycleStartDay`, update tests
- [ ] `internal/api/cycle_settings.go` + `cycle_settings_test.go` -- new merged handler, model, tests covering the I/O matrix (including the mid-save cycle-day-change scenario and the rollback scenario)
- [ ] Delete `internal/api/incomes.go`, `internal/api/incomes_test.go`, `internal/api/settings.go`, `internal/api/settings_test.go`
- [ ] `internal/api/api.go` -- keep `Income`/`Settings`/`FixedCost` models; remove nothing else (repos still wired)
- [ ] `internal/api/router.go` -- swap the three routes
- [ ] repo root -- `go build ./...`, `go vet ./...`, `go test ./internal/api/...` -- zero regressions, and confirm the deleted files' symbols have no remaining references

**Acceptance Criteria:**
- Given a save with `cycleStartDay` changed from 1 to another value, when the request succeeds, then the income written in that same request is keyed to the cycle computed with the NEW day.
- Given `POST /incomes` or `PUT /settings/savings-goal`, when called after this story, then the router returns 404 (route removed).
- Given one write inside the merged transaction fails, when the handler returns, then neither the income nor the settings row reflects the failed request (rollback).

## Implementation Notes

- `gofmt -l` flagged `api.go`/`repo_settings.go`/`router.go`/`repo_settings_test.go` as not gofmt-clean after implementation — confirmed (via byte-diff before/after `gofmt -w`, ignoring line endings) this was pure CRLF-vs-LF drift from this Windows checkout's `core.autocrlf=true`, identical in nature to the same false-positive fixed in Story 1.1's `ratelimit.go`/`router.go`. Ran `gofmt -w` on the four files; content is otherwise byte-identical.

## Spec Change Log

## Review Triage Log

- **medium / patch** — `cycleSettingsInput.SavingsGoal` is a plain `int64`, so an omitted `savingsGoal` key defaults to Go's zero value and passes the `< 0` check identically to an explicit `0` — silently resetting an existing goal to 0 instead of returning a 400. This is a regression of the exact bug already fixed in Story 1.3's now-deleted `PUT /settings/savings-goal` (which used `*int64` with a required-field check) — that fix was not carried into the merged endpoint. Verified: `SavingsGoal int64` confirmed in `cycle_settings.go`. [edge-case-hunter]
- **low / patch** — `Handler.incomes`/`Handler.settings` (built in `NewHandler`) are now unused dead weight: `UpdateCycleSettings` builds its own `NewIncomeRepo(tx)`/`NewSettingsRepo(tx)` scoped to the transaction instead. Verified via repo-wide grep: no remaining reference to `h.incomes`/`h.settings` anywhere. [verification-gap]
- **low / patch** — The deleted `settings_test.go` had a dedicated `TestUpdateSavingsGoal_ZeroIsValid` happy-path test; the new `cycle_settings_test.go` only exercises `savingsGoal: 0` inside two unrelated invalid-`cycleStartDay` tests (which fail before reaching the DB) — no test proves a `savingsGoal` of exactly 0 is accepted and persisted end-to-end through the new handler. [blind-hunter]
- **low / patch** — `cycleStartDay`'s valid upper boundary (31) is never exercised as a happy path — only invalid 0/32 and valid 1/15 are tested. [blind-hunter]
- **low / patch** — The spec's own I/O matrix names the rollback scenario as "simulated DB error on the settings upsert," but the only rollback test injects the failure on the income upsert (the second write) instead — no test proves the transaction rolls back cleanly when the settings write itself is what fails. [blind-hunter]
- **low / patch** — The deliberate, spec-mandated behavior that changing `cycle_start_day` leaves old `incomes` rows as unreferenced history (no effective-dating, nothing deleted) isn't documented in code — worth a one-line comment near the `CycleWindow` call so a future reader doesn't mistake it for a leak. [blind-hunter]
- **low / reject** — `Income` has no upper-bound sanity check. Verified: matches existing convention — `fixed_costs.go`'s `Amount`, `transactions.go`'s `Amount`, etc. are equally uncapped throughout the codebase. Not a new inconsistency. [blind-hunter]
- **low / reject** — Bundling three fields into one endpoint means a client with two invalid fields only learns about one per round trip. Verified: this is the direct, unavoidable consequence of the frozen Intent's explicit mandate (one request, one `h.withTx`, matching the cycle-update-sheet's single Save action) — not a code-level defect introduced carelessly, an inherent tradeoff of the architecture-specified design. [blind-hunter]
- **low / reject** — Test helpers call `CycleWindow(cycleStartDay, time.Now())` independently of the handler's own `time.Now()` call, creating theoretical flakiness if a test run straddles a cycle-boundary rollover between the two calls. Verified: real but requires a clock-injection seam to fix properly (more than a direct correction), and the real-world probability (test execution landing exactly on a cycle-boundary midnight) is negligible. [blind-hunter]
- **defer** — No test proves `POST /incomes`/`PUT /settings/savings-goal` actually 404 now (all tests call handlers directly, bypassing `Register()`/the real router). Verified: the routes are visibly removed from `router.go`'s diff, and this is the same systemic "router wiring untested" gap already logged to `deferred-work.md` during Story 1.3's review — not a new, story-specific gap. [blind-hunter]

## Verification

**Commands:**
- `go build ./...` -- expected: succeeds, no errors
- `go vet ./...` -- expected: succeeds, no warnings
- `go test ./internal/api/... -v` -- expected: all tests pass; no reference to deleted `UpsertIncome`/`UpdateSavingsGoal` symbols remains
- `gofmt -l internal/api cmd/web` -- expected: no output

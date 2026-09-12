---
title: 'Story 1.3: Quản lý Chi phí cố định và Mục tiêu tiết kiệm'
type: 'feature'
created: '2026-09-12'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
context: []
baseline_commit: '56117aaf83aff83cbc97dd28798f99fc9328f77a'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Safe-to-spend (later stories) needs two more inputs that don't exist yet: a live list of fixed costs and a savings goal. Neither has a table, repo, or endpoint today.

**Approach:** Add `fixed_costs` (a live list, not snapshotted per cycle) with full CRUD, and `user_settings` (1:1 with user; `savings_goal` default 0, `cycle_start_day` default 1 — this story only wires `savings_goal`, `cycle_start_day` exists with its default but stays unused/unedited until Story 1.4 per epics.md). **[ASSUMPTION, backend-only]**: following the same precedent as Story 1.2 (user-confirmed), no frontend ships yet — `fixed_costs`/savings-goal UI arrives with the `cycle-update-sheet` in Story 1.5. `savings_goal` is written via its own narrow `PUT /settings/savings-goal` endpoint for now (mirroring Story 1.2's `POST /incomes` pattern); Story 1.4 absorbs it into the shared `PUT /cycle-settings` alongside `cycle_start_day`. `fixed_costs`, unlike income/savings-goal, is architecture's permanent standalone CRUD resource — never merged into `cycle-settings`.

## Boundaries & Constraints

**Always:** Follow `XxxRepo{db dbtx}`/`NewXxxRepo`/`ok()`/`fail()` conventions exactly (see `repo_category.go`/`categories.go` for List/Create/Delete shape, `transactions.go`'s `UpdateTransaction` for the update-by-id-with-not-found shape). Scope every query by `user_id`. New tables via `Migrate()`'s existing slice only — never alter existing tables. `user_settings` is keyed by `user_id` as its own primary key (true 1:1, no separate UUID) so `savings_goal`'s upsert mirrors `IncomeRepo.Upsert`'s INSERT-ON-DUPLICATE-then-re-SELECT shape. Error codes use the next free domain blocks after incomes' `4x` (`fixed_costs` uses `5x`: `40050`/`40051` bind/validation, `40450` not-found, `50050`-`50053` per-operation failures; `settings` uses `6x`: `40060`/`40061` bind/validation, `50060` upsert failure) — verify no collision before finalizing.

**Never:** Do not build `PUT /cycle-settings`, do not wire `cycle_start_day` into `CycleWindow` (Story 1.4's job), do not add any frontend UI, do not compute a "total fixed costs" or touch Safe-to-spend math (Story 1.5's job — this story only persists and lists).

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Add fixed cost | `POST /fixed-costs {name, amount}`, valid | Inserted, appears in `GET /fixed-costs` list | N/A |
| Invalid fixed cost | `name` empty or `amount <= 0` | `fail` 400 (`40051`), no row written | Field validation before repo call |
| Edit fixed cost | `PUT /fixed-costs/:id {name, amount}` on an owned row | Row updated in place, reflected in next `List` | N/A |
| Edit/delete someone else's or a missing fixed cost | `id` not owned by caller | `fail` 404 (`40450`), no row affected | `RowsAffected() == 0` check, mirroring `UpdateTransaction` |
| Set savings goal | `PUT /settings/savings-goal {savingsGoal}`, `>= 0` | Upserts the user's single settings row; `cycle_start_day` keeps its default (1) if the row is new | ON DUPLICATE KEY UPDATE, same shape as `IncomeRepo.Upsert` |
| Invalid savings goal | `savingsGoal < 0` | `fail` 400 (`40061`), no write | Field validation |

</frozen-after-approval>

## Code Map

- `internal/api/db.go` `Migrate()` -- append `fixed_costs (id CHAR(36) PK, user_id CHAR(36), name VARCHAR(120), amount BIGINT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, INDEX idx_fixed_costs_user(user_id), FK fk_fixed_costs_user->users ON DELETE CASCADE)` and `user_settings (user_id CHAR(36) PK, savings_goal BIGINT NOT NULL DEFAULT 0, cycle_start_day TINYINT NOT NULL DEFAULT 1, FK fk_user_settings_user->users ON DELETE CASCADE)`, same `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` style as existing tables.
- `internal/api/repo_fixed_cost.go` (new) -- `FixedCostRepo{db dbtx}` + `NewFixedCostRepo`. `List`, `Create`, `Update(ctx, userID, id string, name string, amount int64) (bool, error)` (bool = found, mirrors `TransactionRepo.Update`'s rows-affected pattern), `Delete`.
- `internal/api/repo_settings.go` (new) -- `SettingsRepo{db dbtx}` + `NewSettingsRepo`. `UpsertSavingsGoal(ctx, userID string, savingsGoal int64) (Settings, error)` mirroring `IncomeRepo.Upsert`'s shape (INSERT ... ON DUPLICATE KEY UPDATE savings_goal, then re-SELECT both columns).
- `internal/api/fixed_costs.go` (new) -- `FixedCost` model (in `api.go` per the established model-location convention from Story 1.2's review), handlers `ListFixedCosts`/`CreateFixedCost`/`UpdateFixedCost`/`DeleteFixedCost` mirroring `categories.go`/`transactions.go` shapes exactly.
- `internal/api/settings.go` (new) -- `Settings` model (in `api.go`, same convention), handler `UpdateSavingsGoal`.
- `internal/api/api.go` -- add `fixedCosts *FixedCostRepo`, `settings *SettingsRepo` fields to `Handler`; wire in `NewHandler`; add `FixedCost`/`Settings` structs alongside existing models.
- `internal/api/router.go` -- add `/fixed-costs` (GET/POST/PUT :id/DELETE :id) and `PUT /settings/savings-goal` to the authed group.
- `internal/api/repo_income.go`, `incomes.go` -- reference pattern for the settings upsert shape; no changes.

## Tasks & Acceptance

**Execution:**
- [x] `internal/api/db.go` -- add `fixed_costs` and `user_settings` tables to `Migrate()`
- [x] `internal/api/api.go` -- add `FixedCost`/`Settings` models, `fixedCosts`/`settings` repo fields, wire in `NewHandler`
- [x] `internal/api/repo_fixed_cost.go` + `repo_fixed_cost_test.go` -- `FixedCostRepo` CRUD, tests for List/Create/Update(found+not-found)/Delete
- [x] `internal/api/repo_settings.go` + `repo_settings_test.go` -- `SettingsRepo.UpsertSavingsGoal`, tests for insert + update-in-place (default `cycle_start_day` preserved)
- [x] `internal/api/fixed_costs.go` + `fixed_costs_test.go` -- handlers, tests for happy path + validation failure + not-found-on-update/delete
- [x] `internal/api/settings.go` + `settings_test.go` -- handler, tests for happy path + negative-goal rejection
- [x] `internal/api/router.go` -- mount all five new routes
- [x] repo root -- `go build ./...`, `go vet ./...`, `go test ./internal/api/...` -- zero regressions

**Acceptance Criteria:**
- Given no fixed costs exist, when one is created, then `GET /fixed-costs` includes it.
- Given an owned fixed cost, when its amount is edited or it is deleted, then the change is reflected immediately in the next `List` call (no separate cache to invalidate).
- Given a savings goal is set, when `SettingsRepo` is queried again, then it returns the saved `savings_goal` and the still-default `cycle_start_day` (1).

## Implementation Notes

- `internal/api/db.go`: appended `fixed_costs` and `user_settings` `CREATE TABLE IF NOT EXISTS` statements to `Migrate()`'s existing slice, matching the established style exactly (`ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`, `ON DELETE CASCADE` FKs to `users`). `user_settings.user_id` is the table's own primary key (true 1:1, no separate UUID), per the spec.
- `internal/api/repo_fixed_cost.go` (new): `FixedCostRepo{db dbtx}` + `NewFixedCostRepo`, mirroring `CategoryRepo`/`TransactionRepo` shapes. `List`/`Create` follow the established patterns; `Update(ctx, userID, id, name string, amount int64) (bool, error)` and `Delete(ctx, userID, id string) (bool, error)` both return a `RowsAffected() > 0` bool (mirroring `TransactionRepo.Update`) so the handler can turn a no-op into a 404 — `Delete` needed this too (unlike `CategoryRepo.Delete`/`TransactionRepo.Delete`, which don't report it) because the I/O matrix explicitly requires 404 on deleting someone else's/a missing fixed cost.
- `internal/api/repo_settings.go` (new): `SettingsRepo{db dbtx}` + `NewSettingsRepo`. `UpsertSavingsGoal` mirrors `IncomeRepo.Upsert`'s INSERT-ON-DUPLICATE-then-re-SELECT shape, including its known re-select-fallback behavior (swallows a non-`ErrNoRows` re-SELECT failure and returns the just-written values, with `cycle_start_day` defaulting to `1`) — left as-is per the same "mirror, don't fix" precedent Story 1.2 established. The `ON DUPLICATE KEY UPDATE savings_goal = VALUES(savings_goal)` clause only ever touches `savings_goal`, so `cycle_start_day` keeps its column default on insert and stays untouched on every subsequent update.
- `internal/api/fixed_costs.go` (new): `ListFixedCosts`/`CreateFixedCost`/`UpdateFixedCost`/`DeleteFixedCost` handlers mirroring `categories.go`/`transactions.go` shapes. Validation (`name` non-empty after trim, `amount > 0`) runs before any repo call, per the I/O matrix.
- `internal/api/settings.go` (new): `UpdateSavingsGoal` handler binds `{savingsGoal int64}`, rejects `savingsGoal < 0` (`40061`) or unparseable JSON (`40060`) before touching the repo, then returns the canonical `Settings` row via `ok()`.
- `internal/api/api.go`: added `FixedCost`/`Settings` models alongside the other domain models, and `fixedCosts *FixedCostRepo`/`settings *SettingsRepo` fields wired in `NewHandler`.
- `internal/api/router.go`: mounted `GET/POST /fixed-costs`, `PUT/DELETE /fixed-costs/:id`, and `PUT /settings/savings-goal` in the authed group.
- Error codes used: fixed costs `40050` (bad JSON), `40051` (validation), `40450` (not found on update/delete), `50050`-`50053` (per-operation failures); settings `40060` (bad JSON), `40061` (negative goal), `50060` (upsert failure) — confirmed free of collision by grepping all existing `400xx`/`404xx`/`500xx` literals in `internal/api` before assigning them.
- No changes made to `PUT /cycle-settings`, `CycleWindow`, any frontend code, or Safe-to-spend math — all explicitly out of scope per the spec.
- All I/O & Edge-Case Matrix scenarios are covered and pass: add fixed cost (`TestFixedCostRepo_Create`, `TestCreateFixedCost_HappyPath`), invalid fixed cost (`TestCreateFixedCost_ValidationFailure_EmptyName/NonPositiveAmount`), edit fixed cost (`TestFixedCostRepo_Update_Found`, `TestUpdateFixedCost_HappyPath`), edit/delete not-owned-or-missing (`TestFixedCostRepo_Update_NotFound`, `TestUpdateFixedCost_NotFound`, `TestFixedCostRepo_Delete_NotFound`, `TestDeleteFixedCost_NotFound`), set savings goal (`TestSettingsRepo_UpsertSavingsGoal_Insert/_UpdateInPlace`, `TestUpdateSavingsGoal_HappyPath`), invalid savings goal (`TestSettingsRepo` validation via handler test `TestUpdateSavingsGoal_InvalidGoal_Negative`).
- `go build ./...`, `go vet ./...`, `go test ./internal/api/... -v` (all tests passing, zero regressions) and `gofmt -l internal/api cmd/web` (no output) all confirmed clean.

## Spec Change Log

## Review Triage Log

- **medium / patch** — `PUT /settings/savings-goal`'s `settingsInput.SavingsGoal` is a plain `int64`, so an omitted `savingsGoal` key in the request body defaults to Go's zero value and passes the `>= 0` validation identically to an explicit `0` — silently resetting an existing goal to 0 instead of returning a 400. Unlike every other numeric input in the app (`amount`, `limit`), where "missing" and "invalid" both fail the same `> 0` check, `savings_goal`'s valid range legitimately includes 0, so this is a genuinely new ambiguity, not a pre-existing pattern. Verified: `SavingsGoal int64 \`json:"savingsGoal"\`` confirmed in `settings.go`. [edge-case-hunter]
- **low / patch** — No test covers a whitespace-only `name` (e.g. `"   "`) for fixed costs; the trim-then-reject behavior (`strings.TrimSpace` then `== ""` check) is already correct but unverified by any test. Trivial addition. [blind-hunter]
- **low / reject** — `fixedCostInput.validate()` doesn't bound `Name`'s length against the `VARCHAR(120)` column, so an overlong name surfaces as a 500 instead of a 400. Verified: matches existing convention exactly — `categories.go`/`transactions.go` apply the same "no app-layer length cap" pattern for their own `VARCHAR` columns. Not a new inconsistency. [blind-hunter, edge-case-hunter]
- **low / reject** — None of the five new generic DB-Exec-failure branches (`50050`-`50053`, `50060`) has a test. Verified: matches existing convention — the equivalent Exec-failure branches in `categories.go`/`budgets.go`/`transactions.go` (`50010`, `50012`, `50013`, etc.) are equally untested throughout the codebase; not a gap unique to this diff. [blind-hunter]
- **low / reject** — No `GET /settings` endpoint exists to read the current `savingsGoal`/`cycleStartDay` without a write. Verified: out of scope per this story's own frozen Approach, which deliberately mirrors Story 1.2's narrow-write-endpoint-only precedent — reading is deferred to Story 1.5's `GET /cycle/summary`, not omitted by oversight. [blind-hunter]
- **low / reject** — `FixedCostRepo.List` has no pagination or per-user cap. Verified: matches existing convention — `CategoryRepo.List`/`TransactionRepo.List`/`BudgetRepo.List` are equally unbounded; also low real-world likelihood (a personal app's recurring fixed-cost list stays small). [blind-hunter]
- **low / reject** — `List`'s `ORDER BY created_at` has no tiebreaker for same-second inserts. Verified: matches existing convention exactly — `CategoryRepo.List` uses the identical `ORDER BY created_at` with no tiebreaker. Not a new inconsistency. [blind-hunter]
- **low / reject** — `UpdateFixedCost` echoes the request input in its response instead of re-reading the persisted row. Verified: matches the explicitly-specified reference pattern this story was told to mirror — `UpdateTransaction` does the identical echo-input-don't-reread thing. [blind-hunter]
- **low / reject** — No handler-level test asserts `GET /fixed-costs` serializes an empty list as `[]` rather than `null`. Verified: marginal — `TestFixedCostRepo_List_Empty` already confirms the repo returns a non-nil `make([]FixedCost, 0)` slice at the Go level, which is what determines JSON serialization; the residual risk this test would catch is negligible. [blind-hunter]
- **defer** — `SettingsRepo.UpsertSavingsGoal`'s INSERT-then-re-SELECT is non-atomic, so a concurrent upsert for the same user between the two steps can return another request's value. Verified: identical non-atomic shape already deferred twice (`BudgetRepo.Upsert` in Story 1.1, `IncomeRepo.Upsert` in Story 1.2), here deliberately mirrored again per this story's own spec direction. [edge-case-hunter]
- **defer** — The five new routes are wired only in `router.go` and asserted only by calling handler methods directly in tests, never by routing an actual HTTP request through `h.Register(...)` — a method/handler mismatch (e.g. `PUT`/`DELETE` swapped) would ship undetected. Verified: this gap is systemic across the entire `internal/api` package (transactions/budgets/categories/incomes have the exact same untested-router-wiring gap, predating this story) — closing it properly means one router-level test pass for the whole package, not a per-story patch. [verification-gap]

## Verification

**Commands:**
- `go build ./...` -- expected: succeeds, no errors
- `go vet ./...` -- expected: succeeds, no warnings
- `go test ./internal/api/... -v` -- expected: all tests pass, including new fixed-cost/settings suites
- `gofmt -l internal/api cmd/web` -- expected: no output

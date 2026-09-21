- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: BudgetRepo.Upsert can return a Budget with an ID that doesn't exist in the DB when the post-write canonical re-SELECT fails for a reason other than "not found," and the error is swallowed instead of surfaced.
  evidence: Confirmed pre-existing via `git show` at the story's baseline commit — the pre-refactor `UpsertBudget` handler had byte-identical fallback logic (`return Budget{ID: id, ...}, nil` on any re-SELECT error). Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: Category-ownership check and the subsequent transaction/budget write are two non-atomic steps; a category deleted in between turns a would-be 400 "category not found" into a raw FK-violation 500.
  evidence: Confirmed pre-existing — the pre-refactor `ownsCategory`-then-`Exec` pattern in `transactions.go`/`budgets.go` was equally non-atomic (no transaction wrapped the check + write). Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: RegisterUser's EmailExists check runs outside any transaction before the insert, so two concurrent registrations with the same email can both pass it, and the loser's insert fails with a raw duplicate-key DB error mapped to a generic 500 instead of a 409.
  evidence: Confirmed pre-existing via `git show` at the story's baseline commit — the pre-refactor handler had the identical check-then-insert shape with no transaction and no duplicate-key error mapping. Not introduced by the Story 1.1 refactor.

## Deferred from: code review of spec-1-1-repo-refactor-verification.md (2026-09-13)

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: `TransactionRepo.Update`'s `RowsAffected()==0 → 404` check reports "not found" for a genuine same-value no-op update on an existing transaction (MySQL reports 0 affected rows when no column value actually changed, even though the WHERE clause matched a row).
  evidence: Confirmed byte-identical pre-existing behavior via `git show` at the story's baseline commit — the pre-refactor handler used the same `RowsAffected()==0` check. Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: `LoadConfig()`'s weak/default `JWT_SECRET` warning now logs unconditionally at process startup; pre-refactor it only logged inside the branch where the DB connected successfully.
  evidence: Confirmed via `git show` at baseline (`cmd/web/main.go`) that the warning previously lived inside the `else` branch of the `api.Connect()` result. This is a genuine, verified deviation from Story 1.1's "zero behavior change" charter, introduced by moving config loading to the top of `main()`. Accepted rather than reverted: harmless (log-timing only), arguably more informative (warns about a weak secret even when the DB is unreachable).

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-declare-income.md`
  summary: IncomeRepo.Upsert's INSERT-then-re-SELECT is non-atomic; a concurrent upsert to the same (user_id, cycle_start_date) row between the two steps can make the response reflect another request's amount instead of the caller's own write.
  evidence: Same non-atomic two-step upsert shape already deferred for BudgetRepo.Upsert in Story 1.1 (see above); Story 1.2 deliberately mirrors that pattern onto the new incomes table per its own spec's explicit direction. Low real-world likelihood for a single-user personal app, but worth a shared fix (e.g. wrap Upsert in a transaction, or use `SELECT ... FOR UPDATE`) if either repo is revisited.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-fixed-costs-savings-goal.md`
  summary: SettingsRepo.UpsertSavingsGoal's INSERT-then-re-SELECT is non-atomic, same class of race as BudgetRepo.Upsert (Story 1.1) and IncomeRepo.Upsert (Story 1.2) — a concurrent upsert for the same user between the two steps can return another request's value.
  evidence: Third occurrence of the same deliberately-mirrored pattern; a shared fix (transaction-wrap or `SELECT ... FOR UPDATE`) across all three repos would be more valuable than patching one at a time.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-fixed-costs-savings-goal.md`
  summary: The internal/api router's HTTP-method-to-handler wiring (router.go) is never exercised by any test — every handler test calls the Go method directly, bypassing gin's routing — so a copy-paste method/handler mismatch (e.g. swapping PUT and DELETE on adjacent route registrations) would ship undetected.
  evidence: Systemic across the whole package (transactions/budgets/categories/incomes/fixed-costs/settings all share this gap), not unique to any one story. Closing it means one router-level (httptest.Server + h.Register) test pass covering every mounted route, not a per-story patch.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-5-safe-to-spend-dashboard.md`
  summary: Story 1.5's frontend half — the `safe-to-spend-hero`, `CycleUpdateSheet` (income/fixed-costs/savings-goal/cycle-day editing via the existing `Modal`), and `DataContext` wiring (cycleSummary/fixedCosts state, saveCycleSettings/fixed-cost mutations, cross-refetch after transaction/settings changes) — split off from the backend `GET /cycle/summary` work to keep each spec under the token budget.
  evidence: User-approved split at the token-count gate; the backend half ships first as its own reviewable/testable unit (consistent with Stories 1.2-1.4's backend-first pattern), and this frontend half is meant to be picked up immediately after in the same session, not deferred indefinitely.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-5-safe-to-spend-frontend.md`
  summary: ~~**Priority follow-up, not a someday item.** `CycleUpdateSheet` pre-fills savings-goal/cycle-start-day from a client-only `DEFAULT_SETTINGS = {0, 1}` state that's never fetched from the server (no `GET /settings` exists — only `PUT /cycle-settings`, which is write-only). A returning user with real, previously-saved non-default values who opens the sheet in a new session and clicks Save without touching those two fields will have them silently overwritten back to `{0, 1}`.~~
  status: **RESOLVED 2026-09-13.** Folded `savingsGoal`/`cycleStartDay` into `GET /cycle/summary`'s existing response (backend: `CycleSummary` struct in `api.go` + `cycle_summary.go`; frontend: `CycleSummary` type in `types.ts` + `DataContext`'s initial-load effect and `refreshCycleSummary` now call `setSettings(...)` from the fetched summary instead of leaving `DEFAULT_SETTINGS` in place until the first save). `go build`/`go vet`/`go test ./internal/api/...` and frontend `tsc -b --noEmit` all clean after the change.
  evidence: Self-identified by the implementer and independently confirmed by review with a concrete demonstration. Root cause: no backend endpoint existed to read the current `user_settings` row without writing.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-5-safe-to-spend-frontend.md`
  summary: ~~The hero's "may be inaccurate" hint for fixed-costs/savings-goal can't distinguish "never set" (shows the same `0` default) from "intentionally set to 0" — same root cause as the settings-defaults gap above (no read endpoint to know whether a `user_settings` row exists at all).~~
  status: **RESOLVED 2026-09-13** — same fix as the entry above closes the root cause (real settings now load on every session).
  evidence: N/A — resolved as a side effect, no separate work needed.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-5-safe-to-spend-frontend.md`
  summary: A background `refreshCycleSummary()` call in `DataContext` can have its response land after logout (or a fast account switch) with no cancellation guard, momentarily writing stale or another session's cycle data into state.
  evidence: Real but low-probability (requires a logout exactly while a refresh is in flight); a proper fix needs a request-token or mounted-ref pattern akin to the `cancelled` flag the initial mount effect already uses, more than a direct correction — worth doing if this pattern is revisited for other reasons.

- source_spec: `_bmad-output/implementation-artifacts/spec-transaction-note-autocomplete.md`
  summary: The note-autocomplete dropdown in `TransactionModal.tsx` can be clipped by the modal's `overflow-y-auto` scroll container on short viewports since it's absolutely positioned inside that scroll box rather than portaled.
  evidence: Real on mobile/short-viewport modals; fixing needs portal-based positioning (e.g. a floating-UI style anchor), not a simple correction — worth doing if the modal gains more overlay-style widgets.

- source_spec: `_bmad-output/implementation-artifacts/spec-transaction-note-autocomplete.md`
  summary: The frontend has no test framework or test files anywhere in the project; `TransactionModal.tsx` (including the new autocomplete interaction logic) has zero coverage.
  evidence: Pre-existing gap across the whole frontend, not introduced by this change — confirmed no vitest/jest config or `*.test.*`/`*.spec.*` files exist under `frontend/src`. Worth addressing as its own initiative (pick a test runner, seed the first few component tests) rather than folding into a single feature spec.

- source_spec: `_bmad-output/implementation-artifacts/spec-transaction-note-autocomplete.md`
  summary: Note-autocomplete only recalls `note` + `categoryId`; recurring transactions (e.g. "Coffee") usually also repeat the same `amount`, which the current feature doesn't offer to fill.
  evidence: Deliberately out of scope for this spec's frozen Intent (note + category only); a natural follow-up if the feature proves useful.

- source_spec: `_bmad-output/implementation-artifacts/spec-transaction-note-autocomplete.md`
  summary: `noteSuggestions` in `TransactionModal.tsx` linearly scans all of `DataContext.transactions` on every keystroke with no memoized index.
  evidence: Not a real problem at single-user personal-finance data volumes (hundreds to low thousands of rows); worth indexing only if transaction history grows much larger than that.

- source_spec: `_bmad-output/implementation-artifacts/spec-recurring-transactions.md`
  summary: A recurring template whose category is later deleted (`ON DELETE SET NULL`) can never again be updated — `UpdateRecurringTransaction`'s validation unconditionally rejects an empty `CategoryID`, and `RecurringSection.tsx` has no category picker to fix it, so pause/resume/amount-edit become permanently unavailable for that template (delete-and-recreate, losing its schedule, is the only escape).
  evidence: Confirmed pre-existing — `transactionInput.validate()` for plain `transactions` has the identical `CategoryID == ""` rejection, so editing any regular transaction whose category was deleted already hits the same wall; not introduced by the recurring-transactions story. The recurring case makes it more likely to bite in practice since pause/resume is a normal, frequent interaction (unlike editing an old one-off transaction). A shared fix — letting an update keep an already-empty `CategoryID` — would resolve both.

- source_spec: `_bmad-output/implementation-artifacts/spec-recurring-transactions.md`
  summary: `TransactionModal.tsx`'s `submit()` has no in-flight/disabled guard, so a fast double-click on "Add" or "Confirm" can fire two concurrent requests before the first commits — for confirming a recurring draft specifically, this can create two `transactions` rows from one tap-intent.
  evidence: Confirmed pre-existing — the same unguarded `submit()` already permits duplicate-click double-adds for plain transactions; the recurring-transactions story inherits the gap for its new "confirm" action rather than introducing it. Fix (disable the button while a request is in flight) is a small, shared, worth-doing-once change across both flows.

- source_spec: `_bmad-output/implementation-artifacts/spec-recurring-transactions.md`
  summary: Neither `recurringTransactionInput.validate()` nor `recurringTransactionUpdateInput.validate()` bounds `Note` to the `recurring_transactions.note VARCHAR(255)` column limit, so an over-length note fails as a raw DB error (500) instead of a 400.
  evidence: Confirmed pre-existing — `transactionInput.validate()` for plain transactions has the identical gap against `transactions.note VARCHAR(255)`; no note-length validation exists anywhere in the codebase today. A shared fix (one length check, reused by both validators) would close both at once.

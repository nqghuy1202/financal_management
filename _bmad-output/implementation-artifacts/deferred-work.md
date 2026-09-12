- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: BudgetRepo.Upsert can return a Budget with an ID that doesn't exist in the DB when the post-write canonical re-SELECT fails for a reason other than "not found," and the error is swallowed instead of surfaced.
  evidence: Confirmed pre-existing via `git show` at the story's baseline commit — the pre-refactor `UpsertBudget` handler had byte-identical fallback logic (`return Budget{ID: id, ...}, nil` on any re-SELECT error). Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: Category-ownership check and the subsequent transaction/budget write are two non-atomic steps; a category deleted in between turns a would-be 400 "category not found" into a raw FK-violation 500.
  evidence: Confirmed pre-existing — the pre-refactor `ownsCategory`-then-`Exec` pattern in `transactions.go`/`budgets.go` was equally non-atomic (no transaction wrapped the check + write). Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: RegisterUser's EmailExists check runs outside any transaction before the insert, so two concurrent registrations with the same email can both pass it, and the loser's insert fails with a raw duplicate-key DB error mapped to a generic 500 instead of a 409.
  evidence: Confirmed pre-existing via `git show` at the story's baseline commit — the pre-refactor handler had the identical check-then-insert shape with no transaction and no duplicate-key error mapping. Not introduced by the Story 1.1 refactor.

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
  summary: **Priority follow-up, not a someday item.** `CycleUpdateSheet` pre-fills savings-goal/cycle-start-day from a client-only `DEFAULT_SETTINGS = {0, 1}` state that's never fetched from the server (no `GET /settings` exists — only `PUT /cycle-settings`, which is write-only). A returning user with real, previously-saved non-default values who opens the sheet in a new session and clicks Save without touching those two fields will have them silently overwritten back to `{0, 1}`.
  evidence: Self-identified by the implementer and independently confirmed by review with a concrete demonstration. Root cause: no backend endpoint exists to read the current `user_settings` row without writing. Fixing it needs a small backend addition (e.g. `GET /settings`, or folding `savingsGoal`/`cycleStartDay` into `GET /cycle/summary`'s existing response) — explicitly out of scope for the frontend-only spec that surfaced it. Should be picked up before this app sees real multi-session usage, since it can silently destroy real user data.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-5-safe-to-spend-frontend.md`
  summary: The hero's "may be inaccurate" hint for fixed-costs/savings-goal can't distinguish "never set" (shows the same `0` default) from "intentionally set to 0" — same root cause as the settings-defaults gap above (no read endpoint to know whether a `user_settings` row exists at all).
  evidence: Would be resolved by the same backend read-endpoint fix as the entry above; no separate work needed once that lands.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-5-safe-to-spend-frontend.md`
  summary: A background `refreshCycleSummary()` call in `DataContext` can have its response land after logout (or a fast account switch) with no cancellation guard, momentarily writing stale or another session's cycle data into state.
  evidence: Real but low-probability (requires a logout exactly while a refresh is in flight); a proper fix needs a request-token or mounted-ref pattern akin to the `cancelled` flag the initial mount effect already uses, more than a direct correction — worth doing if this pattern is revisited for other reasons.

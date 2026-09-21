---
title: 'Recurring/scheduled transactions'
type: 'feature'
created: '2026-09-21'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
context: []
baseline_commit: '79dbc2c7a4eaff452092c3100d74a7fa0aeea66f'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Logging the same recurring expense/income (rent, subscriptions, salary) requires fully retyping it every cycle. Competitive research (`_bmad-output/planning-artifacts/research/competitive-personal-finance-apps-convenience-featur-2026-09-21/research.md`) found this is the most-shipped, most-evidenced gap versus other personal-finance apps, with zero evidenced push-back against the concept and Monefy users explicitly wanting *more* recurring visibility.

**Approach:** Let a user mark a transaction as recurring with a frequency (**weekly or monthly only** — decided; no custom interval, no yearly), stored as a separate template (not bolted onto `FixedCost`, which is a budgeting constant, not a transaction-entry convenience — confirmed distinct during investigation). When a template's next occurrence is due, the app surfaces it as a one-tap-confirm suggested draft — never a silently auto-created transaction — consistent with the "suggest, don't auto-apply" precedent set by the just-shipped note/category autocomplete. If multiple occurrences are overdue (user hasn't opened the app in a while), show a **single "catch up" suggested draft** for the most recent occurrence and silently advance the schedule past the missed ones (decided — no per-occurrence flood). Recurring templates are viewed/edited/paused/deleted from a **collapsible section within the existing Transactions page** (decided — no new nav tab/section, no folding into the settings sheet). No backend scheduler exists in this codebase; due-item detection is computed on demand (e.g. on the transactions list load), the same pattern `cycle/summary` already uses.

</frozen-after-approval>

## Boundaries & Constraints

**Always:**
- New DB table only (e.g. `recurring_transactions`), added via `CREATE TABLE IF NOT EXISTS` appended to `Migrate()` in `internal/api/db.go` — this codebase has no ALTER/versioned-migration mechanism (confirmed in architecture doc and by inspection); never modify the existing `transactions` table's schema.
- A suggested draft is confirmed through the existing `TransactionModal` (pre-filled, still editable — amount/date/note/category can be adjusted before saving), producing a normal row in `transactions` exactly as if hand-entered. Confirming never bypasses existing validation.
- Due-item computation happens on request (e.g. inside `ListTransactions` or a new endpoint), not via a background job — no scheduler exists in this codebase and none should be introduced for this feature.
- Follow existing conventions: REST resource naming (`/recurring-transactions`), repo pattern (`internal/api/repo_recurring_transaction.go` wrapping `dbtx`), i18n dotted keys added to both `vi`/`en` dicts in `frontend/src/lib/i18n.ts`.

**Never:**
- Never unify this with `FixedCost` — that struct feeds the Safe-to-Spend calculation as a live, unsnapshotted budgeting constant; conflating it with transaction generation would change its semantics.
- Never silently create a real `transactions` row from a template without a user tapping confirm.
- Never introduce a cron/scheduler library or background worker for this feature.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Create recurring template | User marks a transaction recurring with a frequency | New row in `recurring_transactions`; no immediate `transactions` row created | 400 if frequency/amount/category invalid, same validation as a normal transaction |
| Due item, app opened | Today >= template's next-due-date | Suggested draft appears (surfaced list/banner), pre-filled `TransactionModal` on tap | N/A |
| Confirm suggested draft | User taps a suggested draft, adjusts nothing, saves | Normal `transactions` row created; template's next-due-date advances by one frequency step | Same as normal `CreateTransaction` errors |
| Edit before confirm | User taps a suggested draft, changes amount/date, saves | Transaction saved with edited values; template's next-due-date still advances from its *original* schedule, not the edited date | N/A |
| Delete/pause template | User deletes or pauses a recurring template | No further suggested drafts from it; already-confirmed transactions untouched | 404 if template not found/not owned |
| Multiple missed occurrences | Today is 3+ frequency-steps past `next_due_date` | A single "catch up" suggested draft appears for the most recent occurrence; `next_due_date` is recomputed to the next future occurrence on confirm (skipped occurrences leave no separate record) | N/A |

## Code Map

- `internal/api/db.go:38-131` (`Migrate`) — append a new `CREATE TABLE IF NOT EXISTS recurring_transactions` here, following the same pattern as `fixed_costs` (db.go:96-104) and `budget_alert_state` (db.go:113-124, closest analog for dismissable/trackable state).
- `internal/api/db.go:60-72` — existing `transactions` table shape (id, user_id, type, amount, category_id, note, date, created_at) to mirror for the template's transaction-shaped fields, plus new `frequency` (ENUM('weekly','monthly')), `next_due_date` (DATE), `active` (BOOL, for pause) columns.
- `internal/api/api.go:96-103` — `Transaction` struct to mirror for a new `RecurringTransaction` struct (own file or same file, matching existing convention).
- `internal/api/repo_transaction.go` (`Create` 39, `Update` 51, `Delete` 64, `List` 15) — pattern to follow for a new `internal/api/repo_recurring_transaction.go`.
- `internal/api/transactions.go` (`CreateTransaction` 47, `ListTransactions` 35, etc.) — pattern for a new `internal/api/recurring_transactions.go` handler file; due-item computation likely belongs in `ListRecurringTransactions` or a small helper computing "due today or overdue" from `next_due_date`.
- `internal/api/router.go` — existing route block (transactions, budgets, fixed-costs, cycle-settings, alerts dismiss) — add `GET/POST /recurring-transactions`, `PUT/DELETE /recurring-transactions/:id`, and a confirm action (e.g. `POST /recurring-transactions/:id/confirm` mirroring the existing `POST /alerts/:categoryId/:threshold/dismiss` pattern) that both creates the `transactions` row and advances `next_due_date` in one server-side operation (avoid two separate client round-trips that could desync).
- `internal/api/repo_fixed_cost.go` + `internal/api/fixed_costs.go` — confirmed different concept (budgeting constant, no date/frequency/transaction link); do not touch or unify.
- `frontend/src/context/DataContext.tsx:78` (`transactions` state), `:94-138` (parallel-fetch `useEffect`), `:158-183` (`addTransaction`/`updateTransaction`/`deleteTransaction`) — add a 6th parallel fetch for recurring templates + due-drafts, plus CRUD + confirm callbacks following the same optimistic-update pattern.
- `frontend/src/pages/Transactions.tsx:120-124` (modal open/editing state), `:185` (add button) — add a collapsible section above the table with two parts: (a) suggested-draft cards (if any due templates) — tapping one opens the existing `TransactionModal` pre-filled/editable; (b) a list of active recurring templates with edit/pause/delete actions. No new nav item or route.
- `frontend/src/components/TransactionModal.tsx` (`Props` at 10-18, save logic ending ~114) — extend with an optional "make recurring" toggle + frequency picker (only relevant when adding, not when confirming a draft or editing a past transaction), and support being opened pre-filled from a suggested draft.
- `frontend/src/lib/i18n.ts` — add new dotted keys (e.g. `recurring.*`) to both `vi` and `en` dicts.

## Tasks & Acceptance

**Execution:**
- [x] `internal/api/db.go` -- append `CREATE TABLE IF NOT EXISTS recurring_transactions` (id, user_id, type, amount, category_id, note, frequency ENUM('weekly','monthly'), next_due_date DATE, active BOOL DEFAULT TRUE, created_at) with FK/index mirroring `transactions` -- new persistent store for templates
- [x] `internal/api/api.go` -- add `RecurringTransaction` struct (JSON tags matching `Transaction`'s camelCase convention, plus `Frequency`, `NextDueDate`, `Active`) -- API/DB shape
- [x] `internal/api/repo_recurring_transaction.go` (new) -- `List`, `Get`, `Create`, `Update`, `Delete`, `AdvanceNextDueDate` mirroring `repo_transaction.go`. Implementation note: the atomic confirm+advance is orchestrated in the handler (`ConfirmRecurringTransaction`) rather than a single `ConfirmAndAdvance` repo method, reusing the existing `TransactionRepo.Create` and `checkBudgetThreshold` inside one `h.withTx` — same atomicity guarantee, less duplicated logic -- core recurrence logic
- [x] `internal/api/recurring_transactions.go` (new) -- `ListRecurringTransactions`, `CreateRecurringTransaction`, `UpdateRecurringTransaction`, `DeleteRecurringTransaction`, `ConfirmRecurringTransaction` handlers, same validation pattern as `transactions.go` (category ownership check, positive amount, valid frequency); `CreateRecurringTransaction` sets the template's first `next_due_date` to one frequency step after the transaction being marked recurring (that transaction itself is logged normally via the existing `POST /transactions`, the template only governs *future* occurrences) -- HTTP layer
- [x] `internal/api/router.go` -- register `GET/POST /recurring-transactions`, `PUT/DELETE /recurring-transactions/:id`, `POST /recurring-transactions/:id/confirm` under the existing authed group -- routing
- [x] `frontend/src/types.ts` -- add `RecurringTransaction` type mirroring the backend struct -- frontend/backend contract
- [x] `frontend/src/context/DataContext.tsx` -- add `recurringTransactions` state + parallel fetch in the existing load effect; add `addRecurringTransaction`, `updateRecurringTransaction`, `deleteRecurringTransaction`, `confirmRecurringTransaction` following the existing optimistic-update + `refreshCycleSummary()` pattern -- state management
- [x] `frontend/src/components/TransactionModal.tsx` -- add an optional recurring toggle + weekly/monthly picker, shown only when adding a fresh (non-editing, non-draft-confirm) transaction; when opened to confirm a suggested draft, pre-fill from the draft and hide the recurring toggle (a draft confirms into a normal transaction, it doesn't create a nested recurring template) -- entry point for marking recurring
- [x] `frontend/src/pages/Transactions.tsx` -- add a collapsible section above the table: due suggested-draft cards (tap opens `TransactionModal` pre-filled) and an active-templates list (edit/pause/delete) -- surfacing + management UI (implemented as new `frontend/src/components/RecurringSection.tsx`, rendered from Transactions.tsx)
- [x] `frontend/src/lib/i18n.ts` -- add `recurring.*` keys to both `vi` and `en` dicts for all new UI strings
- [x] Unit-test the catch-up math (weekly/monthly, single-miss and multi-miss cases) in Go, matching the I/O matrix scenarios above -- implemented as pure functions (`stepFrequency`, `AdvanceRecurrence`, `MostRecentOccurrence`, `attachDueDraft` in `internal/api/recurrence.go`) with dedicated unit tests in `recurrence_test.go`, plus handler/repo-level tests

**Acceptance Criteria:**
- Given a user creates a weekly recurring template dated today, when today reaches the next 7-day mark, then a suggested draft appears and confirming it creates a normal transaction with `next_due_date` advanced by 7 days.
- Given a monthly template is 3 months overdue (user hasn't opened the app), when the user opens the Transactions page, then exactly one "catch up" suggested draft appears (not three), and confirming it advances `next_due_date` to the next future monthly occurrence.
- Given a user edits the amount/date on a suggested draft before saving, when they save, then the transaction reflects the edited values while the template's own schedule still advances from its original (unedited) due date.
- Given a user pauses or deletes a recurring template, when a request lists recurring transactions afterward, then no further suggested draft is generated from it, and previously-confirmed transactions are unaffected.
- Given a request to confirm/update/delete a recurring template belonging to another user, when the handler processes it, then it returns 404, matching the ownership-check pattern used elsewhere (e.g. category ownership in `CreateTransaction`).

## Implementation Notes

- Backend: new table `recurring_transactions`, pure recurrence-math functions in `internal/api/recurrence.go` (unit-tested directly), `RecurringTransactionRepo`, `recurring_transactions.go` handlers, 5 new routes. Confirm is atomic (`h.withTx`: INSERT into `transactions` + `checkBudgetThreshold` + `next_due_date` advance, all-or-nothing).
- Frontend: `RecurringTransaction`/`RecurringFrequency` types, `DataContext` state + CRUD/confirm, a "make recurring" toggle in `TransactionModal` (hidden while editing/confirming a draft), new `RecurringSection.tsx` rendered above the Transactions table (due-draft cards + inline-editable templates list), i18n keys in both `vi`/`en`.
- Verified myself (not just the implementer's report): read the full diff, ran `go build`/`go vet`/`go test ./internal/api/...` and `npm run build`/`npm run lint` independently, confirmed the migration against a real local MySQL instance (`DESCRIBE recurring_transactions` matches spec exactly), and did an end-to-end Playwright click-through (demo login → mark a transaction recurring → template appears in the section with the correct next-due-date → normal transaction also logged) with a screenshot.
- Review: 3 layers (blind-hunter, edge-case-hunter, verification-gap) ran in parallel on the full diff. 6 `patch` findings (missing tests for the paused-confirm branch, the due-draft handler wiring, Update/Confirm validation-failure and ownership-failure symmetry, a duplicated category-name line in the UI, and a concurrent-delete race returning 500 instead of 404) were sent back to the original implementation subagent and fixed; re-verified independently after (25 recurring-specific tests passing, build/vet/lint clean). 1 `false` finding rejected (confirming a not-yet-due template is a no-op by design, not a bug). 3 `low` findings rejected as cosmetic/not worth the added complexity. 3 findings deferred to `deferred-work.md`, all confirmed **pre-existing** patterns shared with the plain-transaction code path (not introduced by this story): category-deleted-blocks-update, no double-submit guard, and no note-length validation.

## Design Notes

**Catch-up advancement algorithm** (in `ConfirmAndAdvance`): starting from the template's current `next_due_date`, repeatedly add one frequency step (7 days for weekly; same day-of-month next calendar month for monthly, clamped to the month's last day if the day doesn't exist, e.g. template due the 31st in a 30-day month) until the result is strictly after today. This both handles the common single-occurrence case and collapses any number of missed occurrences into the single next future date, per the decided "one catch-up draft" behavior — no loop bound needed since each iteration strictly advances the date.

**Confirm as one atomic operation:** `POST /recurring-transactions/:id/confirm` takes the (possibly edited) transaction fields in its body, and server-side does both the `transactions` INSERT and the template's `next_due_date` UPDATE inside one DB transaction — avoids a client doing two separate calls (create transaction, then separately advance the template) that could partially fail and leave the template's schedule stale or double-advanced.

## Verification

**Commands:**
- `go build ./...` -- expected: compiles clean
- `go vet ./...` -- expected: no issues
- `go test ./internal/api/... -v` -- expected: new recurrence-math tests pass, existing tests still pass
- `cd frontend && npm run build` -- expected: `tsc -b && vite build` succeeds
- `cd frontend && npm run lint` -- expected: `tsc -b --noEmit` clean

**Manual checks (if no CLI):**
- Create a weekly recurring template with a past due date, confirm the suggested-draft catch-up shows exactly one draft, confirm it, verify the resulting transaction and the template's new `next_due_date` in the Transactions page and (if convenient) via a DB query.

## Review Triage Log

- `patch` — `ConfirmRecurringTransaction`'s `!rt.Active` rejection branch (paused template) has no test; only the `rt.ID == ""` branch is covered. Verified by reading `recurring_transactions_test.go`: `TestConfirmRecurringTransaction_NotFound` only mocks `sql.ErrNoRows`, never an `Active: false` row. (verification-gap + blind-hunter, same finding)
- `patch` — `ListRecurringTransactions`'s `attachDueDraft` loop is never exercised with an actual due row at the handler level; `TestListRecurringTransactions_HappyPath` mocks zero rows. The pure function is well-tested directly, but the wiring through the handler isn't. (verification-gap)
- `patch` — `UpdateRecurringTransaction` has no validation-failure or ownership-failure test, unlike `CreateRecurringTransaction` which has both for the identical checks. (blind-hunter)
- `patch` — `ConfirmRecurringTransaction` has no invalid-body or ownership-failure test, unlike `CreateRecurringTransaction`. (blind-hunter)
- `patch` — `RecurringSection.tsx`: when a template has no note, the title falls back to the category name (`rt.note || cat?.name || '—'`) and the line directly below it shows the category name again (`cat?.name ?? '—'`) — the same text rendered twice in a row. Verified by reading the component. (blind-hunter)
- `low, patch` — `UpdateRecurringTransaction`: if the template is deleted by a concurrent request between the `Update()` write and the immediately-following `Get()` re-fetch, `saved.ID == ""` is folded into the same branch as a real `Get()` error and returns 500 instead of 404. Verified by reading the code — real but requires a race within the same user's own concurrent requests; fix is a direct correction (branch on `saved.ID == ""` specifically). (edge-case-hunter)
- `false` — Confirming a template whose `NextDueDate` is still in the future (only `Active` is checked, not due-ness) supposedly "leaves the schedule un-advanced." Checked: `AdvanceRecurrence` no-op'ing when not due is its documented, tested, correct behavior (`TestAdvanceRecurrence_NotYetDue`) — not a bug. The UI never reaches this state (confirm is only offered for rows with `dueDraftDate` set); a direct API call would just log an early transaction without disturbing the schedule, which is reasonable, not harmful. (edge-case-hunter)
- `low, rejected` — `RecurringSection.tsx`'s inline edit only exposes amount/frequency, not note/category/type. Real, but matches the spec's designed minimal management UI (Code Map: "edit/pause/delete"); a full edit form is more than a direct correction, and delete-and-recreate is an acceptable workaround. (blind-hunter)
- `low, rejected` — Templates list has no separate sort/grouping for paused/dormant templates. Cosmetic; due items are already separated into their own section above, so the plain list's order rarely matters in practice; fix would add non-trivial grouping logic. (blind-hunter)
- `low, rejected` — Frequency picker uses two different controls (toggle buttons in `TransactionModal`, `<select>` in `RecurringSection`) for the same weekly/monthly choice. Cosmetic inconsistency, doesn't affect correctness. (blind-hunter)
- `low, rejected` — `ConfirmRecurringTransaction`'s `withTx` still opens/commits a transaction on the not-found path. Negligible overhead for an endpoint used rarely in a single-user app; fix adds a branch for no real benefit. (blind-hunter)
- `defer` — A recurring template whose category is later deleted (`ON DELETE SET NULL`) can never again be updated — `UpdateRecurringTransaction`'s validation unconditionally rejects an empty `CategoryID`, and `RecurringSection.tsx` has no category picker to fix it, so pause/resume/amount-edit become permanently unavailable (delete-and-recreate is the only escape, losing the schedule). Confirmed **pre-existing**: `transactionInput.validate()` for plain transactions has the identical `CategoryID == ""` rejection, so editing any transaction whose category was deleted already hits this same wall — not introduced by this story, though the recurring case makes pause/resume specifically (a very normal interaction) hit it more often than editing an old one-off transaction would. (blind-hunter + verification-gap + edge-case-hunter, same root cause, three independent reports)
- `defer` — No double-submit guard on `TransactionModal`'s submit button; a fast double-click on "Confirm" could create two transactions before the first request's DB transaction commits. Confirmed **pre-existing**: the same unguarded `submit()` already allows duplicate-click double-adds for plain transactions today; this story doesn't introduce the gap, just inherits it for the confirm action too. (blind-hunter)
- `defer` — Neither `recurringTransactionInput.validate()` nor `recurringTransactionUpdateInput.validate()` bounds `Note` to the column's 255-char limit, so an over-length note fails as a raw DB error (500) instead of a 400. Confirmed **pre-existing**: `transactionInput.validate()` for plain transactions has the identical gap — no note-length check exists anywhere in the codebase today. (blind-hunter + edge-case-hunter, same finding)

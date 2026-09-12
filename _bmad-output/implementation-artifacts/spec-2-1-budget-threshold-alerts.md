---
title: 'Story 2.1: Phát cảnh báo khi giao dịch vượt ngưỡng ngân sách'
type: 'feature'
created: '2026-09-12'
status: 'done'
review_loop_iteration: 2
followup_review_recommended: false
context: []
warnings: ['oversized']
deferred: []
baseline_revision: '01a37bb46f34b6721a11248f0e7404c8f0a04aa5'
---

<intent-contract>

## Intent

**Problem:** A category can silently blow past its budget — the user only finds out at cycle-end review, when nothing can be done about it. Budgets exist (`budgets` table) but nothing watches spend against them as transactions are saved.

**Approach:** When an expense transaction is created or updated, check — inside the same DB transaction as the write — whether the category's spend in the current cycle just crossed 70%, 90%, or 100% of its budget for that cycle, and if so, record exactly one new `budget_alert_state` row for the single highest threshold newly crossed. This spec is backend-only: no dismiss endpoint, no `GET /cycle/summary`/`GET /budgets` wiring, no UI (Stories 2.2/2.3).

## Boundaries & Constraints

**Always:** Run the threshold check inside the same `h.withTx` as the transaction insert/update (not a separate post-commit write) — applies to both `CreateTransaction` and `UpdateTransaction` (either can raise a category's spend; `DeleteTransaction` only lowers it, so it's out of scope). Only `type == "expense"` transactions count, and only when their `date` falls inside the current cycle `[cycleStart, cycleEnd)` from `CycleWindow(settings.CycleStartDay, time.Now())`. If the category has no budget row for the current cycle's month (`cycleStart.Format("2006-01")`), skip the check entirely — no budget means no percentage. At most one new alert row per save, for the single highest threshold newly crossed. The alert write is an idempotent `INSERT ... ON DUPLICATE KEY UPDATE` so two concurrent requests crossing the same threshold never 500 or duplicate.

**Never:** Do not populate `GET /cycle/summary.activeAlerts`, add a dismiss endpoint, or touch any UI — that's Story 2.2. Do not add `spent`/`percent`/`status` to `GET /budgets` — that's Story 2.3. Do not un-trigger or delete an alert row when a transaction is later reduced or deleted. Do not use `INSERT IGNORE` — no precedent in this codebase; use `ON DUPLICATE KEY UPDATE` like every other upsert here.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| First crossing | Category budget exists, category spend below 70%; saved transaction pushes it past 70% | One `budget_alert_state` row inserted for threshold 70 | N/A |
| Already alerted at this threshold | A 90% alert row already exists this cycle; another transaction keeps spend over 90% but under 100% | No new row inserted | N/A |
| Multi-threshold jump | One large transaction pushes spend from 40% to 105% | Exactly one new row, for threshold 100 (not 70 and 90 too) | N/A |
| Concurrent same-threshold crossing | Two requests both cross 100% for the same category+cycle at once | Both succeed (200), only one row ends up persisted, no 500 | Idempotent upsert absorbs it |
| No budget set | Category has no `budgets` row for the current cycle's month | No threshold check performed, no row inserted | N/A |
| Income transaction | Transaction `type == "income"` | No threshold check performed | N/A |
| Backdated transaction | Transaction `date` falls outside `[cycleStart, cycleEnd)` | No threshold check performed for the current cycle | N/A |

</intent-contract>

## Code Map

- `internal/api/db.go` -- `Migrate()` (stmts slice, ~line 113): add `CREATE TABLE IF NOT EXISTS budget_alert_state (id CHAR(36) NOT NULL PRIMARY KEY, user_id CHAR(36) NOT NULL, category_id CHAR(36) NOT NULL, cycle_start_date DATE NOT NULL, threshold INT NOT NULL, triggered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, dismissed_at TIMESTAMP NULL, UNIQUE KEY uq_alert (user_id, category_id, cycle_start_date, threshold), CONSTRAINT fk_alert_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE, CONSTRAINT fk_alert_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` -- copy the `budgets` table's shape exactly (same file, ~line 74).
- `internal/api/cycle.go` -- add pure `func CrossedThreshold(prevSpent, newSpent, limit int64) (threshold int, crossed bool)`: if `limit <= 0` return `(0, false)`; else for thresholds `[100, 90, 70]` (checked highest-first), return the first `t` where `prevSpent*100/limit < int64(t) && newSpent*100/limit >= int64(t)`. No I/O, unit-testable like `SafeToSpend`/`DaysRemaining` in the same file.
- `internal/api/repo_transaction.go` -- add `SumExpensesInCategoryExcluding(ctx, userID, categoryID string, from, to time.Time, excludeID string) (int64, error)`: `SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND category_id=? AND type='expense' AND date>=? AND date<? AND id != ?`. **Always pass the transaction's own id as `excludeID`, on both create and update** — `Create`/`Update` both run *before* this check inside the same `h.withTx`, so by the time this query runs, the row already exists (or is already changed) in the DB with its real id; passing `""` would not exclude it.
- `internal/api/repo_budget.go` -- add `GetByCategoryMonth(ctx, userID, categoryID, month string) (Budget, error)`: `SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id=? AND category_id=? AND month=?`; propagate `sql.ErrNoRows` as-is (caller uses `ignoreNoRows`, `repo_helpers.go`).
- `internal/api/repo_alert_state.go` (new) -- `AlertStateRepo{db dbtx}` / `NewAlertStateRepo(db dbtx) *AlertStateRepo`, matching `repo_budget.go`'s shape exactly. `Trigger(ctx, userID, categoryID string, cycleStart time.Time, threshold int) error`: `INSERT INTO budget_alert_state (id, user_id, category_id, cycle_start_date, threshold) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE threshold = VALUES(threshold)` with a fresh `uuid.NewString()` id, same style as `repo_budget.go`'s `Upsert`.
- `internal/api/transactions.go` -- `CreateTransaction` (~line 42) and `UpdateTransaction` (~line 76): wrap the existing insert/update in `h.withTx` (pattern from `cycle_settings.go:51-70`). On create, generate the transaction's id (e.g. `uuid.NewString()`) *before* the closure so it's available as `excludeID`. Inside the closure, after building `NewTransactionRepo(tx)` and performing the write: if the saved transaction is `type=="expense"` and its `date` is within `CycleWindow(settings.CycleStartDay, time.Now())` (fetch settings via `NewSettingsRepo(tx).Get(ctx, userID)`), look up the category's budget via `NewBudgetRepo(tx).GetByCategoryMonth(...)` (skip silently on `sql.ErrNoRows`); if found, compute `prevSpent` via `SumExpensesInCategoryExcluding(ctx, userID, categoryID, cycleStart, cycleEnd, excludeID)` — **`excludeID` is the transaction's own id in both handlers, never `""`**, since the row already exists under that id by the time this query runs — and `newSpent = prevSpent + amount`; call `CrossedThreshold(prevSpent, newSpent, limit)`; if crossed, call `NewAlertStateRepo(tx).Trigger(ctx, userID, categoryID, cycleStart, threshold)`.
- `internal/api/api.go` -- no `Handler` field needed: `AlertStateRepo` (like `SettingsRepo`/`IncomeRepo` inside `UpdateCycleSettings`) is only ever constructed inline over `tx` inside the write transaction, per the existing tx-scoped-repo precedent in `cycle_settings.go`.

## Tasks & Acceptance

**Execution:**
- `internal/api/db.go` -- add `budget_alert_state` table to `Migrate()` -- new persistence target for alert rows
- `internal/api/cycle.go` + test -- `CrossedThreshold` pure function -- covers the I/O matrix's threshold-crossing math in isolation
- `internal/api/repo_transaction.go` + test -- `SumExpensesInCategoryExcluding` -- per-category, exclusion-aware spend total needed for both create and update paths
- `internal/api/repo_budget.go` + test -- `GetByCategoryMonth` -- fetch the one budget row relevant to a threshold check
- `internal/api/repo_alert_state.go` (new) + test -- `AlertStateRepo.Trigger` -- idempotent upsert, verify a second call with the same key doesn't error and doesn't need a duplicate-row assertion (upsert makes duplicates structurally impossible)
- `internal/api/transactions.go` + test -- rewrite `CreateTransaction`/`UpdateTransaction` to use `h.withTx` and invoke the threshold check; cover: crossing, already-alerted, multi-threshold jump, no-budget, income-type, backdated-date, and the write-failure/rollback path
- repo root -- `go build ./...`, `go vet ./...`, `go test ./internal/api/... -v` -- zero regressions

**Acceptance Criteria:**
- Given a category's budget for the current cycle and a transaction that pushes usage past 70%, 90%, or 100%, when the transaction is saved, then exactly one new `budget_alert_state` row is written in the same `h.withTx` as the transaction write.
- Given a threshold already alerted this cycle, when another transaction keeps usage above that threshold but below the next one, then no new row is written for it.
- Given one transaction that jumps usage from 40% to 105%, when saved, then only the 100% row is written, never separate 70/90/100 rows.
- Given two concurrent requests that both cross the same new threshold, when both commit, then neither request errors and only one row persists for that (category, cycle, threshold).

## Spec Change Log

### 2026-09-12 — bad_spec amendment (review pass 1)
- **Triggering findings:** edge-case-hunter, verification-gap, and intent-alignment all independently traced the same defect: on create, the Code Map instructed `checkBudgetThreshold(..., excludeID="")`, but `Create` runs *before* the check inside the same `h.withTx` and the transaction's id is a real UUID assigned before `Create` — so the just-inserted row was never excluded from `prevSpent`, double-counting its own amount on every expense create against a budgeted category.
- **What was amended:** the Code Map bullets for `repo_transaction.go` and `transactions.go`, and the Design Notes paragraph, now state that `excludeID` must be the transaction's own id on **both** create and update (never `""`), since both writes land before the check runs in the same tx.
- **Known-bad state avoided:** silently wrong or missing budget-threshold alerts on essentially every real expense transaction saved against a budgeted category — the core guarantee this story exists to provide.
- **KEEP (verified correct, must survive re-derivation):** `CrossedThreshold`'s pure-function design and its `[100,90,70]` highest-first check; the `budget_alert_state` schema shape (including the `UNIQUE(user_id, category_id, cycle_start_date, threshold)` key); `AlertStateRepo.Trigger`'s idempotent `ON DUPLICATE KEY UPDATE` upsert; the update path's use of its own row id as `excludeID` (this was already correct — only create's `""` was wrong); the expense-only/current-cycle-only/budget-existence-gated scoping; and the `h.withTx`-wrapped atomicity of the whole check.

## Review Triage Log

### 2026-09-12 — Review pass 1
- verdicts: 20 findings — high 4, medium 0, low 13, false 3, maybe-false 0
- findings:
  - `[low]` `[reject]` blind-hunter: `checkBudgetThreshold` calls `time.Now()` directly instead of taking an `asOf` param, unlike `cycle.go`'s pure functions — refuted: `cycle_summary.go:22` and `cycle_settings.go:62` already call `time.Now()` directly at the same handler-adjacent layer; the right comparison class is handlers, not `cycle.go`'s pure functions, and handlers already do this throughout the codebase.
  - `[low]` `[reject]` blind-hunter: two concurrent transactions each individually below a threshold could combine to cross it without either detecting it (REPEATABLE READ visibility) — real but not addressed by the intent (AC4 only covers two requests crossing the *same* threshold); rejected as unlikely for a single-user personal-finance app and the fix (row locking/serializable tx) is far more than a direct correction.
  - `[low]` `[patch]` blind-hunter: `epic-2-context.md` says the check runs on create/update/**delete** tx, but `DeleteTransaction` is untouched — moot this pass (code is being reverted/re-derived); may be re-raised next pass.
  - `[low]` `[patch]` blind-hunter: no test exercises an update that changes `categoryId` — moot this pass.
  - `[low]` `[reject]` blind-hunter: no dedicated rollback test for `UpdateTransaction`'s threshold-check failure — rejected: the generic `h.withTx` rollback-on-error mechanism is already proven by `TestCreateTransaction_ThresholdCheckFailure_RollsBack` in this diff and by pre-existing `cycle_settings_test.go` precedent; it is not Update-specific.
  - `[low]` `[patch]` blind-hunter: only "backdated" (date < cycleStart) is tested, not the symmetric date ≥ cycleEnd case — moot this pass.
  - `[low]` `[patch]` blind-hunter: `budget_alert_state.threshold` has no `CHECK`/`ENUM` restricting it to {70,90,100}, unlike this schema's existing `type ENUM(...)` columns — moot this pass.
  - `[low]` `[reject]` blind-hunter: failure-path tests only cover the settings-lookup erroring; no dedicated test for `GetByCategoryMonth`/`SumExpensesInCategoryExcluding`/`Trigger` erroring — rejected: same generic `h.withTx` rollback mechanism already proven, not fallible-call-specific.
  - `[false]` `[reject]` blind-hunter: no test verifies a decreasing update doesn't fire a spurious alert — refuted: `CrossedThreshold`'s own pure-function tests (`cycle_test.go`) already establish the monotonic "no crossing on lower spend" property; `checkBudgetThreshold` adds no increase/decrease-specific logic on top of it.
  - `[low]` `[defer]` blind-hunter: no test runs the new `CREATE TABLE budget_alert_state` DDL against a real DB — pre-existing, whole-codebase gap (no table in `Migrate()` is DB-tested anywhere), not introduced by this diff.
  - `[high]` `[bad_spec]` edge-case-hunter: on create, `checkBudgetThreshold(ctx, tx, uid, t, "")` passes `excludeID=""`, but `t.ID` is a real UUID assigned before `Create` runs, so the just-inserted row is not excluded from `prevSpent` — verified real; root cause is this spec's Code Map, which explicitly (and wrongly) instructed `excludeID=""` on create.
  - `[high]` `[bad_spec]` edge-case-hunter: consequence of the same bug — the "40%→105%" multi-threshold-jump AC would not actually fire the 100% alert against a real DB, since `prevSpent` would already include the new transaction's amount — same root cause as above.
  - `[low]` `[reject]` edge-case-hunter: `prevSpent*100`/`newSpent*100` in `CrossedThreshold` could theoretically overflow `int64` for astronomically large amounts — rejected: input amounts at that magnitude (~9.2e16 VND) are unreachable for a personal-finance app, and no arithmetic elsewhere in this codebase (e.g. `SafeToSpend`) guards against overflow either; fix would introduce unprecedented complexity.
  - `[high]` `[bad_spec]` verification-gap: same create-path double-counting defect, independently derived with concrete numbers from the diff's own test fixtures — filed disposition `patch`, but the traced root cause is this spec's Code Map/Design Notes, which explicitly directed `excludeID=""` on create; reclassified `bad_spec` per the "prefer bad_spec when in doubt" rule since the implementation faithfully followed the (wrong) spec.
  - `[low]` `[reject]` intent-alignment: dedup-by-recomputation never reads `budget_alert_state`, so behavior could "silently change" if a budget's `limit_amount` is edited mid-cycle after a threshold fired — refuted: always recomputing against the *current* limit is the correct, self-correcting behavior; no requirement says otherwise, and Story 2.1 doesn't address budget-limit edits at all.
  - `[low]` `[reject]` intent-alignment: AC4's concurrency guarantee is asserted by code shape (idempotent upsert) but never exercised by a real concurrent-DB test — rejected: matches the whole codebase's existing testing limits (no real-DB concurrency tests exist anywhere, Docker unavailable here), not a gap unique to this diff; a personal single-user app rarely exercises true concurrent writes to one category.
  - `[false]` `[reject]` intent-alignment: scope preconditions (expense-only, cycle-window-only, budget-existence-gated) are the spec's own additions, not literal AC text — refuted as a defect: these are the only defensible reading (e.g. an income transaction has no meaningful "% of budget"), which is exactly the intent-resolution step-01/02 require, not scope creep.
  - `[false]` `[reject]` intent-alignment: the AC lists both `INSERT IGNORE` and `ON DUPLICATE KEY UPDATE` as acceptable; the diff uses only the latter — auditor itself notes this is "within the letter of the intent," not a divergence.
  - `[high]` `[bad_spec]` intent-alignment: same create-path double-counting defect (the audit's most consequential divergence) — same root cause as above.
  - `[low]` `[defer]` intent-alignment: "new table via `Migrate()`" is verified only by source presence, never by executing the DDL against a real database — same pre-existing, whole-codebase gap as blind-hunter's DDL finding above; grouped with it.

### 2026-09-13 — Review pass 2 (post bad_spec re-derivation)
- verdicts: 16 findings — high 0, medium 1, low 13, false 2, maybe-false 0
- findings:
  - `[low]` `[patch]` blind-hunter: pass-1's own Review Triage Log summary line miscounted its findings ("low 11, false 5" vs. the itemized list's actual low 13/false 3) — verified true (arithmetic error). **Fixed:** corrected the pass-1 summary line above directly.
  - `[false]` `[reject]` blind-hunter: `epic-2-context.md` lists `INSERT IGNORE`/`ON DUPLICATE KEY UPDATE` while the spec forbids the former — refuted: not a contradiction, the epic doc describes the AC's stated *options* (both explicitly permitted), the spec makes a valid, reasoned narrower choice within that permitted set.
  - `[medium]` `[patch]` edge-case-hunter: `checkBudgetThreshold` parses `t.Date` with `time.Parse` (always UTC) but compares it against `cycleStart`/`cycleEnd` from `CycleWindow(..., time.Now())` (server-local location) — verified real via manual instant-arithmetic: for any negative-UTC-offset server timezone, a transaction dated exactly on the cycle-start day is wrongly excluded (and other boundary shifts are possible). Low likelihood for this app's probable UTC/Vietnam(+7)-offset deployment, but it's a genuine, cheap-to-fix correctness defect, not a style nit — graded up per "harm real but severity uncertain, pick the higher grade." **Fixed:** `time.Parse` → `time.ParseInLocation(dateLayout, t.Date, cycleStart.Location())`.
  - `[low]` `[patch]` blind-hunter: `checkBudgetThreshold`'s `excludeID` parameter is always `t.ID` at both call sites — keeping it as a separate parameter is an unnecessary footgun for a future call site to reintroduce the just-fixed bug — verified true, fix is a simple internal simplification (use `t.ID` directly, drop the parameter). **Fixed:** parameter removed; function now uses `t.ID` internally; both call sites updated.
  - `[low]` `[patch]` blind-hunter (carried from pass 1): no test exercises an update that changes `categoryId` while crossing a threshold — still true, no longer moot since this pass has no bad_spec/intent_gap. **Fixed:** added `TestUpdateTransaction_CategoryChange_CrossesThresholdInNewCategory`.
  - `[low]` `[patch]` blind-hunter (carried) + verification-gap (pre-verified, filed `patch`): only `date < cycleStart` is tested; no symmetric test for `date >= cycleEnd` — grouped, same root cause (missing future/next-cycle-date test case). **Fixed:** added `TestCreateTransaction_FutureCycleDate_SkipsCheck`.
  - `[low]` `[patch]` blind-hunter (carried from pass 1): `budget_alert_state.threshold` has no `CHECK` restricting it to {70,90,100}, unlike this schema's `ENUM` columns — still true, no longer moot. **Fixed:** added `CHECK (threshold IN (70, 90, 100))` to the `CREATE TABLE budget_alert_state` statement.
  - `[false]` `[reject]` blind-hunter: no coverage for a budget row with non-positive `limit_amount` — refuted: `UpsertBudget`'s input validation (`budgets.go`) already rejects `Limit <= 0` at write time, so this state is unreachable in practice; the exact `limit<=0` branch is separately unit-tested at the pure-function level (`TestCrossedThreshold_NoBudget_LimitZeroOrNegative`).
  - `[false]` `[reject]` blind-hunter: Design Notes claim `CrossedThreshold`'s arithmetic is "reusable" for Story 2.3 but no exported percent-helper exists — refuted as a defect for *this* story: the note's precision doesn't affect Story 2.1's own correctness, and Story 2.3's own spec process will re-derive what it needs regardless.
  - `[low]` `[patch]` blind-hunter: `checkBudgetThreshold`'s doc comment lists 3 no-op conditions but omits the 4th (`CrossedThreshold` itself returning `crossed=false`) — true, cosmetic, bundled into the same patch pass as a one-line comment addition. **Fixed:** added a fourth bullet to the doc comment.
  - `[low]` `[reject]` edge-case-hunter: Update path doesn't independently re-test already-alerted/multi-jump/income/backdated/rollback (only Create does) — rejected: `checkBudgetThreshold` is one shared function with no Create/Update-specific branching in those paths; Update's one actual point of difference (`excludeID`=own row) already has dedicated coverage (`TestUpdateTransaction_ThresholdCrossing_ExcludesOwnRow`), so duplicating the rest would re-test identical code.
  - `[low]` `[reject]` edge-case-hunter (carried from pass 1): `int64` overflow in `CrossedThreshold`'s `*100` math — same reasoning as pass 1, still rejected (unrealistic magnitude, no precedent for overflow guards elsewhere).
  - `[low]` `[reject]` intent-alignment (carried, `carried`): dedup-by-recomputation vs. persisted-state surface mismatch — same as pass 1, code and reasoning unchanged.
  - `[low]` `[reject]` intent-alignment (carried, `carried`): AC4 concurrency guarantee unverified by real-DB test — same as pass 1; this pass's auditor independently confirms it was "raised twice... and dispositioned reject" already.
  - `[false]` `[reject]` intent-alignment (carried, `carried`): implicit gating (expense/cycle-window/budget-existence) as an "interpretive addition" — same as pass 1, still the only defensible reading.
  - `[low]` `[defer]` intent-alignment (carried, `carried`): `Migrate()` DDL never executed against a real DB — same pre-existing whole-codebase gap as pass 1.

## Design Notes

`CrossedThreshold` takes raw spent/limit amounts rather than pre-computed percentages so its integer math (`spent*100/limit`) is the single source of truth for "what percent is this" — Story 2.3 will need equivalent percent math for `GET /budgets` and should reuse this arithmetic rather than re-deriving it. `SumExpensesInCategoryExcluding`'s `excludeID` parameter is what makes one function correct for both create and update: because `Create`/`Update` both run *before* the threshold check inside the same `h.withTx`, the row being saved already exists under its real id by the time this query runs — on **both** paths, `excludeID` must be that row's own id, or its own amount gets double-counted into `prevSpent`. This also handles an update's category change correctly: filtering the "before" total by the transaction's *new* category and excluding its own id naturally leaves out only the other transactions, regardless of what category the row was in previously.

## Verification

**Commands:**
- `go build ./...` -- expected: succeeds, no errors
- `go vet ./...` -- expected: no warnings
- `go test ./internal/api/... -v` -- expected: all tests pass, including new `TestCrossedThreshold_*`, `TestBudgetRepo_GetByCategoryMonth_*`, `TestTransactionRepo_SumExpensesInCategoryExcluding_*`, `TestAlertStateRepo_Trigger_*`, and updated `TestCreateTransaction_*`/`TestUpdateTransaction_*` suites
- `gofmt -l internal/api` -- expected: no output

## Auto Run Result

**Summary:** Implemented compute-on-write budget threshold alerts: saving an expense transaction (create or update) now checks, inside the same DB transaction as the write, whether the category's spend in the current cycle newly crossed 70/90/100% of its budget, and if so writes exactly one `budget_alert_state` row for the single highest threshold crossed. Went through one `bad_spec` loopback (a real double-counting defect traced to this spec's own Code Map) and a second review pass that applied 5 low/medium patches; both fully re-verified independently, not just via subagent report.

**Files changed:**
- `internal/api/db.go` -- added `budget_alert_state` table to `Migrate()`, with `CHECK (threshold IN (70,90,100))`.
- `internal/api/cycle.go` (+`cycle_test.go`) -- added pure `CrossedThreshold(prevSpent, newSpent, limit) (threshold int, crossed bool)`.
- `internal/api/repo_transaction.go` (+test) -- added `SumExpensesInCategoryExcluding`.
- `internal/api/repo_budget.go` (+test) -- added `GetByCategoryMonth`.
- `internal/api/repo_alert_state.go` (new, +test) -- `AlertStateRepo.Trigger`, idempotent `ON DUPLICATE KEY UPDATE` upsert.
- `internal/api/transactions.go` -- `CreateTransaction`/`UpdateTransaction` now run inside `h.withTx`; shared `checkBudgetThreshold(ctx, tx, userID, t)` helper (no `excludeID` parameter -- uses `t.ID` internally); date-boundary comparison uses `time.ParseInLocation` (timezone-safe).
- `internal/api/transactions_test.go` -- scenario coverage for all 7 I/O matrix rows plus category-change-on-update and future-cycle-date cases.
- `_bmad-output/implementation-artifacts/epic-2-context.md` -- corrected to say create/update (not delete) and `ON DUPLICATE KEY UPDATE` only.

**Review findings breakdown:**
- Pass 1: 20 findings -- 4 `high`/`bad_spec` (one shared root cause: create-path double-counting, fully re-derived), 13 `low`, 3 `false`. All 13 lows and 3 falses were rejected/deferred/mooted that pass (bad_spec superseded them).
- Pass 2 (post re-derivation): 16 findings -- 0 high, 1 `medium`/patch (UTC/local timezone mismatch in the new date-boundary check -- fixed), 6 `low`/patch (own pass-1 log tally error; `excludeID` parameter footgun; missing category-change-update test; missing future-cycle-date test; missing `CHECK` constraint; incomplete doc comment -- all fixed), 2 `false`/reject, 2 `low`/reject (redundant Update-path re-testing; carried int64-overflow rejection), 3 carried `low`/`false` rejects (dedup-surface, AC4-concurrency-test, implicit-gating), 1 carried `low`/defer (DDL never run against a real DB).
- All patches applied and independently re-verified by re-running `go build`/`go vet`/`gofmt`/`go test ./internal/api/... -v` myself (not just trusting the implementation subagent's report), including direct source `grep` confirming the `time.ParseInLocation` and `CHECK` constraint are actually present.

**Follow-up review recommendation:** `false`. This pass patched 1 `medium` (not `high`) and no other medium — below the "two or more medium" bar, so no follow-up pass is warranted per the workflow's convergence rule.

**Verification performed:**
- `go build ./...`, `go vet ./...` -- clean.
- `gofmt -l internal/api` -- no output.
- `go test ./internal/api/... -v` -- all tests pass (verified test count and explicit `--- PASS` lines for every one of the 7 I/O matrix scenarios, re-run after patching).
- `go build ./...` and `go test ./...` at repo root -- only 2 pre-existing, unrelated failures: `internal/database` (needs Docker, unavailable on this host) and `internal/pkg/tests/basic` (pre-existing intentionally-broken scaffold test).
- Manual diff review of the full unified diff against baseline `01a37bb46f34b6721a11248f0e7404c8f0a04aa5`, twice (once before the bad_spec loopback, once after), reading actual source lines rather than relying solely on subagent summaries.

**Residual risks (all explicitly triaged `reject`/`defer` above, not blocking):**
- AC4's concurrency guarantee is proven by code shape (`ON DUPLICATE KEY UPDATE`) and a sequential-call idempotency test, not a real concurrent-database test -- consistent with this codebase's total absence of real-DB concurrency testing infrastructure.
- The new `budget_alert_state` DDL is never executed against a real MySQL instance in this environment (Docker unavailable) -- same pre-existing gap as every other table in `Migrate()`.
- Dedup is arithmetic-recomputation-based, not a lookup against `budget_alert_state` -- correct and self-correcting under the current design, but means the alert table itself is never read as part of the check.

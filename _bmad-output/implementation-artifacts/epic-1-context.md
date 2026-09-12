# Epic 1 Context: Biết mỗi ngày còn tiêu được bao nhiêu (Safe-to-spend)

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Give the user, the moment they open the app, a single trustworthy number for how much they can still safely spend today — computed from declared income, fixed costs, and a savings goal, divided across the days left in their budget cycle. This epic also finishes an in-progress backend repository-layer refactor, since every new capability in this and later epics is built on that foundation; the refactor itself delivers no user-facing value and is bundled here rather than split into its own epic.

## Stories

- Story 1.1: Complete & verify the in-progress repository-layer refactor (no regressions in existing auth/transactions/categories/budgets behavior)
- Story 1.2: Declare income for the current cycle
- Story 1.3: Manage fixed costs and savings goal
- Story 1.4: Configure the budget cycle start day
- Story 1.5: Show Safe-to-spend on the Dashboard

## Requirements & Constraints

- v1 assumes a single, fixed, monthly income entered manually per cycle — no multi-source or irregular income handling.
- Income never carries over automatically between cycles; each new cycle starts as "not declared" and requires re-confirmation.
- Missing fixed costs or savings goal default to 0 rather than blocking Safe-to-spend, but the user must be able to see a value is unconfirmed.
- Cycle start day defaults to day 1 and is user-configurable; changing it must immediately affect Safe-to-spend and days-remaining calculations.
- Safe-to-spend is a signed integer (VND) and must never be clamped to 0 — negative means overspent.
- Dashboard must recompute Safe-to-spend on every load (no stale cache) and reflect a just-saved transaction immediately.
- Reliability: after the refactor, every existing CRUD behavior (transactions, categories, budgets, auth, dashboard) must produce results identical to pre-refactor for the same input — no data loss, no discrepancies.
- Performance: Safe-to-spend and budget status should load in under ~1s on a typical mobile connection.
- Security: financial data stays strictly scoped to its owning user (existing JWT + bcrypt + user-scoping); no new third-party tracking/analytics.
- Cost: no new paid infrastructure — stays self-hosted/free-tier.
- Success signal for this epic: users actually open the Dashboard and check Safe-to-spend multiple times per week across a full cycle, not just as a technical pass/fail.

## Technical Decisions

- Extend the existing `internal/api` package using its established layered-lite convention only: `XxxRepo{db dbtx}` + `NewXxxRepo`, handlers as `*Handler` methods registered in `router.go`, `ok()`/`fail()` response envelopes. Do not introduce a controller/service layer.
- Never read, modify, or take convention cues from the parallel "learning scaffold" (`internal/controller`, `internal/services`, `internal/repo`, `internal/routers`, `internal/server`, `internal/database`, `internal/initialize`, `internal/middlewares`, `global/`, `config/*.yaml`, `cmd/api`, `cmd/server`, `cmd/cli`) — it is an unrelated, already-flagged experiment.
- New tables are added only via the existing `Migrate()` (`CREATE TABLE IF NOT EXISTS`); the four original tables are never altered. This epic introduces `user_settings` (1:1 with user; `cycle_start_day` default 1, `savings_goal` default 0), `fixed_costs` (a live list, not snapshotted per cycle), and `incomes` (`UNIQUE(user_id, cycle_start_date)`, upserted the same way the existing `BudgetRepo.Upsert` works).
- A single pure function, `CycleWindow(cycleStartDay, asOf) (start, end)` in `cycle.go`, is the one source of truth for "what cycle is this" across the whole app; `end` is always exclusive. Every later story and epic imports it rather than reimplementing it. Changing `cycle_start_day` takes effect immediately for future calls — prior `incomes` rows tied to the old cycle become unreferenced history, not deleted (no effective-dating in v1).
- `budgets.month` keeps its existing column and format; it is reinterpreted everywhere it's read — including the existing Budgets page, not just new endpoints — as "the calendar month containing the current cycle's start date," so there is exactly one notion of "current period" app-wide.
- `PUT /cycle-settings` writes income and settings (`savings_goal`, `cycle_start_day`) together in one `h.withTx`, matching the cycle-update-sheet's single Save action; `fixed_costs` stays separate CRUD.
- `GET /cycle/summary` is computed once in the backend and returned as-is to the frontend (no client-side recomputation of thresholds or totals): `safeToSpend` (signed int, VND), `daysRemaining`, `income`/`previousIncome` (`int64` or `null` — `null` means "not declared this cycle," distinct from `0`).
- Conventions inherited from the rest of the codebase: IDs are UUID `CHAR(36)`; money is `BIGINT` VND with no decimals; dates are SQL `DATE` / `"2006-01-02"` / JSON `"yyyy-mm-dd"`; every new table is scoped by `user_id`.
- Story 1.1 must land first: finish splitting `repo_budget.go`, `repo_category.go`, `repo_helpers.go`, `repo_transaction.go`, `repo_user.go`, `config.go` (already present, uncommitted) and confirm every existing endpoint (`/auth/*`, `/categories`, `/transactions`, `/budgets`) behaves identically to pre-refactor before adding any new repo.

## UX & Interaction Patterns

- `safe-to-spend-hero`: always the first element on the Dashboard, full-width at every breakpoint, brand-50 background, the number rendered in the `display-number` style (40px/800); a pencil icon opens the cycle-update sheet/modal.
- Missing-income state: the hero shows "Update income to see Safe-to-spend" instead of a misleading number. If income is declared but fixed costs/savings goal are still unconfirmed, the hero still shows a number (treating them as 0) with a small `ink-500` hint that it may be inaccurate — not an alert banner.
- `cycle-update-sheet`: bottom sheet on mobile (`<md`), centered modal on desktop (`≥md`). Income field pre-fills the previous cycle's value as a suggestion only, never auto-applied. Fixed costs and savings goal are shown and editable by default; the "Advanced" section (cycle start day) is collapsed by default. Invalid income (`≤0` or blank) shows an inline field error and does not close the sheet or discard other entered values.
- Cold load shows a skeleton for the hero and stat cards, preserving layout to avoid jank.
- Voice and tone: state the event and the number, no judgment, no emoji/exclamation points, no gamification (e.g. "Today you can still spend 185,000đ", not "You're doing great!").
- Accessibility: minimum 40×40px touch target for the pencil icon (existing `.btn-icon` standard); new colors meet WCAG AA; respect existing `prefers-reduced-motion`.

## Cross-Story Dependencies

- Story 1.1 (repository refactor) is the foundation for every other story in this epic and for Epics 2 and 3, which add further repos on top of it.
- Story 1.2 creates `CycleWindow()` in `cycle.go` using a hardcoded default start day of 1; Story 1.4 later edits that same function to read `user_settings.cycle_start_day` instead — not a rewrite, a completion of the same function.
- Story 1.4's `PUT /cycle-settings` merges the income write from Story 1.2 and the settings write (`savings_goal` from Story 1.3, `cycle_start_day`) into a single transaction, so it depends on both of those stories' data shapes existing first.
- Story 1.5 (Dashboard display) consumes the outputs of Stories 1.2–1.4 but must correctly render "not declared" states even before a user has completed them.
- Epics 2 and 3 both depend on `CycleWindow()`/`PreviousCycles()` established in this epic as their single source of truth for cycle boundaries.

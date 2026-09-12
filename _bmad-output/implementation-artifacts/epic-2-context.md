# Epic 2 Context: Được cảnh báo trước khi vượt ngân sách

<!-- Generated from planning artifacts. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Let the user know, inside the app and at the moment it happens, that a spending category has crossed 70%, 90%, or 100% of its budget for the current cycle — instead of discovering it only at cycle-end review. The same underlying computation also powers an always-available three-level status view (within/near/over) per category on both the Dashboard and the Budgets page, so a user can check status proactively even without a new alert. Builds directly on `CycleWindow()` from Epic 1; no dependency on Epic 3.

## Stories

- Story 2.1: Trigger alerts when a transaction crosses a budget threshold
- Story 2.2: View and dismiss active budget alerts on the Dashboard
- Story 2.3: View per-category 3-level budget status

## Requirements & Constraints

- Alert thresholds are fixed at 70% / 90% / 100% of a category's budget for the current cycle; no other percentages.
- Each (category, threshold, cycle) combination alerts at most once — repeated transactions that keep a category above an already-triggered threshold never create a duplicate alert. A new cycle resets this state implicitly (state is always scoped to the current cycle's start date).
- A dismissed alert never reappears in the same cycle, but dismissal only hides it from the Dashboard — it does not un-trigger the underlying threshold state.
- An alert from a past cycle must never surface in the current cycle's view, even if it was never dismissed.
- The three status levels are exactly: within (<70%), near (70–99%), over (≥100%) — no fourth state, and 0% spent is still "within."
- The Budgets page and Dashboard must always agree on a category's status — there is exactly one computation, not two independent ones.
- Alerts and status must load as part of the existing ~1s Dashboard performance budget (NFR2); computation happens once on the backend, never recomputed on the client (consistent with Epic 1's `GET /cycle/summary`).
- Accessibility floor (NFR5) applies to both new components: `alert-banner` needs `role="alert"` and a dismiss control with an `aria-label` naming the category; neither component may rely on color alone to convey meaning.
- Success signal: over a full cycle, the user receives at least one active alert before a category is fully exhausted (100%) — not just a technical pass/fail. Counter-signal: alert volume should stay roughly ≤5–7/week even with many categories; a higher rate signals the thresholds need revisiting, not that the feature is working harder.
- All new writes stay scoped to the owning user (existing JWT/user-scoping convention); no new endpoint returns unscoped data.

## Technical Decisions

- All cycle/threshold math reuses the single `CycleWindow()` (and, for status, no `PreviousCycles()` — that's Epic 3 only) already established in `cycle.go`; this epic does not define its own notion of "current cycle."
- New table `budget_alert_state` (via existing `Migrate()`, not an ALTER): `id`, `user_id`, `category_id`, `cycle_start_date`, `threshold` (70/90/100), `triggered_at`, `dismissed_at` (nullable). `UNIQUE(user_id, category_id, cycle_start_date, threshold)`.
- `budget_alert_state` is append-only except for `dismissed_at`. Reads for "active alerts" always filter by `cycle_start_date = CycleWindow(asOf).start` in addition to `dismissed_at IS NULL` — never one without the other.
- Compute-on-write: the threshold check and any resulting insert into `budget_alert_state` run inside the *same* `h.withTx` as the transaction create/update, not as a separate write after commit — `delete` is out of scope (Story 2.1) since removing a transaction only lowers spend, never crosses a threshold upward. At most one new alert row is inserted per category per write — the single highest threshold newly crossed (e.g., a jump from 40% to 105% in one save produces only the 100% row, not three). Inserts use `ON DUPLICATE KEY UPDATE` (Story 2.1 chose this over `INSERT IGNORE` for consistency with every other upsert in this codebase) so two concurrent requests triggering the same threshold never surface as a 500.
- `budgets.month` is interpreted everywhere (including the existing Budgets page) as "the calendar month containing the current cycle's start date," not the system clock's month — this was established in Epic 1 (AD-4) and this epic's Story 2.3 is the first consumer that depends on it for correctness once `cycle_start_day != 1`.
- `GET /budgets` (existing endpoint) is extended with `spent`, `percent`, `status` per row, computed by the exact same function `GET /cycle/summary` uses — the Budgets page consumes these fields rather than recomputing them client-side.
- `GET /cycle/summary.activeAlerts` returns entries shaped `{ categoryId, threshold, status }`; `status` is always the lowercase English string `"within" | "near" | "over"`, never a number or Vietnamese text.
- New endpoint `POST /alerts/:categoryId/:threshold/dismiss` sets `dismissed_at`; it does not delete or reset the row.
- Conventions inherited from the rest of the codebase (and Epic 1): UUID `CHAR(36)` ids, `BIGINT` VND with no decimals, dates as SQL `DATE`/`"2006-01-02"`/JSON `"yyyy-mm-dd"`, `{code, message, data}` response envelope via `ok()`/`fail()`, new repo as `AlertStateRepo{db dbtx}` + `NewAlertStateRepo`, handlers as `*Handler` methods registered in `router.go`. No controller/service layer, and no reading/referencing the "learning scaffold" packages excluded by AD-2.

## UX & Interaction Patterns

- `alert-banner`: card with a thick left border, amber for "near" / rose for "over," with a dismiss (X) button; stacked vertically (not a grid) directly under `safe-to-spend-hero`, full-width at every breakpoint. Shown up to 3 at a time — "over" banners always take priority over "near," with any overflow collapsed into "+N alerts, see Budgets." Never auto-dismisses like a Toast; the user must actively close it.
- No active alerts means no banner at all — deliberate silence, not an empty-state message.
- `budget-status-pill`: reuses existing brand/amber/rose chip styling (no new colors) to replace the current binary indicator on the Budgets page and Dashboard summary; maps exactly to within/near/over.
- Voice and tone: banners state the category, the percentage, and days remaining, e.g. "Ăn uống: đã dùng 92% ngân sách, còn 6 ngày" — never judgmental phrasing, no emoji or exclamation points, no gamification.
- Accessibility: `alert-banner` uses `role="alert"` so screen readers announce it on appearance; the dismiss control's `aria-label` names the category; respects existing `prefers-reduced-motion` (fade only, no slide/bounce).

## Cross-Story Dependencies

- Story 2.1's compute-on-write is the source of the rows Story 2.2 reads and dismisses — 2.2 cannot be meaningfully tested without 2.1 already writing `budget_alert_state`.
- Story 2.3 (per-category status on the Budgets page) depends only on the shared `cycle.go` status computation and the extended `GET /budgets`, not on the alert rows from 2.1/2.2 — it can be built and verified independently of them, but must produce results identical to whatever `GET /cycle/summary.budgets` shows for the same category.
- This entire epic depends on `CycleWindow()` established in Epic 1 (Story 1.1–1.4) as its only source of "what cycle is this." It does not depend on, and is not depended on by, Epic 3.

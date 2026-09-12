---
title: 'Story 1.5 (frontend): safe-to-spend-hero + cycle-update-sheet'
type: 'feature'
created: '2026-09-12'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
context: []
baseline_commit: '1887742cfecf1c11bd2cf83677d60e428be577e5'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `GET /cycle/summary` and `PUT /cycle-settings`/`/fixed-costs` all exist server-side (Stories 1.2-1.5 backend), but there is still zero frontend UI for any of it — no way to see or edit income, fixed costs, savings goal, or cycle start day.

**Approach:** This is the frontend half of Story 1.5, split from the backend half (`GET /cycle/summary`, already shipped) at the token-count gate. Add `safe-to-spend-hero` (first element on the Dashboard) showing the number/skeleton/missing-income state, a pencil icon opening `CycleUpdateSheet` (income + fixed-costs list + savings goal + collapsed "Advanced" cycle-day section, saved via the existing `PUT /cycle-settings`), and wire `DataContext` to fetch/refresh `cycleSummary`/`fixedCosts` and expose the new mutations.

**[ASSUMPTION, implementation-detail scope decisions, not product intent]:**
- Reuse the existing `Modal` component (centered at all breakpoints) instead of building a new responsive bottom-sheet — deviates from EXPERIENCE.md's mobile-sheet/desktop-modal split; a deliberate, documented simplification, revisit later if it matters in practice.
- Fixed costs render as a small editable list (name + amount + delete, plus an "add" row) inside the sheet, matching PRD FR-3's plural "thêm/sửa/xoá các khoản" and the backend's already-built full CRUD.
- Validation errors show as one message near the relevant field (mirrors `TransactionModal.tsx`'s existing single-`error`-string convention exactly — not a new per-field error system) — this already satisfies EXPERIENCE.md's "don't close the sheet, don't lose other fields" requirement; only the exact visual positioning differs from the UX doc's ideal.
- Fixed-cost add/edit/delete call their own `/fixed-costs` endpoints immediately (per architecture: "fixed_costs stays separate CRUD"); income/savings-goal/cycle-day are bundled into the sheet's one `PUT /cycle-settings` Save action.

## Boundaries & Constraints

**Always:** Mirror `DataContext.tsx`'s existing `useEffect`+`useState`+`Promise.all` pattern exactly — no new state library, no new context. Mirror `TransactionModal.tsx`'s structure (`useData()`+`useToast()`+`useI18n()`, `Modal` wrapper, footer Cancel/Save buttons) for `CycleUpdateSheet`. Use `.card`/`.input`/`.label`/`.btn-primary`/`.btn-icon`/`.btn-outline`/`.tnum` tokens as-is; add the one new `.display-number` token to `index.css` per DESIGN.md (40px/800/line-height 1.1/letter-spacing -0.01em). Refetch `cycleSummary` after any mutation that can change it: `saveCycleSettings`, fixed-cost add/edit/delete, and the existing `addTransaction`/`updateTransaction`/`deleteTransaction` (Safe-to-spend now depends on transactions). All new user-facing strings go through `useI18n()`'s `t()`, matching every existing string in the codebase (new keys added to `frontend/src/lib/i18n.ts`).

**Never:** Do not touch the Budgets page or `GET /budgets`. Do not build `budgets[].status`/`activeAlerts[]` UI (empty arrays from the backend — nothing to render yet, Epic 2's job). Do not add a new bottom-sheet/responsive-drawer component. Do not change any existing backend contract.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Full inputs declared | `cycleSummary.income` is a number | Hero shows `safeToSpend` (`.display-number`) + "Còn N ngày trong chu kỳ này" | N/A |
| Income not declared | `cycleSummary.income === null` | Hero shows "Cập nhật Lương để xem Safe-to-spend" instead of a number, with a way to open the sheet | N/A |
| Fixed costs / savings goal unset | Empty `fixedCosts` / `settings.savingsGoal === 0` and never confirmed | Hero still shows a number, plus a small `ink-500` hint it may be inaccurate | N/A |
| Cold load | Data not yet fetched | Skeleton placeholder preserving hero + stat-card layout (no layout jump) | N/A |
| Sheet: invalid income | User enters `0` or blank, clicks Save | Inline error near the Lương field, sheet stays open, other fields untouched | Mirrors `TransactionModal`'s error-string pattern |
| Sheet: save succeeds | Valid income/savings-goal/cycle-day, Save clicked | Sheet closes, toast success, Dashboard's `cycleSummary` reflects new values without a manual reload | `saveCycleSettings` → refetch `cycleSummary` |
| Fixed-cost list: add/edit/delete | User adds/edits/removes a row in the sheet | Each action calls its own endpoint immediately, list updates in place | Existing endpoint's validation/404 shown as a toast |
| New expense saved elsewhere | User adds an expense on the Transactions page, returns to Dashboard | Hero's number is lower, reflecting the new spend | `addTransaction` → refetch `cycleSummary` |

</frozen-after-approval>

## Code Map

- `frontend/src/types.ts` -- add `Income {id, cycleStartDate, amount}`, `FixedCost {id, name, amount}`, `Settings {savingsGoal, cycleStartDay}`, `CycleSummary {safeToSpend, daysRemaining, income: number|null, previousIncome: number|null, budgets: BudgetStatus[], activeAlerts: ActiveAlert[]}` (minimal placeholder shapes for the always-empty arrays, matching architecture's JSON contract), `CycleSettingsInput {income, savingsGoal, cycleStartDay}` (mirrors backend `cycleSettingsInput`).
- `frontend/src/index.css` -- add `.display-number` in the components layer, alongside `.tnum`/`.chip`/etc.
- `frontend/src/context/DataContext.tsx` -- add `cycleSummary: CycleSummary | null`, `fixedCosts: FixedCost[]` state; fetch `GET /cycle/summary` + `GET /fixed-costs` in the existing mount `Promise.all` (alongside categories/transactions/budgets); add a `refreshCycleSummary()` helper and call it from the end of `addTransaction`/`updateTransaction`/`deleteTransaction` plus the three new mutations below; add `saveCycleSettings(input: CycleSettingsInput): Promise<void>` (`PUT /cycle-settings`), `addFixedCost`/`updateFixedCost`/`deleteFixedCost` (own REST endpoints, update local `fixedCosts` state) mirroring `upsertBudget`/`deleteBudget`'s exact shape.
- `frontend/src/components/SafeToSpendHero.tsx` (new) -- renders the three states from the I/O matrix (number / missing-income / cold-load skeleton), `.card` variant per DESIGN.md (`bg-brand-50`, `border-brand-500/20`), pencil `.btn-icon` opening `CycleUpdateSheet`.
- `frontend/src/components/CycleUpdateSheet.tsx` (new) -- mirrors `TransactionModal.tsx`: local state for income/fixedCosts-draft-list/savingsGoal/cycleStartDay, collapsed "Nâng cao" section for cycle day, inline validation, Save → `saveCycleSettings` + any pending fixed-cost row actions, uses `Modal` (`size="lg"` or similar).
- `frontend/src/pages/Dashboard.tsx` -- insert `<SafeToSpendHero />` as the first child of the root `space-y-6` div, before the existing header row.
- `frontend/src/lib/i18n.ts` -- add new keys for hero copy, sheet field labels, and validation messages (Vietnamese, matching the existing `dash.*`/`txm.*` naming convention).
- `frontend/src/components/Modal.tsx`, `TransactionModal.tsx` -- reference patterns only, no changes.

## Tasks & Acceptance

**Execution:**
- [x] `frontend/src/types.ts` -- new types
- [x] `frontend/src/index.css` -- `.display-number`
- [x] `frontend/src/lib/i18n.ts` -- new copy keys
- [x] `frontend/src/context/DataContext.tsx` -- new state, fetches, `refreshCycleSummary`, new mutations, cross-refetch wiring on existing transaction mutations
- [x] `frontend/src/components/SafeToSpendHero.tsx` -- new component, all three states
- [x] `frontend/src/components/CycleUpdateSheet.tsx` -- new component
- [x] `frontend/src/pages/Dashboard.tsx` -- hero insertion + wiring
- [x] frontend build/typecheck -- zero errors, zero regressions to existing pages

**Acceptance Criteria:**
- Given income/fixed-costs/savings-goal all declared, when the Dashboard loads, then `safe-to-spend-hero` shows `cycleSummary.safeToSpend` in `.display-number` style plus days remaining.
- Given income not declared, when the Dashboard loads, then the hero shows the "update income" prompt instead of a number.
- Given the sheet's Save succeeds, when it closes, then the Dashboard's hero reflects the new values without a manual page reload.
- Given a new expense is saved on the Transactions page, when the user returns to the Dashboard, then the hero's number is updated to reflect it.

## Implementation Notes

- `DataContext` has no backend endpoint to fetch a user's current `Settings` (savingsGoal/cycleStartDay) ahead of time — only `PUT /cycle-settings` returns it, and `GET /cycle/summary` doesn't carry it. Added a client-only `settings: Settings` state (default `{savingsGoal: 0, cycleStartDay: 1}`, matching the backend's own column defaults) that's populated from each `saveCycleSettings` response; `CycleUpdateSheet` prefills the savings-goal/cycle-day fields from it. Until the user saves once this session, those two fields open blank-equivalent (0 / day 1) rather than reflecting a prior save from an earlier session — a real gap, but there is no contract to close it without violating "do not change any existing backend contract."
- The "fixed costs / savings goal unset -> inaccuracy hint" I/O-matrix row is implemented as an OR (`fixedCosts.length === 0 || settings.savingsGoal === 0`), not an AND, since either alone is a legitimate reason the number could be off.
- Fixed-cost list rows are locally editable (draft state per row) and only call `updateFixedCost` when a row's draft actually differs from its saved value (a small Check button appears); add/delete call their endpoints immediately per the spec.

## Spec Change Log

## Review Triage Log

- **high / patch** — `CycleUpdateSheet`'s main Save button only calls `saveCycleSettings` (income/savingsGoal/cycleStartDay); it never flushes fixed-cost rows the user has edited in-place but not yet confirmed via their own small per-row Check icon. A user who edits an amount and clicks the sheet's obvious, prominent Save button (not the small per-row check) silently loses that edit. Verified by reading `handleSave`/`handleUpdateFixedCost` — no dirty-draft flush occurs. [edge-case-hunter]
- **medium / patch** — `handleAddFixedCost`'s only guard is `if (!name && !amount) return` — a partial row (name filled, amount blank, or vice versa) is silently POSTed to the backend, which 400s with only a generic toast. The `err.fixedCostInvalid` i18n key already exists but is never used. Same gap applies to `handleUpdateFixedCost` (no pre-flight validation on the Check icon). [blind-hunter, edge-case-hunter, verification-gap]
- **medium / patch** — No distinct "failed to load" state: if the initial `GET /cycle/summary` fetch itself fails, `cycleSummary` stays `null` and the hero renders the same "update your income" prompt as a user who simply hasn't declared income yet — a backend/network outage is silently mislabeled as a data-entry prompt. [blind-hunter, edge-case-hunter]
- **medium / patch** — A negative `safeToSpend` (overspent) renders with the same neutral `text-ink-900` styling as a healthy balance — no color/icon distinction, unlike the app's own established rose/warning convention for over-budget amounts elsewhere (`Budgets.tsx`). The dashboard's single most prominent number gives no visual signal exactly when the user most needs one. [blind-hunter]
- **low / patch** — `DataContext`'s failed-refetch `.catch` block resets `categories`/`transactions`/`budgets`/`cycleSummary`/`fixedCosts` but not `settings` back to `DEFAULT_SETTINGS`, unlike the "no user" branch a few lines above which does reset it — stale settings persist after a failed refresh. [blind-hunter]
- **low / patch** — `refreshCycleSummary()` is `await`ed inside all 6 mutations (transactions ×3, `saveCycleSettings`, fixed-cost ×3 — wait, listed as best-effort/errors-swallowed already) adding a second sequential round trip before each action's success toast/close, even though its own comment calls it best-effort. Removing the `await` (fire-and-forget) keeps the primary action responsive. [blind-hunter]
- **low / patch** — No in-flight guard on the fixed-cost row Add/Update/Delete buttons, unlike the main Save button's `saving`-flag disable — a fast double-click can fire duplicate requests (e.g. two identical fixed costs from one Add click). [blind-hunter, edge-case-hunter]
- **low / reject** — Extending the mount-time `Promise.all` to 5 endpoints widens the blast radius of its existing all-or-nothing `.catch` (a failure in either new endpoint now also clears already-successful categories/transactions/budgets). Verified: this is the identical pre-existing pattern already used for the first 3 domains, not new logic introduced by this diff; a proper fix (partial-failure handling) is a broader refactor out of this story's scope. [blind-hunter, verification-gap]
- **low / reject** — `saveCycleSettings` receives `CycleSettingsResult.income`/settings but discards them, doing a redundant `GET /cycle/summary` refetch instead of using the response directly. Verified: negligible overhead for a personal single-user app, matching the same "not optimized for scale" reasoning already applied to other rejected findings in this epic. [blind-hunter]
- **defer** — No way to distinguish "savings goal never set" (shows `0`) from "intentionally set to 0" — same root cause as the settings-defaults gap below; needs a backend read endpoint. [blind-hunter]
- **defer** — `CycleUpdateSheet` pre-fills savings-goal/cycle-start-day from a client-only `DEFAULT_SETTINGS = {0, 1}` state that's never fetched from the server (no `GET /settings` exists). A returning user with real, previously-saved non-default values who opens the sheet and clicks Save without touching those two fields will have them silently overwritten back to `{0, 1}` — a genuine value-destroying bug for real usage, self-identified by the implementer in this spec's Implementation Notes. Fixing it needs a backend read endpoint, which this spec's Boundaries explicitly forbid changing — flagging as a priority follow-up before this reaches real multi-session usage, not a "someday" item. [verification-gap]
- **defer** — A background `refreshCycleSummary()` response can land after logout (or a fast account switch) with no cancellation guard, momentarily writing stale or another session's data into state. Real but low-probability (requires logout exactly mid-flight); proper fix needs a request-token/mounted-ref pattern, more than a direct correction. [edge-case-hunter]

## Verification

**Commands:**
- `npm run lint` (in `frontend/`; runs `tsc -b --noEmit`) -- ran after `npm install` (node_modules was absent): succeeds, zero type errors across the whole app.
- `npm run build` (in `frontend/`; runs `tsc -b && vite build`) -- succeeds, `dist/` produced with no warnings beyond the pre-existing recharts deprecation notice.
- Manual smoke check (no test framework exists yet in `frontend/`): NOT performed — Docker Desktop's engine isn't running in this environment, so the MySQL-backed API couldn't be started to drive an authenticated browser session. Static verification instead: cross-checked every new `t()` key against both `vi`/`en` dicts (all present, dicts stay key-for-key symmetric at 166 entries each), and traced the backend contracts (`internal/api/cycle_summary.go`, `cycle_settings.go`, `fixed_costs.go`, `api.go`) field-by-field against the new frontend types/payloads. Recommend the user (or a follow-up session with Docker running) do the actual click-through before merging.

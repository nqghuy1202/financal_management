---
title: 'Story 2.2: Xem và đóng cảnh báo ngân sách trên Dashboard'
type: 'feature'
created: '2026-09-13'
status: 'done'
review_loop_iteration: 2
followup_review_recommended: false
context: []
warnings: ['oversized']
deferred: []
baseline_revision: 'be244abcd89c096fd334b596011b7e0bc98a3b99'
---

<intent-contract>

## Intent

**Problem:** Story 2.1 writes `budget_alert_state` rows when a category crosses 70/90/100%, but nothing surfaces them — the user has no way to see or acknowledge an active alert.

**Approach:** Populate `GET /cycle/summary.activeAlerts` with real data (currently a hardcoded `[]`), add `POST /alerts/:categoryId/:threshold/dismiss`, and render up to 3 `alert-banner` components directly under `safe-to-spend-hero` on the Dashboard, "over" alerts before "near", overflow collapsed into a "+N" link to the Budgets page. Dismissing sets `dismissed_at` and removes the banner for the rest of the current cycle.

## Boundaries & Constraints

**Always:** `activeAlerts` entries are `{categoryId, threshold, status}` (architecture's fixed contract — `status` is `"near"` for threshold 70/90, `"over"` for 100; both computed by a pure `AlertStatus(threshold int) string` in `cycle.go`, reusable by Story 2.3). Listing always filters `cycle_start_date = <current cycle's start>` **and** `dismissed_at IS NULL` together — never one without the other, so a past-cycle alert never surfaces even if never dismissed. Dismiss also scopes to the current cycle's `cycle_start_date` server-side (recomputed from settings, not client-supplied) and is idempotent: dismissing an already-dismissed or non-existent alert still returns 200, never a 404/500. The frontend resolves `categoryId` → display name locally via the already-loaded `categories` list (`categoryById`, `DataContext.tsx`) — the backend does not embed a category name. `daysRemaining` in banner text reuses the existing top-level `cycleSummary.daysRemaining` (shared across all banners, not per-alert). Each `alert-banner` has `role="alert"`; its dismiss control has an `aria-label` naming the category (NFR5). No banner at all when `activeAlerts` is empty — deliberate silence, no empty-state message.

**Never:** Do not compute or return an exact live percent-used number — that data (per-category `spent`/`percent`) doesn't exist system-wide until Story 2.3 extends `GET /budgets`; banner copy uses threshold-derived phrasing ("đã vượt ngân sách" / "đã dùng gần hết ngân sách"), not a specific "92%"-style figure. Do not add `spent`/`percent`/`status` to `GET /budgets` or touch the Budgets page's own status pills — that's Story 2.3. Do not delete or "un-trigger" `budget_alert_state` rows on dismiss — only set `dismissed_at`. Do not auto-dismiss a banner (no toast-style timeout) — only the user's own dismiss click removes it.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Active alerts exist | 2 undismissed alerts this cycle (one over, one near) | Both render, "over" banner listed before "near" | N/A |
| More than 3 active alerts | 5 undismissed alerts this cycle | Top 3 shown (over-priority), "+2 cảnh báo khác" link to Budgets | N/A |
| No active alerts | Zero undismissed rows this cycle | No banner rendered at all | N/A |
| Dismiss | User clicks dismiss on a rendered banner | `dismissed_at` set via the dismiss endpoint; banner removed from the list immediately, does not reappear this cycle | N/A |
| Past-cycle alert never dismissed | A prior cycle has an undismissed row | Never listed in the current cycle's `activeAlerts` | N/A |
| Dismiss an already-dismissed/unknown alert | Repeat dismiss click, or stale (categoryId, threshold) | 200 OK, no error, no new state change | Idempotent no-op, never 404/500 |
| Category deleted while it has an active alert | `deleteCategory` succeeds, then `cycleSummary` is refreshed | The alert for that category no longer appears (refresh drops it); if a stale banner is somehow still on screen and dismissed, the frontend removes it locally on the resulting 400 rather than leaving it stuck | 400 on dismiss is treated as "already gone", not a user-facing error |

</intent-contract>

## Code Map

- `internal/api/cycle.go` -- add pure `func AlertStatus(threshold int) string`: `if threshold >= 100 { return "over" }; return "near"`. Same file/spirit as `CrossedThreshold`; Story 2.3 reuses it for `GET /budgets`'s per-category status.
- `internal/api/repo_alert_state.go` -- add `ActiveAlert` struct (`CategoryID string \`json:"categoryId"\``, `Threshold int \`json:"threshold"\``, `Status string \`json:"status"\``) and `func (r *AlertStateRepo) ListActive(ctx, userID string, cycleStart time.Time) ([]ActiveAlert, error)`: `SELECT category_id, threshold FROM budget_alert_state WHERE user_id=? AND cycle_start_date=? AND dismissed_at IS NULL ORDER BY threshold DESC, triggered_at DESC` (threshold DESC naturally puts 100/"over" before 90/70/"near"); build `ActiveAlert{CategoryID, Threshold, Status: AlertStatus(threshold)}` per row; return `[]ActiveAlert{}` (never nil) when empty. Add `func (r *AlertStateRepo) Dismiss(ctx, userID, categoryID string, cycleStart time.Time, threshold int) error`: `UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP WHERE user_id=? AND category_id=? AND cycle_start_date=? AND threshold=? AND dismissed_at IS NULL` -- 0 rows affected is not an error (idempotent).
- `internal/api/api.go` -- change `CycleSummary.ActiveAlerts` field type from `[]any` to `[]ActiveAlert`.
- `internal/api/cycle_summary.go` -- in `GetCycleSummary`, replace the hardcoded `ActiveAlerts: []any{}` with `alerts, err := NewAlertStateRepo(h.db).ListActive(ctx, userID, cycleStart)` (reuse the `cycleStart` already computed at line ~30; on error, `fail(c, 500, 50085, "Không thể tải cảnh báo")` -- next free code in this file's existing 50080s block); pass `alerts` as `ActiveAlerts`.
- `internal/api/alerts.go` (new) -- `func (h *Handler) DismissAlert(c *gin.Context)`: parse `categoryId := c.Param("categoryId")`, `threshold, err := strconv.Atoi(c.Param("threshold"))`; if `err != nil` or threshold not in {70,90,100}, `fail(c, 400, 40090, "Ngưỡng không hợp lệ")`. Check ownership via `h.categories.Owns(ctx, uid, categoryId)` (existing helper); not-owned/not-found → `fail(c, 400, 40091, "Danh mục không tồn tại")`. Compute current cycle via `h.settings.Get(ctx, uid)` + `CycleWindow(settings.CycleStartDay, time.Now())` (same pattern as `GetCycleSummary`). Call `NewAlertStateRepo(h.db).Dismiss(ctx, uid, categoryId, cycleStart, threshold)`; on error, `fail(c, 500, 50090, "Không thể đóng cảnh báo")`; else `ok(c, gin.H{"dismissed": true})`.
- `internal/api/router.go` -- add `auth.POST("/alerts/:categoryId/:threshold/dismiss", h.DismissAlert)` near the `/cycle/summary` route.
- `frontend/src/types.ts` -- replace the `ActiveAlert = Record<string, unknown>` placeholder with `interface ActiveAlert { categoryId: string; threshold: 70 | 90 | 100; status: 'near' | 'over' }`.
- `frontend/src/lib/i18n.ts` -- add both `vi`/`en` entries: `alert.near`, `alert.over` (`{category}`/`{days}` vars), `alert.dismissAria` (`{category}` var), `alert.moreCount` (`{count}` var), `alert.unknownCategory` (a generic label, e.g. "Danh mục đã xóa" / "Deleted category"). Do not add an unused `alert.seeBudgets` key -- the overflow link's full label is `alert.moreCount` alone (matches the literal AC text "+N cảnh báo khác"); nothing else references a separate "see budgets" string.
- `frontend/src/context/DataContext.tsx` -- add `dismissAlert(categoryId: string, threshold: number)`: call `apiSend('POST', \`/alerts/${categoryId}/${threshold}/dismiss\`)`; on success, filter the `(categoryId, threshold)` pair out of `cycleSummary.activeAlerts` locally (no full refetch needed). On failure: if the thrown `ApiError`'s `status === 400` (category no longer owned/found -- the stale-alert case), still remove it locally (self-healing) and skip the toast; for any other failure, `toast.error(...)` (pattern from `CycleUpdateSheet.tsx`) and leave the banner in place. Also: `deleteCategory` (existing function, same file) must call `refreshCycleSummary()` after its delete succeeds, immediately dropping any `activeAlerts` entry for the deleted category (its `budget_alert_state` rows are already `ON DELETE CASCADE`'d server-side) instead of leaving a stale entry until some unrelated later refresh.
- `frontend/src/components/AlertBanner.tsx` (new) -- one alert: thick left border + `bg-amber-50 text-amber-600` for `status==='near'`, `bg-rose-50 text-rose-600` for `status==='over'` (reuse `.chip`-adjacent color convention from `Budgets.tsx`); resolve `categoryName = categoryById(alert.categoryId)?.name ?? t('alert.unknownCategory')` (a real fallback string, never the raw id); text via `t('alert.near'|'alert.over', { category: categoryName, days: cycleSummary.daysRemaining })`; dismiss (X) button with `aria-label={t('alert.dismissAria', { category: categoryName })}` calling `dismissAlert`; root element `role="alert"`; the decorative icon gets `aria-hidden="true"` (it adds no information a screen reader needs beyond the text).
- `frontend/src/pages/Dashboard.tsx` -- directly under `<SafeToSpendHero />`: if `cycleSummary.activeAlerts.length`, sort/take first 3 (already over-first from backend ordering) and render `<AlertBanner>` for each, stacked full-width; if more than 3, render a trailing `+N` link (`t('alert.moreCount', { count })`) to `/budgets`.

## Tasks & Acceptance

**Execution:**
- `internal/api/cycle.go` + test -- `AlertStatus` pure function
- `internal/api/repo_alert_state.go` + test -- `ListActive` (empty, some, ordering, past-cycle excluded), `Dismiss` (found, idempotent-on-repeat, query error)
- `internal/api/api.go`, `cycle_summary.go` + test -- wire real `ActiveAlerts` into `GetCycleSummary`
- `internal/api/alerts.go` (new) + test -- `DismissAlert`: happy path, invalid threshold, not-owned category, repo error
- `internal/api/router.go` -- mount the dismiss route
- `frontend/src/types.ts`, `lib/i18n.ts` -- concrete `ActiveAlert` type, vi/en banner strings
- `frontend/src/context/DataContext.tsx` -- `dismissAlert`: wait for the POST to succeed before removing locally (never optimistic), self-heal (remove locally anyway) on a 400, leave the banner in place on any other failure
- `frontend/src/components/AlertBanner.tsx` (new) -- banner UI, `role="alert"`, `aria-label` dismiss
- `frontend/src/pages/Dashboard.tsx` -- render up to 3 banners under the hero + overflow link
- repo root -- `go build ./...`, `go vet ./...`, `go test ./internal/api/... -v`; `cd frontend && npm run lint` (tsc -b --noEmit) -- zero regressions

**Acceptance Criteria:**
- Given undismissed alerts exist for the current cycle, when the Dashboard loads, then up to 3 `alert-banner`s render directly under `safe-to-spend-hero`, "over" before "near", overflow collapsed into a single "+N" link to Budgets.
- Given the user clicks a banner's dismiss button, when the request succeeds, then the banner disappears immediately and does not reappear later in the same cycle (`dismissed_at` is set, filtered by `ListActive`).
- Given no undismissed alerts exist for the current cycle, when the Dashboard loads, then no banner renders at all -- no empty-state message.
- Given an alert exists only in a previous cycle and was never dismissed, when viewing the Dashboard in the current cycle, then it never appears in `activeAlerts`.

## Spec Change Log

### 2026-09-13 — bad_spec amendment (review pass 1)
- **Triggering findings:** blind-hunter, edge-case-hunter, and intent-alignment all independently traced the same defect: deleting a category that currently has an active alert never refreshes `cycleSummary`, so the stale `activeAlerts` entry lingers in the frontend; its banner falls back to the raw `categoryId` (a UUID) instead of a real label, and if the user clicks dismiss on it, the backend's `Owns` check correctly 400s (the category is truly gone) but the frontend's error handler leaves the stale banner in place — permanently stuck until an unrelated mutation happens to call `refreshCycleSummary`. Root cause: this spec's own Code Map told `AlertBanner.tsx` to fall back to `?? alert.categoryId`, directly contradicting the intent-contract's I/O matrix promise of "a generic fallback label instead of crashing" — and never told `deleteCategory` to refresh `cycleSummary` at all. (The matrix's framing of this scenario as "category deleted after its alert fired" was itself imprecise: `budget_alert_state` has `ON DELETE CASCADE` on `category_id`, so the DB row is gone instantly — the real, reachable mechanism is the frontend's own `cycleSummary` cache going stale, not a live DB row referencing a dead category.)
- **What was amended:** (1) I/O matrix's scenario corrected to describe the real mechanism (stale cached `activeAlerts` after a category delete, not a literal DB-level dangling reference). (2) Code Map for `DataContext.tsx`'s `deleteCategory` now calls `refreshCycleSummary()` after the delete succeeds, so a deleted category's alert entry is dropped immediately. (3) Code Map for `AlertBanner.tsx` now uses a real i18n fallback string (`alert.unknownCategory`) instead of the raw `categoryId` when `categoryById` returns `undefined`. (4) Code Map for `dismissAlert` now also removes the (categoryId, threshold) pair from local state when the dismiss request fails with a 400 specifically (category no longer owned/found) — self-healing even before the next refresh, as defense-in-depth alongside fix (2).
- **Known-bad state avoided:** a confusing raw-UUID banner that the user can never dismiss, persisting until an unrelated action happens to refresh the cycle summary.
- **KEEP (verified correct, must survive re-derivation):** the entire backend half (`AlertStatus`, `ListActive`, `Dismiss`, `DismissAlert`, router wiring) — all four reviewers confirmed this is correct and well-tested; only the frontend's category-deletion interaction and its fallback text were wrong. `role="alert"`/`aria-label` accessibility wiring, the over-before-near backend ordering, the idempotent-dismiss behavior, and the "+N" overflow link to `/budgets` are all correct as implemented and should not be re-derived differently.

## Design Notes

`AlertStatus` deliberately lives in `cycle.go` next to `CrossedThreshold` rather than in `repo_alert_state.go`, because Story 2.3 needs the identical threshold→status mapping for `GET /budgets`'s per-category status field and the architecture requires both surfaces to "always agree... same computation, not two independent ones" -- putting it in `cycle.go` now means Story 2.3 imports/calls it rather than re-deriving it. The exact live percent shown in the epic's illustrative banner copy ("đã dùng 92% ngân sách") is not implemented literally in this story: the architecture's own contract for `activeAlerts` is `{categoryId, threshold, status}` with no percent field, and per-category `spent`/`percent` doesn't exist anywhere in the API until Story 2.3 extends `GET /budgets` -- inventing a parallel percent computation here would duplicate (and risk diverging from) that future single source of truth. Banner copy instead names the threshold band ("near"/"over") in prose, which is fully supported by data this story already has.

## Verification

**Commands:**
- `go build ./...` -- expected: succeeds, no errors
- `go vet ./...` -- expected: no warnings
- `go test ./internal/api/... -v` -- expected: all tests pass, including new `TestAlertStatus_*`, `TestAlertStateRepo_ListActive_*`, `TestAlertStateRepo_Dismiss_*`, `TestDismissAlert_*`, and `TestGetCycleSummary_*` updated for real `ActiveAlerts`
- `gofmt -l internal/api` -- expected: no output
- `cd frontend && npm run lint` (`tsc -b --noEmit`) -- expected: no type errors

### 2026-09-13 — Review pass 2 (post bad_spec re-derivation)
- verdicts: 21 findings — high 0, medium 0, low 17, false 4, maybe-false 0
- findings:
  - `[low]` `[patch]` blind-hunter: this spec file had a duplicate, empty, orphaned `## Design Notes` heading at the very end (a leftover from a prior edit) -- verified true. **Fixed:** removed directly (doc-only, no re-derivation needed).
  - `[false]` `[reject]` blind-hunter: `sprint-status.yaml` still lists this story as `backlog` -- not a defect: the tracker is updated at Finalize/commit time, per this project's established pattern (see Story 2.1), not mid-review.
  - `[low]` `[patch]` blind-hunter: English `'alert.moreCount': '+{count} more alerts'` is grammatically wrong at `count===1` ("+1 more alerts") -- verified true. **Fixed:** added `alert.moreCountOne`, `Dashboard.tsx` picks it when `overflowCount===1`.
  - `[low]` `[reject]` blind-hunter: the dismiss button's shared `.btn-icon` class is neutral-gray, visually clashing with the colored banner it sits in -- real but purely cosmetic; a proper per-status color variant is more than a direct correction (no matching amber `.btn-icon` variant exists in this codebase yet).
  - `[low]` `[patch]` blind-hunter (carried from pass 1): `dismissAlert(categoryId, threshold: number)` widens the precise `70 | 90 | 100` union back to `number` -- still true, no longer moot. **Fixed:** narrowed to `ActiveAlert['threshold']` in both the interface and implementation.
  - `[low]` `[patch]` blind-hunter: `TestAlertStateRepo_ListActive_PastCycleExcluded` never actually seeds a past-cycle row -- it's functionally identical to `_Empty` and overclaims coverage in its own assert message -- verified true. **Fixed:** rewritten to mock the expectation keyed to the current cycle's date, call `ListActive` with a past `cycleStart`, and assert the resulting argument-mismatch error -- proving `cycleStart` is a real, live parameter, not hardcoded -- with a comment noting sqlmock's inherent limitation for asserting true cross-call DB isolation.
  - `[low]` `[reject]` blind-hunter (carried from pass 1): `DismissAlert` reuses error code `50090` for three distinct failure branches -- same precedent as Story 2.1 and this story's pass 1; still rejected.
  - `[low]` `[reject]` blind-hunter (carried from pass 1): no pending/disabled state on the dismiss button -- still rejected, same reasoning (idempotency absorbs the harm).
  - `[low]` `[reject]` blind-hunter: `AlertBanner` calls `useData()` independently per instance instead of receiving `daysRemaining` as a prop -- real but negligible (≤3 banners, already-memoized context), a stylistic preference rather than a bad outcome.
  - `[false]` `[reject]` edge-case-hunter: a threshold dismissed then legitimately re-crossed later in the same cycle never reappears (`Trigger`'s upsert never clears `dismissed_at`) -- refuted: `epic-2-context.md`'s own Technical Decisions section explicitly states "a dismissed alert never reappears in the same cycle" with no re-crossing exception -- this is deliberate, already-settled architecture, not a defect.
  - `[low]` `[reject]` edge-case-hunter (carried from pass 1): no final ordering tiebreaker for same-threshold/same-timestamp alerts -- still rejected, same reasoning (same-tier only, over-before-near unaffected).
  - `[low]` `[defer]` edge-case-hunter (carried from pass 1): same-category simultaneous 70+100 alerts can occupy 2 of 3 slots -- still deferred, same reasoning (product-level UX decision, not a defect against the literal AC).
  - `[low]` `[reject]` edge-case-hunter (carried from pass 1): `refreshCycleSummary()` race could resurrect a just-dismissed banner -- still rejected, same reasoning (narrow window, single-user app, fix is nontrivial).
  - `[low]` `[patch]` edge-case-hunter: this spec's own Tasks & Acceptance line for `dismissAlert` said "optimistic update + error rollback-safety," but the actual (correct) design is wait-then-remove with 400-triggered self-heal -- verified true, a genuine wording mismatch in the spec itself (not the code, which is correct). **Fixed:** reworded the Tasks line directly (doc-only).
  - `[low]` `[reject]` edge-case-hunter: same race as above framed against AC2's "does not reappear" wording -- same root cause and disposition as the `refreshCycleSummary` race finding above.
  - `[low]` `[defer]` verification-gap (pre-verified, filed `defer`): the stale-alert self-heal mechanism (`deleteCategory` + `refreshCycleSummary`, `dismissAlert`'s 400 handling) has no test proving it -- closing this means introducing a frontend test framework from scratch; grouped with this story's other no-frontend-test-framework findings below.
  - `[low]` `[defer]` intent-alignment: every AC is phrased as on-screen Dashboard behavior, but every test in the diff stops at the API/DB layer -- same pre-existing, whole-codebase gap (no frontend test framework anywhere in this repo); grouped with the verification-gap finding above.
  - `[low]` `[defer]` intent-alignment: the `role="alert"`/`aria-label` accessibility clause is asserted in JSX but never confirmed to reach the DOM -- grouped with the same no-frontend-test-framework gap.
  - `[low]` `[defer]` intent-alignment: "no banner when empty" is tested backend-side (empty array) but never on the frontend consequence (nothing rendered) -- grouped with the same gap.
  - `[false]` `[reject]` intent-alignment: the category-deletion self-healing behavior is a "scope addition" not present in the literal AC text -- refuted: this is exactly the bad_spec fix from pass 1, reasoned and necessary, not an unjustified addition.
  - `[false]` `[reject]` intent-alignment: linking the overflow text to `/budgets` sources from `epic-2-context.md` rather than the literal AC -- already established in pass 1 as a direct requirement (the epic's own UX section says exactly this), not an invented addition.

## Auto Run Result

**Summary:** Populated `GET /cycle/summary.activeAlerts` with real data (`AlertStateRepo.ListActive`, cycle-scoped + over-before-near ordered), added the idempotent `POST /alerts/:categoryId/:threshold/dismiss` endpoint, and rendered up to 3 `alert-banner` components under `safe-to-spend-hero` with an overflow link to Budgets. Went through one `bad_spec` loopback (this spec's own Code Map contradicted its intent-contract's "generic fallback label" promise, and never told `deleteCategory` to refresh the cycle summary — leaving a stale, undismissable, raw-UUID banner reachable by deleting a category with an active alert) and a second review pass that applied 3 low-severity code patches plus 2 doc-only spec corrections; all fully re-verified independently.

**Files changed:**
- `internal/api/cycle.go` (+test) -- pure `AlertStatus(threshold) string`.
- `internal/api/repo_alert_state.go` (+test) -- `ActiveAlert` struct, `ListActive`, `Dismiss`.
- `internal/api/api.go` -- `CycleSummary.ActiveAlerts` now `[]ActiveAlert`.
- `internal/api/cycle_summary.go` (+test) -- wired real `ListActive` call, error code `50085`.
- `internal/api/alerts.go` (new, +test) -- `DismissAlert` handler.
- `internal/api/router.go` -- mounted the dismiss route.
- `frontend/src/types.ts` -- concrete `ActiveAlert` interface.
- `frontend/src/lib/i18n.ts` -- `alert.*` vi/en strings, including singular/plural overflow text and a real `alert.unknownCategory` fallback.
- `frontend/src/context/DataContext.tsx` -- `dismissAlert` (wait-then-remove, 400-self-heals); `deleteCategory` now also calls `refreshCycleSummary()`.
- `frontend/src/components/AlertBanner.tsx` (new) -- banner UI, `role="alert"`, `aria-hidden` icon, `aria-label` dismiss, real fallback label.
- `frontend/src/pages/Dashboard.tsx` -- renders up to 3 banners + overflow link.

**Review findings breakdown:**
- Pass 1: 24 findings -- 4 `medium`/`bad_spec` (one shared root cause: stale `activeAlerts` after category deletion, raw-UUID fallback, permanently-stuck dismiss -- fully re-derived), 17 `low`, 3 `false`. All non-bad_spec findings were mooted/carried that pass.
- Pass 2 (post re-derivation): 21 findings -- 0 high/medium, 3 `low`/patch applied to code (English overflow-text pluralization; `dismissAlert`'s threshold type narrowed back to the literal union; a repo test rewritten to actually prove cycle-scoping instead of duplicating the empty-case test), 2 `low`/patch applied directly to this spec doc (an orphaned duplicate heading removed; a Tasks-list line's "optimistic update" wording corrected to match the actual, correct wait-then-remove design), 5 `low`/reject (established precedents: duplicate error codes, no pending-button state, cosmetic icon-button color mismatch, no ordering tiebreaker, useData-per-instance), 1 `false`/reject (a dismissed-then-re-crossed alert never reappearing -- confirmed deliberate per `epic-2-context.md`), 4 `low`/defer (all the same pre-existing whole-codebase gap: no frontend test framework exists anywhere in this repo, so the UI-facing ACs and the self-heal mechanism are verified by direct code inspection, not automated tests), 1 `low`/defer (same-category simultaneous alerts, carried), 3 `false`/reject (all already-settled from pass 1: intentional ordering design, the category-deletion self-heal as a justified bad_spec fix, and the overflow-link-to-Budgets requirement sourced from the epic's own UX section).

**Follow-up review recommendation:** `false`. Pass 2 patched only `low`-severity entries -- below the "any high, or two-or-more medium" bar for a first-pass recommendation.

**Verification performed:**
- `go build ./...`, `go vet ./...`, `gofmt -l internal/api` -- clean, both passes.
- `go test ./internal/api/... -v` -- all tests pass both passes, including every test family the spec names by name (`TestAlertStatus_*`, `TestAlertStateRepo_ListActive_*`, `TestAlertStateRepo_Dismiss_*`, `TestDismissAlert_*`, updated `TestGetCycleSummary_*`).
- `cd frontend && npm run lint` (`tsc -b --noEmit`) -- clean, both passes.
- `go build ./...` and `go test ./...` at repo root -- only 2 pre-existing, unrelated failures (`internal/database` needs Docker; `internal/pkg/tests/basic` is a pre-existing intentionally-broken scaffold test), confirmed unrelated by direct inspection, not just by name.
- Manual diff review of the full unified diff against baseline `be244abcd89c096fd334b596011b7e0bc98a3b99`, twice (before and after the bad_spec loopback), including direct source `grep`/`Read` confirmation that every claimed fix (fallback string, `refreshCycleSummary` call, `aria-hidden`, pluralization, type narrowing, rewritten test) actually landed in the files, not just in subagent prose.

**Residual risks (all explicitly triaged `reject`/`defer` above, not blocking):**
- No frontend test framework exists anywhere in this repository, so every UI-facing acceptance criterion (banner ordering/cap/overflow, no-banner-when-empty, the category-deletion self-heal) is verified by direct source-code inspection rather than an automated test that would catch a future regression.
- A single category can have two simultaneous active alerts (e.g. an undismissed 70 plus a later 100), which can occupy 2 of the 3 visible banner slots -- not contradicted by the literal AC, but a candidate for a future per-category grouping decision.
- The dismiss icon button uses the app's neutral `.btn-icon` styling rather than a status-colored variant -- cosmetic only.

## Review Triage Log

### 2026-09-13 — Review pass 1
- verdicts: 24 findings — high 0, medium 4, low 17, false 3, maybe-false 0
- findings:
  - `[low]` `[patch]` blind-hunter: spec's own Tasks/Verification sections require `TestAlertStatus_*` tests for the new pure `AlertStatus` function, but none were added -- moot this pass (code being reverted/re-derived); may be re-raised next pass.
  - `[low]` `[patch]` blind-hunter: `alert.seeBudgets` i18n key added to both dictionaries but never referenced by any component -- dead string. **Amended in this pass:** Code Map now explicitly says not to add it.
  - `[medium]` `[bad_spec]` blind-hunter: `AlertBanner.tsx`'s fallback `categoryById(...)?.name ?? alert.categoryId` shows the user a raw internal UUID instead of "a generic fallback label" as the spec's own I/O matrix promised -- verified real; root cause is this spec's own Code Map, which explicitly told the implementer to fall back to the raw id, directly contradicting the intent-contract's promise.
  - `[low]` `[defer]` blind-hunter: a single category can have two simultaneous active alerts (e.g. 70 and 100, if the 70 was never dismissed before a later save crossed 100) -- `budget_alert_state`'s unique key is per-threshold, not per-category, so both rows coexist and could occupy 2 of the 3 visible banner slots. Real, but not contradicted by the literal AC (each is a genuinely distinct undismissed alert); any per-category merge/grouping is a product-level UX decision beyond a direct-correction patch.
  - `[low]` `[reject]` blind-hunter: `DismissAlert` reuses error code `50090` for three distinct failure branches (ownership-check error, settings-lookup error, dismiss-exec error) -- same class of finding already rejected in Story 2.1's review (doesn't affect behavior, just log-level diagnosability); same reasoning applies here.
  - `[low]` `[reject]` blind-hunter: dismiss button has no pending/disabled state while the request is in flight, inviting duplicate clicks -- rejected: server-side idempotency already absorbs any duplicate POST harmlessly; proper loading-state UI is more than a direct correction.
  - `[low]` `[patch]` blind-hunter: decorative `AlertTriangle` icon inside the `role="alert"` banner has no `aria-hidden="true"` -- true, cheap accessibility fix. **Amended in this pass:** Code Map now specifies `aria-hidden="true"` on the icon.
  - `[low]` `[patch]` blind-hunter: `types.ts` gives `ActiveAlert.threshold` the precise union `70 | 90 | 100`, but `dismissAlert(categoryId, threshold: number)` widens it back to plain `number`, discarding that type safety -- true, moot this pass.
  - `[low]` `[defer]` blind-hunter: no frontend test exercises any UI-facing AC (ordering, 3-banner cap, overflow text, no-banner-when-empty) -- pre-existing, whole-codebase gap (no frontend test framework exists anywhere in this repo), not introduced by this diff.
  - `[low]` `[reject]` blind-hunter: `TestDismissAlert_HappyPath`/`_RepoError` each call `time.Now()` independently to compute the expected mock argument rather than sharing one frozen "now" -- same class of latent test-time-dependency already identified and accepted in Story 1.4/1.5's reviews (negligible real-world probability, needs a clock-injection seam to properly fix).
  - `[low]` `[reject]` edge-case-hunter: alert ordering has no final tiebreaker (e.g. `id`) for two alerts with the same threshold and identical `triggered_at` -- rejected: only affects ordering *among* same-tier (both "near" or both "over") alerts, never violates the over-before-near AC; a same-second collision between two different categories is a narrow edge case.
  - `[low]` `[reject]` edge-case-hunter: a slower in-flight `refreshCycleSummary()` (from an unrelated mutation) could resolve after `dismissAlert` already filtered the list, resurrecting a just-dismissed banner -- rejected: narrow race window, unlikely for a single-user app's realistic usage pattern; a proper fix (request sequencing/cancellation) is more than a direct correction.
  - `[medium]` `[bad_spec]` edge-case-hunter: dismissing an alert whose category was deleted gets a correct 400 from the backend, but the frontend's error handler doesn't remove the stale banner, leaving it permanently stuck until an unrelated refresh -- verified real (confirmed `deleteCategory` never calls `refreshCycleSummary`); same root cause as the fallback-label finding above.
  - `[low]` `[patch]` edge-case-hunter (same claim as blind-hunter's finding above): missing `TestAlertStatus_*` tests -- grouped, moot this pass.
  - `[medium]` `[bad_spec]` edge-case-hunter (same claim as blind-hunter's fallback finding above): raw-id fallback contradicts the spec's own "generic fallback label" promise -- grouped, same root cause.
  - `[low]` `[patch]` verification-gap (pre-verified, filed `patch`): `DismissAlert`'s ownership-check DB-error branch (as opposed to not-found) is untested, unlike the established `TestUpsertBudget_OwnershipCheckDBError` precedent for the same `Owns` pattern elsewhere -- code is already correct, just untested; moot this pass.
  - `[low]` `[patch]` verification-gap (pre-verified, filed `patch`): `DismissAlert`'s `settings.Get` error branch is untested, unlike `GetCycleSummary`'s two equivalent tests for the same lookup -- code is already correct, just untested; moot this pass.
  - `[low]` `[defer]` intent-alignment: the ACs are phrased entirely as on-screen Dashboard behavior, but every test in the diff exercises the API/DB layer only, never the rendered component tree -- same pre-existing whole-codebase gap as blind-hunter's frontend-test finding above; grouped with it.
  - `[false]` `[reject]` intent-alignment: "over before near" ordering is achieved solely by backend `ORDER BY`, with the frontend trusting array order as-is -- not a defect, this is a correct, intentional design (single source of truth for ordering); purely descriptive.
  - `[low]` `[defer]` intent-alignment: `role="alert"`/`aria-label` exist in the JSX but nothing confirms they reach the DOM correctly -- same pre-existing whole-codebase no-frontend-test-framework gap; grouped with the two frontend-test findings above.
  - `[medium]` `[bad_spec]` intent-alignment (same claim as above): shipped fallback is the raw `categoryId`, not the "generic fallback label" the spec's own I/O matrix promised -- grouped, same root cause.
  - `[low]` `[patch]` intent-alignment (same claim as blind-hunter's dead-string finding above): `alert.seeBudgets` is added but never referenced -- grouped, moot this pass.
  - `[false]` `[reject]` intent-alignment: linking the "+N" overflow text to `/budgets` is called out as an "interpretive addition" beyond the literal AC text -- refuted: `epic-2-context.md`'s own UX section explicitly says "any overflow collapsed into '+N alerts, see Budgets'", so this is a direct requirement, not an invented one.
  - `[false]` `[reject]` intent-alignment: `dismissAlert`'s code comment calls the update "optimistic" when it actually waits for the POST to succeed first -- refuted as a defect: waiting for success before removing is the objectively correct behavior here (matches the spec's own explicit "on failure do not remove the banner" requirement) and the only issue is imprecise comment wording, not a functional problem.

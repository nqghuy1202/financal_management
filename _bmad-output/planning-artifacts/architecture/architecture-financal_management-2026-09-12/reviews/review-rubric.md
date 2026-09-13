---
reviewed: ARCHITECTURE-SPINE.md
review-date: '2026-09-12'
reviewer: rubric (good-spine checklist)
verdict: CONDITIONAL-PASS
---

# Review — Architecture Spine (HL Personal Finance — Cải tổ)

## Verdict

**Conditional pass** — the spine is well-formed, internally consistent, and ratifies the real brownfield codebase, but has one **High** correctness gap (budget-limit lookup under a customized cycle start day, which FR-4 explicitly requires to work) and two **Medium** gaps (rolling-average cycle enumeration for anomaly baseline; alert-state reversal semantics on transaction edit/delete) that are real divergence points the spine's own AD-3/AD-4/AD-6 were meant to close but don't fully close. Fixing AD-4's scope and adding one more small rule/function would close the gap; nothing here requires re-architecting.

## Finding counts

- High: 1
- Medium: 2
- Low: 1
- Informational: 2

## Findings

### H1 — AD-4 does not resolve which `budgets.month` row governs a non-calendar-aligned cycle (FR-4 × FR-5/6/7)

**Checklist criterion:** "fixes the real divergence points for the level below and misses none."

AD-4's own text concedes the gap: *"Khi `cycle_start_day = 1` (mặc định), chu kỳ trùng khớp tháng dương lịch nên `month` vẫn đúng."* It then states aggregation must use the cycle date-range (AD-3), not the `month` string — which fixes the *transaction-summing* side. But it never says which `budgets.month` row's `limit_amount` a non-default cycle should compare against. Once `cycle_start_day != 1`, most cycles straddle two calendar months (e.g. `cycle_start_day=15` → a cycle running Jan 15–Feb 14 touches both `budgets` rows for `2026-01` and `2026-02`). FR-4 explicitly requires this to work ("Chu kỳ hiện tại... được lưu và áp dụng nhất quán cho cả Safe-to-spend lẫn Cảnh báo ngân sách"), and FR-5/6/7 (budget alerts, dashboard status) depend on a single limit per category per cycle. Nothing in the spine, the PRD, or the UX docs (`DESIGN.md`, `EXPERIENCE.md`) picks a rule (e.g. "the limit is the `budgets.month` row matching the cycle's start month," or "prorate across the two overlapping months," or "budgets become cycle-keyed too"). This is exactly the kind of decision the level below (implementer) will make ad hoc and two people (or the same person on two features) could pick differently — the divergence AD-4 exists to prevent, left half-closed.

**Fix:** extend AD-4 with an explicit rule for `cycle_start_day != 1`, e.g. "the applicable budget limit for a cycle is the `budgets` row whose `month` equals the cycle's start date truncated to `YYYY-MM`" (or whatever choice is made) — one sentence closes it.

### M1 — No canonical function for enumerating "N previous cycles" needed by the anomaly rolling average (FR-8, FR-10)

**Checklist criterion:** "every AD's Rule is enforceable and actually prevents its stated divergence" / "misses none."

AD-3 gives one pure function, `CycleWindow(cycleStartDay, asOf) -> (start, end)`, for *the current* cycle containing `asOf`. FR-8's anomaly threshold is "1.5× trung bình 3 chu kỳ gần nhất," and FR-10 gates on "≥1 chu kỳ đầy đủ dữ liệu" — both require stepping back N cycles from the current one. With `cycle_start_day = 1` this is trivial (subtract calendar months), but the whole point of AD-3/AD-4 is to generalize past that case, and cycles of a custom start day are not fixed-length (they inherit calendar-month irregularity, e.g. `cycle_start_day=31`). The spine's Structural Seed lists only `CycleWindow()`/`SafeToSpend()`/`BudgetStatus()` in `cycle.go` — no `PreviousCycleWindow`/equivalent is named. Two implementations of "3 chu kỳ gần nhất" (e.g. `AddDate(0,-1,0)` on the start date vs. `AddDate(0,0,-daysInCycle)`) will not always agree once the cycle length varies, which is precisely the scenario AD-3/AD-4 were written to guard against.

**Fix:** add a second pure function alongside `CycleWindow` (e.g. `PreviousCycleWindow(cycleStartDay int, w Window) Window`) and state in AD-3 (or a new AD) that it is the only way to step between cycles.

### M2 — Alert-state "un-trigger" semantics on transaction update/delete are unspecified (FR-6 × AD-6/AD-7)

**Checklist criterion:** "every AD's Rule is enforceable and actually prevents its stated divergence."

AD-7 correctly extends compute-on-write to `UpdateTransaction`/`DeleteTransaction`, and AD-6 makes `budget_alert_state` the single source of truth, written once at trigger. But the spine doesn't say what happens when an edit or delete brings a category's spend back *below* an already-triggered threshold within the same cycle — do `dismissed_at`-style semantics apply, does the row get deleted so the threshold can re-fire later if spend climbs again, or does it stay (meaning the user is never re-warned even though they're now well under 70%)? FR-6 only specifies the "don't repeat within a cycle while over" case; the reversal case is a real behavior a dev must invent, and different inventions produce different (and both plausible) user-facing behavior. Given AD-7 explicitly binds this rule to the delete/update path, this should be pinned down.

**Fix:** one sentence in AD-6 or AD-7, e.g. "an edit/delete that drops spend below a previously-triggered threshold deletes that `budget_alert_state` row, allowing it to re-fire later in the same cycle" (or the opposite choice, explicitly).

### L1 — Deployment/ops dimension is covered but minimally (one Deferred bullet)

**Checklist criterion:** "a whole dimension left silent... is a finding."

Not silent: Deferred explicitly states infra is unchanged (Docker/Docker Compose, self-host/free-tier) and ties this to the PRD's Non-Goal on Cost (§7). This is a legitimate "ratify the status quo" decision, appropriate for a feature-altitude spine that inherits infra from a higher level. Flagged only as Low/informational because the checklist calls this dimension out by name and the coverage is a single sentence with no mention of what changes operationally when the compute-on-write path is added to the transaction handlers (e.g., no new failure mode called out for a migration failure blocking the new tables, though `Migrate()`'s existing "warn and continue" behavior already implicitly answers this the same way it does today). No action required unless the team wants more explicit operational notes.

## Informational (not defects)

- **Stack table versions verified exact-match against `go.mod`**: Go 1.26.3, gin v1.12.0, gin-contrib/cors v1.7.7, go-sql-driver/mysql v1.10.0, golang-jwt/jwt/v5 v5.3.1, google/uuid v1.6.0, golang.org/x/crypto v0.53.0 — all present verbatim. Frontend versions (React ^19, TypeScript ^5.7, Vite ^6.1, Tailwind ^4.0) verified against `frontend/package.json` — matches the spine's "19/5/6/v4" claim. I did not independently verify these are each package's actual latest/current upstream release (would require a web check unavailable here); internal consistency with the repo's own `go.mod`/`package.json` is confirmed, which is the strongest check available offline. The Stack table's other go.mod deps (viper, zap, testify, testcontainers, godotenv) are correctly omitted as not relevant to the new repo/handler work this spine scopes.
- **Brownfield ratification spot-checks all passed**: `internal/api/api.go` and `cmd/web/main.go` do carry the exact "learning scaffold" / "experimental controller/service/repo scaffold" comments cited by AD-2; `internal/api/db.go`'s `Migrate()` is exactly the idempotent `CREATE TABLE IF NOT EXISTS`-only mechanism described in AD-5 (no ALTER support exists); `internal/api/repo_budget.go` and `budgets.go` match the `XxxRepo{db dbtx}` / `NewXxxRepo` / `*Handler` method / `ok()`/`fail()` shape claimed in AD-1 verbatim, including the `Upsert`-via-`ON DUPLICATE KEY` pattern the spine says `IncomeRepo` should mirror; `router.go` confirms the "everything but `/auth/register`, `/auth/login`, `/auth/demo` sits behind `AuthMiddleware()`" claim used to justify "new endpoints go under the existing auth-required group." No contradictions found between the spine and the real code.

## PRD coverage

All ten FRs (FR-1…FR-10) are covered in the Capability → Architecture Map, each with backing ADs. Cross-cutting NFRs (§7): Reliability and Security/Privacy are structurally supported (AD-2's scope fence protects existing behavior from regression; the "every new table has `user_id` + `userIDFrom(c)` scoping" convention satisfies the privacy NFR); Cost is explicitly addressed in Deferred; Performance (<1s dashboard) is not separately addressed but is reasonable to leave unaddressed given the PRD itself marks it a soft, unbenchmarked target for a single-user, small-data app on the existing indexed schema.

## Deferred section check

Reviewed each Deferred bullet for latent divergence risk: fixed-cost snapshotting, migration versioning, response-envelope duplication, learning-scaffold removal, and Phase-2 shared-ledger prep are all genuinely inert deferrals (each has an explicit fallback rule already governing current behavior, so no two units can silently diverge while the deferred item sits unresolved). The infra bullet is a decision-in-disguise ("no change"), not a true deferral, and is fine as such (see L1).

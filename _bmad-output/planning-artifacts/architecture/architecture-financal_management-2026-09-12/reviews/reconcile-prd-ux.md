---
title: Reconciliation — PRD + UX vs Architecture Spine
inputs:
  - '{planning_artifacts}/prds/prd-financal_management-2026-09-11/prd.md'
  - '{planning_artifacts}/ux-designs/ux-financal_management-2026-09-11/DESIGN.md'
  - '{planning_artifacts}/ux-designs/ux-financal_management-2026-09-11/EXPERIENCE.md'
output_reviewed: '{planning_artifacts}/architecture/architecture-financal_management-2026-09-12/ARCHITECTURE-SPINE.md'
date: 2026-09-12
---

# Reconciliation: PRD + UX → Architecture Spine

## 1. FR Coverage (FR-1 … FR-10)

All ten FRs have an explicit row in the spine's **Capability → Architecture Map**. None are silently dropped.

| FR | Spine home | Verdict |
|---|---|---|
| FR-1 Safe-to-spend hiển thị | `GET /cycle/summary` → `cycle.go` (AD-3, AD-8) | Covered, but see Gap D (response contract for "chưa xác nhận Lương" state is not spelled out) |
| FR-2 Khai báo Lương | `GET/PUT /income` → `IncomeRepo` (AD-5) | Covered, but see Gap E (prefill-from-previous-cycle needs a query shape not defined) |
| FR-3 Chi phí cố định + Mục tiêu tiết kiệm | `/fixed-costs` CRUD + `/settings` → `FixedCostRepo`, `SettingsRepo` (AD-5) | Covered. "Living list, not per-cycle snapshot" tradeoff is explicitly acknowledged in Deferred. |
| FR-4 Ngày bắt đầu chu kỳ | `/settings` (`cycle_start_day`) + `cycle.go` (AD-3, AD-4, AD-5) | Covered |
| FR-5 Phát cảnh báo ngưỡng | Compute-on-write in transaction handlers → `AlertStateRepo`; read via `GET /cycle/summary` (AD-4, AD-6, AD-7) | Covered |
| FR-6 Không lặp cảnh báo trong chu kỳ | Same as FR-5; uniqueness via `UNIQUE(user_id, category_id, cycle_start_date, threshold)` on `budget_alert_state` | Covered — the "reset trạng thái khi chu kỳ mới bắt đầu" requirement falls out automatically from keying state rows by `cycle_start_date` (no explicit reset job needed, which is correct) |
| FR-7 Tổng quan trạng thái ngân sách | `GET /cycle/summary` (AD-4, AD-8) | Covered on Dashboard. See Gap F for the Ngân sách page. |
| FR-8 Gắn cờ bất thường | Compute-on-write → `FlagRepo`; `flagged/flagReason` on `GET /transactions` (AD-6, AD-7) | Covered, but see Gap G (which transaction gets flagged when a category crosses the threshold cumulatively is not decided) |
| FR-9 Đánh dấu đã xem | `POST /transactions/:id/ack-flag` → `transaction_flags.acknowledged_at` (AD-6, AD-7) | Covered for the happy path; see **Gap A** for the forever-gone contradiction risk |
| FR-10 Không gắn cờ khi thiếu baseline | `cycle.go` (checks ≥1 full cycle before computing baseline) (AD-3) | Covered |

**Conclusion on FR coverage:** structurally complete — every FR has a named endpoint/table/governing AD. The gaps below are about behavioral completeness and contract precision, not missing capabilities.

## 2. New UX component → data/API backing

| Component | Backing in spine | Verdict |
|---|---|---|
| `safe-to-spend-hero` | `GET /cycle/summary` (safe-to-spend number, days remaining) | Backed, but the response shape needed to drive the hero's three states (number / "cần cập nhật Lương" / "thiếu chi phí cố định") is not defined — see Gap D |
| `alert-banner` | `GET /cycle/summary` (active, undismissed alerts) + `POST /alerts/:categoryId/:threshold/dismiss` | Backed |
| `budget-status-pill` | `GET /cycle/summary` (per-category % + 3-state status) | Backed on Dashboard; integration with the existing Ngân sách page's own listing endpoint is unaddressed — see Gap F |
| `anomaly-badge` | `GET /transactions` (`flagged`, `flagReason` fields) + `POST /transactions/:id/ack-flag` | Backed, subject to Gap A |
| `cycle-update-sheet` | `GET/PUT /income`, `/fixed-costs` CRUD, `/settings` | Backed as three separate calls with no atomicity/error story for the sheet's single "Lưu" action — see Gap C |

## 3. Gaps, contradictions, and under-specifications

### Gap A — FR-9 "dismiss-then-forever-gone" is not provably durable against AD-7's compute-on-write (highest priority)

PRD FR-9 and EXPERIENCE.md are explicit: once a flagged transaction is acknowledged, the badge is gone **"vĩnh viễn cho giao dịch đó (không hiện lại dù mở lại trang)."** The PRD even allows an internal log to persist, but the flag must never resurface for that transaction.

The spine's `transaction_flags` table uses `PK = transaction_id` (one row per transaction, with `acknowledged_at` nullable), and AD-7 states the anomaly check "chạy đồng bộ ngay trong handler tạo/sửa/xoá giao dịch" for **Create/Update/Delete**, i.e. an `UpdateTransaction` call re-runs the FR-8 check.

Nothing in the spine says what the upsert into `transaction_flags` does when a row already exists with `acknowledged_at` set and the user then edits the transaction (e.g., fixes a typo in the amount, or just re-saves it). If the compute-on-write path does a naive upsert of `(reason, flagged_at)` on every update — which is the natural reading of AD-7 — it can silently clear or ignore the existing `acknowledged_at`, resurrecting a badge the user already dismissed forever. This is a direct, unaddressed collision between AD-7 (recompute on every write) and AD-6/FR-9 (state is durable, "ghi một lần tại thời điểm kích hoạt"). The spine needs an explicit rule, e.g. "compute-on-write for FR-8 only inserts when no row exists for that transaction, or never re-flags a transaction whose `acknowledged_at` is already set, even if edited" — this is currently missing.

### Gap B — "im lặng khi không có gì bất thường" — verified safe, not a gap, but not documented as a spine intent

DESIGN.md's silence principle and EXPERIENCE.md's State Patterns (no `alert-banner` when nothing is active; no "đang học" indicator for FR-10) are satisfied *by construction* in the spine: `GET /cycle/summary` returning an empty alerts array and `GET /transactions` returning `flagged:false`/absent rows both degrade to "nothing rendered" without any extra state to model. No contradiction found. Flagging only because the spine itself never states this as a design intent it is honoring — a one-line note in AD-6/AD-7 ("absence of a row = silence, by design, matching DESIGN.md's quiet-by-default principle") would make the connection traceable instead of coincidental.

### Gap C — `cycle-update-sheet`'s single "Lưu" action maps to 3+ uncoordinated API calls

EXPERIENCE.md Flow 1 treats the sheet as one atomic action ("Minh nhập Lương... xem lại Chi phí cố định... xác nhận Mục tiêu tiết kiệm, nhấn Lưu" → one climax where "Dashboard tự cập nhật ngay tại chỗ"). The spine backs this with three independent resources: `PUT /income`, `/fixed-costs` CRUD (potentially multiple calls, one per fixed cost row), and `PUT /settings` (savings goal + cycle_start_day). AD's `withTx` guidance only covers atomicity *within* a single handler's multi-table writes, not atomicity *across* these separate HTTP requests triggered by one client-side Save click. The spine does not say:
- what order the frontend should call these in,
- what happens to the UI/state if the income PUT succeeds but a fixed-cost update fails (partial save), or
- whether the sheet's single inline validation error ("Lương phải lớn hơn 0") is expected to prevent all three calls from firing, or only the income one.

This is a real under-specification for a flow the UX document treats as one indivisible unit.

### Gap D — `GET /cycle/summary` response contract doesn't distinguish "no income declared" from "income = 0"

FR-2's consequence is explicit: if no Lương has been declared for the current cycle, Safe-to-spend must **not** show a misleading number (e.g. 0 or negative) — it must prompt for input instead (UX: hero shows "Cập nhật Lương để xem Safe-to-spend"). This requires the summary endpoint to expose a distinct signal (e.g. `incomeDeclared: bool`) separate from the numeric fields, not just an amount that happens to be 0. The spine names the endpoint and the governing ADs (AD-3, AD-8) but never sketches the response shape, so this state-vs-value distinction — which the UX explicitly depends on (`safe-to-spend-hero`'s three states in Component Patterns / State Patterns) — is not guaranteed to survive into implementation.

### Gap E — No API shape defined for "prefill Lương with the previous cycle's value as a suggestion"

EXPERIENCE.md Flow 1 step 3 and Component Patterns require: "Trường Lương điền sẵn giá trị chu kỳ trước làm gợi ý, không tự áp dụng." `incomes` is keyed `UNIQUE(user_id, cycle_start_date)`, so the data to satisfy this exists, but `GET/PUT /income` as named in the Capability Map implies fetching/writing the *current* cycle only. There's no stated way for the frontend to fetch "the previous cycle's income value" to use as a placeholder — either the endpoint needs a cycle-date query parameter, or a dedicated field in `GET /cycle/summary`. Currently unspecified.

### Gap F — `budget-status-pill` on the existing Ngân sách page isn't wired to the new computation

FR-7 and the IA table both place `budget-status-pill` on **both** Dashboard and the existing Ngân sách (Budgets) page ("thay hiển thị nhị phân cũ" — replacing the old binary display there). The spine's only named home for the 3-state computation is `GET /cycle/summary` (Dashboard-shaped). The pre-existing Budgets page presumably has its own listing endpoint (`BudgetRepo`-backed); the spine doesn't say whether that page will also call `/cycle/summary`, or whether the existing budgets endpoint needs the same % + status fields merged in. Left unresolved, the two surfaces risk computing/displaying budget status two different ways — the exact class of bug AD-8 is written to prevent.

### Gap G — FR-8's flagging target ("giao dịch hoặc Danh mục") is inherited-ambiguous, not resolved by the spine

PRD FR-8 says the check compares category-cycle totals against history, then flags "giao dịch hoặc Danh mục" (transaction *or* category) — already ambiguous in the PRD. The spine's `transaction_flags` table (PK = `transaction_id`) implicitly resolves this to per-transaction flagging, which matches the UX's per-row `anomaly-badge`, and is a reasonable resolution — but the spine never states this as a deliberate decision, nor addresses the underlying case: if a category crosses the anomaly threshold cumulatively (several small transactions together, none individually large), which transaction(s) get the badge — only the one that tipped the running total over, or none of the earlier ones, or all of them retroactively? Compute-on-write firing per-transaction-write suggests only the newest transaction at the moment of crossing gets flagged, but this is not stated as a rule anywhere, so a future implementer could go either way.

### Minor note — Performance NFR not reflected in Structural Seed

PRD §7 sets a soft <1s Dashboard load target for Safe-to-spend/budget status. `GET /cycle/summary` is compute-heavy on read (per-category sums over a date range, potentially joined against `fixed_costs`/`incomes`/`budget_alert_state`), yet the spine's Structural Seed doesn't mention indexes (e.g. `transactions(user_id, category_id, date)`) needed to keep that endpoint fast. Not a correctness gap, but worth a line in Consistency Conventions or Deferred given it's a named NFR.

## 4. Summary verdict

- All 10 FRs are present in the Capability → Architecture Map — no FR is silently dropped.
- All 5 new UX components have a named data/API backing in the spine — no component is left floating.
- The most consequential issue is **Gap A**: AD-7's blanket "recompute on every transaction write" is not reconciled with FR-9's "acknowledged forever" guarantee, and as written could let an edit resurrect a dismissed anomaly badge.
- Gaps C, D, E are contract-level omissions (no defined request/response shape) for behaviors the UX treats as load-bearing (atomic sheet save, income-not-yet-declared state, prior-cycle prefill).
- Gap F is a cross-surface consistency risk (Dashboard vs. Ngân sách page computing/serving budget status differently).
- The UX's "quiet by default" principle (Gap B) is actually satisfied by the spine's design, just not documented as intentional.

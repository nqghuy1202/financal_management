---
title: Reconciliation — PRD/Brief vs UX Spine (DESIGN.md + EXPERIENCE.md)
created: 2026-09-11
inputs:
  - prds/prd-financal_management-2026-09-11/prd.md
  - prds/prd-financal_management-2026-09-11/addendum.md
  - briefs/brief-financal_management-2026-09-11/brief.md
outputs:
  - ux-designs/ux-financal_management-2026-09-11/DESIGN.md
  - ux-designs/ux-financal_management-2026-09-11/EXPERIENCE.md
---

# Reconciliation: PRD/Brief → UX Spine

## 1. FR-1..FR-10 coverage in EXPERIENCE.md

All ten FRs have at least a named surface/component/state. Detail:

| FR | Coverage | Where |
|---|---|---|
| FR-1 (show Safe-to-spend on open) | Full | `safe-to-spend-hero` component pattern, Flow 1 |
| FR-2 (declare/update Lương) | Full | "Cập nhật chu kỳ" surface, State Patterns row 1, Flow 1 |
| FR-3 (Chi phí cố định + Mục tiêu tiết kiệm) | Full | Same surface, State Patterns row 2 |
| FR-4 (configurable cycle start day) | **Thin** | One IA-table line ("nằm trong mục Nâng cao... hiếm khi đổi"); no Flow or State Pattern shows the interaction, validation, or the recompute feedback that FR-4's own consequences require ("đổi ngày bắt đầu chu kỳ tính lại đúng số ngày còn lại") |
| FR-5 (fire alert at threshold) | Full | `alert-banner`, Flow 2 — banner copy example matches FR-5 consequence (category, %, days left) almost verbatim |
| FR-6 (no repeat same threshold/cycle) | Full | Component Patterns row for `alert-banner`, Flow 2 failure case; reset-at-new-cycle consequence is not restated but not contradicted |
| FR-7 (dashboard budget status, 3 tiers) | Full | `budget-status-pill`, IA table |
| FR-8 (flag anomalous transaction) | Full | `anomaly-badge`, Flow 3 |
| FR-9 (mark reviewed) | Full | "Đã xem, không vấn đề" action, Flow 3 |
| FR-10 (no flagging without baseline) | Full | State Patterns row 4 |

**Gap:** FR-4 is the one FR with no dedicated Flow or State Pattern — every other FR gets an explicit interaction/edge-case treatment, FR-4 gets a single IA-table mention. Given the PRD's own `[NOTE FOR PM]` under FR-4 calls this "an important requirement," the UX spine under-specifies it relative to its stated importance (see §5 below — this compounds with the silently-resolved open question).

## 2. UJ-1/UJ-2/UJ-3 vs Key Flows

All three PRD journeys have a matching Key Flow, 1:1, including edge cases:

- UJ-1 → Flow 1 (cập nhật chu kỳ mới). Edge case (no fixed-cost data yet → default 0 + soft warning) is carried into State Patterns row 2.
- UJ-2 → Flow 2 (cảnh báo giữa chu kỳ). Edge case (no repeat alert on same threshold) carried into the Flow's failure case and FR-6 cross-reference.
- UJ-3 → Flow 3 (xem lại khoản chi bị gắn cờ). Edge case (insufficient history) carried into State Patterns row 4.

No missing journey coverage.

## 3. Scope-creep check against Non-Goals / Out-of-Scope

Checked every Non-Goal (§5 PRD) against DESIGN.md + EXPERIENCE.md:

- Shared/family ledger (Giai đoạn 2) — not implied anywhere; "Single-tenant" stated explicitly in Foundation.
- Email/push/bot alerts — not implied; `alert-banner` is explicitly in-app only, Toast (existing, in-app) reused for errors.
- Bank sync, investment/asset/debt tracking, bill negotiation/subscription cancellation — no mention.
- Portfolio/recruiter-facing polish — DESIGN.md explicitly disclaims this ("không phải một sản phẩm tiêu dùng cần gây ấn tượng").
- Fraud/security detection via anomaly flag — DESIGN.md explicitly reinforces the PRD's exclusion (violet, not a warning triangle, "không phải cảnh báo lỗi/bảo mật").

**No scope creep found.** One thing worth a second look, not creep: EXPERIENCE.md invents a rule not in the PRD — "tối đa 3 banner cùng lúc, over ưu tiên trước near, phần còn lại gộp thành '+N cảnh báo khác'." This is a reasonable UX filling of a gap the PRD left open (PRD never addresses simultaneous multi-category alerts), not a scope violation, but it is a UX-originated product decision that the PM should ratify since it changes visible behavior (some alerts become link-only) beyond what FR-5/FR-6 describe.

## 4. Qualitative intent from the brief (tone, no gamification, in-app-only)

All three checked, all intact:

- **"Proactive not passive"** — carried faithfully: hero is unmissable, alerts fire immediately after the triggering transaction, anomaly badges appear inline rather than in a buried report. Matches brief's "chủ động" framing precisely.
- **No gamification / no portfolio feel** — Voice and Tone table explicitly bans celebratory copy ("Bạn đang làm rất tốt! 🎉", exclamation marks, emoji) and DESIGN.md explicitly rejects visual "impressiveness" as a goal. Consistent with brief's "không phải một sản phẩm tiêu dùng cần gây ấn tượng" and Non-Goal on portfolio/recruiter framing.
- **In-app-only alerts** — `alert-banner` and Toast are both explicitly in-app; nothing implies email/push/bot.

**Minor tension worth flagging, not a contradiction:** the `anomaly-badge` icon choice ("lucide Sparkles") leans toward a delight/reward visual vocabulary that sits slightly closer to "gamification" iconography than the doc's own stated intent ("đáng chú ý", not "có gì sai"). The reasoning given (avoid the warning-triangle language of amber/rose) is sound, but Sparkles specifically carries a positive/achievement connotation elsewhere in UI conventions (unlocked, magic, highlight) that could read as mildly celebratory for what's meant to be a neutral self-reflection prompt. Worth a second icon candidate (e.g., `Info` or `Eye`, both of which DESIGN.md already lists as an alternative) rather than `Sparkles` specifically.

## 5. PRD `[NOTE FOR PM]` / Open Questions vs UX docs

PRD has 4 Open Questions and 2 inline `[NOTE FOR PM]` markers (§4.1 FR-4 Notes, §4.3 FR-10 Notes) plus one in §5 Non-Goals (data model prep for household — an architecture concern, not UX).

- **Open Question 1 (FR-4 Notes): "ask cycle-start day at onboarding, or default to day 1 and let user change later?"** PRD explicitly says this decision belongs at `bmad-ux`. **EXPERIENCE.md does resolve it** — cycle-start-day config is placed inside a collapsed "Nâng cao" (Advanced) sub-section of the "Cập nhật chu kỳ" sheet, reachable only via the pencil icon on the hero, with the IA table justifying this as "hiếm khi đổi". This is a real answer to Open Question 1 (default day 1, no onboarding prompt). **But EXPERIENCE.md never flags this as resolving the PM's open question** — it reads as an incidental placement decision, not a called-out resolution. Given the PRD's own FR-4 note calls the salary-date mismatch case important enough to require confirmation "khi vào bmad-ux," this should have been an explicit, labeled decision (e.g., a `[RESOLVED: Open Question 1]` callout), not something inferable only from where a settings row was placed. Recommend adding an explicit note in EXPERIENCE.md's Foundation or State Patterns section stating this resolution and its rationale, so it can be traced back to the PRD by whoever builds it next (architecture/dev).
- **Open Question 2 (threshold values 70/90/100%, 1.5x/3-cycle)** — explicitly deferred to post-launch usage data in the PRD; UX docs correctly just consume the given values without re-litigating them. No gap.
- **Open Question 3 (`TAI_LIEU_KY_THUAT.docx` unreviewed)** — scoped to `bmad-architecture`, not a UX concern. No gap.
- **Open Question 4 (README recruiter language cleanup)** — a repo-hygiene item, not a UX concern. No gap.
- **FR-10 Notes (thresholds are a starting point, revisit later)** — informational only, nothing for UX to act on. No gap.

## Additional finding (outside the 5 checks, worth noting)

EXPERIENCE.md's Information Architecture section says "→ Tham chiếu bố cục: `mockups/dashboard.html`" as the authoritative layout reference, but no `mockups/` directory or `dashboard.html` file exists anywhere under this UX package (`ux-financal_management-2026-09-11/`) — confirmed by directory listing (only `DESIGN.md`, `EXPERIENCE.md`, `.memlog.md`, and two empty `.working`/`imports` folders exist). This is a dangling reference: either the mockup was never produced/committed, or it lives outside this folder and the path is wrong. Whoever consumes this spine next (dev/architecture) will hit a broken link for the one concrete visual reference the doc points to.

---

## Summary of gaps

1. **FR-4 under-specified relative to other FRs** — only an IA-table mention, no Flow/State Pattern for changing the cycle start day or its recompute feedback.
2. **Open Question 1 silently resolved, not flagged as a resolution** — cycle-start-day config is tucked into a collapsed "Advanced" section (default day-1, no onboarding ask), which does answer the PRD's open question, but EXPERIENCE.md doesn't call this out as a decision tracing back to that open question.
3. **Un-PRD'd multi-banner cap rule** (max 3 banners, over-before-near, "+N khác" rollup) — a sensible but PM-unratified behavior addition beyond FR-5/FR-6.
4. **Anomaly-badge icon choice ("Sparkles") leans mildly celebratory**, in slight tension with the brief's "no gamification" tone intent, though not a hard contradiction.
5. **Dangling reference**: `mockups/dashboard.html` cited as the layout source of truth does not exist in the UX package.

No FR is missing UX coverage; all three UJs have matching Key Flows; no Non-Goal is contradicted or scope-crept into the UX docs.

# Reconciliation: Brief vs. PRD — HL Personal Finance (Cải tổ)

Compared:
- Input: `briefs/brief-financal_management-2026-09-11/brief.md` + `addendum.md`
- Output: `prds/prd-financal_management-2026-09-11/prd.md` + `addendum.md`

## Verdict on the four specific checks requested

1. **Vision/scope match (brief Executive Summary + Scope vs. PRD §1 Vision)** — Matches well overall. PRD §1 restates the "sổ ghi chép thụ động → trợ lý kiểm soát dòng tiền chủ động" framing near-verbatim, keeps the three pillars, and keeps the "vài tuần tới" timeframe. One piece of the brief's *forward-looking* Vision section (not the Exec Summary) is lost — see Gap 2 below.
2. **Phase-2 family-ledger deferral** — Carried forward correctly and consistently: PRD §2.2 (non-users), Glossary ("Sổ chung gia đình"), §5 Non-Goals, §6.2 (with the `[NOTE FOR PM]` about preparing the data model for group ownership, matching the brief addendum's architectural heads-up almost 1:1).
3. **In-app-only alert channel decision** — Carried forward correctly, including the brief's *reasoning* (avoid building email/push infra now; push becomes more valuable once a family ledger exists because members won't all have the app open at once). See PRD §5 and §6.2.
4. **"Not a portfolio project" framing** — Carried forward at the framing level (PRD §2.2 lists recruiters/portfolio-viewers as explicit non-users; §5 Non-Goals repeats "không nhắm tới mục đích portfolio/tuyển dụng"). However, the brief addendum's one *concrete* action item tied to this framing (README currently has recruiter-facing language — "recruiters can explore" — that should be reviewed/removed) is not tracked anywhere in the PRD as actual work; see Gap 3.

## Gaps found

### Gap 1 — "What Makes This Different" narrative is dropped entirely
The brief's `## What Makes This Different` section is a distinct, qualitative piece of reasoning: this project isn't competing with Money Lover/Spendee/YNAB, and its value is **ownership + personal fit** — a tool built exactly around the user's own salary/spend/save cycle, fully user-controlled data, and customizable safe-to-spend/anomaly logic. Crucially, the brief explicitly self-disclaims: *"Đây là lợi thế của việc tự xây, không phải một lợi thế cạnh tranh khó sao chép (moat) về kỹ thuật hay thị trường."* (This is the advantage of building it yourself, not a defensible competitive moat.)

Nothing in the PRD restates this. There is no competitive-positioning section, no mention of Money Lover/Spendee/YNAB as reference points for differentiation (the PRD addendum *does* cite YNAB/Mint/PocketGuard/Monarch/Rocket Money, but only as benchmarks for picking numeric thresholds — a different, narrower purpose). The "why build this yourself instead of adopting an existing app" rationale — arguably central to the project's motivation and to anyone later asking "why not just use YNAB" — is silently lost. This is a "why" gap, not a feature gap: nothing functional depends on it, but a reader of the PRD alone would not know this reasoning exists.

**Recommendation:** add a short subsection (or fold into §1 Vision) preserving the ownership/personal-fit rationale and the explicit "not a moat" disclaimer, so downstream readers don't mistake this for a market-competitive product.

### Gap 2 — Long-range Vision aspiration and closing tone line dropped
The brief's `## Vision` section (distinct from the Exec Summary) describes two things the PRD does not carry forward:
- The safe-to-spend logic eventually being smart enough to not just alert but **suggest adjustments** (e.g., "dồn tiết kiệm cho một mục tiêu cụ thể" — redirect savings toward a specific goal) — a step beyond the family-ledger deferral itself.
- The closing tone statement: *"một hệ thống nhỏ nhưng vững chắc — dễ bảo trì, dữ liệu đáng tin cậy — phản ánh đúng tinh thần ban đầu: một công cụ được xây để thực sự dùng, không phải để trưng bày."* (A small, sturdy, maintainable system — built to actually be used, not to be shown off.)

PRD §1 Vision covers the family-ledger data-layer prep but stops there — it does not mention the "smarter suggestions" aspiration, and the "built to be used, not shown off" framing survives only indirectly (scattered across §2.2 non-users and §5 Non-Goals) rather than as part of the Vision narrative itself. Low functional impact (the smarter-suggestions idea is explicitly beyond v1 either way), but it is a genuine trimming of the brief's articulated long-term ambition and voice.

**Recommendation:** optional — a one-sentence addition to PRD §1 noting the longer-range ambition (smarter, suggestion-capable safe-to-spend) would preserve continuity for whoever picks this up post-Phase-2, without adding scope now.

### Gap 3 — README recruiter-language cleanup has no tracked home in the PRD
The brief addendum flags a concrete, actionable item: the repo's current README uses recruiter-facing language ("recruiters can explore") that "nên được xem lại/loại bỏ vì không còn phản ánh đúng mục đích dự án." The PRD only cross-references this back to the brief addendum (§2.2, §5) rather than capturing it as a trackable item — it's not in §6.1 MVP Scope, not in §9 Open Questions, and not phrased as any FR or NFR. Since the PRD (plus epics/stories generated from it) is what downstream work will actually be driven from, this small but concrete cleanup task risks falling through the cracks — nobody scans a brief addendum once a PRD exists.

**Recommendation:** either add it as a trivial line item in §6.1 (MVP in-scope housekeeping) or §9 Open Questions, so it survives into `bmad-create-epics-and-stories`.

### Gap 4 (minor) — Usage-frequency success criterion is operationalized without being flagged as an assumption
Brief Success Criteria: *"Người dùng thực sự dùng app hàng ngày/hàng tuần trong ít nhất một chu kỳ lương trọn vẹn."* PRD SM-2 turns this into a specific, testable number: "≥3 lần/tuần trong suốt một chu kỳ ngân sách trọn vẹn." That's a reasonable and probably necessary operationalization (a metric needs a number), and the "full cycle" qualifier is preserved — but the specific "3x/week" threshold is an interpretive choice not present in the brief and is not marked `[ASSUMPTION]` the way other numeric choices in the PRD are (e.g., the 70/90/100% alert thresholds, the 1.5x anomaly multiplier both are marked and listed in §10 Assumptions Index). For consistency, this one should probably be flagged the same way.

**Recommendation:** add "SM-2's 3x/week threshold" to §10 Assumptions Index for consistency with how the other invented numbers were handled.

## Things explicitly checked and found NOT to be gaps
- All four brief Solution bullets (safe-to-spend, proactive budget alerts, anomaly flagging, reliable backend) map cleanly to PRD §4.1–4.3 and the Reliability NFR (§7).
- All Success Criteria bullets map to Success Metrics (§8), including the "existing features must not regress" criterion (SM-3) and the "alert before, not after, 100%" criterion (SM-1).
- Brief Scope (in-scope / out-of-scope for the first iteration) matches PRD §6.1/§6.2 and §5 Non-Goals point for point, including "no bank sync," "no investment/debt tracking," "no bill negotiation/subscription-cancellation features."
- Technical-constraint items from the brief addendum (stack pinning, the in-progress un-committed repo-layer refactor, the unreviewed `TAI_LIEU_KY_THUAT.docx`) are all correctly surfaced in PRD §9 Open Questions and in the PRD's own addendum's "Ghi chú cho bmad-architecture" section.
- Currency/locale (VND, Vietnam context) — not explicit in the brief but reasonably logged by the PRD as an assumption in §10; no contradiction.
- No feature or decision in the PRD contradicts anything stated in the brief or its addendum.

## Net assessment
No structural or functional gaps — the PRD is a faithful, well-organized superset of the brief's functional content, and the three specifically-flagged decisions (Phase-2 family deferral, in-app-only alerts, not-a-portfolio-project) all carry forward correctly, including their stated rationale. The gaps that do exist are qualitative/tonal: the brief's explicit "why build this yourself, not a competitive moat" narrative (Gap 1) and part of its long-range ambition/closing tone (Gap 2) did not survive translation into the FR-structured PRD, and one concrete cleanup action item (README language, Gap 3) has no tracked home going forward.

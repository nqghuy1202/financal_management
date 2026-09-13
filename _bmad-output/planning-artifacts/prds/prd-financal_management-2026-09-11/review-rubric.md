# PRD Quality Review — HL Personal Finance — Cải tổ

## Overall verdict

This is a well-crafted PRD for what it is: a solo-builder, brownfield, chain-top PRD with a genuine thesis (passive ledger → proactive cash-flow control), trade-offs surfaced honestly against named alternatives (PocketGuard, Monarch, YNAB, Mint) in the addendum, and FRs that mostly carry crisp, testable consequences. The one gap serious enough to block downstream work cleanly is a missing functional requirement: nothing in the PRD specifies how "Lương" (income) — the first term in the Safe-to-spend formula that is the flagship feature — is entered or derived, and this omission is not flagged anywhere as an `[ASSUMPTION]` or `[NOTE FOR PM]`. Close that gap and tighten one FR-6 threshold mapping, and this PRD is ready to feed `bmad-ux`/`bmad-architecture` as-is.

## Decision-readiness — strong

The PRD states decisions rather than hedging them. FR-2 commits to "v1 chỉ hỗ trợ khai báo thủ công — không tự động suy luận chi phí cố định từ lịch sử giao dịch," and the addendum names the alternative considered and why it was rejected ("giữ v1 đơn giản, tránh logic phát hiện 'giao dịch định kỳ' phức tạp"). The addendum's "Lựa chọn đã cân nhắc và từ chối" section is genuinely rare in PRDs: it names Monarch Money's multi-bucket model and YNAB's outright rejection of safe-to-spend forecasting, and states what the PRD gives up by choosing a single number ("có thể xem xét tách bucket ở giai đoạn sau nếu một con số duy nhất gây hiểu lầm" — naming the risk, not just the choice).

Open Questions (§9) are genuinely open rather than rhetorical: Q1 (onboarding timing for cycle-start date) is explicitly deferred to `bmad-ux` with a stated non-blocking status; Q2 (whether the 70/90/100% and 1.5x thresholds hold up) is deferred to real usage data. `[NOTE FOR PM]` callouts land at real tensions — FR-3's note about salary-date mismatch risk, FR-9's note that the 1.5x/3-cycle threshold is a starting point, and §6.2's note that the data model should prepare for Phase 2 sharing without building it. None of these read as safe-checkpoint theater.

No findings — this dimension needs none.

## Substance over theater — strong

Single persona (Minh) used consistently across all three UJs and tied to specific FR realizations — not persona theater. The addendum grounds every `[ASSUMPTION]` value in named competitor precedent (PocketGuard's real-time recompute, Rocket Money's variable-income handling, Mint's non-ML heuristic for anomaly detection) while explicitly caveating "chỉ mang tính định hướng, không phải benchmark chính thức" — honest about the shallowness of the research rather than overclaiming novelty. The Vision statement (§1) names the specific prior stack (Go + React + MySQL) and the specific failure mode ("chưa từng được dùng thật" — never actually used) — it could not swap into another PRD unchanged.

NFRs (§7) are mostly specific rather than boilerplate: Reliability names an actual before/after regression test method; Security & Privacy inherits a named mechanism (JWT + bcrypt + user scoping) and states a concrete Phase 2 rule (sharing must be opt-in, not default); Cost commits to no new paid infrastructure. The one soft spot is Performance ("gần như tức thời" with no numeric bound) — see Done-ness clarity finding below.

## Strategic coherence — strong

The thesis is explicit: "biến app từ 'sổ ghi chép' thành 'trợ lý kiểm soát dòng tiền chủ động'" (§1). All three new capabilities (Safe-to-spend, proactive budget alerts, anomaly flagging) serve that thesis directly, and the "harden existing foundation" work is explicitly bucketed separately from it rather than blended in as a fourth capability. Success Metrics validate the thesis rather than measuring activity: SM-1 checks that a warning arrives *before* a category is fully blown (not just that warnings exist), and SM-2 is explicitly framed as "thước đo hành vi dùng thật, không phải số liệu kỹ thuật" — a deliberate avoidance of a vanity technical metric. Counter-metrics are present and correctly paired: SM-C1 bounds alert-fatigue against SM-1, SM-C2 bounds false-positive flagging against SM-4. This is a problem-solving-shaped MVP with scope logic that follows from the thesis, not a backlog with headings.

## Done-ness clarity — adequate

Eight of the nine FRs carry genuinely testable consequences — concrete, verifiable, free of "handles gracefully"/"user-friendly" language. FR-4 specifies exact alert content (category name, % used, days remaining); FR-7 specifies the exact flagging formula and gives example copy; FR-9 specifies the exact data-sufficiency bar. This is above-average rigor for FR consequences.

### Findings

- **critical** No FR specifies how "Lương" (income) is entered or derived (§3 Glossary, §4.1, FR-2) — The Glossary defines Safe-to-spend as `(Lương − Chi phí cố định − Mục tiêu tiết kiệm) chia đều cho số ngày còn lại trong Chu kỳ ngân sách` (§3), but FR-2 — the only FR governing declarable inputs to this formula — covers only "các khoản Chi phí cố định và một Mục tiêu tiết kiệm," never Lương. No other FR addresses it either. An engineer building the flagship feature (FR-1, realizing UJ-1) has no spec for where the first term of the formula comes from: a manual per-cycle field, a settings value, or something inferred from income transactions in Giao dịch. *Fix:* add an FR (e.g., FR-2b, "Khai báo/cập nhật Lương cho chu kỳ hiện tại") specifying the entry mechanism and its consequences, and add a "Lương" entry to the Glossary (§3) alongside Chi phí cố định and Mục tiêu tiết kiệm.
- **medium** FR-6's three status buckets aren't mapped to thresholds (§4.2, FR-6) — FR-6 requires Dashboard to show "trạng thái hiện tại (trong hạn mức / gần vượt / đã vượt)" per category, but never states the % ranges that separate the three states. FR-4 defines 70/90/100% for alerts; it's a reasonable guess that FR-6 reuses these, but the PRD doesn't say so, leaving the boundary between "trong hạn mức" and "gần vượt" (e.g., is it 70%, or some other value) to implementer inference. *Fix:* state the mapping explicitly in FR-6, e.g. "trong hạn mức: <70%; gần vượt: 70–99%; đã vượt: ≥100%" or cross-reference FR-4's thresholds directly.
- **low** NFR Performance has no numeric bound (§7, Performance) — "phải hiển thị gần như tức thời" is an adjective, not a bound. Given the explicit hobby/personal-data-scale framing right in the same sentence ("dữ liệu cá nhân, quy mô nhỏ — không cần tối ưu cho tải lớn"), this is low-severity, but even a soft number (e.g., "<1s Dashboard load on typical mobile connection") would give the solo builder a concrete definition of done rather than a feeling.

## Scope honesty — adequate

Non-Goals (§5) is thorough and specific — seven explicit exclusions, each with a one-line rationale (e.g., no bank sync, no investment tracking, no fraud detection framed explicitly as "không phải bảo mật, chỉ là gợi ý"). Phase 2 (Sổ chung gia đình) is consistently deferred with an explicit `[NOTE FOR PM]` in §6.2 that the data model should still prepare for it. De-scoping (e.g., FR-2's manual-only income declaration) is done openly with rationale in the addendum, not silently.

The exception is the Lương gap noted under Done-ness clarity: it is a real omission that is *not* flagged anywhere with `[ASSUMPTION]` or `[NOTE FOR PM]` — the reader has to notice it by cross-checking the Glossary formula against the FR list rather than being told about it. That's the one place this PRD's otherwise-strong "no silent gaps" discipline lapses.

Open-items density (3 Open Questions + 6 indexed assumptions + 4 `[NOTE FOR PM]` callouts) is appropriate for a hobby-scale, non-green-light-to-enterprise PRD — not excessive.

## Downstream usability — adequate

This PRD is chain-top (feeds `bmad-ux`, `bmad-architecture`, `bmad-create-epics-and-stories`), so this dimension carries real weight. ID hygiene is clean: FR-1…FR-9 contiguous and unique, UJ-1…UJ-3 contiguous, SM-1…SM-4 plus SM-C1/C2 clearly labeled by role. Every "Realizes"/"Validates" cross-reference checked resolves correctly (UJ-1→FR-1/2/3, UJ-2→FR-4/5/6, UJ-3→FR-7/8/9, SM-1→FR-4/5, SM-2→FR-1, SM-3→NFR Reliability §7, SM-4→FR-7). All three UJs have a named, consistent protagonist (Minh) with stated entry state and edge case — no floating UJs.

The Glossary (§3) is otherwise solid, but the missing "Lương" entry (see Done-ness clarity finding) is exactly the kind of gap this dimension is meant to catch: a term load-bearing in a formula but absent from the Glossary that downstream skills are told to rely on verbatim (§0: "thuật ngữ được định nghĩa một lần trong Glossary (§3) và dùng nguyên văn xuyên suốt").

## Shape fit — strong

Calibration matches the stated shape well. The document explicitly frames itself for a solo builder ("chính người xây dựng dự án, đóng vai trò vừa PM vừa dev," §0), and the rigor is light without dropping substance: one persona (not persona-theater inflation), three UJs (appropriate since this is a consumer-shaped mobile app, not an internal single-operator tool where UJs would be overhead), and NFRs scaled to personal/hobby stakes rather than enterprise boilerplate. Brownfield handling is accurate and honest: existing capabilities (auth, giao dịch, danh mục, ngân sách, dashboard) are explicitly marked "đã tồn tại trong hệ thống hiện tại" in the Glossary and separated from new capabilities in MVP scope (§6.1); Open Question 3 honestly flags that `TAI_LIEU_KY_THUAT.docx` hasn't been reviewed yet and defers that check to `bmad-architecture` rather than silently ignoring an unverified technical reference.

## Mechanical notes

- **Assumptions Index roundtrip gap (§10 vs. body):** 3 of the 6 indexed entries have no matching inline `[ASSUMPTION]` tag in the body text — they're stated as plain FR requirements or narrative instead: the FR-5 "one alert per threshold per cycle" rule, the FR-9 "minimum 1 full cycle of data" rule, and the document-wide VND currency assumption (nowhere tagged inline, only asserted in the Index as "kế thừa từ brief, chưa hỏi lại trực tiếp"). This doesn't break anything, but it means scanning for `[ASSUMPTION]` inline undercounts what the Index itself claims are assumptions.
- **Minor glossary gap:** "Ngưỡng bất thường" (the anomaly threshold used in §4.3's `[ASSUMPTION]` and FR-7) is never given its own Glossary entry, unlike its sibling "Ngưỡng cảnh báo" which is. Low value fix if the term is reused downstream.
- Title still marked "Working title — confirm" (line 9) — presumably intentional and pending the builder's own confirmation, not a defect.
- No broken cross-references, no ID gaps or duplicates found across FR/UJ/SM numbering.

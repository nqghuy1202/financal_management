# Spine Pair Review — HL Personal Finance

## Overall verdict

This is a well-disciplined delta spec, correctly scaled to a solo hobby project extending an existing Tailwind design system rather than a greenfield spine — it inherits colors/type/shape wholesale, names the existing frontend file it inherits from, and adds only what the three new capabilities require. Terminology and ID references (UJ-1..3, FR-1..10, component names) are used verbatim and consistently across PRD → EXPERIENCE.md → DESIGN.md with no drift found. The two real defects are a citation to a mockup file that was never created (`mockups/dashboard.html`) and one new surface — the "Cập nhật chu kỳ" cycle sheet — that never gets a proper Components/Component Patterns row of its own, its behavior instead scattered across the IA table, one Key Flow, and Interaction Primitives. Neither is disqualifying at this project's stakes, but both are worth a five-minute fix before an architecture/dev pass consumes this spine.

## 1. Flow coverage — strong

Sources frontmatter → PRD §2.3 gives UJ-1, UJ-2, UJ-3. `EXPERIENCE.md → Key Flows` has all three, each with a named protagonist (Minh), numbered steps, a bolded `**Climax:**` beat, and a `Thất bại:` failure path. No misses.

### Findings
None.

## 2. Token completeness — adequate

Frontmatter `colors` (brand/ink/anomaly scales), `typography.display-number`, `rounded`, and `components` all cross-check cleanly: every `{path.to.token}` reference used in the `components` block (`{colors.brand-50}`, `{colors.brand-500}`, `{rounded.2xl}`, `{typography.display-number}`, `{colors.anomaly-50}`, `{colors.anomaly-700}`, `{rounded.full}`) resolves to a value defined earlier in the same frontmatter. `amber`/`rose` are used only as literal Tailwind utility strings (`'bg-amber-50 text-amber-700'`), not `{path}` refs, so they don't need a frontmatter entry to resolve — not a gap.

### Findings
- **low** `spacing:` (DESIGN.md line 36) has no keys, only a trailing comment — it parses as `spacing: null` rather than an object, against the type rule in `references/design-md-spec.md` ("Scale levels … → dimensions"). Harmless today since nothing references `{spacing.*}`, but the moment something does, resolution breaks silently. *Fix:* either give it an empty-but-typed placeholder or a one-line note like the `typography` field does, so it stays an object.

## 3. Component coverage — adequate

The four flagship components (`safe-to-spend-hero`, `budget-status-pill`, `alert-banner`, `anomaly-badge`) each have a real row in `DESIGN.md → Components` (anatomy, color usage) and a real row in `EXPERIENCE.md → Component Patterns` (behavioral rules, not one-word descriptions) — e.g. `alert-banner`'s pattern row specifies max-3-visible, rose-before-amber priority, and "+N cảnh báo khác" overflow, which is genuine behavioral spec, not filler.

### Findings
- **medium** The "Cập nhật chu kỳ" sheet (EXPERIENCE.md IA table, line 25) is a real new surface — a form with Lương/Chi phí cố định/Mục tiêu tiết kiệm fields plus a collapsed "Nâng cao" section for cycle-start-day — but it has no row in `DESIGN.md → Components` and no row in `EXPERIENCE.md → Component Patterns`. Its behavior (prefill-not-auto-apply, inline validation "Lương phải lớn hơn 0", sheet-on-mobile/modal-on-desktop) is recoverable only by piecing together the IA table one-liner, Key Flow 1 (lines 87–95), and Interaction Primitives (line 71). A consumer doing per-component source-extraction (the exact test this rubric runs) will not find it under Component Patterns. *Fix:* add one row for the cycle sheet under both DESIGN.md.Components (even if it's "inherits `.input`/`.label`/modal chrome, no new visual tokens") and EXPERIENCE.md.Component Patterns consolidating the behavior already stated in the flow.
- **low** `EXPERIENCE.md → Component Patterns` (line 54) has a row for "Bảng Giao dịch (đã có)" stating it's structurally unchanged, but `DESIGN.md`'s "Không đổi" list of untouched components (line 106: `.btn-primary/.btn-ghost/.btn-outline/.btn-icon`, `.input`, `.label`, `StatCard`, `CategoryIcon`, Recharts charts) never names the transactions table itself. *Fix:* add it to that list for symmetry.

## 4. State coverage — adequate

`EXPERIENCE.md → State Patterns` covers the states that matter most: unconfirmed-salary hero, partial-cycle-data hero, empty alert list (explicitly tied back to the "silent when nothing is wrong" brand principle), insufficient-baseline for anomaly flagging, cold-load skeleton, and load errors. This is the strongest section in the pair — each row ties back to an FR/UJ and states a concrete treatment, not a placeholder.

### Findings
- **low** The cold-load/skeleton row (line 64) is scoped to Dashboard only. Budgets and Transactions pages also gain new elements this cycle (`budget-status-pill`, `anomaly-badge`) but their loading treatment isn't addressed — presumably inherited from the existing pages' current behavior, but that inheritance isn't stated the way it is elsewhere (e.g. Elevation & Depth explicitly says "kế thừa nguyên vẹn").
- **low** The cycle sheet's validation-error state ("Lương phải lớn hơn 0") only appears in Key Flow 1's failure path (line 95), not as a row in the State Patterns table alongside the other surfaces' states.

## 5. Visual reference coverage — broken

`imports/` exists but is empty; there is no `mockups/` or `wireframes/` directory anywhere in this workspace or the repo.

### Findings
- **high** `EXPERIENCE.md → Information Architecture` (line 30) states: "→ Tham chiếu bố cục: `mockups/dashboard.html`. Spine … thắng khi có mâu thuẫn." — this cites a file that was never created. It isn't an orphan (an unreferenced file sitting around) but the inverse: a dead pointer a downstream consumer will follow and find nothing. *Fix:* either drop the sentence (the prose already fully describes the one layout change — a new top row for hero + banners) or actually produce `mockups/dashboard.html` if a visual reference was intended.

## 6. Bloat & overspecification — strong

Appropriately lean for a delta spec at this scale: no pixel-pushing where tokens/Tailwind classes already cover it, no restatement of PRD personas/FRs beyond what's needed to justify a decision, tables used instead of prose for Do's/Don'ts and Voice/Tone. `DESIGN.md → Brand & Style` carries editorial voice (permitted); `EXPERIENCE.md` stays behavioral except for one narrative paragraph in Foundation (line 16) justifying the mobile-first `[ASSUMPTION]` — acceptable since it's explicitly flagged as reasoning for an assumption, not decorative color.

### Findings
None.

## 7. Inheritance discipline — strong

`sources:` in EXPERIENCE.md frontmatter (lines 4–6) resolves to real files (`prd.md`, `brief.md`). UJ-1/2/3 and FR-1 through FR-10 are used verbatim throughout both Key Flows and the State/Component Pattern tables with no ID mismatches against the PRD. Component names (`safe-to-spend-hero`, `budget-status-pill`, `alert-banner`, `anomaly-badge`) are spelled identically across DESIGN.md's frontmatter, DESIGN.md's body, and every EXPERIENCE.md table. No glossary drift — terms like "Chu kỳ ngân sách," "Ngưỡng cảnh báo," "Ngưỡng bất thường" are used the same way everywhere they appear.

### Findings
None.

## 8. Shape fit — adequate

DESIGN.md sections run in canonical order (Brand & Style → Colors → Typography → Layout & Spacing → Elevation & Depth → Shapes → Components → Do's and Don'ts). EXPERIENCE.md has all eight required defaults (Foundation, IA, Voice and Tone, Component Patterns, State Patterns, Interaction Primitives, Accessibility Floor, Key Flows).

### Findings
- **medium** `DESIGN.md → Layout & Spacing` (lines 85–87) states the one layout change — a new top row for `safe-to-spend-hero` + alert banners — but never says how that row behaves across breakpoints, even though the very next sentence points at the existing stat-card grid that does define one (`sm:grid-cols-2 xl:grid-cols-4`). This is a responsive web app with an explicit multi-breakpoint grid already in place, so the omission is a real gap rather than an inapplicable section. *Fix:* one sentence — e.g. does the hero stay full-width at all breakpoints, do banners wrap or stack on desktop.
- **low** No "Inspiration & Anti-patterns" section, though PRD assumptions for FR-6 and FR-8 explicitly cite a reference product by name ("tương tự cách Mint gộp cảnh báo," "mô hình … như Mint dùng"), which is the stated trigger for this required-when-applicable section. Weak finding: the Mint references are about computation/business-logic assumptions (threshold %, dedup rule) already justified in the PRD, not lifted interaction patterns the way the worked example (Quill) uses this section — so its absence is defensible, just worth a conscious call rather than a silent drop.

## Mechanical notes

- Broken cross-reference: `mockups/dashboard.html` (EXPERIENCE.md line 30) — file does not exist anywhere in the repo (see Finding under §5).
- `spacing:` frontmatter key (DESIGN.md line 36) parses as `null`, not an object — inert today, flagged under §2.
- No name inconsistencies found between DESIGN.md and EXPERIENCE.md for colors, components, or FR/UJ IDs.
- No Mermaid diagrams present in either file.
- Frontmatter completeness: both files have all fields their respective spec/examples call for (DESIGN.md: `name`, `description`, token blocks; EXPERIENCE.md: `name`, `status`, `sources`, `updated`).

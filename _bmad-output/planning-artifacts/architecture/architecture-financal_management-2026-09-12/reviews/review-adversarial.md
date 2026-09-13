---
type: architecture-review
lens: adversarial
target: ARCHITECTURE-SPINE.md (HL Personal Finance — Cải tổ)
target_path: '{planning_artifacts}/architecture/architecture-financal_management-2026-09-12/ARCHITECTURE-SPINE.md'
created: '2026-09-12'
method: >
  Construct pairs of next-level units (stories/PRs, or FE+BE change pairs) that
  each satisfy every AD in the spine to the letter, yet compose incompatibly.
  Each pair is graded against every AD-1..AD-8 individually before being kept —
  a pair is only reported if no existing Rule closes it.
result: 10 incompatible-pair scenarios found (0 requiring a spec/PRD reading beyond
  the spine itself — all 10 are reachable by two implementers each reading only
  ARCHITECTURE-SPINE.md and picking the more natural of two literal readings).
---

# Adversarial Review — ARCHITECTURE-SPINE.md

## How to read this document

For each scenario: **Unit A** and **Unit B** are two stories/PRs (or a BE+FE
pair) that a sprint-planning split could plausibly hand to two different
developers or two different autonomous dev-agent runs, in parallel or in
either order. Both are checked against AD-1..AD-8 and the Consistency
Conventions table; both pass. The "Clash" is what breaks anyway when they are
integrated. "Closes it" proposes the AD (new or tightened) that would remove
the ambiguity.

---

## 1. "Transaction" is overloaded in AD-7 itself — two legal atomicity boundaries

**Quote (AD-7, line 75):**
> "kiểm tra ngưỡng ngân sách (FR-5) và kiểm tra bất thường (FR-8) chạy đồng bộ
> ngay trong handler tạo/sửa/xoá giao dịch, **ngay sau khi transaction
> commit** — ghi/cập nhật `budget_alert_state`/`transaction_flags` như một
> side-effect của cùng request."

**Quote (Consistency Conventions, line 89):**
> "Ghi nhiều bảng cùng lúc (ví dụ tạo giao dịch + cập nhật
> `budget_alert_state`) dùng `h.withTx` khi tính atomic thực sự cần thiết cho
> đúng đắn dữ liệu; đọc-rồi-ghi đơn giản (không đa bảng) không bắt buộc phải
> bọc transaction."

The word "transaction" in AD-7 is ambiguous between (a) the financial
**giao dịch** record just inserted, and (b) the SQL **`*sql.Tx`** committing
(the exact type `h.withTx` wraps, confirmed in `internal/api/api.go:51`).
Both readings are grammatically valid Vietnamese-English code-mixing in this
codebase, and the spine never disambiguates.

- **Unit A (FR-5, budget alert write)** reads AD-7 literally as "after the
  SQL transaction commits": `CreateTransaction` inserts the row, commits, and
  only *then* calls `AlertStateRepo.Insert` as a plain, unwrapped second
  write. This is defensible under the Convention row too — one extra
  single-table insert is "đọc-rồi-ghi đơn giản (không đa bảng)" from Unit A's
  point of view (it doesn't touch `transactions`, only `budget_alert_state`).
  Fully AD-7-compliant, fully Convention-compliant.
- **Unit B (FR-8, anomaly flag write)** reads the Convention row as
  authoritative and decides atomicity *is* "thực sự cần thiết cho đúng đắn dữ
  liệu" here, because `transaction_flags.transaction_id` is a PK+FK with `ON
  DELETE CASCADE` (line 149) — a flag orphaned from a not-yet-committed
  transaction would violate that FK. So Unit B wraps the transaction insert
  **and** the flag insert in one `h.withTx`, which requires writing the flag
  *before* the outer commit — directly contradicting AD-7's "ngay sau khi
  transaction commit" read literally as "after the DB transaction commits."

**Clash:** the same handler (`CreateTransaction`) now needs two different
crash-consistency guarantees for its two side-effects, built by two people
each with a legitimate, opposite reading of "transaction." Worse: if a crash
happens between commit and the unwrapped alert-write (Unit A's design), a
budget crossing silently never alerts — a correctness gap AD-6 ("nguồn sự
thật duy nhất") assumes can't happen, since it assumes the alert table is
written reliably at the trigger moment.

**Closes it:** Add a Rule that names the atomicity boundary unambiguously,
e.g.: "the side-effect writes in AD-7 MUST be inside the same `h.withTx` as
the triggering `INSERT/UPDATE/DELETE` on `transactions`; 'ngay sau khi commit'
in AD-7 refers to *logical ordering within that one DB transaction*, not a
separate DB round-trip." Also rename the AD-7 phrase to avoid "transaction"
meaning two different things in one sentence.

---

## 2. `cycle.go` / `CycleWindow` has no assigned owner or interval contract

**Quote (AD-3, line 51):** "tồn tại đúng một hàm thuần (pure)
`CycleWindow(cycleStartDay int, asOf time.Time) (start, end time.Time)`."

**Quote (AD-4, line 57):** aggregation "lọc theo khoảng ngày `[cycle_start,
cycle_end)`" — half-open, but this constraint is stated in **AD-4**, about
*aggregation call sites*, not in AD-3's own contract for what `end` means.

AD-3 binds **eight** FRs (FR-1, FR-4–FR-10) to one function that doesn't
exist yet. Nothing in the spine names which story creates `cycle.go` first,
or forbids two stories from each scaffolding it independently if run in
parallel (a likely outcome given the user's stated preference for autonomous,
hands-off BMAD story execution).

- **Unit A (FR-1, Safe-to-spend)** needs `end` to mean "last inclusive
  instant of the cycle" so it can render "N days left in cycle" and compute
  a same-day-inclusive spend total for "today." It implements `CycleWindow`
  returning `end = nextCycleStart.Add(-time.Nanosecond)` and queries with
  `<=`.
- **Unit B (FR-8/FR-10, anomaly baseline)** is built against AD-4's explicit
  words and implements/queries `CycleWindow` with `end = nextCycleStart`
  (exclusive) and `<`.

Both satisfy AD-3's literal text (a pure function with that exact signature
exists and is the single call site everyone uses) and both satisfy AD-4
("no hardcoded month-boundary logic elsewhere"). If Unit A lands first,
Unit B's PR either duplicates `CycleWindow` under a different call
convention or silently reuses Unit A's inclusive-`end` version while still
writing `<` comparisons — an off-by-one day at every cycle boundary, and it
only manifests once every cycle rollover, making it hard to catch in review.

**Clash:** boundary semantics of the one shared primitive are decided by
whichever story merges first, and the second story has no spine text to
check itself against — a silent contract violation, not a merge conflict.

**Closes it:** Pin the exact contract in AD-3 itself, not delegate it to
AD-4: "`CycleWindow` returns `end` **exclusive** (`[start, end)`); every
caller compares with `>= start && < end`, no exceptions for display code."
Also name a single story/file as authoritative owner of `cycle.go` that all
other FR-1/4–10 stories must depend on (sprint-order dependency), not just
"binds."

---

## 3. Two owners of `transaction_flags`: `FlagRepo` vs. the transaction handler's own write/join

**Quote (AD-1, line 39):** "mọi repo mới đều là `XxxRepo{db dbtx}` +
`NewXxxRepo(db dbtx) *XxxRepo`... mọi handler mới là method trên `*Handler`."
This pins repo *shape*, but never says **only** `FlagRepo` may write/read
`transaction_flags`.

**Quote (Capability Map, line 184):** FR-8/9 lives in "Compute-on-write
trong handler giao dịch → `FlagRepo`; trường `flagged`/`flagReason` trên
`GET /transactions`."

- **Unit A (FR-9, ack-flag endpoint)** builds `repo_flag.go` with
  `FlagRepo{db dbtx}` and `POST /transactions/:id/ack-flag` → straightforward
  AD-1-compliant CRUD-style repo, sets `acknowledged_at`.
  It treats the row as immutable-once-flagged truth per AD-6: `flagged`
  stays `true` forever once the row exists; only `acknowledged` toggles.
- **Unit B (`GET /transactions` change to add `flagged`/`flagReason`)** is
  scoped as a change to the **existing** `TransactionRepo`/handler (it is
  the pre-existing endpoint being extended, not a "new repo" under AD-1 — so
  AD-1's repo-shape rule doesn't even apply to it). To avoid an N+1 query
  per row, Unit B's dev adds a `LEFT JOIN transaction_flags` straight inside
  `TransactionRepo`'s existing list query, and — reasonably, from a UX
  standpoint not visible anywhere in the spine — decides `flagged` in the
  JSON response should reflect "still needs attention," i.e.
  `flagged := reason IS NOT NULL AND acknowledged_at IS NULL`.

**Clash:** Unit A's contract ("flagged is forever true once flagged") and
Unit B's contract ("flagged flips to false after ack") are both compatible
with AD-6/AD-7's letter (AD-6 only forbids *recomputing from raw transaction
history*; deriving `flagged` from two existing columns of the flag row itself
isn't "suy luận lại từ lịch sử giao dịch thô," so Unit B isn't violating
AD-6). The two units are built by different people against the same table
with two different read semantics for one JSON field name, and nothing
forces them to agree — this is a wire-contract collision, not a code bug in
either PR alone.

**Closes it:** Add a Rule (tighten AD-6 or add AD-9) that pins the exact
read-projection for every "state" field exposed over the API — e.g.: "the
JSON field `flagged` on `GET /transactions` is `reason IS NOT NULL` and is
**never** re-derived from `acknowledged_at`; acknowledgement is exposed
separately as `flagAcknowledged`." And state explicitly that `FlagRepo` is
the only code path allowed to construct SQL against `transaction_flags`
(even for the join, via a method `FlagRepo.AttachTo(...)` or a shared
fragment), so the projection logic lives in one place.

---

## 4. Two owners of `budget_alert_state` on delete: "write once" vs. "recheck on delete"

**Quote (AD-6, line 69):** "`budget_alert_state`... được ghi **một lần tại
thời điểm kích hoạt**; mọi lần đọc sau chỉ đọc lại hai bảng này, không suy
luận lại."

**Quote (AD-7, line 75):** the compute-on-write check runs in the handler for
"tạo/sửa/xoá giao dịch" (create/edit/**delete**) — explicitly including
delete.

- **Unit A (DeleteTransaction)** reads AD-7's explicit inclusion of "xoá
  giao dịch" as a mandate: when a transaction is deleted, the category's
  cycle total drops, so a previously triggered 90% alert may no longer be
  true. Unit A's dev adds logic to **delete** (or reset) the now-stale
  `budget_alert_state` row for that category/cycle so the UI doesn't keep
  showing a warning for spend that no longer exists.
- **Unit B (`GET /cycle/summary`, FR-6/FR-7)** reads AD-6 literally: state is
  written once, "không suy luận lại" — it treats the presence of an
  undismissed row as permanent history/audit trail ("you *did* cross 90% at
  some point this cycle") and never expects rows to be deleted or mutated
  outside of `dismissed_at`. It may even rely on row *count* for a "how many
  times you approached this limit" stat.

**Clash:** Unit A silently deletes/rewrites rows Unit B assumes are
append-only. Both cite AD-6/AD-7 in their PR description as justification,
correctly — the spine simply never says whether "kiểm tra bất thường/ngưỡng
... khi ... xoá giao dịch" means "recompute and possibly retract" or "just
run the same one-way trigger check again" (which on a delete would almost
always find nothing new to trigger, making the delete-time check a no-op in
practice, a third valid reading).

**Closes it:** AD-6 should state explicitly whether `budget_alert_state`
rows are ever deleted/updated post-write other than `dismissed_at`, e.g.:
"rows in `budget_alert_state` are append-only once inserted; `DeleteTransaction`
runs the same threshold-check as create/update but **never** deletes or
un-triggers an existing row — a lowered total simply does not insert a new
row for a threshold no longer crossed."

---

## 5. Undismissed alerts from past cycles resurfacing forever (no cycle-scope on the read)

**Quote (AD-6, line 69):** "mọi lần đọc sau chỉ đọc lại hai bảng này, không
suy luận lại." **Quote (ERD, line 155):** `UNIQUE(user_id, category_id,
cycle_start_date, threshold)` — a *new* row is created every cycle.

- **Unit A (compute-on-write, FR-5)** relies on the UNIQUE constraint
  including `cycle_start_date` specifically so each cycle gets its own,
  independent alert rows — correct and AD-6/AD-4-compliant.
- **Unit B (`GET /cycle/summary` "danh sách cảnh báo đang hoạt động", FR-6/FR-7)**
  takes AD-6's "không suy luận lại" (don't re-infer/add logic) at face value
  and queries `WHERE user_id=? AND dismissed_at IS NULL` — deliberately
  *not* adding a `cycle_start_date = CycleWindow(...).start` filter, since
  from Unit B's reading, adding extra filtering criteria beyond what's
  literally stored **is** a form of "suy luận" the AD tells it not to do.

**Clash:** a threshold=100 alert from three cycles ago that the user never
dismissed (dismissal is a manual, easy-to-forget action per FR-6/FR-9) stays
"active" forever and keeps showing in the current cycle's summary,
contradicting FR-7 ("tổng quan trạng thái ngân sách" for the *current*
cycle). Both units are individually AD-6-compliant; the incompatibility is
in what "đọc lại" is scoped to.

**Closes it:** Tighten AD-6 (or AD-4) with: "reads of `budget_alert_state`
for 'active alerts' MUST filter `cycle_start_date = CycleWindow(asOf).start`
in addition to `dismissed_at IS NULL`; alerts do not carry over between
cycles regardless of dismissal."

---

## 6. Old month-based budget view vs. new cycle-window view disagree once `cycle_start_day ≠ 1`

**Quote (AD-4, line 57):** "Khi `cycle_start_day = 1` (mặc định), chu kỳ
trùng khớp tháng dương lịch nên `month` vẫn đúng." — the AD itself flags that
this equivalence **breaks** once `cycle_start_day != 1`, but stops there.

Confirmed in the current code (`internal/api/budgets.go`): `ListBudgets`,
`UpsertBudget`, `DeleteBudget` filter/key strictly by the `month` CHAR(7)
string, unmodified by this overhaul (correctly so — AD-2/AD-5 forbid
touching this old convention, and it's outside the Capability Map).

- **Unit A (FR-4, `/settings` cycle_start_day)** lets a user set
  `cycle_start_day = 15`. Fully AD-3/AD-4/AD-5-compliant: it only adds a new
  `user_settings` row, doesn't touch `budgets`.
- **Unit B (any story that leaves the existing budgets-by-month
  screen/endpoint as-is)** is *correct to do nothing* — AD-2 explicitly says
  the pre-existing convention isn't in scope, and AD-5 forbids altering the
  `budgets` table.

**Clash:** the shipped product now has two budget views live at once for the
same user — the legacy "this month" view (calendar month, `budgets.month`)
and the new "this cycle" view (`GET /cycle/summary`, day-range from
`CycleWindow`) — and for any user with `cycle_start_day != 1` these
legitimately show **different numbers for "budget used this period"** with
no UI or API signal that they mean different things. Neither Unit A nor Unit
B violates any Rule; the product-level contradiction is exactly the
consequence AD-4 names and then leaves open.

**Closes it:** Add an explicit Rule closing the gap AD-4 admits, e.g.: "when
`cycle_start_day != 1`, the legacy month-based budget screen/endpoint MUST
be either (a) hidden behind a note that it uses calendar months regardless
of the configured cycle, or (b) deprecated in favor of `/cycle/summary` in
the same release that ships FR-4." This is a product decision the spine
should force explicitly rather than leave as an emergent UX inconsistency.

---

## 7. Changing `cycle_start_day` mid-flight invalidates already-persisted cycle-keyed rows

`incomes` has `UNIQUE(user_id, cycle_start_date)` (line 154) and
`budget_alert_state` keys on `cycle_start_date` too (line 155). Both are
computed by calling `CycleWindow(cycleStartDay, asOf)` — but `cycleStartDay`
is itself mutable, live, user-editable state (`user_settings.cycle_start_day`,
FR-4).

- **Unit A (`PUT /settings`, FR-4)** simply updates
  `user_settings.cycle_start_day`. AD-1/AD-5-compliant: one repo, one new
  table, no cascading writes mentioned anywhere in the spine as required.
- **Unit B (FR-2, `GET/PUT /income`)** computes `cycle_start_date` for
  "today's income row" by calling `CycleWindow(currentSettings.cycle_start_day,
  now)` at write time, per AD-3 — the only sanctioned way to get a cycle
  start date. Fully AD-3-compliant.

**Clash:** a user sets `cycle_start_day = 1`, logs income for the cycle
starting 2026-09-01 (`incomes` row keyed `cycle_start_date = 2026-09-01`),
then on 2026-09-10 changes `cycle_start_day` to 15 (a perfectly legal FR-4
action Unit A supports with no guard). Unit B, and every FR-1/5/6/7/8
consumer, now compute "current cycle" as starting 2026-09-15 or (depending
on rollback direction) 2026-08-15 — a date that has **no matching**
`incomes` row, so Safe-to-spend silently treats the user as having declared
₫0 income for the "current" cycle, and `budget_alert_state` rows keyed to
the old `2026-09-01` start become permanently orphaned/unreachable (they'll
never match any future `CycleWindow(...).start` again, so they neither get
read by Unit B's summary nor ever get cleaned up). Neither AD-3 nor AD-4 nor
AD-5 says anything about what happens to already-written cycle-keyed rows
when the setting that produced their key changes — this is exactly the kind
of "ambiguous ordering" the prompt asks to hunt for: Unit A's write and
Unit B's read are individually correct, but their *relative timing* against
a third mutable input (the setting) is unconstrained.

**Closes it:** Add a Rule: either (a) `cycle_start_day` changes only take
effect starting the *next* cycle boundary (freeze the value used by
`CycleWindow` for any `asOf` before the change's effective date — requires
storing an effective-from timestamp, not just the current value), or (b)
explicitly accept the breakage and require a one-time reconciliation job/
warning on settings change ("changing this will orphan N income/alert
records"). Silence, as currently written, is the hole.

---

## 8. Cardinality mismatch: one over-large transaction can cross multiple thresholds at once

Thresholds are `70/90/100` (line 142), each its own row under
`UNIQUE(user_id, category_id, cycle_start_date, threshold)` (line 155).

- **Unit A (compute-on-write, FR-5)** handles "a single new transaction
  jumps category spend from 40% to 105% in one commit" by inserting a row
  **per threshold newly crossed** (70, 90, and 100 all in the same request) —
  a natural, AD-6/AD-7-compliant reading: each threshold independently
  "kích hoạt," so each gets its own recorded trigger moment.
- **Unit B (`GET /cycle/summary` "danh sách cảnh báo đang hoạt động", FR-6/FR-7,
  and the UX's 3-level pill described in AD-8)** assumes at most one
  *effective* alert state per category per cycle (a single 3-level pill:
  ok/warning/over) and implements the summary query as
  `SELECT ... GROUP BY category_id ORDER BY triggered_at DESC LIMIT 1`,
  or worse, without the `LIMIT 1`/aggregation at all, iterates all matching
  rows and renders three stacked banners for the one category.

**Clash:** nothing in AD-6/AD-7 states whether a single write event may
produce 1..N new alert rows or exactly the highest-newly-crossed one, and
nothing in AD-8 (which governs the FE-visible "3-level status") reconciles
"3 possible severities" with "N possible rows per category per cycle" at the
API layer. Unit A and Unit B are each defensible in isolation; together they
either under- or over-notify.

**Closes it:** Tighten AD-6/AD-7 with: "a single compute-on-write pass
inserts **at most one** new `budget_alert_state` row per category per
request — the highest threshold newly crossed since the last check — even
if multiple thresholds are crossed in one jump." And pin the summary read to
"one row (the highest undismissed, current-cycle threshold) per category."

---

## 9. AD-8 says "computed once at backend" but pins no wire schema — BE and FE stories can each be "AD-8 compliant" and still disagree

**Quote (AD-8, line 81):** "Safe-to-spend, trạng thái 3 mức ngân sách, danh
sách cảnh báo đang hoạt động, và cờ bất thường được **tính một lần ở
backend**, trả qua API mới." The spine's only wire-format text at all is the
Consistency Conventions envelope row (`{code, message, data}` — line 88);
`data`'s inner shape for any of these endpoints is never specified.

- **Unit A (backend, `cycle_summary.go`, FR-1/FR-7)** implements the 3-level
  budget status as a string enum matching the UX doc's category names
  literally: `status: "an_toan" | "canh_bao" | "vuot_han"`.
- **Unit B (frontend, `budget-status-pill` component, built from the UX
  DESIGN.md/EXPERIENCE.md color-token descriptions, in parallel, against
  the *capability map's* endpoint name only — `GET /cycle/summary` — since
  the spine names the endpoint but not its payload)** is written expecting
  `status: 0 | 1 | 2` (matching the three design tokens' declared order) and
  a `safeToSpend` field that is pre-clamped to `>= 0`.

**Clash:** both changes satisfy AD-8's actual Rule — the number is computed
exactly once, in the backend, and the frontend performs no threshold logic
of its own (`analytics.ts` isn't touched, per AD-8's `Prevents`) — yet
integration fails immediately: wrong enum representation, and a real
negative safe-to-spend (overspend) either gets silently clamped to 0 by a
frontend expecting only non-negative values, or crashes a `.toFixed()` call
somewhere, depending on which side "wins" the ambiguity. AD-8 closes the
*duplicate-computation* hole it names, but opens no contract for the shape
of the single computation's output.

**Closes it:** The spine (or a companion doc it should reference — currently
`companions: []`, line 16) needs a pinned response schema per new endpoint,
minimally: field names, enum vs. int encoding for status, and explicit
sign/clamping semantics for `safeToSpend` (allowed negative = overspend, not
clamped). This is exactly the kind of "shared-data-shape" gap the adversarial
brief asks to hunt for, and it's currently unconstrained by any AD.

---

## 10. Concurrent requests racing the same UNIQUE key with no idempotency rule

`budget_alert_state` has `UNIQUE(user_id, category_id, cycle_start_date,
threshold)` (line 155), and the only atomicity guidance is the ambiguous
Convention row already flagged in Finding 1: "đọc-rồi-ghi đơn giản (không đa
bảng) không bắt buộc phải bọc transaction."

- **Unit A (`CreateTransaction`, category X)** and **Unit B
  (`UpdateTransaction`, a different transaction also in category X, changing
  its amount)** can legitimately fire from two concurrent HTTP requests
  (multi-tab, or a bulk-import flow the PRD doesn't rule out) for the same
  user/category/cycle. Both handlers, per AD-7, independently: (1) recompute
  the category's cycle total, (2) determine "70% threshold newly crossed,"
  (3) `SELECT` to check no row exists yet for `(user, category, cycle,
  70)`, then (4) `INSERT`. Each individually follows AD-7's "chạy đồng bộ
  ngay trong handler" and the Convention's "đọc-rồi-ghi đơn giản không bắt
  buộc bọc transaction" to the letter — a single-table read-then-write, no
  `withTx` required by the Rule as written.

**Clash:** between steps (3) and (4) in each request, the other request can
interleave and insert first; the UNIQUE constraint then turns Unit B's (or
Unit A's, whichever loses the race) plain `INSERT` into a duplicate-key
error. Nothing in AD-6/AD-7 specifies that this must be treated as a benign,
already-triggered no-op (`INSERT ... ON DUPLICATE KEY UPDATE` no-op, or a
caught/ignored duplicate-key error) rather than surfaced as a 500 to the
user who simply happened to save two transactions close together. This is
precisely "a race condition the Rules don't close": both units are
individually spec-compliant, and the failure only appears under concurrency
neither unit's author was required to consider.

**Closes it:** Add to AD-6 or the Convention row: "inserts into
`budget_alert_state` and `transaction_flags` MUST be race-safe against the
UNIQUE constraint (e.g. `INSERT IGNORE`/`ON DUPLICATE KEY UPDATE` treated as
success, not error) — concurrent triggers of the same threshold are expected
and must not surface as request failures."

---

## Summary table

| # | Colliding units | Category |
| --- | --- | --- |
| 1 | FR-5 alert-write vs. FR-8 flag-write, same handler | Ambiguous atomicity boundary ("transaction" overload) |
| 2 | FR-1 Safe-to-spend vs. FR-8/10 anomaly baseline | Shared-primitive ownership + interval-inclusivity ambiguity |
| 3 | FR-9 ack-flag vs. `GET /transactions` flagged field | Two owners of one entity + field-semantics clash |
| 4 | DeleteTransaction recheck vs. `GET /cycle/summary` read | Two owners of one entity + mutation-vs-append-only clash |
| 5 | Compute-on-write vs. active-alerts read | Missing cycle-scope on shared-state read |
| 6 | FR-4 cycle_start_day vs. legacy month-based budgets | Two co-existing state-mutation/read paths, product-level contradiction |
| 7 | `PUT /settings` vs. FR-2 income / FR-5 alert writes | Ambiguous ordering across a mutable shared input |
| 8 | Multi-threshold jump write vs. single-pill summary read | Cardinality mismatch, shared-data shape |
| 9 | Backend `cycle_summary.go` vs. frontend `budget-status-pill` | Missing wire schema, shared-data shape |
| 10 | Concurrent CreateTransaction vs. UpdateTransaction | Race condition on UNIQUE constraint, no idempotency rule |

**10 incompatible-pair scenarios found**, all reachable by two implementers
each reading only `ARCHITECTURE-SPINE.md` (no need to consult the PRD/UX docs
to find the ambiguity, though the UX doc's 3-level pill was cited where
relevant to ground Finding 9 in something concrete). Each maps to a
concrete AD tightening or a new AD proposed inline above.

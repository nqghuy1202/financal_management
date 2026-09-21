---
title: 'Expense entry autocomplete'
type: 'feature'
created: '2026-09-21'
status: 'done'
route: 'oneshot'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Adding a transaction in `TransactionModal.tsx` requires retyping the same note (merchant/description) and re-picking the same category every time, even though the user's own transaction history already has that pairing.

**Approach:** Add a client-side autocomplete dropdown on the note field, sourced from the notes already loaded in `DataContext.transactions` (no new API). Suggestions are distinct past `note` values, restricted to transactions whose `type` matches the currently selected type toggle (expense/income), case-insensitive substring match against the current input, deduplicated, ordered most-recent-first, capped at 8. Selecting a suggestion fills `note` and also sets `categoryId` to that historical transaction's category — but only if the user has not manually picked a different category during this modal session (a simple "user touched category" flag, reset each time the modal opens). Suggestions require at least 2 characters typed, are keyboard-navigable (Up/Down/Enter/Escape), and dismiss on outside click or Escape, styled to match the existing `.input`/Tailwind conventions in this file. New UI strings (if any, e.g. an empty-suggestion hint is not needed) go under the `txm.*` prefix in both `vi` and `en` dicts in `frontend/src/lib/i18n.ts`.

</frozen-after-approval>

## Implementation Notes

- All changes contained in `frontend/src/components/TransactionModal.tsx`; no backend, no new files, no new i18n keys needed (dropdown shows raw historical note text, no new labels).
- `useData()` already exposes `transactions`, so no context change was needed.
- Suggestions computed via `useMemo` filtering `transactions` by matching `type`, min 2 chars typed, case-insensitive substring match, deduped by lowercase note (first occurrence wins — list is already ordered most-recent-first by the backend query), capped at 8.
- `categoryTouched` flag (reset on modal open/edit) prevents a selected suggestion from overriding a category the user explicitly clicked; the category button `onClick` now also sets this flag.
- Dropdown closes on outside click (`mousedown` listener scoped to a wrapping ref, only attached while open) and on Escape; suggestion buttons use `onMouseDown={preventDefault}` so clicking a suggestion doesn't blur-close the list before the click registers.
- `tsc -b` (build) and `tsc -b --noEmit` (lint) both pass clean.

## Review Triage Log

- medium — Editing a transaction pre-filled `categoryId` without marking it "touched", so picking an autocomplete suggestion while editing could silently overwrite the transaction's real category. Confirmed by reading the reset effect and `selectSuggestion`. Fixed: `categoryTouched` is now seeded to `true` when opening in edit mode.
- low — Suggestion dropdown had no `onBlur` handler, so tabbing away (not clicking) left it rendered. Confirmed: only dismiss path was the outside-click `mousedown` listener. Fixed: added `onBlur` to close it; suggestion buttons already use `onMouseDown` preventDefault so mouse selection still works.
- medium — No ARIA combobox semantics, so screen-reader users got no indication of the suggestion list or which item was highlighted. Confirmed by inspecting the markup. Fixed: added `role="combobox"`/`aria-expanded`/`aria-controls`/`aria-activedescendant` on the input and `role="listbox"`/`role="option"`/`aria-selected` on the list.
- low, rejected — Dropdown can clip inside the modal's `overflow-y-auto` scroll container on short viewports. Real, but fixing it needs portal-based positioning, not a simple correction; deferred.
- low, rejected — No frontend test coverage for the new interaction logic. Confirmed the frontend has no test framework or test files anywhere in the project, so this is a pre-existing convention gap, not something this one change should introduce; deferred.
- maybe-false, rejected as scope — "Autocomplete should also recall amount." Not a defect: the frozen Intent explicitly scopes the fill to note + category only. Logged as a future idea, not fixed here.
- false — "No 'no matches' feedback state." The frozen Intent explicitly decided against an empty-state hint; this is the intended behavior, not a gap.
- low, rejected — Suggestion list does a linear scan of all transactions per keystroke with no memoized index. Not reachable as a real problem at single-user personal-finance data volumes; fixing would add an indexing structure disproportionate to the benefit. Deferred as a future idea if data volume ever grows.


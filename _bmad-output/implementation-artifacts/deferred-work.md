- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: BudgetRepo.Upsert can return a Budget with an ID that doesn't exist in the DB when the post-write canonical re-SELECT fails for a reason other than "not found," and the error is swallowed instead of surfaced.
  evidence: Confirmed pre-existing via `git show` at the story's baseline commit — the pre-refactor `UpsertBudget` handler had byte-identical fallback logic (`return Budget{ID: id, ...}, nil` on any re-SELECT error). Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: Category-ownership check and the subsequent transaction/budget write are two non-atomic steps; a category deleted in between turns a would-be 400 "category not found" into a raw FK-violation 500.
  evidence: Confirmed pre-existing — the pre-refactor `ownsCategory`-then-`Exec` pattern in `transactions.go`/`budgets.go` was equally non-atomic (no transaction wrapped the check + write). Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-repo-refactor-verification.md`
  summary: RegisterUser's EmailExists check runs outside any transaction before the insert, so two concurrent registrations with the same email can both pass it, and the loser's insert fails with a raw duplicate-key DB error mapped to a generic 500 instead of a 409.
  evidence: Confirmed pre-existing via `git show` at the story's baseline commit — the pre-refactor handler had the identical check-then-insert shape with no transaction and no duplicate-key error mapping. Not introduced by the Story 1.1 refactor.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-declare-income.md`
  summary: IncomeRepo.Upsert's INSERT-then-re-SELECT is non-atomic; a concurrent upsert to the same (user_id, cycle_start_date) row between the two steps can make the response reflect another request's amount instead of the caller's own write.
  evidence: Same non-atomic two-step upsert shape already deferred for BudgetRepo.Upsert in Story 1.1 (see above); Story 1.2 deliberately mirrors that pattern onto the new incomes table per its own spec's explicit direction. Low real-world likelihood for a single-user personal app, but worth a shared fix (e.g. wrap Upsert in a transaction, or use `SELECT ... FOR UPDATE`) if either repo is revisited.

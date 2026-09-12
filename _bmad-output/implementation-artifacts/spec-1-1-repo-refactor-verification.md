---
title: 'Story 1.1: Hoàn thiện & xác thực refactor repository layer hiện có'
type: 'refactor'
created: '2026-09-12'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
context: []
baseline_commit: '447509701e475b97dccd23f97e2158338d95a9af'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `internal/api` was mid-refactor (uncommitted) from inline SQL-in-handlers to a `XxxRepo{db dbtx}` layer. Investigation confirms the code-move itself is already complete and builds clean — but nothing verifies the four existing surfaces (auth, categories, transactions, budgets) still behave identically to pre-refactor. No test exists at all for `internal/api` today.

**Approach:** Add automated tests that pin current handler+repo behavior using `go-sqlmock` (no live DB/Docker required, matching this environment's constraints) — repo-level tests per aggregate, plus `httptest`-driven handler tests covering the request/response contract (status codes, envelope shape, auth/ownership checks). Fix the two pre-existing `gofmt` violations noticed in the untouched `ratelimit.go`/`router.go` only if trivial; do not otherwise touch their logic.

## Boundaries & Constraints

**Always:** Keep the `XxxRepo{db dbtx}` / `NewXxxRepo` / `ok()`/`fail()` / `withTx` conventions exactly as they exist today — this story verifies, it does not redesign. Every new test must fail if handler behavior silently diverges from what's on disk right now (status code, envelope `code`/`message`, response `data` shape, DB scoping by `user_id`). Use `github.com/DATA-DOG/go-sqlmock` (`go get` it as a test-only dependency) for repo/handler tests — its mock `*sql.DB` satisfies `dbtx` directly.

**Never:** Do not add or modify any production repo/handler logic to make tests pass — if a test reveals a real behavior gap, stop and report it in Implementation Notes rather than silently patching scope. Do not touch the dead "learning scaffold" tree. Do not attempt real MySQL/testcontainers integration tests — Docker is unavailable in this environment (confirmed: daemon unreachable).

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Register + duplicate email | `POST /auth/register` twice, same email | 2nd call: `fail` 409-class envelope, no 2nd row inserted | Assert `EmailExists` short-circuits `Create` |
| Login wrong password | Valid email, wrong password | `fail` with generic auth error, no user data leaked in response | bcrypt compare failure path |
| Category ownership on transaction create | User A's JWT, User B's `category_id` | `fail` (ownership check via `CategoryRepo.Owns`), no row inserted | `Owns` returns false → handler rejects before `TransactionRepo.Create` |
| Budget upsert idempotency | `PUT` same category+month twice with different amounts | 2nd call updates in place (ON DUPLICATE KEY), `List` shows one row with latest amount | Re-SELECT-after-write path in `Upsert` |
| Transaction date round-trip | Create with `"2026-09-12"`, then `List` | Returned date string is exactly `"2026-09-12"` (no TZ drift) | Uses shared `dateLayout` const |

</frozen-after-approval>

## Code Map

- `internal/api/repo_user.go`, `repo_category.go`, `repo_transaction.go`, `repo_budget.go`, `repo_helpers.go` -- refactor is complete here; write `*_test.go` siblings for each, mocking `dbtx` via `sqlmock.New()`.
- `internal/api/auth.go` -- handlers `RegisterUser`, `Login`, `Me`, `Demo`; `withTx` usage at ~L117-122 (register) and ~L189-197 (demo seed). Test via `httptest` + `gin.CreateTestContext`, sqlmock backing `Handler.db`.
- `internal/api/categories.go`, `transactions.go`, `budgets.go` -- thin handlers delegating to repos + `Owns` checks; test the delegation/status-code contract, not repo internals twice.
- `internal/api/api.go` -- `dbtx`, `Handler`, `NewHandler`, `withTx`, `ok`/`fail` envelope helpers; reference only, no changes expected.
- `internal/api/router.go`, `ratelimit.go` -- pre-existing `gofmt` drift, unrelated to refactor; fix formatting only if a one-line `gofmt -w` is safe (no logic diff).
- `go.mod`/`go.sum` -- add `github.com/DATA-DOG/go-sqlmock` (test-only; verify it lands under a `require` block, no impact on the production binary).

## Tasks & Acceptance

**Execution:**
- [x] `go.mod` -- run `go get github.com/DATA-DOG/go-sqlmock@latest` -- adds test-only mock DB dependency
- [x] `internal/api/repo_user_test.go` -- cover `EmailExists`, `Create`, `FindByEmail` (found/not-found), `FindByID` -- pins existing SQL/error-mapping behavior
- [x] `internal/api/repo_category_test.go` -- cover `List`, `Create`, `Delete`, `Owns` (true/false), `ByName`, `SeedDefaults` -- pins ownership + seeding behavior
- [x] `internal/api/repo_transaction_test.go` -- cover `List`, `Create`, `Update` (found/not-found), `Delete`, date round-trip via `dateLayout` -- pins CRUD + date handling
- [x] `internal/api/repo_budget_test.go` -- cover `List`, `Upsert` (insert + update-in-place), `Delete` -- pins ON-DUPLICATE-KEY upsert behavior
- [x] `internal/api/auth_test.go` -- cover register success, duplicate-email rejection, login success/failure, `Me` with valid/invalid JWT, `Demo` seeding -- pins the `withTx` orchestration paths
- [x] `internal/api/categories_test.go`, `transactions_test.go`, `budgets_test.go` -- cover one happy path + one ownership/validation failure path per handler -- pins request/response contract
- [x] repo root -- run `go build ./...`, `go vet ./...`, `go test ./internal/api/...` -- confirm zero regressions and all new tests pass

**Acceptance Criteria:**
- Given the refactored `internal/api` code as it exists now, when the new test suite runs, then all tests pass without any change to production repo/handler logic.
- Given a category owned by another user, when a transaction or budget references it, then the handler rejects with the same ownership-failure envelope as today.
- Given `go vet ./...` and `go build ./...`, then both succeed with zero new warnings introduced by test files.

## Implementation Notes

- Added `github.com/DATA-DOG/go-sqlmock v1.5.2` as a direct (test-only) dependency via `go get` + `go mod tidy`; it is imported only from `_test.go` files in `internal/api`, so it has no effect on the production binary. `go mod tidy` also confirmed no other module drift existed.
- Fixed the two pre-existing `gofmt` violations in `internal/api/ratelimit.go` and `internal/api/router.go`. Root cause: both files had CRLF line endings on disk while every other file in the package uses LF; `gofmt -w` rewrote them with LF endings and produced a **byte-for-byte identical diff to HEAD** (`git diff` shows 0 bytes changed) — confirming this was pure whitespace/line-ending drift with zero logic change, safe per the spec's "only if trivial" constraint.
- New test files added (all `internal/api`): `repo_user_test.go`, `repo_category_test.go`, `repo_transaction_test.go`, `repo_budget_test.go`, `auth_test.go`, `categories_test.go`, `transactions_test.go`, `budgets_test.go`, plus a shared `handler_test_helpers_test.go` (sqlmock-backed `Handler` + `httptest`/gin context builders, kept out of the production build via the `_test.go` suffix).
- All five I/O & Edge-Case Matrix scenarios are covered explicitly and pass: register+duplicate-email (`TestRegisterUser_DuplicateEmail`), login wrong password with no data leakage (`TestLogin_WrongPassword`), category-ownership rejection on transaction create (`TestCreateTransaction_OwnershipFailure`), budget-upsert idempotency/update-in-place (`TestBudgetRepo_Upsert_UpdateInPlace`, `TestUpsertBudget_Idempotency`), and the transaction date round-trip via `dateLayout` with no TZ drift (`TestTransactionRepo_List_DateRoundTrip`).
- No production repo/handler logic was changed. Investigation found no real behavior gaps — every handler already had the ownership/validation/error-mapping behavior the spec asked to pin.
- `go test ./...` (whole repo, not just `internal/api`) has two pre-existing, out-of-scope failures untouched by this story: `internal/database` (needs Docker/testcontainers, confirmed unavailable in this environment) and `internal/pkg/tests/basic` (the intentionally-broken "learning scaffold" tree the spec says not to touch). Both were failing before this story's changes and are unrelated to `internal/api`.

## Spec Change Log

## Review Triage Log

- **medium / patch** — `CategoryRepo.Owns` (repo_category.go:50-59) now genuinely propagates non-`ErrNoRows` DB errors (new behavior vs. the pre-refactor `ownsCategory`, which always folded any error into `false`), and this new error branch — plus its three call sites' 500 responses (budgets.go `50034`, transactions.go Create `50025`/Update `50026`) — has zero test coverage. Verified: confirmed via `git show` at baseline that old `ownsCategory` returned `bool` only and could never surface a DB error; new `Owns` can. [blind-hunter, edge-case-hunter x0, verification-gap]
- **medium / patch** — `withTx`'s rollback path (api.go:51-67), used by `RegisterUser`/`Demo` for atomic signup/seeding, has no test that injects a mid-transaction failure and asserts `tx.Rollback()` fires and no token/user is returned. Verified: `TestRegisterUser_Success`/`TestDemo_Seeding` only mock the all-succeed path; a broken rollback would not fail either test. [blind-hunter, verification-gap]
- **medium / patch** — `internal/api/config.go` (`LoadConfig`), a new file in this diff, has no test at all — including the weak/default `JWT_SECRET` fallback-and-warning branch, which is security-relevant. Verified: file exists, no `config_test.go` present. [blind-hunter]
- **low / patch** — `TestDemo_Seeding` (auth_test.go) hardcodes `sampleTxCount = 13`/`sampleBudgetCount = 5` instead of deriving them from the actual sample-data slice lengths, so an unrelated future edit to the demo dataset breaks this test with an opaque sqlmock error. Verified: literals confirmed in test file. Fix is a direct substitution (use `len(...)`), not new complexity, so kept despite low severity. [blind-hunter]
- **defer** — `BudgetRepo.Upsert` (repo_budget.go:49-58): when the post-write canonical re-SELECT fails for a reason other than "not found," the error is swallowed and a `Budget` with a possibly-nonexistent ID is returned. Verified pre-existing: `git show` at `baseline_commit` shows the pre-refactor `UpsertBudget` handler had byte-identical fallback logic. Not caused by this story. [blind-hunter, edge-case-hunter]
- **defer** — Category-ownership check and the subsequent transaction/budget write are two non-atomic steps; a category deleted in between turns a would-be 400 "category not found" into a raw FK-violation 500. Verified pre-existing: the pre-refactor `ownsCategory`-then-`Exec` pattern was equally non-atomic. Not caused by this story. [edge-case-hunter]
- **defer** — `RegisterUser`'s `EmailExists` check runs outside any transaction before insert; two concurrent registrations with the same email can both pass it, and the loser's insert fails with a raw duplicate-key error mapped to a generic 500 instead of 409. Verified pre-existing: `git show` at baseline shows the identical check-then-insert shape with no transaction. Not caused by this story. [blind-hunter, edge-case-hunter]
- **low / reject** — `withTx` discards the error from `tx.Rollback()` in both the error and panic-recovery paths, so a rollback failure is silently lost. Verified: real gap in newly-introduced code, but unlikely to be hit in practice (requires rollback itself to fail) and the fix adds a new branch (error-check + logging), which fails the reject bar's "more than a direct correction" test. [blind-hunter, edge-case-hunter]

## Verification

**Commands:**
- `go build ./...` -- expected: succeeds, no errors
- `go vet ./...` -- expected: succeeds, no warnings
- `go test ./internal/api/... -v` -- expected: all tests pass, including the new repo/handler suites
- `gofmt -l internal/api cmd/web` -- expected: no output (or only pre-existing `router.go`/`ratelimit.go` if left untouched by choice)

# Lottery Admin Usability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a fully translated and understandable lottery administration workflow with direct group/user selection and MySQL-safe plan creation.

**Architecture:** Keep the existing lottery API contract and split frontend responsibilities into form conversion helpers, a reusable user multi-select, and the page component. Reuse existing group and user APIs. Fix database compatibility at the model default and schema definition.

**Tech Stack:** Go, GORM, Gin, React 19, TypeScript, React Hook Form, Zod, TanStack Query, Base UI/shadcn components, i18next, Bun.

---

### Task 1: Protect the lottery algorithm database contract

**Files:**
- Modify: `model/lottery.go`
- Test: `model/lottery_test.go`

- [ ] Add a regression test that creates a plan, reloads it, and verifies the default algorithm identifier is non-empty and no longer than the legacy 32-character MySQL column.
- [ ] Run `go test ./model -run TestLotteryPlanDefaultAlgorithmFitsLegacyColumn -count=1` and confirm it fails with the current 35-character identifier.
- [ ] Define the stable algorithm identifier in `model/lottery.go`, use it in `BeforeCreate`, and widen the GORM column tag to `varchar(64)`.
- [ ] Re-run the targeted test and existing lottery model tests.

### Task 2: Isolate lottery form payload conversion

**Files:**
- Create: `web/default/src/features/lottery/lib/admin-form.ts`
- Create: `web/default/src/features/lottery/lib/admin-form.test.ts`
- Modify: `web/default/src/features/lottery/types.ts`

- [ ] Write table tests for group selection, user selection, quota prizes, subscription prizes, and claim-expiry day conversion.
- [ ] Run `bun test src/features/lottery/lib/admin-form.test.ts` and confirm the helper is missing.
- [ ] Add typed form values, defaults, date conversion, and `buildLotteryPlanPayload`.
- [ ] Re-run the test and confirm exact payloads.

### Task 3: Add direct group and user selectors

**Files:**
- Create: `web/default/src/features/lottery/components/lottery-user-multi-select.tsx`
- Modify: `web/default/src/features/lottery/admin.tsx`
- Modify: `web/default/src/features/lottery/api.ts`

- [ ] Add query wrappers for existing group and user-search endpoints.
- [ ] Build a searchable user multi-select that keeps selected users visible and displays display name, username, ID, and group.
- [ ] Use the existing `MultiSelect` for groups and conditionally render the correct selector for the chosen eligibility mode.
- [ ] Ensure switching eligibility mode clears selections that no longer apply.

### Task 4: Rebuild the lottery creation form for readability

**Files:**
- Modify: `web/default/src/features/lottery/admin.tsx`

- [ ] Replace the unlabeled grid with `FieldGroup`, `Field`, `FieldLabel`, `FieldDescription`, and `FieldError`.
- [ ] Split the form into basic information, participation eligibility, and prize settings sections separated by headings and separators.
- [ ] Show only reward-relevant controls and replace raw claim seconds with claim-expiry days.
- [ ] Translate plan statuses and add schedule, capacity, and eligibility summaries to the plan list.
- [ ] Keep published-plan editing and participant weight/preset management behavior unchanged.

### Task 5: Complete translations

**Files:**
- Temporary: `web/default/scripts/add-missing-keys.mjs`
- Modify: `web/default/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json`

- [ ] Extract every `t('...')` key used by lottery and extended redemption-code screens.
- [ ] Add reviewed values for all supported locales through `add-missing-keys.mjs`.
- [ ] Run the script and `bun run i18n:sync`, then remove the temporary script.
- [ ] Verify no changed-screen translation key is missing from any locale.

### Task 6: Verify and publish

**Files:**
- Verify all changed backend and frontend files.

- [ ] Run `go test ./model ./controller`.
- [ ] Run frontend unit tests, `bun run typecheck`, targeted oxlint/oxfmt, and `bun run build`.
- [ ] Inspect `git diff --check` and the final diff for unrelated changes.
- [ ] Commit to `main`, push `origin/main`, and confirm the GitHub image workflow succeeds.

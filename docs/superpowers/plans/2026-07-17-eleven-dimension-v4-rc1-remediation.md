# Eleven Dimension V4-RC1 Remediation Implementation Plan

> **For agentic workers:** Execute inline in the current workspace. The project explicitly forbids build and test commands, and no Git commit is authorized.

**Goal:** Close the reviewed V4-RC1 rule-priority, completeness-statistics, and operation-snapshot gaps before the rebuild command is allowed to run.

**Architecture:** Keep workbook parsing and rebuilding in the existing command. Correct route precedence in the generated JavaScript, then reuse one CRM Service helper for P01-P11 field recognition, completeness statistics, and rule snapshots. Preserve the existing rule execution and dynamic-form contracts.

**Tech Stack:** Go, Dever ORM/Service, JavaScript RuleScript, React/TypeScript

---

### Task 1: Correct generated route precedence

**Files:**
- Modify: `cmd/rebuild-eleven-dimension/rule_script.go`

- [x] Replace numeric route ranks with the workbook precedence `R4 > R3 > R0 > R2 > R1`.
- [x] Route option tags, combination rules, and T-node defaults through the same `applyRoute` function.
- [x] Remove the separate `requiresR0` branch so R0 can override R2 but not R3/R4.

### Task 2: Centralize P01-P11 recognition

**Files:**
- Create: `package/crm/service/eleven_dimension.go`
- Modify: `package/crm/service/work.go`
- Modify: `package/crm/service/admin_summary.go`

- [x] Recognize both new flat `P01`-`P11` fields and the current grouped probe fields during the transition.
- [x] Count only the 11 probe selections in workbench completeness and admin statistics.
- [x] Exclude the seven shared fields and P12 from completeness totals and dimension rows.

### Task 3: Complete operation-backed snapshots

**Files:**
- Modify: `package/crm/service/eleven_dimension.go`
- Modify: `package/crm/service/work_todo_execute.go`

- [x] Capture the exact P01-P11 input map in the automatic-rule operation snapshot.
- [x] Generate `JUDGMENT-<operation_id>` after the operation row exists.
- [x] Add the generated number to eleven-dimension output fields before persisting the diagnosis record.

### Task 4: Align the existing admin UI copy

**Files:**
- Modify: `package/crm/front/src/nodes/show/admin-stats.tsx`

- [x] Change P01-P12 copy and display limits to P01-P11.
- [x] Do not change layout or introduce new UI behavior.

### Task 5: Static verification only

**Files:**
- Review: all files changed above

- [x] Run `gofmt` on changed Go files.
- [x] Run the Dever static audit and `git diff --check`.
- [x] Confirm the rebuild command remains unexecuted and the database remains unchanged.
- [x] Do not run `go test`, `go run`, `npm run build`, `dever build`, or any migration command.

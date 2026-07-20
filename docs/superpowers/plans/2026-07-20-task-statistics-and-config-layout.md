# Task Statistics And Config Layout Implementation Plan

> Statistic configuration is superseded by `2026-07-20-task-capability-simplification.md`. Statistics are enabled on reusable data fields, not selected separately per task.

> **For agentic workers:** Execute inline in the current workspace. The project explicitly forbids build/test commands and Git commits; use static verification only.

**Goal:** Keep task configuration compact while replacing the retired system-purpose statistics path with a task-scoped, visible statistics workflow.

**Architecture:** Reuse the existing task-through-relation pattern and `StatFieldValue` snapshot table. Page JSON owns the task form and embedded configuration modals; one small CRM node renders reusable capability summary rows. Runtime synchronization and dashboard aggregation remain focused service functions.

**Tech Stack:** Dever Model/Page JSON/Provider, Go services, PostgreSQL migration, React/TypeScript CRM front plugin.

---

### Task 1: Task statistic relation and configuration options

**Files:**
- Create: `model/task_stat_field.go`
- Create: `service/setting/task_statistics.go`
- Modify: `model/task.go`
- Modify: `service/setting/options.go`
- Modify: `service/setting/workflow.go`

- [x] Add a unique `task_id + data_field_id` through model and expose `stat_field_ids` from `Task`.
- [x] Build one reusable helper that expands enabled group fields from the selected task form.
- [x] Return those fields from `OptionService.LoadTaskStatFieldOptions` using `parentId=form.form_id`.
- [x] Validate submitted statistic fields against the effective task form and clear them for task types without a form.

### Task 2: Advanced configuration UI

**Files:**
- Modify: `front/src/plugin.ts`
- Modify: `front/page/admin/task/update.json`

- [x] Add “流程控制”和“业务能力” section headings below the existing advanced switch.
- [x] Use direct finance and statistic multi-selects; do not add a second configuration dialog.
- [x] Clear `stat_field_ids` whenever task type or task form changes.

### Task 3: Statistic snapshot synchronization

**Files:**
- Create: `service/work_statistics.go`
- Modify: `service/work_audit.go`

- [x] Load task statistic relations once per saved record and ignore fields not selected for the task.
- [x] Infer statistic type from the data field type and reuse existing display/option normalization.
- [x] Insert or update the existing owner-and-field snapshot with task and operation provenance.
- [x] Synchronize only values present in the submitted record so hidden or omitted fields do not overwrite history.

### Task 4: Business dashboard consumption

**Files:**
- Modify: `service/admin_summary.go`
- Modify: `front/src/nodes/show/admin-stats.tsx`

- [x] Aggregate configured statistic snapshots by workflow and selected date range.
- [x] Return counts plus numeric totals/average/min/max, time min/max, or top value distributions.
- [x] Render one scan-friendly field-statistics table in the business dashboard.
- [x] Replace stale finance copy that still refers to financial-purpose fields.

### Task 5: Migration and static verification

**Files:**
- Create: `migrations/postgres/027_task_statistics.sql`

- [x] Create the relation table and indexes idempotently.
- [x] Convert legacy statistic-purpose bindings that still map to enabled task form fields, including rows retired by migration 024.
- [x] Run `gofmt` on changed Go files.
- [x] Parse changed JSON with `jq empty` and inspect SQL for idempotent guards.
- [x] Run the Dever static audit and `git diff --check`.
- [x] Do not run `npm run build`, `go test`, or any equivalent build/test command.

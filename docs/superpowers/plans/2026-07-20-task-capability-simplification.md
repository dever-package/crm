# Data Field Capability Implementation Plan

**Goal:** Move finance and statistics configuration from workflow tasks to reusable data fields while preserving all historical records.

**Architecture:** `DataField` is the single configuration source for finance and statistics. Task configuration retains only workflow behavior. Form submission resolves the submitted data fields once and dispatches finance/statistic side effects from their metadata.

**Constraints:** Reuse existing Dever model/page/service patterns. Do not run build or tests. Do not delete historical tables or records.

### Task 1: Add data-field capabilities

**Files:**
- Modify: `model/data_field.go`
- Modify: `service/setting/data_field.go`
- Modify: `service/setting/data_template.go`
- Modify: `front/page/admin/data_field/update.json`
- Modify: `front/page/admin/data_field/child_update.json`
- Modify: `front/page/admin/data_template/field_list.json`
- Add: `migrations/postgres/028_data_field_capabilities.sql`

- [x] Add `finance_type_id` and `stat_enabled` to data fields.
- [x] Expose and validate both properties in top-level and child-field editors.
- [x] Backfill field metadata from preserved ledger, snapshot, usage, and task-relation data.

### Task 2: Remove task-level duplication

**Files:**
- Modify: `model/task.go`
- Modify: `service/setting/workflow.go`
- Modify: `service/setting/options.go`
- Modify: `front/page/admin/task/update.json`
- Retain for audit only: `model/task_finance_type.go`
- Retain for audit only: `model/task_stat_field.go`
- Delete: `service/setting/task_statistics.go`

- [x] Remove finance/statistic fields and relations from task editing.
- [x] Remove task-specific validation and option loaders.
- [x] Keep the underlying historical tables through audit-only models and migrations.

### Task 3: Use field metadata at runtime

**Files:**
- Modify: `service/work_operation_side_effect.go`
- Modify: `service/work_statistics.go`
- Modify: `service/work_form_input.go`
- Modify: `service/work_todo_execute.go`
- Modify: `service/work_form_changes.go`
- Modify: `service/work.go`
- Modify: `service/admin_summary_fields.go`

- [x] Stop generating finance controls from task relations.
- [x] Resolve submitted data fields once and reuse them for finance/statistic side effects.
- [x] Write finance ledgers with the actual field ID and `field:<id>` source key.
- [x] Read statistic fields directly from enabled field metadata in the dashboard.

### Task 4: Export and static review

**Files:**
- Modify: `service/setting/field_export_columns.go`
- Modify: `service/setting/field_export.go`

- [x] Export finance type and statistic participation with data-field definitions.
- [x] Format edited Go files and parse edited JSON files.
- [x] Check SQL and stale task-level references without running build or tests.

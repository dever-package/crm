# CRM Communication Groups Implementation Plan

> **For agentic workers:** Execute inline in the current task. Project instructions prohibit sub-agent dispatch, commits, build and automated tests for this work.

**Goal:** Add configurable communication-group types, case-owned communication-group records, related staff, workbench management, admin visibility, and a Feishu history import target.

**Architecture:** Use ordinary Dever Models and Page JSON for configuration and global lists. Put cross-table invariants, permissions, staff synchronization and operation logging in focused CRM services. Reuse one generic department-first staff picker in both schedules and communication groups.

**Tech Stack:** Go, Dever ORM/Page JSON, PostgreSQL migrations, React/TypeScript, existing CRM front plugin.

---

### Task 1: Domain schema and models

**Files:**
- Create: `model/communication_group_type.go`
- Create: `model/communication_group.go`
- Create: `model/communication_group_staff.go`
- Modify: `model/options.go`
- Create: `migrations/postgres/036_communication_group.sql`

- [x] Define enabled/disabled communication-group types with a stable unique code.
- [x] Define case-owned communication groups with active/dissolved status, business dates and a nullable import source key.
- [x] Define the unique group/staff relationship and preserve a relation role.
- [x] Seed the `enterprise_wechat` type and add a partial unique index allowing only one active group per workflow instance.

### Task 2: Domain and workbench services

**Files:**
- Create: `service/communication_group.go`
- Create: `service/work_communication_group.go`
- Modify: `service/work_schedule_query.go`
- Modify: `service/work.go`
- Modify: `service/work_operation_business.go`
- Modify: `api/work.go`

- [x] Centralize enabled department/staff option construction and reuse it from schedules.
- [x] Validate workflow ownership, enabled type, enabled related staff and single-active-group invariant.
- [x] Save group and staff relationships transactionally while retaining historical source roles for unchanged staff.
- [x] Dissolve an active group transactionally and record the operation.
- [x] Return communication groups and maintenance permission from customer detail.
- [x] Expose thin work APIs for people options, save and dissolve actions.

### Task 3: Shared staff picker and workbench UI

**Files:**
- Create: `front/src/nodes/show/work-people-types.ts`
- Create: `front/src/nodes/show/work-people-picker.tsx`
- Modify: `front/src/nodes/show/work-schedule-types.ts`
- Modify: `front/src/nodes/show/work-schedule-participant-picker.tsx`
- Delete: `front/src/nodes/show/work-schedule-participant-dialog.tsx`
- Create: `front/src/nodes/show/work-communication-groups.tsx`
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-customer-detail.tsx`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [x] Extract the schedule department-first multi-select dialog into a domain-neutral staff picker.
- [x] Keep schedule-specific locked people and badges in the schedule wrapper.
- [x] Add communication-group types, records and people-option contracts.
- [x] Add a group tab only to customer/asset details, not lead details.
- [x] Build create/edit and dissolve dialogs, refresh through the existing `crm-work-refresh` event, and keep read-only users viewable.

### Task 4: Admin pages and validation

**Files:**
- Create: `service/setting/communication_group.go`
- Create: `front/page/admin/communication_group/list.json`
- Create: `front/page/admin/communication_group_type/list.json`
- Create: `front/page/admin/communication_group_type/update.json`

- [x] Add a global communication-group list under customer management.
- [x] Add group-type maintenance as a child modal, seeded with enterprise WeChat.
- [x] Keep the global group list read-only; all group facts are edited from the workbench so audit and staff synchronization cannot be bypassed.
- [x] Validate type code/name and prevent disabled types from being selected for new workbench records.

### Task 5: Feishu import mapping

**Files:**
- Modify: `.trellis/tasks/07-22-crm-feishu-history-import/prd.md`
- Modify: `.trellis/tasks/07-22-crm-feishu-history-import/design.md`
- Modify: `.trellis/tasks/07-22-crm-feishu-history-import/implement.md`
- Modify: `.trellis/tasks/07-22-crm-feishu-history-import/research/field-mapping.md`

- [x] Replace the temporary historical-template destination with the communication-group model.
- [x] Map group facts and role-aware staff relations explicitly.
- [x] Keep stage/signing fields as reconciliation-only evidence.
- [x] Reserve migration number 037 for history import audit after the communication-group migration.

### Task 6: Static verification

- [x] Run `gofmt` on changed Go files.
- [x] Parse changed JSON page files.
- [x] Run Dever static audit on changed CRM paths.
- [x] Run `git diff --check` and inspect all new references.
- [x] Do not run build or automated tests; hand off browser verification to the user.

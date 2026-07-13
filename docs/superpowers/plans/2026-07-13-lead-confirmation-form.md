# Lead Confirmation Form Implementation Plan

> **For agentic workers:** Execute inline in the current CRM package. Do not dispatch subagents. Do not run build or test commands because the project instructions explicitly prohibit them.

**Goal:** Add a configurable default lead-confirmation task form and bind it to the existing default lead workflow task.

**Architecture:** Reuse the existing `crm_form` / `crm_form_field` models and work task-form runtime. Add one idempotent PostgreSQL migration; no new Service, API, model, or React component is needed.

**Tech Stack:** Dever CRM, Go model metadata, PostgreSQL migrations, React work frontend already present.

---

### Task 1: Create the default lead confirmation form

**Files:**
- Create: `migrations/postgres/015_lead_confirmation_form.sql`

- [ ] Insert or update the enabled `线索确认` form.
- [ ] Rebuild its default fields from the eight lead main fields.
- [ ] Append enabled fields from the `线索补充信息` data template.
- [ ] Keep only `姓名` required; rely on the existing cross-field validation for phone or WeChat.

### Task 2: Bind the default lead workflow task

**Files:**
- Modify: `migrations/postgres/015_lead_confirmation_form.sql`

- [ ] Locate the default enabled lead workflow and its first enabled stage.
- [ ] Reuse the existing `确认线索` task when present.
- [ ] Change the task to `form`, bind the new form, and retain stage-based assignment.
- [ ] Insert the task only when the default stage does not already contain it.

### Task 3: Static review and migration application

**Files:**
- Review: `service/setting/workflow.go`
- Review: `service/work_form_input.go`
- Review: `front/src/nodes/show/work-lead.tsx`

- [ ] Confirm backend validation rejects customer/asset fields on a lead workflow task.
- [ ] Confirm completing the form does not expose terminal stage completion in the lead dialog.
- [ ] Confirm `转客户` becomes available only after required non-todo tasks are complete.
- [ ] Run `git diff --check` and the Dever static audit only; do not run build or tests.
- [ ] Execute migration 015 explicitly through the existing PostgreSQL container; this project does not auto-run `migrations/postgres`.
- [ ] Restart the existing Dever development environment so the latest service code is active.

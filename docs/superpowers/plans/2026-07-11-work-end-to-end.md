# CRM Work End-to-End Implementation Plan

> **For agentic workers:** Execute inline and verify each checkpoint in the real work site. Do not run build or test commands.

**Goal:** Make the CRM work site complete the existing customer -> asset -> P01-P12 -> automatic decision path.

**Architecture:** Add the missing page-level task modal, centralize backend form-field resolution, and render resolved groups with the existing dynamic form controls. Keep task execution and stage transition services unchanged unless browser evidence exposes a separate defect.

**Tech Stack:** Dever page JSON, Go services/models, React/TypeScript plugin, Playwright Chromium, PostgreSQL.

---

### Task 1: Work Entry and Task Modal

**Files:**
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/page/work/work.json`

- [ ] Change `getWorkEntryPath()` to return the configured CRM work route under the current site root.
- [ ] Add a `feedback-shell` layout slot, a reusable `feedback-modal`, an initially empty `work-task-form-section`, task action-target data, and `state.dialog.workTask`.
- [ ] Reload the live site and verify login lands on the work table and the create-customer task opens a form.

### Task 2: Shared Group Field Resolution

**Files:**
- Create: `service/work_form_fields.go`
- Modify: `service/work_form_input.go`
- Modify: `service/work.go`

- [ ] Implement one resolver that loads enabled form fields and expands a `group` marker into its ordered child data fields.
- [ ] Use the resolver when collecting submitted values so rendered and accepted field sets cannot diverge.
- [ ] Use the resolver when building work task responses, including group names and child options.
- [ ] Inspect `/crm/work/tasks` in the browser and confirm P01-P12 contains real select/textarea/upload children.

### Task 3: Group Rendering

**Files:**
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] Add group metadata to the work form field contract.
- [ ] Build section-heading nodes when the group changes and render each child using the existing field renderer.
- [ ] Keep submit field mappings pointed at the child data field IDs.
- [ ] Verify all twelve probe sections render without overflow on a 1440px viewport.

### Task 4: Browser Workflow Walkthrough

**Files:**
- Update only files implicated by reproduced failures.

- [ ] Log in as MKT and create a customer through the modal.
- [ ] Log in as NPL, complete first contact, create the first asset, and reach S04.
- [ ] Fill representative P01-P12 values and complete collection.
- [ ] Confirm the automatic decision task executes and the asset reaches the expected next stage.
- [ ] Confirm browser console/network has no relevant errors and capture final screenshots.

### Task 5: Cleanup and Static Audit

**Files:**
- Review all modified CRM files.

- [ ] Remove duplicated field-loading/rendering logic and keep names domain-specific.
- [ ] Run the Dever static audit against changed files.
- [ ] Record browser steps, resulting customer/asset IDs, decision result, and final stage.

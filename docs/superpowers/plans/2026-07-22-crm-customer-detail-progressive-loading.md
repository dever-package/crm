# CRM Customer Detail Progressive Loading Implementation Plan

> **For agentic workers:** Execute inline in the current CRM workspace. The project explicitly forbids build/test commands and Git commits, so verification is limited to static checks and source review.

**Goal:** Split the customer detail workspace into independently loaded timeline, detail, and attachment streams and reorganize the three-column UI.

**Architecture:** Extract one shared access/target resolver in the CRM work service. Keep the legacy aggregate endpoint and add focused profile, operation, and attachment endpoints. The operation endpoint reuses the common query path without assembling unrelated todos. The React detail hook owns three independent request states and the workspace remains a presentational component.

**Tech Stack:** Go, Dever Service/API, React, TypeScript, component-owned CSS.

---

### Task 1: Extract reusable detail service boundaries

**Files:**
- Create: `service/work_customer_detail.go`
- Modify: `service/work.go`

- [x] Move customer-detail target resolution and profile assembly out of the large work service file.
- [x] Centralize customer, asset, workflow-instance, and dispatcher-scope authorization.
- [x] Keep `CustomerDetail` as a compatibility aggregate that combines profile and operations.
- [x] Add `CustomerProfile` without operation loading.
- [x] Add `CustomerOperations` without todo loading.
- [x] Add `CustomerAttachments` with file deduplication and source metadata.

### Task 2: Expose thin work APIs

**Files:**
- Modify: `api/work.go`

- [x] Add `GetCustomerProfile` and pass only customer/asset/workflow identifiers.
- [x] Add `GetCustomerOperations` with the same shared access contract.
- [x] Add `GetCustomerAttachments` with the same identifier contract.
- [x] Keep `GetCustomerDetail` unchanged for compatibility.
- [x] Keep `/crm/work/operations` unchanged for existing callers.

### Task 3: Split frontend request state

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-customer-detail-workspace.tsx`

- [x] Define profile, operation, and attachment response types.
- [x] Replace the aggregate reload with independent profile, operation, and attachment requests.
- [x] Preserve prefetched profile data for schedule/search entry points.
- [x] Add independent loading, error, retry, and stale-response protection per column.
- [x] Remove client-side attachment derivation from complete sections and operations.

### Task 4: Reorganize and refine the workspace

**Files:**
- Modify: `front/src/nodes/show/work-customer-detail-workspace.tsx`
- Modify: `front/src/nodes/show/work-customer-detail.tsx`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [x] Move timeline to the left and attachments to the right.
- [x] Make responsive two-column layout place attachments on the second row.
- [x] Display `系统` when an operation has no operator.
- [x] Refine center view tabs, template tabs, section heading, field labels, and long-value wrapping.
- [x] Preserve attachment preview/download and operation detail opening.

### Task 5: Static verification

**Files:**
- Review all changed CRM files.

- [x] Run `gofmt` on manually changed Go sources.
- [x] Run the Dever static audit on changed Go and TSX files.
- [x] Run `git diff --check` and review the scoped diff for duplicate access/query paths.
- [x] Confirm the plugin dev server remains listening.
- [x] Do not run build, typecheck, unit tests, browser automation, or any equivalent test command.

### Task 6: Include every customer-linked business upload

**Files:**
- Modify: `service/work.go`
- Modify: `service/work_customer_detail.go`

- [x] Treat attachment, file, image, audio, and video data fields as upload fields through the existing file-payload projection.
- [x] Read `crm_attachment` rows belonging to the customer-level and selected-asset scopes, including meeting-arrival videos.
- [x] Read historical operation `file_ids` so replaced arrival videos remain discoverable.
- [x] Reuse the existing attachment collector so entity, operation, and independent attachment sources share one deduplication contract.
- [x] Preserve source and field labels for the existing right-side attachment cards.
- [x] Run Go formatting, Dever static audit, scoped diff review, and `git diff --check`; do not run tests or builds.

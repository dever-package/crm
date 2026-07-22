# CRM Customer Detail Workspace Implementation Plan

> **For agentic workers:** Execute inline in the current CRM workspace. The project explicitly forbids build/test commands and Git commits, so verification is limited to static review and the Dever audit script.

**Goal:** Replace the work customer detail drawer with a large three-column modal that reuses existing CRM attachments, detail fields, communication groups, and operation history.

**Architecture:** Keep `/crm/work/customer_detail` as the only data source. Add one focused presentational component in `work-customer-detail-workspace.tsx`, project existing detail fields and operation summaries into a deduplicated attachment gallery, and pass the existing operation timeline into the right column. Keep shared field rendering in `work-customer-detail.tsx` and data fetching/record-opening behavior in `work-auth.tsx`.

**Tech Stack:** React, TypeScript, Tailwind CSS, Dever front plugin/page JSON.

---

### Task 1: Switch the detail shell to a modal

**Files:**
- Modify: `front/page/work/work.json`
- Modify: `front/page/work/schedule.json`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] Replace the `feedback-drawer` definitions with `feedback-modal` definitions using `dialog.workDetail`.
- [ ] Use a wide modal body with no nested page scrolling.
- [ ] Update both detail-opening paths to open the dialog state and close any stale drawer state.

### Task 2: Build the reusable three-column workspace

**Files:**
- Modify: `front/src/nodes/show/work-customer-detail.tsx`
- Create: `front/src/nodes/show/work-customer-detail-workspace.tsx`
- Modify: `front/src/nodes/show/work-upload.tsx`

- [ ] Export the existing upload preview-kind and preview-URL projections for reuse.
- [ ] Add a deduplicated attachment projection covering dynamic detail fields and operation summary attachments.
- [ ] Render attachments in the left column with image thumbnails, file fallbacks, preview, and download actions.
- [ ] Extend the existing detail section renderer with a horizontal-tab navigation mode for the center column.
- [ ] Render customer/asset information and communication groups as center-column tabs.
- [ ] Accept the existing timeline as right-column content and keep every desktop column independently scrollable.
- [ ] Add responsive two-column and one-column fallbacks without changing business behavior.

### Task 3: Connect current detail data and remove duplicated containers

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] Retain the existing `flow` object returned by `customer_detail` in the detail hook.
- [ ] Replace the separate customer and asset detail wrappers with one shared wrapper.
- [ ] Pass the current customer, asset, flow, sections, operations, communication groups, and existing timeline into the workspace.
- [ ] Reset only view-local state when the selected customer or asset changes.

### Task 4: Static verification

**Files:**
- Review all changed CRM files.

- [ ] Inspect imports, JSX nesting, response types, and dialog state keys.
- [ ] Run the Dever audit script if available for the changed source/page files.
- [ ] Review `git diff --check` and the scoped diff.
- [ ] Do not run `npm run build`, typecheck, unit tests, browser automation, or other test commands; hand browser verification to the user.

### Task 5: Redesign the workspace to match the approved reference

**Files:**
- Modify: `front/page/work/work.json`
- Modify: `front/page/work/schedule.json`
- Modify: `front/src/nodes/show/work-customer-detail-workspace.tsx`
- Modify: `front/src/nodes/show/work-customer-detail.tsx`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] Disable the shared modal's default close footer and visually collapse its duplicate title block.
- [ ] Make the custom summary header a fixed compact grid with four one-row metrics.
- [ ] Change the desktop column tracks to approximately 25% / 43% / 32%.
- [ ] Use a three-column attachment gallery and keep preview/download behavior.
- [ ] Wrap detail categories and render field rows as bordered label/value tables.
- [ ] Add a `rail` timeline variant without changing the existing default card variant.
- [ ] Use the rail variant only inside the customer detail workspace.

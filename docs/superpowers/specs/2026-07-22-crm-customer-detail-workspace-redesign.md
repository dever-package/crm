# CRM Customer Detail Workspace Redesign

## Goal

Match the reference workspace's compact three-column composition while continuing to render only existing CRM customer, asset, attachment, communication-group, and operation data.

## Layout

- The CRM modal keeps only the top-right maximize and close actions from the shared feedback modal.
- A single 92px workspace header contains customer/asset identity on the left and four compact summary cells on the right.
- The content area uses approximately 25% / 43% / 32% tracks for attachments, detail data, and workflow history.
- Each desktop column scrolls independently and fills the remaining modal height.
- The modal has no bottom close footer; the top-right close action remains available.

## Left Column

- Keep the existing attachment projection from dynamic data fields and operation summaries.
- Present a compact section header and three-column thumbnail gallery.
- Preserve existing preview and download behavior.
- Show a restrained empty state when the current business data has no attachments.

## Center Column

- Use one compact "详细信息" toolbar with existing customer-data and communication-group views.
- Render dynamic template categories as wrapped compact tabs instead of a horizontally scrolling row.
- Render fields as a bordered four-cell row pattern: label, value, label, value.
- Keep all field values, completion counts, file links, and communication-group actions unchanged.

## Right Column

- Keep the current-stage summary at the top with status-specific tones.
- Add a workspace-only timeline variant with three tracks: date/time, rail/dot, operation content.
- Remove per-operation cards. Keep operator, action title, stage, badge, description, and click-to-open behavior.
- Preserve the existing card timeline as the default for lead detail and other consumers.

## Data And Behavior

- No backend or API changes.
- `/crm/work/customer_detail` remains the only detail data source.
- No insurance-specific fields, document generation, scoring, or tags are added.
- Empty and terminated states are rendered from existing CRM values rather than fabricated data.

## Verification

- Parse both page JSON files.
- Run Dever static audit and `git diff --check`.
- Do not run build, typecheck, automated tests, or browser automation because the project instructions prohibit test commands; browser acceptance remains manual.

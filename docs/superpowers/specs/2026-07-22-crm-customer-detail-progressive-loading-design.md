# CRM Customer Detail Progressive Loading Design

## Goal

Make the three-column customer detail workspace open quickly and load each business area independently while preserving the current permission model and detail behavior.

## Layout

- Left column: current-stage summary and operation timeline.
- Center column: customer, asset, dynamic template, and communication-group details.
- Right column: deduplicated attachments.
- Operation rows without an operator display `系统`.
- The center column uses compact view tabs, restrained template tabs, clear section headers, and bordered label/value rows.

## API Boundaries

- `/crm/work/customer_profile`: customer, selected asset, flow summary, detail sections, customer products, workflow instances, and communication groups.
- `/crm/work/customer_operations`: operation timeline rows using the shared detail access scope, without unrelated todo assembly.
- `/crm/work/customer_attachments`: deduplicated business uploads projected from lead/customer/asset data, operation snapshots, and independently stored CRM attachment records.
- `/crm/work/customer_detail` remains available as the legacy aggregate response so existing callers are not broken.

All three detail endpoints use the same customer/asset/workflow access resolver. API handlers remain thin and delegate assembly to `WorkService`. The legacy `/crm/work/operations` endpoint still serves its existing callers and keeps todo assembly where they require it.

## Loading Behavior

- Existing list data opens the modal immediately.
- Profile, timeline, and attachments load independently.
- Each column owns its loading and failure state; one failed request does not blank the other columns.
- Schedule and search entry points prefetch only the profile response and reuse it after opening instead of loading the full legacy aggregate twice.
- Refresh events reload all three streams.

## Attachment Projection

- Entity upload fields are read from configured dynamic data templates, including attachment, file, image, audio, and video fields.
- Operation upload fields are read from operation snapshots.
- Independently stored CRM attachments, such as meeting-arrival videos, are included from the customer-level scope and selected-asset scope; other assets remain excluded.
- The aggregation never scans unrelated global upload records; every file must be linked to the visible customer/asset through a business record.
- Current records and historical operation snapshots are both scanned, then files are deduplicated by upload ID with URL/name fallback.
- Each row keeps its source section and field label for the attachment panel.

## Compatibility And Verification

- No model or migration change.
- No permission expansion.
- No changes to generated files or compiled assets.
- Verification is limited to Go formatting, JSON parsing where applicable, Dever static audit, scoped diff review, and `git diff --check` because project instructions prohibit builds and tests.

# CRM Work End-to-End Design

## Goal

Make the existing work site usable from login through customer creation, asset data collection, P01-P12 data entry, automatic decision execution, and stage transition.

## Scope

- Keep the existing Customer, CustomerAsset, Form, Task, RuleScript, and Stage models.
- Fix the work-site entry URL and task modal wiring.
- Reuse the existing dynamic page form controls and task execution API.
- Expand legacy `group` data fields into their real child fields for display and submission.
- Verify the complete S01 -> S05 path in Chromium through the work UI.
- Do not add lead, opportunity, product, contract, or operation domain models in this change.

## Architecture

The work page owns one reusable `feedback-modal` backed by a dynamic `formSection`. Every task button writes the selected task/customer/asset into the existing page store, builds nodes for that task, and opens the same modal. The existing submit controller continues to call `/crm/work/execute`.

The backend resolves form fields once for both rendering and submission. A normal field resolves to itself. A legacy `group` field resolves to an ordered group containing the enabled data fields created after that group marker and before the next selected form field in the same template. The API returns group metadata and child fields; submission validates and saves the same resolved children. This preserves the current P01-P12 configuration without introducing a parallel form engine.

## Data Flow

1. Login redirects to `/work/crm/work`.
2. A task button opens the shared task modal.
3. The work API returns the task form, including expanded group children.
4. The plugin renders group headings and existing input/select/upload controls.
5. Progress save persists partial values without transition.
6. Complete validates required fields, saves data records, triggers after-task decisions, and applies stage transitions.
7. The table refreshes and shows the new stage/available tasks.

## Error Handling

- Missing or empty groups remain visible as a section and do not crash rendering.
- Unsupported field types fall back to text input, matching current behavior.
- API validation errors continue to map back to the corresponding dynamic field.
- Automatic rule failures continue to use the existing operation log behavior.

## Verification

The user forbids build and test commands. Verification is limited to Dever static audit, browser console/network inspection, screenshots, database state inspection, and a Playwright-driven Chromium walkthrough of the real work site.

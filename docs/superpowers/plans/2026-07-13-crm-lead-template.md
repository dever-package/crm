# CRM Lead Template Implementation Plan

> **For agentic workers:** Execute inline in the current session. The user explicitly forbids subagents and automated tests.

**Goal:** Add configurable lead fields, preserve source-lead provenance after conversion, and expose leads in the admin customer-management menu.

**Architecture:** Keep stable lead identity and lifecycle fields in `crm_lead`; store configurable lead values in the existing JSON column using data-field IDs. Reuse current data-template metadata and option helpers, while keeping workflow form collection limited to customer, asset and business categories.

**Tech Stack:** Go, Dever ORM/page JSON, React/TypeScript, PostgreSQL migrations.

---

### Task 1: Add the lead data-template category

**Files:**
- Modify: `model/data_template_cate.go`
- Modify: `service/setting/options.go`
- Modify: `front/page/admin/data_template/list.json`
- Create: `migrations/postgres/011_lead_data_templates.sql`

- [ ] Add category ID 4 and the `lead` target.
- [ ] Include it in data-template options but exclude it from workflow form-field cascaders.
- [ ] Add the lead tab and migration; disable the obsolete customer-source template.

### Task 2: Persist and expose configurable lead fields

**Files:**
- Modify: `service/work_lead.go`
- Modify: `front/src/nodes/show/work-lead.tsx`

- [ ] Return enabled lead templates with field options from the lead-pool endpoint.
- [ ] Accept only enabled lead-template field IDs and store them under `RecordJSON.data_values`.
- [ ] Render supported field types below the fixed lead fields and submit their values.

### Task 3: Preserve and show source-lead provenance

**Files:**
- Modify: `service/work.go`
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-customer-detail.tsx`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] Attach the converted source lead to customer detail responses.
- [ ] Add read-only source-lead core and extension sections to customer details.
- [ ] Show the source lead in the customer overview without duplicating values into customer records.

### Task 4: Add the admin lead list

**Files:**
- Modify: `model/lead.go`
- Create: `service/setting/lead.go`
- Create: `front/page/admin/lead/list.json`

- [ ] Add relations and a focused row-enrichment Provider.
- [ ] Create a read-only searchable lead table under customer management with sort 0.

### Task 5: Static verification and commits

- [ ] Run `gofmt`, `jq empty`, `git diff --check`, focused `rg`, and Dever audit.
- [ ] Do not run build, test, or browser automation.
- [ ] Commit the implementation to local `main`.

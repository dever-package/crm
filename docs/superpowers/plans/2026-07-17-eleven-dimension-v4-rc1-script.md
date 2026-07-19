# Eleven Dimension V4-RC1 Rebuild Script Implementation Plan

> **For agentic workers:** Execute inline in the current workspace. The project explicitly forbids build and test commands, and no Git commit is authorized.

**Goal:** Create a safe Go command that reads the frozen V4-RC1 workbook, previews the impact, and only with `--apply` deletes old eleven-dimension data and rebuilds the P01-P11 configuration in one ORM transaction.

**Architecture:** Keep the command isolated under the application `cmd` directory. Split workbook parsing, rule generation, database rebuilding, and CLI output into focused files. Reuse the existing CRM models and rule runtime; do not add SQL, API, Service, Model, or frontend code.

**Tech Stack:** Go, Dever ORM, CRM Model, Excelize, JavaScript RuleScript

---

### Task 1: Parse and validate the frozen workbook

**Files:**
- Create: `cmd/rebuild-eleven-dimension/source.go`

- [x] Read the `.xlsx` entry from `doc/1.zip` without extracting it into the repository.
- [x] Parse sheets `02_标准候选选项`, `09_判定输出字段`, `10_选项规则标签`, `11_跨维组合规则`, and `12_T节点服务映射_规则` by header name.
- [x] Validate 11 dimensions, 114 unique options, 114 matching tag rows, 35 unique combination rules, 27 outputs, and 11 T-node mappings.
- [x] Reject P12, unknown dimension codes, unsupported rule expressions, duplicate codes, and missing required cells before any database mutation.

### Task 2: Generate the V4-RC1 JavaScript rule

**Files:**
- Create: `cmd/rebuild-eleven-dimension/rule_script.go`

- [x] Compile the workbook condition expressions into a small condition tree supporting equality, inequality, IN, AND, OR, missing input, option flags, and route metrics.
- [x] Generate one JavaScript `main(input)` using the frozen option tags, combination rules, and T-to-service mappings.
- [x] Return the V4 output field keys and product codes through the existing `TaskRuleResult` contract.
- [x] Keep `formal_t` manual and omit `judgment_snapshot_no` from JavaScript output so runtime code can assign the operation-backed snapshot number later.

### Task 3: Preview and rebuild through CRM models

**Files:**
- Create: `cmd/rebuild-eleven-dimension/rebuild.go`
- Create: `cmd/rebuild-eleven-dimension/cleanup.go`
- Create: `cmd/rebuild-eleven-dimension/configuration.go`

- [x] Preflight the eleven-dimension template/form, diagnosis template, rule script, signing workflow, diagnosis stage, and affected tasks.
- [x] Capture the seven shared field definitions before removing the P01-P12 grouped fields.
- [x] Calculate preview counts for records, statistics, attachments, operations, events, form fields, options, usages, finance rows, and old rule-generated candidate products.
- [x] In one `orm.Transaction`, delete only old eleven-dimension data and old-rule derived data, preserving customers, assets, confirmed/processing/completed products, and unrelated records.
- [x] Rebuild 11 flat required select fields plus seven optional shared fields, with no parent/group fields.
- [x] Rebuild the eleven-dimension form, V4 diagnosis outputs, formal-T field/form, rule script, and diagnosis-stage task bindings.

### Task 4: Add the safe CLI entrypoint

**Files:**
- Create: `cmd/rebuild-eleven-dimension/main.go`

- [x] Add `--source` with default `doc/1.zip` and `--apply` with default `false`.
- [x] Always print workbook counts and database impact before the result.
- [x] In preview mode, perform no writes and print the exact apply command.
- [x] In apply mode, return non-zero on any validation, preflight, mutation, or postcondition failure.

### Task 5: Static review only

**Files:**
- Modify: files created in Tasks 1-4

- [x] Run `gofmt -w cmd/rebuild-eleven-dimension/*.go`.
- [x] Run `git diff --check -- cmd/rebuild-eleven-dimension package/crm/docs/superpowers/specs/2026-07-17-eleven-dimension-v4-rc1-design.md package/crm/docs/superpowers/plans/2026-07-17-eleven-dimension-v4-rc1-script.md`.
- [x] Manually inspect that the code contains no raw SQL, the default path cannot write, and all mutations use the transaction context.
- [x] Do not run `go test`, `go run`, `npm run build`, `dever build`, or any migration command.

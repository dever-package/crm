# CRM Product Workflow Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除业务记录类型与流程的重复配置，让产品通过可选服务流程驱动客户产品和独立流程实例，并把客户、资产、业务数据统一沉淀到数据模板体系。

**Architecture:** 配置层只保留产品、流程、任务、资料模板和数据模板。运行层新增客户产品与流程实例，所有待办、操作记录、统计、业务数据和财务流水通过流程实例定位，入口流程完成后按已确认产品幂等启动服务流程。

**Tech Stack:** Go、Dever ORM、Dever page JSON、React/TypeScript CRM front plugin、PostgreSQL migration

---

## Task 1: 建立可回退基线

**Files:**
- No source changes

- [ ] **Step 1: 确认 CRM 组件工作区干净**

Run:

```bash
git status --short --branch
git rev-parse HEAD
```

Expected: `main` 工作区无未提交源码，基线提交为设计文档提交 `ba99b5a` 或其直接后继。

- [ ] **Step 2: 创建备份分支**

Run:

```bash
git branch backup/crm-before-product-workflow-unification-20260713 HEAD
```

Expected: 备份分支指向实施前完整 CRM。

---

## Task 2: 精简产品与流程后台配置

**Files:**
- Create: `model/product_category.go`
- Create: `front/page/admin/product_category/list.json`
- Create: `front/page/admin/product_category/update.json`
- Modify: `model/product.go`
- Modify: `model/workflow.go`
- Modify: `model/task.go`
- Modify: `model/options.go`
- Modify: `service/setting/product.go`
- Modify: `service/setting/workflow.go`
- Modify: `front/page/admin/product/list.json`
- Modify: `front/page/admin/product/update.json`
- Modify: `front/page/admin/workflow/update.json`
- Modify: `front/page/admin/task/update.json`

- [ ] **Step 1: 新增可配置产品分类模型**

Create `ProductCategory` with the exact business fields:

```go
type ProductCategory struct {
    ID        uint64
    Name      string
    Status    int16
    Sort      int
    CreatedAt time.Time
}
```

Use table `crm_product_category`, default order `sort asc,id asc`, status options, and a `status,sort,id` index. Add a standard relation named `productCategoryRelation` in `model/options.go`.

- [ ] **Step 2: 简化产品模型**

Replace fixed `Category` and all signing/review fields with:

```go
CategoryID        uint64 `dorm:"type:bigint;not null;default:0;comment:产品分类"`
ServiceWorkflowID uint64 `dorm:"type:bigint;not null;default:0;comment:服务流程"`
```

Keep `ID/Code/Name/Description/Status/Sort/CreatedAt/UpdatedAt`. Add relations for product category and service workflow. Remove signing direction constants, options and indexes.

- [ ] **Step 3: 删除流程全局后续关系并增加产品任务类型**

Remove `NextWorkflowID` from `Workflow`, its relation/index, setting validation and page field. Add:

```go
const TaskTypeProduct = "product"
```

Expose it as `确认产品` in `taskTypeOptions`. Product tasks do not require `form_id` or `script_id`.

- [ ] **Step 4: 更新产品保存校验**

`ProviderBeforeSaveProduct` must:

- require code, name and category;
- validate `category_id` points to an enabled category;
- allow `service_workflow_id=0`;
- when non-zero, require an enabled non-default-entry workflow;
- reject a service workflow that is the only enabled default entry workflow;
- keep code normalization, status and sort defaults;
- retain option-set synchronization for enabled products.

Remove every normalization/default branch for signing direction and review flags.

- [ ] **Step 5: 更新后台 page JSON**

Product list columns become `name/code/product_category.name/service_workflow.name/status/sort/actions`. Add a `产品分类` header button opening the embedded category list. Product form uses model-backed selects for `category_id` and optional `service_workflow_id`.

Product category list/update use standard Model + page JSON CRUD and remain modal-only (`page.type=2`), so no new sidebar menu appears.

Workflow form removes `next_workflow_id`. Task form displays the existing form/script fields only for their matching task types and includes the new product task option.

- [ ] **Step 6: 格式和静态检查**

Run:

```bash
gofmt -w model/product_category.go model/product.go model/workflow.go model/task.go model/options.go service/setting/product.go service/setting/workflow.go
jq empty front/page/admin/product_category/list.json front/page/admin/product_category/update.json front/page/admin/product/list.json front/page/admin/product/update.json front/page/admin/workflow/update.json front/page/admin/task/update.json
git diff --check
```

Expected: Go formatting complete, JSON syntax valid, no whitespace errors. Do not run build or tests.

- [ ] **Step 7: Commit**

```bash
git add model service/setting front/page/admin/product front/page/admin/product_category front/page/admin/workflow/update.json front/page/admin/task/update.json
git commit -m "refactor: simplify crm product workflow configuration"
```

---

## Task 3: 固定数据模板分类为客户、资产和业务数据

**Files:**
- Modify: `model/data_template_cate.go`
- Modify: `service/setting/options.go`
- Modify: `front/page/admin/data_template/list.json`
- Delete: `front/page/admin/data_template_cate/update.json`

- [ ] **Step 1: 简化数据模板分类模型**

Define exactly three fixed IDs:

```go
const (
    CustomerDataTemplateCateID      uint64 = 1
    CustomerAssetDataTemplateCateID uint64 = 2
    BusinessDataTemplateCateID      uint64 = 3
)
```

Seed names are `客户信息/客户资产/业务数据`. This stage retains the old target columns internally until the runtime has switched, but the page no longer exposes them. Final removal happens in Task 7.

- [ ] **Step 2: 固定分类选项来源**

`ensureBaseDataTemplateCates` must ensure all three fixed rows. Category option providers return only these rows in ID order and never create arbitrary categories.

The third seed temporarily keeps the old `business_object` target so the current runtime remains usable before Task 6. Do not bind business data templates to products or workflows.

- [ ] **Step 3: 删除可编辑扩展主表入口**

Data template list tabs become `客户信息/客户资产/业务数据`. Remove the hard-coded `租赁记录` label and all create/edit actions for data template categories. Delete the category update page. Data template create/edit remains unchanged apart from accepting category 3.

- [ ] **Step 4: 静态检查并提交**

```bash
gofmt -w model/data_template_cate.go service/setting/options.go
jq empty front/page/admin/data_template/list.json
git diff --check
git add model/data_template_cate.go service/setting/options.go front/page/admin/data_template
git commit -m "refactor: fix crm data template categories"
```

Do not run build or tests.

---

## Task 4: 增加客户产品和流程实例领域模型

**Files:**
- Create: `model/customer_product.go`
- Create: `model/workflow_instance.go`
- Modify: `model/work_todo.go`
- Modify: `model/operation_log.go`
- Modify: `model/stat_event.go`
- Modify: `model/data_record.go`
- Modify: `model/stat_field_value.go`
- Modify: `model/finance_ledger.go`
- Modify: `model/options.go`
- Create: `migrations/postgres/009_product_workflow_unification.sql`

- [ ] **Step 1: 新增客户产品模型**

Use table `crm_customer_product` and statuses:

```go
const (
    CustomerProductStatusCandidate  = "candidate"
    CustomerProductStatusConfirmed  = "confirmed"
    CustomerProductStatusProcessing = "processing"
    CustomerProductStatusCompleted  = "completed"
    CustomerProductStatusLost       = "lost"
)
```

Fields are `ID/CustomerID/AssetID/ProductID/SourceWorkflowInstanceID/Status/CreatedAt/UpdatedAt`. Add indexes for source-instance product uniqueness, customer/asset status, and product status.

- [ ] **Step 2: 新增流程实例模型**

Use table `crm_workflow_instance` with fields matching the design:

```go
ID, CustomerID, AssetID, CustomerProductID,
WorkflowID, StageID, OwnerDepartmentID, OwnerStaffID,
Status, StartedAt, CompletedAt, TerminatedAt,
TerminatedReason, UpdatedAt
```

The active uniqueness boundary is `customer_product_id,workflow_id,status,id`; entry instances use `customer_product_id=0` and are found by customer/asset/default-entry workflow. Keep existing progress statuses.

- [ ] **Step 3: 让运行记录关联流程实例**

Add `WorkflowInstanceID` and `CustomerProductID` to `WorkTodo`, `OperationLog`, `StatEvent`, `DataRecord`, `StatFieldValue` and `FinanceLedger` where applicable. Replace business-object indexes and relations with workflow-instance/customer-product indexes and relations. WorkTodo uniqueness becomes `workflow_instance_id,stage_id,task_id`.

- [ ] **Step 4: 编写数据库迁移**

`009_product_workflow_unification.sql` must:

- create product category, customer product and workflow instance tables;
- add `category_id/service_workflow_id` to product and backfill one default category;
- seed data-template category 3 as `业务数据`;
- add workflow/customer-product columns to todo, operation, data, statistics and finance tables;
- copy existing asset progress into entry workflow instances;
- backfill current todo/log rows by customer, asset, workflow and stage;
- recreate indexes with the new ownership keys.

This additive migration does not drop old product columns, asset progress or business-object tables. Destructive cleanup is deferred to Task 7 after active code no longer uses them.

The migration is idempotent through `IF EXISTS/IF NOT EXISTS` and guarded inserts.

- [ ] **Step 5: 静态检查并提交**

```bash
gofmt -w model/customer_product.go model/workflow_instance.go model/work_todo.go model/operation_log.go model/stat_event.go model/data_record.go model/stat_field_value.go model/finance_ledger.go model/options.go
git diff --check
git add model migrations/postgres/009_product_workflow_unification.sql
git commit -m "feat: add crm customer products and workflow instances"
```

Do not run migrations, build or tests in this step.

---

## Task 5: 切换流程运行时到流程实例

**Files:**
- Create: `service/customer_product.go`
- Modify: `service/workflow_runtime.go`
- Modify: `service/workflow_transition.go`
- Modify: `service/workflow_assignment.go`
- Modify: `service/work_todo_execute.go`
- Modify: `service/work_flow_actions.go`
- Modify: `service/work_query.go`
- Modify: `service/work_audit.go`
- Modify: `service/work_ai_fill.go`
- Modify: `service/work.go`
- Modify: `service/setting/workflow.go`

- [ ] **Step 1: 实现客户产品同步服务**

Create focused functions:

```go
func SyncConfirmedCustomerProducts(ctx context.Context, instanceID uint64, productIDs []uint64) ([]*crmmodel.CustomerProduct, error)
func StartConfirmedProductWorkflows(ctx context.Context, entry *crmmodel.WorkflowInstance) error
func CompleteCustomerProductForInstance(ctx context.Context, instance *crmmodel.WorkflowInstance) error
```

The sync function validates enabled products, inserts missing confirmed rows, marks removable confirmed rows lost, and refuses to cancel processing/completed rows. Start is idempotent and creates one active service instance per customer product.

- [ ] **Step 2: 改造入口流程启动**

Replace `StartAssetWorkflow` internals with entry `WorkflowInstance` creation. Keep the public function name temporarily only where existing callers require it, but all internal helpers accept `workflowInstanceID` or `*WorkflowInstance`, never infer the current flow solely from asset ID.

- [ ] **Step 3: 改造阶段、待办、分配和终止**

Replace `CustomerStage` parameters with `WorkflowInstance`. All todo creation, assignment, stage completion and termination filters use `workflow_instance_id`. Flow action payloads require `workflow_instance_id`; asset ID remains response context, not identity.

`nextWorkflowStage` only returns the next stage inside the current workflow. At terminal stage:

- entry instance calls `StartConfirmedProductWorkflows`;
- product instance calls `CompleteCustomerProductForInstance`.

- [ ] **Step 4: 实现产品确认任务执行**

Add `completeProductTodo` handling in the task dispatch. Payload uses `product_ids` as an array of enabled product IDs. It calls `SyncConfirmedCustomerProducts`, records an operation snapshot containing selected IDs, and completes the todo. The task endpoint must reject empty selection for a required product task.

- [ ] **Step 5: 更新查询、审计和 AI 上下文**

Todo detail, workflow detail, operation logs, statistical events and AI form context include `workflow_instance_id` and optional `customer_product_id/product_name`. Permission checks load the exact instance from the todo instead of selecting an arbitrary active flow for the asset.

- [ ] **Step 6: 静态检查并提交**

```bash
gofmt -w service/customer_product.go service/workflow_runtime.go service/workflow_transition.go service/workflow_assignment.go service/work_todo_execute.go service/work_flow_actions.go service/work_query.go service/work_audit.go service/work_ai_fill.go service/work.go service/setting/workflow.go
git diff --check
git add service
git commit -m "refactor: run crm workflows by instance"
```

Do not run build or tests.

---

## Task 6: 切换业务数据、统计和财务归属

**Files:**
- Modify: `service/data_record.go`
- Modify: `service/work_form_input.go`
- Modify: `service/work_operation_side_effect.go`
- Modify: `service/work_todo_execute.go`
- Modify: `service/work_audit.go`
- Modify: `service/admin_summary.go`
- Modify: `front/page/admin/finance_ledger/list.json`

- [ ] **Step 1: 保存业务数据到流程实例**

Customer and asset templates continue using customer/asset ownership. Business-data templates require `workflow_instance_id`; product service instances also persist `customer_product_id`. Replace every `business_object_id` payload, filter and save helper with these two IDs.

- [ ] **Step 2: 更新统计与财务副作用**

Stat field values and finance ledger rows copy workflow/customer-product ownership from the completing todo and operation log. Uniqueness and reversal lookup use `workflow_instance_id + data_field_id + operation_log_id`, not business object.

- [ ] **Step 3: 更新后台财务列表**

Replace the business-object filter/column with customer product and workflow instance columns. Keep customer, asset, finance type, amount, direction, operator and time filters unchanged.

- [ ] **Step 4: 静态检查并提交**

```bash
gofmt -w service/data_record.go service/work_form_input.go service/work_operation_side_effect.go service/work_todo_execute.go service/work_audit.go service/admin_summary.go
jq empty front/page/admin/finance_ledger/list.json
git diff --check
git add service front/page/admin/finance_ledger/list.json
git commit -m "refactor: attach crm business data to workflow instances"
```

---

## Task 7: 用业务处理页面替换业务记录类型和租赁记录

**Files:**
- Create: `front/page/admin/customer_product/list.json`
- Create: `front/page/admin/customer_product/update.json`
- Create: `service/setting/customer_product.go`
- Delete: `front/page/admin/business_object/list.json`
- Delete: `front/page/admin/business_object/update.json`
- Delete: `front/page/admin/business_object_type/list.json`
- Delete: `front/page/admin/business_object_type/update.json`
- Delete: `model/business_object.go`
- Delete: `model/business_object_type.go`
- Delete: `service/setting/business_object.go`
- Modify: `model/data_template_cate.go`
- Modify: `service/setting/data_template.go`
- Modify: `service/setting/options.go`
- Modify: `service/setting/form.go`
- Modify: `service/work_form_input.go`
- Create: `migrations/postgres/010_remove_legacy_business_objects.sql`

- [ ] **Step 1: 新增业务处理标准后台页**

Customer product list page name is `业务处理`, parent is `crm-customer-manage`, and columns are customer, asset, product, product status, active service workflow/stage, owner and updated time. Filters are keyword, product, workflow and status.

The update modal only allows status changes that do not bypass active workflow validation; owner and stage changes continue through workflow actions rather than direct CRUD.

- [ ] **Step 2: 删除旧业务对象配置和页面**

Delete both business-object models, setting hook and four pages. Remove their relations and option providers. No compatibility menu, API or hidden page remains.

- [ ] **Step 3: 完成数据模板固定分类切换**

Reduce `DataTemplateCate` to `ID/Name/Status/Sort/CreatedAt`. Remove `target_table`, `business_object_type_id`, their relations and validators. Replace dynamic target lookup with:

```go
func dataTemplateRecordTarget(cateID uint64) string {
    switch cateID {
    case crmmodel.CustomerAssetDataTemplateCateID:
        return "asset"
    case crmmodel.BusinessDataTemplateCateID:
        return "workflow"
    default:
        return "customer"
    }
}
```

Rename the remaining `businessObjectDataRecords` path to `workflowDataRecords`.

- [ ] **Step 4: 添加破坏性清理迁移**

`010_remove_legacy_business_objects.sql` drops old product signing/review columns, workflow `next_workflow_id`, data-template category target/type columns, asset-progress table, and business-object/type tables only after Task 5 and Task 6 have switched all active code.

- [ ] **Step 5: 静态检查并提交**

```bash
gofmt -w model/data_template_cate.go service/setting/customer_product.go service/setting/data_template.go service/setting/options.go service/setting/form.go service/work_form_input.go
jq empty front/page/admin/customer_product/list.json front/page/admin/customer_product/update.json
rg -n "BusinessObject|business_object|业务记录类型|租赁记录" model service front/page/admin --glob '!front/dist/**'
git diff --check
```

Expected: `rg` only reports migration history or intentional user-facing data labels inside historical SQL; active model/service/page code has no old business-object dependency.

```bash
git add model service/setting service/work_form_input.go front/page/admin migrations/postgres/010_remove_legacy_business_objects.sql
git commit -m "refactor: replace crm business objects with customer products"
```

---

## Task 8: 切换工作台到客户产品和多流程实例

**Files:**
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-task-form.tsx`
- Modify: `service/work.go`
- Modify: `service/work_query.go`
- Modify: `api/work.go`

- [ ] **Step 1: 扩展工作台响应结构**

Customer and asset detail responses include `customer_products` and `workflow_instances`. Each instance returns ID, workflow, stage, owner, status, pending tasks and linked product. Remove `business_objects` and rental-specific response fields.

- [ ] **Step 2: 增加产品确认任务控件**

Render product tasks as a searchable multi-select using enabled product options returned with the task. Submit `workflow_instance_id`, `todo_id` and `product_ids`. Keep form, approval, rule and normal todo components unchanged.

- [ ] **Step 3: 调整客户详情与流程操作**

Replace the rental-record block with `已确认产品/业务处理`. Display entry flow separately, then one compact row per customer product with its service workflow and stage. Every assignment, completion and termination action sends the selected `workflow_instance_id`.

- [ ] **Step 4: 静态检查并提交**

```bash
gofmt -w service/work.go service/work_query.go api/work.go
git diff --check
git add front/src/nodes/show service/work.go service/work_query.go api/work.go
git commit -m "feat: show crm customer product workflows"
```

Do not run front build or tests.

---

## Task 9: 最终静态审计和旧路径清理

**Files:**
- Modify only files identified by the audits below

- [ ] **Step 1: 搜索禁止保留的旧概念**

```bash
rg -n "BusinessObject|business_object|business_object_type|default_signing_business_type|need_(pm|lawyer|ala|finance|contract)_review|next_workflow_id" model service api front --glob '!front/dist/**'
```

Expected: no active-code matches. Migration SQL may retain old column names only inside explicit drop/backfill statements.

- [ ] **Step 2: 运行 Dever 静态协议审计**

```bash
bash /data/project/demo/skills/skills-dever/scripts/audit.sh /data/project/demo/gjj/package/crm
```

Expected: no forbidden page protocol, generated-file edit, CRUD API wrapper or front-dist modification findings.

- [ ] **Step 3: 检查 JSON、格式和提交范围**

```bash
find front/page/admin -name '*.json' -print0 | xargs -0 -n1 jq empty
git diff --check
git status --short
git log --oneline backup/crm-before-product-workflow-unification-20260713..HEAD
```

Expected: JSON valid, no whitespace errors, worktree clean after final fixes, and commits correspond to the planned phases.

- [ ] **Step 4: 提交审计修正**

If static audit required fixes:

```bash
git add model service api front migrations
git commit -m "chore: finish crm workflow unification"
```

Per user instruction, do not run `npm run build`, `dever build`, `go test`, Playwright, or any automated test command. Report that UI and complete business-chain verification remain for user manual testing.

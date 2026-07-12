# CRM Customer Workspace UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 work 前台客户列表、客户详情和动态任务表单改造成紧凑、易搜索、易填写的客户工作区，同时保持现有流程和资料模板模型不变。

**Architecture:** 继续由现有 `ShowCrmWorkCustomerTable`、`ShowCrmWorkDetail` 和动态任务节点负责业务编排，将新增的列表、详情和表单展示拆到三个聚焦的 React 文件。后端只扩展现有客户列表与详情响应：列表返回阶段选项，详情返回按模板组织的只读资料分区；搜索、保存进度、任务完成和流程动作全部复用现有接口。

**Tech Stack:** Go、Dever ORM/Service、React、TypeScript、Dever front plugin、Tailwind 宿主样式、Lucide 图标、Playwright 浏览器探针。

**Project constraints:** 不运行 `npm run build`、`dever build`、任何 test 或等价测试命令。每个阶段使用 `gofmt`、`git diff --check` 和已启动 Dever 环境中的浏览器探针验证。

---

## File Map

- Modify: `service/work.go`
  - 为 `/crm/work/customers` 增加 `stage_options` 响应。
  - 为 `/crm/work/customer_detail` 生成完整的 `detail_sections` 只读投影。
- Modify: `front/src/nodes/show/work-core.ts`
  - 增加综合搜索、阶段选项、详情资料分区和表单布局类型。
  - 提取列表、详情和表单共同使用的纯领域格式化函数。
- Create: `front/src/nodes/show/work-customer-list.tsx`
  - 客户筛选工具栏、桌面表格、移动卡片、任务菜单和分页展示。
- Create: `front/src/nodes/show/work-customer-detail.tsx`
  - 详情上下文头、总览、完整资料分区和单列流程时间线。
- Create: `front/src/nodes/show/work-task-form.tsx`
  - 自适应表单上下文、分组导航、字段控件、填写进度和错误摘要。
- Modify: `front/src/nodes/show/work-auth.tsx`
  - 保留 API/store 编排，改为组合新的聚焦组件。
  - 复用现有任务打开、保存和刷新逻辑。
- Modify: `front/src/plugin.ts`
  - 注册任务表单上下文节点；现有公开节点名保持不变。
- Modify: `front/page/work/work.json`
  - 给任务弹窗和详情抽屉增加稳定的语义 class，供自适应布局使用。
- Create outside repository: `/data/project/demo/gjj/tmp/crm_customer_workspace_probe.py`
  - 浏览器只读/业务验证脚本，不加入 CRM package Git。

## Existing Reuse Points

- 综合搜索直接使用现有 `/crm/work/customers?keyword=` 和 `filterWorkCustomers`。
- 精确筛选继续使用 `customer_no/customer_name/phone/wechat/asset_no`。
- 工作台下钻继续使用 `mode/stage_filter/task_filter/scope`。
- 任务表单继续由 `buildWorkTaskFormState` 和资料模板字段生成。
- 保存继续使用 `/crm/work/execute` 的 `progress`、`complete` 模式。
- 详情继续使用 `/crm/work/customer_detail`、`crm-work-refresh` 和现有权限检查。
- 附件继续使用 `ShowCrmWorkTaskUpload`、`WorkTaskUploadPreviewDialog`。
- 流程动作继续使用 `WorkFlowActions`。

---

### Task 1: Add Customer Workspace Contracts and Defaults

**Files:**
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] **Step 1: Extend search and response types**

Add the following contracts to `work-core.ts` next to the existing customer types:

```ts
export type WorkSearchFilters = {
  keyword: string;
  customerNo: string;
  customerName: string;
  phone: string;
  wechat: string;
  assetNo: string;
  status: string;
};

export type WorkStageOption = {
  id: string;
  value: string;
  code?: string;
  workflowName?: string;
};

export type WorkDetailField = {
  key: string;
  label: string;
  value?: unknown;
  valueType?: string;
  empty?: boolean;
  group?: string;
  files?: unknown[];
};

export type WorkDetailSection = {
  id: string;
  name: string;
  targetType: "customer" | "asset" | "business_object";
  templateId?: string | number;
  objectId?: string | number;
  objectName?: string;
  filled: number;
  total: number;
  percent: number;
  fields: WorkDetailField[];
};

export type WorkTaskLayoutMode = "compact" | "workspace";
```

- [ ] **Step 2: Initialize keyword consistently**

Update `emptyWorkSearchFilters()` in `work-core.ts`:

```ts
export function emptyWorkSearchFilters(): WorkSearchFilters {
  return {
    keyword: "",
    customerNo: "",
    customerName: "",
    phone: "",
    wechat: "",
    assetNo: "",
    status: "",
  };
}
```

Update `workSearchQuery()` and `workSearchFiltersFromURL()` in `work-auth.tsx` so `keyword` is serialized as `keyword` and restored from the URL without removing the exact filters.

- [ ] **Step 3: Make the direct customer-list default `all`**

Change the fallback in `workCustomerModeFromNode()` to:

```ts
const pathname = textValue(window.location.pathname);
return pathname.endsWith("/work/done") || pathname.includes("/work/done/")
  ? "done"
  : "all";
```

Reorder `workTopFilterOptions` to `all`, `pending`, `done` while retaining their existing mode values.

- [ ] **Step 4: Perform static diff validation**

Run:

```bash
git diff --check
```

Expected: no output and exit code 0.

- [ ] **Step 5: Commit contracts and defaults**

```bash
git add front/src/nodes/show/work-core.ts front/src/nodes/show/work-auth.tsx
git commit -m "refactor: add crm customer workspace contracts"
```

---

### Task 2: Add Stage Options and Detail Sections to Existing Responses

**Files:**
- Modify: `service/work.go`

- [ ] **Step 1: Return reusable stage options from the customer list**

Add one helper near the customer list helpers:

```go
func workCustomerStageOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewStageModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	})
	result := make([]map[string]any, 0, len(rows))
	for _, stage := range rows {
		if stage == nil || stage.ID == 0 || strings.TrimSpace(stage.Name) == "" {
			continue
		}
		value := fmt.Sprintf("%d", stage.ID)
		result = append(result, map[string]any{
			"id":    value,
			"value": stage.Name,
			"code":  value,
		})
	}
	return result
}
```

Add `"stage_options": workCustomerStageOptions(ctx)` to both return paths in `WorkService.Customers`.

- [ ] **Step 2: Build complete template field rows**

Add helpers next to `workDataCompletenessTemplates` that return every configured non-group field, including empty fields:

```go
func workDataDetailSections(ctx context.Context, targetType string, cateID uint64, values map[string]any) []map[string]any {
	templates := crmmodel.NewDataTemplateModel().Select(ctx, map[string]any{
		"cate_id": cateID,
		"status":  crmmodel.StatusEnabled,
	})
	result := make([]map[string]any, 0, len(templates))
	for _, template := range templates {
		if template == nil {
			continue
		}
		fields := workDataDetailFields(ctx, template.ID, values)
		if len(fields) == 0 {
			continue
		}
		filled := 0
		for _, field := range fields {
			if !booleanFromAny(field["empty"]) {
				filled++
			}
		}
		percent := 0
		if len(fields) > 0 {
			percent = int(math.Round(float64(filled) / float64(len(fields)) * 100))
		}
		result = append(result, map[string]any{
			"id":            fmt.Sprintf("%s:%d", targetType, template.ID),
			"name":          template.Name,
			"target_type":   targetType,
			"template_id":   template.ID,
			"filled":        filled,
			"total":         len(fields),
			"percent":       percent,
			"fields":        fields,
		})
	}
	return result
}
```

`workDataDetailFields` must:

- select enabled fields by `data_template_id` in sort order;
- skip group rows but retain the parent group name on children;
- use `workDataFieldDisplayValue` for non-empty values;
- set `empty: true` and `value: ""` for missing values;
- retain `value_type`, `files` and other metadata returned by the display converter.

Implement it as:

```go
func workDataDetailFields(ctx context.Context, templateID uint64, values map[string]any) []map[string]any {
	fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	})
	parentNames := workDataCompletenessParentNames(ctx, fields)
	result := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		if field == nil || field.FieldType == "group" {
			continue
		}
		key := fmt.Sprintf("data:%d", field.ID)
		rawValue := values[key]
		empty := emptyWorkFieldValue(rawValue)
		displayValue := ""
		meta := map[string]any{}
		if !empty {
			displayValue, meta = workDataFieldDisplayValue(ctx, field, rawValue)
		}
		item := map[string]any{
			"key":        key,
			"label":      field.Name,
			"value":      displayValue,
			"value_type": "text",
			"empty":      empty,
		}
		if group := parentNames[field.ParentFieldID]; group != "" {
			item["group"] = group
		}
		for metaKey, metaValue := range meta {
			item[metaKey] = metaValue
		}
		result = append(result, item)
	}
	return result
}
```

- [ ] **Step 3: Attach detail sections only in `CustomerDetail`**

After the detail customer and optional asset have been loaded, return:

```go
detailSections := workDataDetailSections(
	ctx,
	crmmodel.DataTemplateTargetCustomer,
	crmmodel.CustomerDataTemplateCateID,
	mapFromAny(customer["data_values"]),
)
if len(asset) > 0 {
	detailSections = append(detailSections, workDataDetailSections(
		ctx,
		crmmodel.DataTemplateTargetCustomerAsset,
		crmmodel.CustomerAssetDataTemplateCateID,
		mapFromAny(asset["data_values"]),
	)...)
}
result["detail_sections"] = detailSections
```

Add business-object sections for each selected asset record:

```go
func workBusinessObjectDetailSections(ctx context.Context, customerID uint64, assetID uint64, asset map[string]any) []map[string]any {
	result := []map[string]any{}
	for _, object := range mapListFromAny(asset["business_objects"]) {
		objectID := inputUint64(object["id"])
		typeID := inputUint64(object["business_object_type_id"])
		if objectID == 0 || typeID == 0 {
			continue
		}
		cate := crmmodel.NewDataTemplateCateModel().Find(ctx, map[string]any{
			"target_table":           crmmodel.DataTemplateTargetBusinessObject,
			"business_object_type_id": typeID,
			"status":                 crmmodel.StatusEnabled,
		})
		if cate == nil {
			continue
		}
		values := workBusinessObjectFormValues(ctx, customerID, assetID, objectID)
		sections := workDataDetailSections(ctx, crmmodel.DataTemplateTargetBusinessObject, cate.ID, values)
		for _, section := range sections {
			section["id"] = fmt.Sprintf("business_object:%d:%v", objectID, section["template_id"])
			section["object_id"] = objectID
			section["object_name"] = firstText(object, "object_name", "object_no")
			result = append(result, section)
		}
	}
	return result
}
```

Append this result after the selected asset sections. This covers configured rental, product, contract and other business-object records without hard-coding their names.

Do not add these complete field definitions to the paginated customer list response.

- [ ] **Step 4: Format and inspect the Go diff**

Run:

```bash
gofmt -w service/work.go
git diff --check
git diff -- service/work.go
```

Expected: `service/work.go` is formatted, diff check exits 0, and only response projection helpers changed.

- [ ] **Step 5: Verify the live response read-only**

Use the existing logged-in browser session or a small Playwright fetch and confirm:

```text
CUSTOMERS_STAGE_OPTIONS > 0
DETAIL_SECTIONS > 0
DETAIL_EMPTY_FIELDS > 0
```

Do not submit any task during this step.

- [ ] **Step 6: Commit response projections**

```bash
git add service/work.go
git commit -m "feat: expose crm customer detail sections"
```

---

### Task 3: Build the Compact Customer List Presentation

**Files:**
- Create: `front/src/nodes/show/work-customer-list.tsx`
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] **Step 1: Define focused list component props**

Create `work-customer-list.tsx` with one public presentation component:

```ts
export type WorkCustomerListViewProps = {
  items: WorkItem[];
  loading: boolean;
  mode: WorkCustomerMode;
  modeCounts: Record<WorkCustomerMode, number>;
  scope: WorkCustomerScope;
  canDispatch: boolean;
  filters: WorkSearchFilters;
  stageFilter: string;
  stageOptions: WorkStageOption[];
  page: number;
  pageSize: number;
  total: number;
  onFiltersChange: (filters: WorkSearchFilters) => void;
  onSearch: () => void;
  onReset: () => void;
  onModeChange: (mode: WorkCustomerMode) => void;
  onScopeChange: (scope: WorkCustomerScope) => void;
  onStageChange: (stage: string) => void;
  onPageChange: (page: number) => void;
  onRefresh: () => void;
  onOpenDetail: (item: WorkItem) => void;
  onOpenTask: (item: WorkItem, task: WorkTask) => void;
};
```

The file owns toolbar, table, cards, task overflow and pagination markup. It must not call APIs or access the Dever store directly.

- [ ] **Step 2: Implement the compact toolbar**

The visible toolbar contains:

```tsx
<Input
  value={filters.keyword}
  placeholder="搜索姓名、手机、微信、客户或资产编号"
  onChange={(event) =>
    onFiltersChange({ ...filters, keyword: event.currentTarget.value })
  }
/>
<select value={stageFilter} onChange={(event) => onStageChange(event.currentTarget.value)}>
  <option value="">全部阶段</option>
  {stageOptions.map((option) => (
    <option key={option.id} value={option.id}>{option.value}</option>
  ))}
</select>
```

“更多筛选” toggles a bordered inline region containing the five exact fields. Hide the legacy `status` input; URL `status` remains supported by the request builder for compatibility.

- [ ] **Step 3: Implement stable five-column rows**

Use this desktop column contract:

```ts
const customerColumns = {
  customer: "w-[18rem] min-w-[18rem]",
  asset: "w-[20rem] min-w-[20rem]",
  stage: "w-[15rem] min-w-[15rem]",
  task: "w-[18rem] min-w-[18rem]",
  action: "w-[10rem] min-w-[10rem]",
};
```

For each row:

- render customer name/code plus phone and WeChat in the first cell;
- render asset title/no and one address field in the second cell;
- render stage, owner and elapsed days in the third cell;
- render the first task and total count in the fourth cell;
- render one primary action in the fifth cell;
- use an `Ellipsis` menu for tasks after the first;
- stop click propagation from task buttons;
- open detail when the row identity area is clicked.

Select the primary action with one shared helper so a read-only rule task never blocks an actionable task:

```ts
export function workPrimaryActionTask(tasks: WorkTask[]): WorkTask | undefined {
  return tasks.find((task) => textValue(task.task_type) !== "rule");
}
```

If every task is a rule task, show the first rule result as status and use “查看” as the action.

- [ ] **Step 4: Implement matching mobile cards**

Mobile cards show customer identity, current asset, stage, first task and one primary button. Extra tasks use the same overflow component instead of duplicating task mapping logic.

- [ ] **Step 5: Replace list markup in the orchestrator**

Keep fetching, mode state and Dever store callbacks in `ShowCrmWorkCustomerTable`. Replace the old toolbar/table/card JSX with `WorkCustomerListView` and map callbacks to existing `openWorkDetail` and `openRowTask`.

Capture `stage_options` from the customer response:

```ts
const [stageOptions, setStageOptions] = useState<WorkStageOption[]>([]);
// after a successful request
setStageOptions(Array.isArray(payload.stage_options) ? payload.stage_options : []);
```

- [ ] **Step 6: Validate and commit the list UI**

Run:

```bash
git diff --check
```

Open `/work/crm/work` and confirm the initial request includes `mode=all`, rows have stable height, and the browser has no horizontal document overflow at 1920px and 390px.

Commit:

```bash
git add front/src/nodes/show/work-customer-list.tsx front/src/nodes/show/work-core.ts front/src/nodes/show/work-auth.tsx
git commit -m "feat: simplify crm customer list"
```

---

### Task 4: Add Customer Detail Workspace Presentation

**Files:**
- Create: `front/src/nodes/show/work-customer-detail.tsx`
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/page/work/work.json`

- [ ] **Step 1: Extend the detail response type**

Add `detail_sections?: WorkDetailSection[]` to `WorkDetailTargetResponse` and store it in `useWorkDetailData` alongside customer, asset, operations, todos and flow.

Move the existing record guard from `work-auth.tsx` into `work-core.ts`, export it, and normalize the snake_case service response once:

```ts
export function workIsRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function normalizeWorkDetailFields(value: unknown): WorkDetailField[] {
  if (!Array.isArray(value)) return [];
  return value.filter(workIsRecord).map((field) => ({
    key: textValue(field.key),
    label: displayText(field.label),
    value: field.value,
    valueType: textValue(field.value_type) || "text",
    empty: Boolean(field.empty),
    group: textValue(field.group),
    files: Array.isArray(field.files) ? field.files : [],
  }));
}

export function normalizeWorkDetailSections(value: unknown): WorkDetailSection[] {
  if (!Array.isArray(value)) return [];
  return value.filter(workIsRecord).map((section) => ({
    id: textValue(section.id),
    name: displayText(section.name),
    targetType: textValue(section.target_type) as WorkDetailSection["targetType"],
    templateId: section.template_id as string | number | undefined,
    objectId: section.object_id as string | number | undefined,
    objectName: textValue(section.object_name),
    filled: Number(section.filled) || 0,
    total: Number(section.total) || 0,
    percent: Number(section.percent) || 0,
    fields: normalizeWorkDetailFields(section.fields),
  }));
}
```

`normalizeWorkDetailFields` maps `value_type`, `empty`, `group` and `files` into the camelCase field contract. Do not read snake_case keys throughout the UI.

- [ ] **Step 2: Create the detail context header**

Create `WorkCustomerDetailHeader` in `work-customer-detail.tsx` with props for customer, asset, flow, pending tasks and primary-action callback. It displays:

```tsx
<header className="crm-customer-detail-header sticky top-0 z-10 border-b bg-background pb-4">
  {/* name/status, customer/asset identifiers, stage/owner/days */}
  {primaryTask ? (
    <Button type="button" onClick={() => onOpenTask(primaryTask)}>
      <ClipboardList className="h-4 w-4" />
      处理当前任务
    </Button>
  ) : null}
</header>
```

Do not expose an unrestricted “编辑资料” action.

- [ ] **Step 3: Build the complete data section view**

Create `WorkCustomerDetailSections` that:

- groups sections by `targetType`;
- renders template name, `filled / total` and progress percentage;
- groups fields by their optional `group` name;
- shows `未填写` for `empty: true`;
- shows file count and uses the existing preview callback for attachments;
- uses a two-column description grid on desktop and one column on mobile.

Use one reusable field row:

```tsx
function WorkDetailFieldValue({ field }: { field: WorkDetailField }) {
  if (field.empty) return <span className="text-muted-foreground">未填写</span>;
  if (field.valueType === "files") return <WorkDetailFiles field={field} />;
  return <span className="break-words text-foreground">{displayText(field.value)}</span>;
}
```

- [ ] **Step 4: Simplify overview and flow**

Move overview presentation to the new file and remove duplicated full information cards. Keep only current stage, pending tasks, key customer/asset facts, completeness summary and five recent operations.

Replace the alternating timeline grid with a single vertical list:

```tsx
<div className="relative grid gap-3 border-l border-border/70 pl-5">
  {operations.map((operation) => (
    <WorkCustomerFlowRow key={operation.id} operation={operation} />
  ))}
</div>
```

Retain the existing all/mine scope filter and record-detail callback.

- [ ] **Step 5: Integrate the three detail tabs**

`ShowCrmWorkDetail` and its data hook remain responsible for loading and store state. Compose:

- `总览` -> new overview component;
- `资料` -> base customer/asset facts plus `detail_sections`;
- `流程` -> `WorkFlowActions` plus single timeline.

Update `work.json` drawer body class to include `crm-customer-detail-drawer-body`, retaining its current 52rem desktop width and full-width mobile behavior.

- [ ] **Step 6: Validate and commit detail UI**

Run `git diff --check`, then open a customer with an asset and confirm:

- the header remains visible while scrolling;
- “资料” includes at least one `未填写` field;
- “流程” is one column;
- record detail still opens;
- no unrestricted edit button appears.

Commit:

```bash
git add front/src/nodes/show/work-customer-detail.tsx front/src/nodes/show/work-core.ts front/src/nodes/show/work-auth.tsx front/page/work/work.json
git commit -m "feat: add crm customer detail workspace"
```

---

### Task 5: Build Reusable Task Form Field Controls

**Files:**
- Create: `front/src/nodes/show/work-task-form.tsx`
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-auth.tsx`

- [ ] **Step 1: Export one normalized field contract**

Move `WorkTaskGroupField` into `work-core.ts` and extend it:

```ts
export type WorkTaskFormField = {
  formKey: string;
  label: string;
  placeholder: string;
  required: boolean;
  readonly?: boolean;
  type: string;
  inputType?: "text" | "number" | "date" | "datetime-local";
  fullWidth?: boolean;
  options?: WorkCommonOption[];
  meta?: Record<string, unknown>;
};
```

Update field normalization to preserve `readonly`, field input type and full-width behavior.

- [ ] **Step 2: Normalize field render metadata once**

Update `workTaskFieldRenderConfig` so:

- boolean -> `form-switch`;
- number/decimal/money -> `form-input` with `inputType: number`;
- date -> `form-input` with `inputType: date`;
- datetime -> `form-input` with `inputType: datetime-local`;
- textarea/upload/multi-select -> `fullWidth: true`;
- readonly from the configured form field is retained.

Do not add a second field-type switch elsewhere.

- [ ] **Step 3: Implement focused controls**

In `work-task-form.tsx`, implement one dispatch map instead of repeated conditional markup:

```ts
const taskFieldRenderers: Record<string, TaskFieldRenderer> = {
  "form-input": renderTaskInput,
  "form-textarea": renderTaskTextarea,
  "form-select": renderTaskSelect,
  "form-switch": renderTaskBoolean,
  "show-crm-work-task-upload": renderTaskUpload,
};
```

Requirements:

- single select uses one menu/select control;
- multi-select renders checkbox rows and stores a string array;
- boolean renders a two-option segmented control (`是` / `否`);
- input uses the normalized `inputType`;
- readonly fields are disabled and visually distinct;
- every control is wrapped by one `WorkTaskField` component with label, required marker, `data-work-form-key`, error and width class.

- [ ] **Step 4: Render sections with a responsive grid**

Replace one-field-per-row markup with:

```tsx
<div className="grid gap-x-5 gap-y-4 md:grid-cols-2">
  {fields.map((field) => (
    <WorkTaskField
      key={field.formKey}
      field={field}
      className={field.fullWidth ? "md:col-span-2" : ""}
    />
  ))}
</div>
```

Change `addWorkTaskFieldSectionNodes` to always generate `show-crm-work-task-field-section` nodes, even when there is only one logical section. This ensures all fields use the same renderer.

- [ ] **Step 5: Validate controls and commit**

Open one todo, one approval and one data form without submitting. Confirm textarea, select, boolean, multi-select and upload controls render from the same normalized field contract.

Run `git diff --check` and commit:

```bash
git add front/src/nodes/show/work-task-form.tsx front/src/nodes/show/work-core.ts front/src/nodes/show/work-auth.tsx
git commit -m "refactor: unify crm task form controls"
```

---

### Task 6: Add the Long-Form Workspace and Error Navigation

**Files:**
- Modify: `front/src/nodes/show/work-task-form.tsx`
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/plugin.ts`
- Modify: `front/page/work/work.json`

- [ ] **Step 1: Calculate layout mode from normalized nodes**

Add one pure helper:

```ts
function workTaskLayoutMode(nodes: WorkTaskFormNode[]): WorkTaskLayoutMode {
  const fields = nodes.flatMap(workTaskNodeFields);
  const groupCount = nodes
    .filter((node) => node.type === "show-crm-work-task-group-tabs")
    .flatMap((node) => normalizeWorkTaskGroupTabs(node.meta?.tabs)).length;
  return fields.length > 6 || groupCount > 1 ? "workspace" : "compact";
}
```

Store the result at `data.actionTarget.workTaskLayout` when preparing the form.

- [ ] **Step 2: Add the task context node**

Insert `show-crm-work-task-context` as the first dynamic node. Register it in `plugin.ts` and render customer, asset, stage, task name and completion count.

The component root must include:

```tsx
<div data-crm-work-task-layout={layoutMode} className="crm-work-task-context">
  {/* context and required-field progress */}
</div>
```

Load task form nodes from their focused module instead of creating a dependency back to `work-auth.tsx`:

```ts
const loadWorkTaskForm = () => import("./nodes/show/work-task-form");
```

Point `show-crm-work-task-context`, `show-crm-work-task-group-tabs` and `show-crm-work-task-field-section` at `loadWorkTaskForm`. Keep `show-crm-work-task-form` on the orchestrator until submission side effects are fully separated.

- [ ] **Step 3: Build desktop group navigation**

For workspace mode, `ShowCrmWorkTaskGroupTabs` renders:

```tsx
<div className="crm-work-task-workspace-grid">
  <nav className="crm-work-task-section-nav">{/* group buttons */}</nav>
  <section className="min-w-0">{/* active group grid */}</section>
</div>
```

Use component-local semantic CSS because plugin-only arbitrary Tailwind classes are not guaranteed to exist in the host stylesheet:

```css
.crm-work-task-workspace-grid {
  display: grid;
  grid-template-columns: 176px minmax(0, 1fr);
  gap: 20px;
}
@media (max-width: 767px) {
  .crm-work-task-workspace-grid { grid-template-columns: minmax(0, 1fr); }
  .crm-work-task-section-nav { display: flex; overflow-x: auto; }
}
```

- [ ] **Step 4: Expand only workspace dialogs**

Use the existing `:has` pattern already used by record details:

```css
[role="dialog"]:has([data-crm-work-task-layout="workspace"]) {
  width: min(1120px, calc(100vw - 32px)) !important;
  max-width: min(1120px, calc(100vw - 32px)) !important;
  max-height: calc(100vh - 32px);
}
```

Keep compact todo and approval dialogs at their current size. Keep footer buttons outside the scrolling body by reusing the existing footer portal.

- [ ] **Step 5: Add error summary and field focus**

Convert current validation errors into `{formKey, label, groupId, message}` rows. When “确认完成” fails:

1. show a summary below the context header;
2. switch to the group containing the first error;
3. focus `[data-work-form-key="<formKey>"]`;
4. leave all entered values untouched.

Use `data.actionTarget.workTaskActiveGroup` as the single store path shared by the error summary and `ShowCrmWorkTaskGroupTabs`. The group component must follow this value when it changes and write the same path when the user selects a tab; do not introduce a second event bus.

Do not show the summary when “保存进度” is used.

- [ ] **Step 6: Browser-verify progress and completion behavior**

Using a CTF customer with a grouped form:

- open the long form and confirm workspace width;
- switch P01 and P12 groups;
- change one non-sensitive test value;
- use “保存进度” and reopen to confirm persistence;
- trigger missing-field validation and confirm the correct group opens;
- do not complete the business flow during this focused step.

- [ ] **Step 7: Commit workspace form behavior**

```bash
git add front/src/nodes/show/work-task-form.tsx front/src/nodes/show/work-auth.tsx front/src/plugin.ts front/page/work/work.json
git commit -m "feat: add crm long form workspace"
```

---

### Task 7: Remove Superseded Markup and Consolidate Reuse

**Files:**
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-customer-list.tsx`
- Modify: `front/src/nodes/show/work-customer-detail.tsx`
- Modify: `front/src/nodes/show/work-task-form.tsx`
- Modify: `front/src/nodes/show/work-core.ts`

- [ ] **Step 1: Remove old duplicate components**

Delete old list toolbar/table/card components, old alternating detail timeline markup and old `WorkTaskGroupInput` branches only after their new callers are active.

- [ ] **Step 2: Consolidate repeated display logic**

Ensure there is one implementation for each of:

- customer/asset identity formatting;
- first actionable task selection;
- task overflow rendering;
- detail field value rendering;
- task field renderer dispatch;
- required-field error collection.

If a helper is used by two UI modules, place the pure helper in `work-core.ts`; do not create a base component or inheritance layer.

- [ ] **Step 3: Check module responsibilities**

Confirm:

- list file has no detail markup and no direct API calls;
- detail file has no list filters and no task submission;
- task form file has no customer-list pagination or detail timeline;
- `work-auth.tsx` contains orchestration and unrelated login/stats code, not duplicate presentation trees.

- [ ] **Step 4: Static cleanup and commit**

Run:

```bash
git diff --check
rg -n "WorkItemTableRow|WorkItemCardList|WorkOperationCard|WorkTaskGroupInput" front/src/nodes/show/work-auth.tsx
```

Expected: diff check exits 0; the superseded component names are absent from `work-auth.tsx` unless retained only as intentional orchestration wrappers.

Commit:

```bash
git add front/src/nodes/show
git commit -m "refactor: separate crm customer workspace views"
```

---

### Task 8: Create and Run the Browser Verification Probe

**Files:**
- Create outside repository: `/data/project/demo/gjj/tmp/crm_customer_workspace_probe.py`

- [ ] **Step 1: Implement read-only list and detail checks**

The probe logs in with the existing CTF account and prints:

```text
LIST_DEFAULT_MODE all
LIST_ROW_COUNT <positive number>
KEYWORD_RESULT_COUNT <positive number>
STAGE_OPTION_COUNT <positive number>
DETAIL_TAB_COUNT 3
DETAIL_SECTION_COUNT <positive number>
DETAIL_EMPTY_FIELD_COUNT <positive number>
FLOW_LAYOUT single-column
```

It captures:

- `customer-list-desktop.png`
- `customer-detail-overview-desktop.png`
- `customer-detail-data-desktop.png`
- `customer-detail-flow-desktop.png`
- `customer-list-mobile.png`
- `customer-detail-mobile.png`

- [ ] **Step 2: Add non-destructive form layout checks**

Open available todo/approval forms and a grouped data form without completing them. Print:

```text
COMPACT_DIALOG_WIDTH <800
WORKSPACE_DIALOG_WIDTH >=1000
WORKSPACE_GROUP_COUNT >=2
FORM_HORIZONTAL_OVERFLOW 0
```

If no grouped form is currently pending, use a new clearly named CTF-only lead/customer and stop after saving progress; do not alter an existing production-like customer.

- [ ] **Step 3: Run the probe at desktop and mobile sizes**

Run:

```bash
python /data/project/demo/gjj/tmp/crm_customer_workspace_probe.py
```

Expected:

- exit code 0;
- document and content horizontal overflow both 0;
- console error list empty;
- failed request list empty;
- all expected screenshots exist.

- [ ] **Step 4: Inspect screenshots and correct visual defects**

Use `view_image` on all six screenshots. Correct only defects within the customer workspace scope: overlap, clipped text, unstable row height, hidden footer, unreadable missing values or mobile overflow.

- [ ] **Step 5: Commit final visual corrections**

If corrections were needed:

```bash
git add service/work.go front/page/work/work.json front/src
git commit -m "fix: polish crm customer workspace"
```

If no correction was needed, do not create an empty commit.

---

### Task 9: Final Review and Branch Handoff

**Files:**
- Review all files changed since `a57642f`.

- [ ] **Step 1: Review the implementation against the spec**

Check every acceptance criterion in `docs/superpowers/specs/2026-07-12-crm-customer-workspace-ux-design.md` against code and browser evidence.

- [ ] **Step 2: Run allowed final checks**

Run:

```bash
gofmt -w service/work.go
git diff --check main...HEAD
git status --short
python /data/project/demo/gjj/tmp/crm_customer_workspace_probe.py
```

Expected: formatting stable, diff check clean, Git worktree clean after commits, browser probe exit 0.

Do not run build, test, lint, `go test`, `npm test`, or `npm run build`.

- [ ] **Step 3: Summarize commits and remaining manual checks**

Report:

- backup archive and bundle paths;
- feature branch name;
- commits created;
- browser routes and viewport sizes verified;
- any CTF-only customer created for form verification;
- explicit note that build/test commands were not run by user instruction.

- [ ] **Step 4: Offer local merge after verification**

Keep `feature/customer-workspace-ux` until the user chooses local merge, push/PR, keep or discard. Never delete the backup files during branch cleanup.

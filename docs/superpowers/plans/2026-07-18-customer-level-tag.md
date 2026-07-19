# Customer Level Tag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan inline. The project does not authorize subagents, builds, automated tests, or Git commits.

**Goal:** 将客户标签改为按四个客户等级配置的结构化标签，并在签约表单中通过单选标签自动判定客户等级，同时保持线索来源和渠道自动继承。

**Architecture:** 新增客户标签及客户标签关系模型，使用一个聚焦的标签领域服务统一完成选项加载、同等级校验、关系同步和客户等级更新。后台在现有客户等级编辑页内维护标签；后台客户表单和工作台任务表单复用同一个分组标签选择组件，不新增标签 CRUD API 或通用规则引擎。

**Tech Stack:** Go、Dever Model/Provider/Page JSON、PostgreSQL、React、TypeScript

---

## 文件结构

新增文件：

- `model/customer_tag.go`：客户标签配置模型。
- `model/customer_tag_relation.go`：客户与标签关联模型。
- `service/customer_tag.go`：标签读取、校验、等级推导和关系同步的唯一业务入口。
- `front/page/admin/customer_tag/update.json`：等级编辑页内嵌标签行表单。
- `front/src/nodes/show/customer-tag-selector.tsx`：后台和工作台共用的分组标签选择组件。
- `migrations/postgres/019_customer_level_tags.sql`：表结构、20 个默认标签和默认接单表单配置。

修改文件：

- `model/customer.go`：声明客户标签关系。
- `model/customer_level.go`：声明等级下的标签关系。
- `model/options.go`：增加标签、客户的 Relation 定义。
- `service/setting/customer.go`：后台客户及客户等级保存适配。
- `service/setting/form.go`：客户主字段配置不再提供可手工填写的客户等级。
- `service/work.go`：任务字段输出标签选项，并给客户详情附加 `tag_ids`。
- `service/work_form_input.go`：任务表单收集并保存结构化标签。
- `front/page/admin/customer/update.json`：使用标签选择组件并移除手工等级选择。
- `front/page/admin/customer_level/update.json`：内嵌维护当前等级标签。
- `front/src/nodes/show/work-core.ts`：补充标签选项类型。
- `front/src/nodes/show/work-auth.tsx`：识别客户标签字段类型和初始值。
- `front/src/nodes/show/work-task-form-fields.tsx`：工作任务字段复用标签选择组件。
- `front/src/plugin.ts`：注册后台标签表单节点。

## Task 1：建立标签模型和统一领域服务

**Files:**

- Create: `model/customer_tag.go`
- Create: `model/customer_tag_relation.go`
- Modify: `model/customer.go`
- Modify: `model/customer_level.go`
- Modify: `model/options.go`
- Create: `service/customer_tag.go`

- [ ] **Step 1：新增客户标签模型**

`CustomerTag` 包含 `ID`、`LevelID`、`Name`、`Status`、`Sort`、`CreatedAt`、`UpdatedAt`。索引固定为：

```go
type CustomerTagIndex struct {
	LevelName struct{} `unique:"level_id,name"`
	LevelSort struct{} `index:"level_id,status,sort,id"`
}
```

`NewCustomerTagModel` 使用表名 `crm_customer_tag`，关联 `customerLevelRelation`，按 `level_id asc,sort asc,id asc` 排序。

- [ ] **Step 2：新增客户标签关系模型**

`CustomerTagRelation` 包含 `ID`、`CustomerID`、`TagID`、`CreatedAt`。索引固定为：

```go
type CustomerTagRelationIndex struct {
	CustomerTag struct{} `unique:"customer_id,tag_id"`
	TagCustomer struct{} `index:"tag_id,customer_id,id"`
}
```

`NewCustomerTagRelationModel` 使用表名 `crm_customer_tag_relation`，并关联客户和客户标签。

- [ ] **Step 3：声明双向关系**

在客户模型增加 `tag_relations` Through 关系，在客户等级模型增加只读的 `level_tags` Through 关系。页面虚拟字段 `tags` 由 Provider 同步，不能与 Through 关系同名，否则 Dever 会在 after hook 前自动保存并删除遗漏子项；关系字段也需要与现有 `Customer.Tags string` 展示快照区分。

- [ ] **Step 4：实现统一标签选项读取**

在 `service/customer_tag.go` 定义：

```go
type CustomerTagSelection struct {
	LevelID   uint64
	LevelName string
	TagIDs    []uint64
	TagNames  []string
}

func CustomerTagOptions(ctx context.Context) []map[string]any
func CustomerTagIDs(ctx context.Context, customerID uint64) []uint64
func ResolveCustomerTagSelection(ctx context.Context, raw any) (*CustomerTagSelection, error)
func SyncCustomerTags(ctx context.Context, customerID uint64, raw any) (*CustomerTagSelection, error)
```

`ResolveCustomerTagSelection` 将输入归一化为去重 ID，校验只能提交一个启用标签。多标签请求直接拒绝，错误文案固定为“客户标签只能选择一个”。

`SyncCustomerTags` 先完成全部校验，再删除该客户旧关联并写入新关联，最后更新客户 `level_id`、逗号分隔的标签名称快照和 `updated_at`。没有标签时清空关系和名称快照，但不猜测新的客户等级。

- [ ] **Step 5：格式化 Go 文件**

Run:

```bash
gofmt -w model/customer_tag.go model/customer_tag_relation.go model/customer.go model/customer_level.go model/options.go service/customer_tag.go
```

Expected: 命令退出码为 0。

## Task 2：在后台配置等级标签和编辑客户标签

**Files:**

- Create: `front/page/admin/customer_tag/update.json`
- Modify: `front/page/admin/customer_level/update.json`
- Modify: `front/page/admin/customer/update.json`
- Modify: `service/setting/customer.go`
- Modify: `service/setting/form.go`
- Create: `front/src/nodes/show/customer-tag-selector.tsx`
- Modify: `front/src/plugin.ts`

- [ ] **Step 1：扩展客户等级编辑表单**

给 `customer_level/update.json` 的 `form` 增加 `_fields: ["name", "tags"]`、`service: "crm.setting.CrmHook.BuildCustomerLevelForm"` 和 `tags: []`。页面在等级名称后增加：

```json
{
  "type": "form-array",
  "name": "等级标签",
  "value": "form.tags",
  "mode": "form",
  "meta": {
    "formLayout": "horizontal",
    "pageRoute": "/crm/customer_tag/update",
    "addText": "添加标签",
    "drag": "sort"
  }
}
```

- [ ] **Step 2：增加内嵌标签行表单**

`customer_tag/update.json` 只提供标签名称输入；隐藏的 `status`、`sort` 沿用数组行默认值。提交前使用 `crm.setting.CrmHook.BeforeSaveCustomerTag`，不增加独立列表菜单。

- [ ] **Step 3：复用等级表单的批量保存模式**

在 `service/setting/customer.go` 增加：

```go
func (CrmHook) ProviderBuildCustomerLevelForm(c *server.Context, params []any) any
func (CrmHook) ProviderBeforeSaveCustomerTag(_ *server.Context, params []any) any
```

扩展 `ProviderBeforeSaveCustomerLevel`，统一校验并规范化标签行，但保留 `tags` 供保存后 Provider 使用。增加 `ProviderAfterSaveCustomerLevel`，通过 `savedRecordID` 取得新建或更新后的等级 ID，再按 `level_id` 替换标签。标签行只允许名称、状态和排序，不接收前端传入的其他等级 ID。`customer_level/update.json` 的 `action.submit.after` 调用该 Provider，保证新增等级和已有等级使用同一条同步路径。

- [ ] **Step 4：创建共享分组标签选择组件**

`customer-tag-selector.tsx` 导出两个层次：

```tsx
export type CustomerTagOption = {
  id: string;
  name: string;
  levelID: string;
  levelName: string;
  levelSort: number;
  sort: number;
};

export function CustomerTagSelector(props: {
  options: CustomerTagOption[];
  value: string[];
  disabled?: boolean;
  error?: string;
  onChange: (tagIDs: string[]) => void;
}): ReactElement;

export function ShowCrmCustomerTagSelector(props: WorkNodeProps): ReactElement;
```

组件按等级排序分组，所有标签共用单选语义。点击任意标签都将当前值替换为该标签 ID，选中项使用高对比实底样式和勾选标记。控件下方显示“自动判定：高意向”等当前等级，不提供等级选择器。

- [ ] **Step 5：注册后台节点并调整客户编辑页**

在 `front/src/plugin.ts` 注册 `form-crm-customer-tags`。客户编辑页：

- 删除 `form.level_id` 手工选择节点。
- 把 `form.tags` 文本输入替换为 `form-crm-customer-tags`，值改成 `form.tag_ids`。
- `data.form` 增加 `tag_ids: []`、`tag_options: []` 和 `service: "crm.setting.CrmHook.BuildCustomerForm"`。
- 标签节点通过 `meta.optionsPath: "form.tag_options"` 读取完整的 `id,name,level_id,level_name,level_sort,sort`，避免通用 option 归一化丢失分组元数据。
- 来源、渠道保持现有选择能力；签约默认表单的只读属性由迁移单独控制。

- [ ] **Step 6：后台保存统一调用标签领域服务**

`ProviderBuildCustomerForm` 写入全部 `tag_options`，并在有客户 ID 时写入 `tag_ids`。`ProviderBeforeSaveCustomer` 在收到 `tag_ids` 时调用 `ResolveCustomerTagSelection`，将派生的 `tags` 和 `level_id` 放入客户记录；保留 `tag_ids` 供 after hook 使用，标准 Model 保存会自动过滤这个虚拟字段。实现现有页面已经声明的 `ProviderAfterSaveCustomer`，通过 `savedRecordID` 获取新增或更新后的客户 ID，再调用 `SyncCustomerTags`；没有提交 `tag_ids` 的 partial save 不改标签关系。

从 `collectMainFields` 的客户主字段中移除 `level_id`，保留 `tags` 作为可配置标签字段，避免新资料模板继续加入手工客户等级。

- [ ] **Step 7：格式化并检查 JSON 语法**

Run:

```bash
gofmt -w service/setting/customer.go service/setting/form.go
jq empty front/page/admin/customer_tag/update.json front/page/admin/customer_level/update.json front/page/admin/customer/update.json
```

Expected: 所有命令退出码为 0。

## Task 3：接入签约任务表单

**Files:**

- Modify: `service/work.go`
- Modify: `service/work_form_input.go`
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-task-form-fields.tsx`

- [ ] **Step 1：给客户详情附加结构化标签**

在 `workCustomerRow`、完成客户详情及线索详情共用的客户数据装配路径中写入：

```go
customer["tag_ids"] = CustomerTagIDs(ctx, customerID)
```

客户列表仍只返回紧凑字段；打开任务时现有 `customer_detail` 请求会取得完整 `tag_ids`。

- [ ] **Step 2：输出任务标签选项**

`mainFieldInputType("tags")` 返回 `customer_tags`，`mainFieldOptions("tags")` 调用 `CustomerTagOptions`。每个选项包含标签和所属等级信息，前端不再解析带“高意向-”前缀的名称。

来源、渠道仍通过现有 `mainFieldOptions` 和 `workEntityFieldValue` 初始化，不创建新字段。

- [ ] **Step 3：初始化标签任务字段**

在 `workMainFieldAliases` 增加：

```ts
tags: ["tag_ids", "tags"]
```

`workTaskFieldRenderConfig` 遇到 `customer_tags` 时返回专用渲染类型 `show-crm-work-customer-tags`，并将后端选项映射为 `CustomerTagOption`。已有结构化 ID 直接回显；仅有历史标签名称快照时，允许按完全相同的标签名匹配，不能模糊猜测。

- [ ] **Step 4：工作任务字段复用选择组件**

在 `work-task-form-fields.tsx` 的 `taskFieldRenderers` 注册 `show-crm-work-customer-tags`，调用 `CustomerTagSelector`。值始终保持字符串 ID 数组，错误状态沿用任务表单现有必填错误展示。

标签选择器自身展示自动判定等级，因此默认接单表单不再渲染独立 `level_id` 字段。

- [ ] **Step 5：收集并保存标签关系**

扩展 `workFormInput`：

```go
customerTagIDs       []uint64
customerTagsProvided bool
```

收集 `main:tags` 时不再写入 `customerFields["tags"]`，而是记录结构化 ID 和是否提交。`mergeWorkFormInput` 同步合并这两个字段。

`saveWorkFormInput` 先保存普通客户字段，再在 `customerTagsProvided` 为真时调用 `SyncCustomerTags`。完成任务时空标签由现有 required 校验拒绝；保存进度时未提交标签不改现有关系。

- [ ] **Step 6：格式化前后端源码**

Run:

```bash
gofmt -w service/work.go service/work_form_input.go
```

Expected: 命令退出码为 0。TypeScript 只做人工静态复核，不运行构建或类型测试。

## Task 4：增加幂等配置迁移

**Files:**

- Create: `migrations/postgres/019_customer_level_tags.sql`

- [ ] **Step 1：创建两张标签表**

迁移使用 `CREATE TABLE IF NOT EXISTS` 和幂等索引建立 `gjj_crm_customer_tag`、`gjj_crm_customer_tag_relation`。外键只按项目现有约定处理，不额外引入级联删除。

- [ ] **Step 2：写入图 2 的 20 个默认标签**

按等级名称查找现有四个等级并幂等写入：

- 高意向：即将逾期、已经逾期、资不抵债。
- 中意向：不认可方案、产权存在争议、单纯咨询/了解观望、房产已查封、房租过低、自己无法做主、资可抵债。
- 无意向：无保房需求、潜在沟通、非保房业务。
- 联系不上：首次未接通、二次未接通、三次未接通、四次未接通、五次未接通（无效）、关机/停机/空号、联系方式不统一。

标签名称不带等级前缀；等级关系由 `level_id` 表达。

- [ ] **Step 3：调整默认接单建档表单**

按表单名“接单建档”和 `main_field` 更新：

- `source_id`、`channel_id`：`readonly = TRUE`。
- `tags`：`required = TRUE`、`readonly = FALSE`。
- 删除该表单的 `level_id` 字段，由标签选择器显示自动判定等级。

不修改线索确认表单；MKT 仍可在线索阶段录入或修正来源、渠道。

- [ ] **Step 4：保留迁移但不执行**

本轮只创建迁移脚本。未经用户再次明确授权，不对当前 PostgreSQL 数据库执行 `019_customer_level_tags.sql`。

## Task 5：静态验证和范围复核

**Files:**

- Review: 本计划列出的全部文件。

- [ ] **Step 1：运行 Dever 静态审计**

Run:

```bash
bash /root/.agents/skills/shemic-dever/scripts/audit.sh /data/project/demo/gjj/package/crm/model
bash /root/.agents/skills/shemic-dever/scripts/audit.sh /data/project/demo/gjj/package/crm/service
bash /root/.agents/skills/shemic-dever/scripts/audit.sh /data/project/demo/gjj/package/crm/front/page
bash /root/.agents/skills/shemic-dever/scripts/audit.sh /data/project/demo/gjj/package/crm/front/src
```

Expected: 四次均输出 `dever skill audit 通过`。

- [ ] **Step 2：检查格式和设计边界**

Run:

```bash
git diff --check -- model service front migrations/postgres/019_customer_level_tags.sql
```

Expected: 命令退出码为 0。确认没有修改生成文件或编译产物，没有新增标签 CRUD API、通用规则引擎、数据模板标签字段或 Git 提交。

- [ ] **Step 3：交付人工验收清单**

由用户在运行中的 `http://127.0.0.1:8082` 手工验证：后台等级标签维护、签约标签单选及替换、选中效果、等级即时显示、保存后回显、后端拒绝多标签、来源渠道继承和默认只读。

按照项目要求不运行 `npm run build`、任何自动化测试或等价命令。

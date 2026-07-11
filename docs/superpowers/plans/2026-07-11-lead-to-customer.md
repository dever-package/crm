# Lead To Customer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 MKT 工作台实现手工录入线索、查重、判无效、确认重复和转客户，并继续复用现有派单至 NPL 流程。

**Architecture:** 使用独立 `Lead` 主表保存线索状态和检索字段，使用 `record_json` 保存现有可配置录入表单的完整值。任务 1 继续复用动态资料模板，但提交动作改为创建线索；转化服务在事务内调用已有 `executeCreateCustomerTask`，不维护第二套客户创建逻辑。

**Tech Stack:** Go、Dever ORM/Service/API、PostgreSQL、React/TypeScript、Dever front plugin/page JSON。

**Verification constraint:** 用户明确禁止 `npm run build`、`go test` 及任何测试命令。本计划只使用格式检查、JSON 解析、Dever audit、运行日志、HTTP 健康检查和 Playwright 浏览器实操验证。

---

### Task 1: 线索与无效原因模型

**Files:**
- Create: `model/lead.go`
- Create: `model/lead_code.go`
- Create: `model/lead_invalid_reason.go`
- Modify: `model/options.go`

- [ ] **Step 1: 定义状态和线索模型**

在 `model/lead.go` 定义 `pending`、`invalid`、`duplicate`、`converted` 四个状态。`Lead` 只保存线索编号、姓名、联系方式、来源渠道、外部线索 ID、城市、初始诉求、状态、重复对象、无效原因、转化客户、负责人/操作人、`record_json` 和时间字段。

手机号和微信号只建普通索引，`Code` 建唯一索引，状态负责人、来源外部 ID 和客户 ID 建查询索引。

- [ ] **Step 2: 定义无效原因配置**

在 `model/lead_invalid_reason.go` 定义名称、状态、排序和创建时间，Seeds 提供空号/错号、测试数据、非目标客户、区域不符。

- [ ] **Step 3: 生成线索编号**

在 `model/lead_code.go` 实现：

```go
func GenerateUniqueLeadCode(ctx context.Context) (string, error)
```

编号使用 `L` 加日期和六位随机数，并查询冲突后返回。

- [ ] **Step 4: 集中配置 Options 和 Relations**

在 `model/options.go` 增加线索状态选项，以及来源、渠道、客户、重复线索、无效原因、负责人等 Relations，页面不重复硬编码。

- [ ] **Step 5: 检查 Go 格式**

```bash
gofmt -d model/lead.go model/lead_code.go model/lead_invalid_reason.go model/options.go
```

预期：无输出。

### Task 2: 线索录入、去重和转化服务

**Files:**
- Create: `service/work_lead.go`
- Modify: `service/work_task_execute.go`
- Modify: `service/work.go`

- [ ] **Step 1: 增加 MKT 权限判断**

通过当前人员 `DepartmentID` 查询启用部门，并判断部门编码为 `MKT`。所有线索写操作都在服务端校验。

- [ ] **Step 2: 复用动态表单创建线索**

实现：

```go
func executeCreateLeadTask(
    ctx context.Context,
    staff *WorkStaffSession,
    task *crmmodel.Task,
    values map[string]any,
) (map[string]any, error)
```

调用 `collectWorkCreateFormInput` 复用必填校验，从 `customerFields` 提取固定字段，将完整 `values` 保存到 `record_json`。继续复用 `validateWorkCustomerContact`，但不拒绝重复线索。

- [ ] **Step 3: 实现统一去重函数**

```go
type workLeadDuplicate struct {
    LeadID     uint64
    CustomerID uint64
    Reason     string
}
```

按已有客户、其他线索、来源外部 ID 检查精确手机号/微信号。保存疑似重复对象和依据，仍允许线索入库。

- [ ] **Step 4: 切换任务 1 的创建行为**

`executeWorkTask` 的 create 分支读取 `task.ConfigJSON.lead_entry_enabled`。为真时调用 `executeCreateLeadTask`，否则保留 `executeCreateCustomerTask`，不影响其他创建任务。

- [ ] **Step 5: 实现线索列表**

在 `WorkService` 增加 `Leads(ctx, staff, payload)`，支持关键词和状态筛选，并附加来源、渠道、无效原因、关联客户显示值。非 MKT 返回 `enabled:false` 和空列表。

- [ ] **Step 6: 实现状态动作入口**

增加 `ActOnLead(ctx, staff, payload)`，支持 `invalid`、`duplicate`、`reopen`、`convert`。无效必须选择启用原因，确认重复必须存在疑似对象，恢复会清理终态字段并重新检查重复。

- [ ] **Step 7: 事务内复用客户创建流程**

`convert` 再次查重，读取 `record_json` 后调用现有：

```go
executeCreateCustomerTask(
    txCtx,
    staff,
    leadEntryTask,
    mapFromAny(lead.RecordJSON),
    newWorkExecutionRuntime(),
)
```

成功后写回客户 ID、转化人和转化时间。已转化线索直接返回原客户 ID，避免重复创建。

- [ ] **Step 8: 检查 Go 格式**

```bash
gofmt -d service/work_lead.go service/work_task_execute.go service/work.go
```

预期：无输出。

### Task 3: Work API 和选项

**Files:**
- Modify: `api/work.go`
- Modify: `service/work.go`

- [ ] **Step 1: 增加薄 API**

在 `api/work.go` 增加 `GetLeads` 和 `PostLeadAction`。API 只解析输入、获取当前人员、调用 Service 并返回 `crmJSON`。

- [ ] **Step 2: 扩展工作台选项**

在 `WorkService.Options` 返回启用的来源、渠道和无效原因，供筛选与弹窗复用。

- [ ] **Step 3: 检查 Go 格式**

```bash
gofmt -d api/work.go service/work.go
```

预期：无输出。

### Task 4: MKT 线索池界面

**Files:**
- Create: `front/src/nodes/show/work-lead.tsx`
- Modify: `front/src/nodes/show/work-core.ts`
- Modify: `front/src/plugin.ts`
- Modify: `front/page/work/work.json`

- [ ] **Step 1: 定义线索前端类型**

在 `work-core.ts` 增加 `WorkLead`、筛选和无效原因类型，复用 `workApi`、`textValue`、`formatWorkDate`、`workRefreshEvent`。

- [ ] **Step 2: 实现独立线索池组件**

`work-lead.tsx` 加载 `/crm/work/leads`，`enabled:false` 时返回 `null`。桌面使用紧凑表格，移动端使用单列记录块，显示联系方式、来源渠道、状态、重复提示和时间。

- [ ] **Step 3: 实现筛选与动作**

提供关键词、状态筛选和刷新。待处理线索可转客户、判无效、确认重复；无效/重复可恢复；已转化显示关联客户。判无效使用原因选择弹窗，转化和确认重复使用确认弹窗。

提交统一调用 `/crm/work/lead_action`，成功后触发 `workRefreshEvent`，同步刷新线索和客户列表。

- [ ] **Step 4: 注册并加入工作台**

在 `plugin.ts` 注册 `show-crm-work-lead-table`，在 `work.json` 客户表格前插入节点。非 MKT 页面保持原样。

- [ ] **Step 5: 检查 JSON 和差异**

```bash
jq empty front/page/work/work.json
git diff --check
```

预期：无输出。

### Task 5: 后台页面和迁移

**Files:**
- Create: `front/page/admin/lead/list.json`
- Create: `front/page/admin/lead_invalid_reason/list.json`
- Create: `front/page/admin/lead_invalid_reason/update.json`
- Create: `migrations/postgres/011_lead_to_customer.sql`

- [ ] **Step 1: 增加只读线索列表**

使用标准 Model + page JSON 展示线索、来源、状态、重复/无效原因、关联客户和时间，不提供直接改状态入口。

- [ ] **Step 2: 增加无效原因 CRUD**

使用标准 list/update page JSON，只录入名称、状态和排序，不新增 CRUD API/Service。

- [ ] **Step 3: 编写幂等迁移**

创建两张表和索引，插入默认原因，并将 S01 的创建任务改名为“录入线索”，把 `lead_entry_enabled:true` 合并进 `config_json`。不回填历史客户。

- [ ] **Step 4: 应用并复跑迁移**

连续运行两次。第一次创建/更新，第二次不报错且不产生重复原因。

- [ ] **Step 5: 执行静态检查**

```bash
rg --files -g '*.json' | xargs -r -n1 jq empty
git diff --check
bash /root/.agents/skills/shemic-dever/scripts/audit.sh --changed .
```

预期：JSON 和 diff 无输出，Dever audit 无错误。

### Task 6: 浏览器全链路验证

**Files:**
- No repository files expected.

- [ ] **Step 1: 确认 Dever 热更新成功**

检查现有 `dever run` 日志和 8082 健康状态，不手改生成文件。

- [ ] **Step 2: 验证正常转化**

MKT 登录后录入新手机号线索，在线索池转为客户，确认客户列表出现且线索变成已转化。

- [ ] **Step 3: 验证重复与无效**

再次录入相同手机号，确认线索被保留并提示重复；确认重复后客户数量不增加。再录入另一条线索，判无效并恢复。

- [ ] **Step 4: 验证后续派单**

对转化客户执行现有“派单至 NPL”，使用 NPL 账号确认能看到接单首呼任务。

- [ ] **Step 5: 验证权限和响应式布局**

非 MKT 账号不显示线索池；桌面和 390px 移动视口无控制台错误、横向溢出或文字按钮重叠。

- [ ] **Step 6: 最终检查**

运行格式检查、JSON 解析、`git diff --check`、Dever audit 和 HTTP 200 检查，并明确记录未运行 build/test。

# CRM Admin Data Permissions And Product Display Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 限制后台客户数据只能编辑，并修复产品分类和签约后流程的展示。

**Architecture:** 只调整现有 page JSON 节点和 Model/Provider 文案，保留现有编辑路由与保存逻辑。产品分类直接使用 Model Relation 已输出的 `category.name`，不新增展示 Provider。

**Tech Stack:** Dever page JSON, Go Model/Provider

---

### Task 1: 收紧客户与资产后台入口

**Files:**
- Modify: `front/page/admin/customer/list.json`
- Modify: `front/page/admin/customer_asset/list.json`

- [ ] **Step 1: 移除客户新增入口**

在 `customer/list.json` 删除“新增客户” `show-button`、`dialog.create` 弹窗与 state，保留“编辑客户”按钮和 `dialog.edit`。

- [ ] **Step 2: 移除资产新增入口**

在 `customer_asset/list.json` 删除“新增资产” `show-button`、`dialog.create` 弹窗与 state，保留“编辑资产”按钮和 `dialog.edit`。

- [ ] **Step 3: 移除资产详情入口**

从资产列表操作按钮中删除：

```json
{
  "icon": "eye",
  "description": "查看资产详情",
  "to": "/crm/customer_asset/detail"
}
```

保留 `front/page/admin/customer_asset/detail.json` 文件，本次不扩大到路由删除。

- [ ] **Step 4: 检查页面再无创建状态**

Run:

```bash
rg -n 'dialog\.create|新增客户|新增资产|查看资产详情' \
  front/page/admin/customer/list.json \
  front/page/admin/customer_asset/list.json
```

Expected: 无输出。

- [ ] **Step 5: 提交后台入口调整**

```bash
git add front/page/admin/customer/list.json front/page/admin/customer_asset/list.json
git commit -m "fix: restrict admin customer data creation"
```

### Task 2: 修复产品分类和签约后流程展示

**Files:**
- Modify: `front/page/admin/product/list.json`
- Modify: `front/page/admin/product/update.json`
- Modify: `model/product.go`
- Modify: `service/setting/product.go`
- Modify: `service/setting/workflow.go`
- Modify: `service/customer_product.go`
- Modify: `front/page/admin/customer_product/list.json`

- [ ] **Step 1: 修正产品分类 Relation 路径**

将产品列表的：

```json
"value": "product_category.name"
```

改为：

```json
"value": "category.name"
```

- [ ] **Step 2: 统一页面文案**

将产品列表、产品编辑和客户产品列表中用户可见的“服务流程”改为“签约后流程”。产品表单占位文案改为：

```json
"placeholder": "不选择则签约流程完成后结束"
```

- [ ] **Step 3: 统一 Model 与校验提示**

将 `Product.ServiceWorkflowID` 的 comment 改为“签约后流程”。将产品保存、流程保存和签约后流程启动的错误文案统一为“签约后流程”，不修改 `service_workflow_id` 字段名和运行逻辑。

- [ ] **Step 4: 格式化 Go 文件**

```bash
gofmt -w model/product.go service/setting/product.go service/setting/workflow.go service/customer_product.go
```

- [ ] **Step 5: 提交产品展示修复**

```bash
git add front/page/admin/product/list.json front/page/admin/product/update.json \
  front/page/admin/customer_product/list.json model/product.go \
  service/setting/product.go service/setting/workflow.go service/customer_product.go
git commit -m "fix: clarify product post-signing workflow"
```

### Task 3: 静态验证

**Files:**
- Verify: all files changed in Task 1 and Task 2

- [ ] **Step 1: 解析全部后台 JSON**

```bash
find front/page -name '*.json' -print0 | xargs -0 -n1 jq empty
```

Expected: exit code 0, 无输出。

- [ ] **Step 2: 执行 Dever 静态审计**

```bash
bash /root/.agents/skills/shemic-dever/scripts/audit.sh --changed .
```

Expected: `dever skill audit 通过`。

- [ ] **Step 3: 检查补丁完整性**

```bash
git diff --check
git status --short
```

Expected: `git diff --check` 无输出；除实施计划进度外无未提交业务变更。

> 按项目 `AGENTS.md` 要求，不运行 `npm run build`、`dever build`、`go test` 或其它测试命令。

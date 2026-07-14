# Lead Owner Assignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan inline. The user explicitly forbids subagents, builds and automated tests.

**Goal:** 让部门负责人和管理员可以在线索池把未完成线索分配给当前负责部门的启用人员。

**Architecture:** 后端在现有工作流权限层增加统一的负责人改派策略，并继续复用现有候选人查询和负责人变更事务。前端提取一个独立的负责人选择弹窗，供流程详情和线索行共同使用，避免重复请求与表单逻辑。

**Tech Stack:** Go、Dever Service/API、React、TypeScript

---

### Task 1: 统一部门负责人改派权限

**Files:**
- Modify: `service/work.go`
- Modify: `service/workflow_access.go`
- Modify: `service/work_flow_actions.go`
- Modify: `service/workflow_assignment.go`

- [ ] **Step 1: 将人员类型带入工作台会话**

在 `WorkStaffSession` 增加 `StaffType string`，并在 `CurrentWorkStaff`、`workStaffPayload`、`workStaffSessionPayload` 中从 `crmmodel.Staff.StaffType` 读取或返回该值。现有 `CanDispatch` 继续表示管理员级流程调度权限。

- [ ] **Step 2: 提取统一改派权限函数**

在 `service/workflow_access.go` 增加：

```go
func canChangeWorkflowOwner(staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	if staff == nil || staff.ID == 0 || instance == nil || instance.Status != crmmodel.ProgressStatusActive {
		return false
	}
	if staff.CanDispatch {
		return true
	}
	return staff.StaffType == crmmodel.StaffTypeLeader &&
		staff.DepartmentID > 0 && staff.DepartmentID == instance.OwnerDepartmentID
}
```

`canViewWorkflowInstance` 在原负责人和本人待办之外复用该函数，使部门负责人可以看到本部门活动流程，但不修改 `canManageLeadWorkflow`，确保其不能处理别人名下的线索。

- [ ] **Step 3: 三处业务入口复用同一权限**

- `WorkService.FlowAssignees` 的 `target=current_owner` 分支使用 `canChangeWorkflowOwner`。
- `ChangeWorkflowInstanceOwner` 读取活动流程后使用 `canChangeWorkflowOwner`，错误文案改为“只有当前负责部门负责人或流程调度员可以更换负责人”。
- `workFlowDetail` 的 `can_change_owner` 使用 `canChangeWorkflowOwner`。

目标人员仍通过 `enabledStaffInDepartment` 校验，负责人变更事务继续调用 `reassignStageOwnerTodos` 并同步线索 owner 字段。

- [ ] **Step 4: 格式化 Go 文件**

Run:

```bash
gofmt -w service/work.go service/workflow_access.go service/work_flow_actions.go service/workflow_assignment.go
```

Expected: 命令退出码为 0，不运行 `go test` 或构建。

### Task 2: 提取可复用负责人分配弹窗

**Files:**
- Create: `front/src/nodes/show/work-flow-owner-dialog.tsx`
- Modify: `front/src/nodes/show/work-flow-actions.tsx`

- [ ] **Step 1: 创建负责人选择弹窗**

新增 `WorkFlowOwnerDialog`，接口保持聚焦：

```tsx
type WorkFlowOwnerDialogProps = {
  flow?: WorkFlowDetail | null;
  open: boolean;
  title?: string;
  onOpenChange: (open: boolean) => void;
};
```

弹窗打开后调用：

```ts
/crm/work/flow_assignees?workflow_instance_id=${workflowInstanceID}&target=current_owner
```

提交继续调用 `/crm/work/change_flow_owner`。候选项使用“姓名（流程 N / 待办 N）”，成功后关闭弹窗、发送 `workRefreshEvent`，失败时保留弹窗并显示服务端错误。

- [ ] **Step 2: 流程详情复用新弹窗**

`WorkFlowActions` 删除 `owner` picker 分支及对应重复请求，把“更换负责人”按钮改为打开 `WorkFlowOwnerDialog`。任务分配和选择下一阶段负责人继续使用原有内联 picker，不改变其行为。

### Task 3: 在线索行增加直接分配入口

**Files:**
- Modify: `front/src/nodes/show/work-lead.tsx`

- [ ] **Step 1: 接入分配状态和共享弹窗**

在线索池组件增加 `assignLead` 状态，在根部渲染：

```tsx
<WorkFlowOwnerDialog
  flow={assignLead?.flow}
  open={Boolean(assignLead)}
  title="分配线索"
  onOpenChange={(open) => !open && setAssignLead(null)}
/>
```

刷新事件同时清理 `assignLead`，避免分配完成后保留旧流程对象。

- [ ] **Step 2: 增加单条线索分配按钮**

扩展 `WorkLeadRowProps` 和桌面、移动端共用的 `LeadActions`。仅当线索状态为 `pending` 且 `flow.can_change_owner` 为真时显示“分配”，点击后传递当前线索；不把 `can_change_owner` 合并到 `canManage`，避免部门负责人获得编辑、判无效或转客户按钮。

### Task 4: 静态验证与范围复核

**Files:**
- Review: `service/work.go`
- Review: `service/workflow_access.go`
- Review: `service/work_flow_actions.go`
- Review: `service/workflow_assignment.go`
- Review: `front/src/nodes/show/work-flow-owner-dialog.tsx`
- Review: `front/src/nodes/show/work-flow-actions.tsx`
- Review: `front/src/nodes/show/work-lead.tsx`

- [ ] **Step 1: 运行 Dever 静态审计**

Run:

```bash
bash /root/.agents/skills/shemic-dever/scripts/audit.sh /data/project/demo/gjj/package/crm/service
bash /root/.agents/skills/shemic-dever/scripts/audit.sh /data/project/demo/gjj/package/crm/front/src/nodes/show
```

Expected: 两次均输出 `dever skill audit 通过`。

- [ ] **Step 2: 检查差异格式和实现边界**

Run:

```bash
git diff --check -- service/work.go service/workflow_access.go service/work_flow_actions.go service/workflow_assignment.go front/src/nodes/show/work-flow-owner-dialog.tsx front/src/nodes/show/work-flow-actions.tsx front/src/nodes/show/work-lead.tsx
```

Expected: 命令退出码为 0。确认没有新增 API、模型、跨部门候选人或批量分配逻辑；按照项目要求不运行 build 或任何自动化测试。


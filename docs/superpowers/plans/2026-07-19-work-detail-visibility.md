# Work Detail Visibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure any staff member whose assigned workflow record appears in the workbench can open its lead, customer, and asset details after completing the task.

**Architecture:** Keep one shared read-access rule in `service/workflow_access.go`. Treat any task assignment on the workflow instance as participation, regardless of todo status, while leaving every mutation and task-execution permission unchanged.

**Tech Stack:** Go, Dever ORM, CRM work service

**Verification Constraint:** Do not run builds or automated tests. Use source tracing, `gofmt`, the Dever static audit, and diff checks only.

---

### Task 1: Align workflow detail access with list visibility

**Files:**
- Modify: `service/workflow_access.go:92`

- [ ] **Step 1: Update the shared participation check**

Remove the todo status filter from `canViewAssignedWorkflowInstance` so both pending and completed assignees are recognized:

```go
return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
	"workflow_instance_id": instance.ID,
	"assignee_staff_id":    staff.ID,
}) > 0
```

- [ ] **Step 2: Confirm read and write permission boundaries**

Use source search to confirm lead/customer/asset detail paths call `canViewWorkflowInstance` or `canViewWorkflowInstanceInScope`, while task execution still calls `canOperateWorkTodo` and workflow mutations retain their existing owner/dispatcher checks.

- [ ] **Step 3: Run allowed static checks**

Run `gofmt` on `service/workflow_access.go`, the Dever static audit for that file, and `git diff --check`. Do not run build or test commands.

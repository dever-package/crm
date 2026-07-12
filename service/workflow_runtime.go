package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

var ErrNoAvailableWorkflow = errors.New("未配置可用流程")

func StartAssetWorkflow(ctx context.Context, customerID, assetID uint64) error {
	return orm.Transaction(ctx, func(txCtx context.Context) error {
		return startAssetWorkflow(txCtx, customerID, assetID)
	})
}

func startAssetWorkflow(ctx context.Context, customerID, assetID uint64) error {
	if customerID == 0 || assetID == 0 {
		return fmt.Errorf("客户和资产不能为空")
	}
	if crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{
		"id":          assetID,
		"customer_id": customerID,
	}) == nil {
		return fmt.Errorf("客户资产不存在")
	}

	progressModel := crmmodel.NewCustomerStageModel()
	if progressModel.Find(ctx, map[string]any{"asset_id": assetID}) != nil {
		return nil
	}
	workflow, stage := firstAvailableWorkflowStage(ctx)
	if workflow == nil || stage == nil {
		return ErrNoAvailableWorkflow
	}

	now := time.Now()
	progressID := uint64(progressModel.Insert(ctx, map[string]any{
		"customer_id":         customerID,
		"asset_id":            assetID,
		"workflow_id":         workflow.ID,
		"stage_id":            stage.ID,
		"owner_department_id": stage.OwnerDepartmentID,
		"owner_staff_id":      uint64(0),
		"status":              crmmodel.ProgressStatusActive,
		"started_at":          now,
		"updated_at":          now,
	}))
	if progressID == 0 {
		return fmt.Errorf("资产流程启动失败")
	}
	progress := progressModel.Find(ctx, map[string]any{"id": progressID})
	if progress == nil {
		return fmt.Errorf("资产流程启动失败")
	}
	recordWorkStageChange(ctx, progress, 0, stage.ID, "entered")
	return createStageTodos(ctx, progress, stage)
}

func firstAvailableWorkflowStage(ctx context.Context) (*crmmodel.Workflow, *crmmodel.Stage) {
	workflows := crmmodel.NewWorkflowModel().Select(
		ctx,
		map[string]any{"status": crmmodel.StatusEnabled},
		map[string]any{"order": "sort asc,id asc"},
	)
	for _, workflow := range workflows {
		if workflow == nil {
			continue
		}
		if stage := firstEnabledStage(ctx, workflow.ID); stage != nil {
			return workflow, stage
		}
	}
	return nil, nil
}

func enterWorkflowStage(ctx context.Context, progress *crmmodel.CustomerStage, workflow *crmmodel.Workflow, stage *crmmodel.Stage) error {
	if progress == nil || workflow == nil || stage == nil {
		return fmt.Errorf("流程阶段不能为空")
	}
	now := time.Now()
	updated := crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"id": progress.ID,
	}, map[string]any{
		"workflow_id":         workflow.ID,
		"stage_id":            stage.ID,
		"owner_department_id": stage.OwnerDepartmentID,
		"owner_staff_id":      uint64(0),
		"status":              crmmodel.ProgressStatusActive,
		"started_at":          now,
		"updated_at":          now,
	})
	if updated == 0 {
		return fmt.Errorf("资产流程阶段更新失败")
	}
	progress.WorkflowID = workflow.ID
	progress.StageID = stage.ID
	progress.OwnerDepartmentID = stage.OwnerDepartmentID
	progress.OwnerStaffID = 0
	progress.Status = crmmodel.ProgressStatusActive
	progress.StartedAt = now
	progress.UpdatedAt = now
	return createStageTodos(ctx, progress, stage)
}

func createStageTodos(ctx context.Context, progress *crmmodel.CustomerStage, stage *crmmodel.Stage) error {
	if progress == nil || stage == nil {
		return fmt.Errorf("资产进度和阶段不能为空")
	}
	tasks := crmmodel.NewTaskModel().Select(ctx, map[string]any{
		"stage_id": stage.ID,
		"status":   crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	now := time.Now()
	ruleTodos := make([]struct {
		todo *crmmodel.WorkTodo
		task *crmmodel.Task
	}, 0)
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
			"asset_id": progress.AssetID,
			"stage_id": stage.ID,
			"task_id":  task.ID,
		}) != nil {
			continue
		}
		departmentID, staffID, err := resolveTaskAssignee(ctx, stage, task)
		if err != nil {
			return err
		}
		var dueAt *time.Time
		if task.DueDays > 0 {
			deadline := now.AddDate(0, 0, task.DueDays)
			dueAt = &deadline
		}
		todoID := uint64(crmmodel.NewWorkTodoModel().Insert(ctx, map[string]any{
			"customer_id":            progress.CustomerID,
			"asset_id":               progress.AssetID,
			"workflow_id":            progress.WorkflowID,
			"stage_id":               stage.ID,
			"task_id":                task.ID,
			"assignee_department_id": departmentID,
			"assignee_staff_id":      staffID,
			"required":               task.Required,
			"status":                 crmmodel.WorkTodoStatusPending,
			"due_at":                 dueAt,
			"result":                 "",
			"created_at":             now,
			"updated_at":             now,
		}))
		if todoID == 0 {
			return fmt.Errorf("阶段待办创建失败")
		}
		if task.TaskType == crmmodel.TaskTypeRule {
			ruleTodos = append(ruleTodos, struct {
				todo *crmmodel.WorkTodo
				task *crmmodel.Task
			}{
				todo: crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{"id": todoID}),
				task: task,
			})
		}
	}
	for _, current := range ruleTodos {
		if err := executePendingRuleTodo(ctx, current.todo, current.task); err != nil {
			return err
		}
	}
	return advanceAssetProgressIfReady(ctx, progress.ID)
}

func resolveTaskAssignee(ctx context.Context, stage *crmmodel.Stage, task *crmmodel.Task) (uint64, uint64, error) {
	if stage == nil || task == nil {
		return 0, 0, fmt.Errorf("阶段和任务不能为空")
	}
	switch task.AssigneeMode {
	case crmmodel.TaskAssigneeStage:
		if enabledDepartment(ctx, stage.OwnerDepartmentID) {
			return stage.OwnerDepartmentID, 0, nil
		}
		return 0, 0, fmt.Errorf("阶段负责部门不存在或已停用")
	case crmmodel.TaskAssigneeDepartment:
		if enabledDepartment(ctx, task.AssigneeDepartmentID) {
			return task.AssigneeDepartmentID, 0, nil
		}
		return 0, 0, fmt.Errorf("任务负责部门不存在或已停用")
	case crmmodel.TaskAssigneeStaff:
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
			"id":     task.AssigneeStaffID,
			"status": crmmodel.StatusEnabled,
		})
		if staff != nil {
			return staff.DepartmentID, staff.ID, nil
		}
		return 0, 0, fmt.Errorf("任务负责人不存在或已停用")
	default:
		return 0, 0, fmt.Errorf("任务负责方式无效")
	}
}

func enabledDepartment(ctx context.Context, departmentID uint64) bool {
	return departmentID > 0 && crmmodel.NewDepartmentModel().Find(ctx, map[string]any{
		"id":     departmentID,
		"status": crmmodel.StatusEnabled,
	}) != nil
}

func advanceAssetProgressIfReady(ctx context.Context, progressID uint64) error {
	progress := crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"id":     progressID,
		"status": crmmodel.ProgressStatusActive,
	})
	if progress == nil {
		return nil
	}
	if crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"asset_id": progress.AssetID,
		"stage_id": progress.StageID,
		"required": true,
		"status":   crmmodel.WorkTodoStatusPending,
	}) > 0 {
		return nil
	}

	now := time.Now()
	crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
		"asset_id": progress.AssetID,
		"stage_id": progress.StageID,
		"required": false,
		"status":   crmmodel.WorkTodoStatusPending,
	}, map[string]any{
		"status":     crmmodel.WorkTodoStatusCanceled,
		"updated_at": now,
	})

	if stage := nextEnabledStage(ctx, progress.WorkflowID, progress.StageID); stage != nil {
		workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
			"id":     progress.WorkflowID,
			"status": crmmodel.StatusEnabled,
		})
		if workflow == nil {
			return fmt.Errorf("当前流程不存在或已停用")
		}
		recordWorkStageChange(ctx, progress, progress.StageID, stage.ID, "entered")
		return enterWorkflowStage(ctx, progress, workflow, stage)
	}

	workflow := nextEnabledWorkflow(ctx, progress.WorkflowID)
	for workflow != nil {
		if stage := firstEnabledStage(ctx, workflow.ID); stage != nil {
			auditProgress := *progress
			auditProgress.WorkflowID = workflow.ID
			recordWorkStageChange(ctx, &auditProgress, progress.StageID, stage.ID, "entered")
			return enterWorkflowStage(ctx, progress, workflow, stage)
		}
		workflow = nextEnabledWorkflow(ctx, workflow.ID)
	}

	recordWorkStageChange(ctx, progress, progress.StageID, 0, "completed")
	crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{"id": progress.ID}, map[string]any{
		"status":     crmmodel.ProgressStatusCompleted,
		"updated_at": now,
	})
	return nil
}

func nextEnabledStage(ctx context.Context, workflowID, stageID uint64) *crmmodel.Stage {
	current := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": stageID, "workflow_id": workflowID})
	if current == nil {
		return nil
	}
	stages := crmmodel.NewStageModel().Select(ctx, map[string]any{
		"workflow_id": workflowID,
		"status":      crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	for _, stage := range stages {
		if stage != nil && (stage.Sort > current.Sort || stage.Sort == current.Sort && stage.ID > current.ID) {
			return stage
		}
	}
	return nil
}

func nextEnabledWorkflow(ctx context.Context, workflowID uint64) *crmmodel.Workflow {
	current := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": workflowID})
	if current == nil {
		return nil
	}
	workflows := crmmodel.NewWorkflowModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	for _, workflow := range workflows {
		if workflow != nil && (workflow.Sort > current.Sort || workflow.Sort == current.Sort && workflow.ID > current.ID) {
			return workflow
		}
	}
	return nil
}

func firstEnabledStage(ctx context.Context, workflowID uint64) *crmmodel.Stage {
	return crmmodel.NewStageModel().Find(ctx, map[string]any{
		"workflow_id": workflowID,
		"status":      crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
}

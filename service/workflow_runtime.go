package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

var ErrNoAvailableWorkflow = errors.New("未配置可用的默认入口流程")

func StartAssetWorkflow(ctx context.Context, customerID, assetID uint64, ownerStaffID ...uint64) error {
	return orm.Transaction(ctx, func(txCtx context.Context) error {
		return startAssetWorkflow(txCtx, customerID, assetID, ownerStaffID...)
	})
}

func startAssetWorkflow(ctx context.Context, customerID, assetID uint64, ownerStaffID ...uint64) error {
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
	workflow, stage := defaultEntryWorkflowStage(ctx)
	if workflow == nil || stage == nil {
		return ErrNoAvailableWorkflow
	}
	requestedOwnerID := firstRequestedOwnerID(ownerStaffID)
	owner, err := resolveStageOwner(ctx, stage, requestedOwnerID)
	if err != nil {
		return err
	}

	now := time.Now()
	progressID := uint64(progressModel.Insert(ctx, map[string]any{
		"customer_id":         customerID,
		"asset_id":            assetID,
		"workflow_id":         workflow.ID,
		"stage_id":            stage.ID,
		"owner_department_id": owner.DepartmentID,
		"owner_staff_id":      owner.ID,
		"status":              crmmodel.ProgressStatusActive,
		"started_at":          now,
		"terminated_reason":   "",
		"updated_at":          now,
	}))
	if progressID == 0 {
		return fmt.Errorf("资产流程启动失败")
	}
	progress := progressModel.Find(ctx, map[string]any{"id": progressID})
	if progress == nil {
		return fmt.Errorf("资产流程启动失败")
	}
	if recordWorkStageChange(ctx, nil, progress, workStageChange{
		ToWorkflowID: workflow.ID,
		ToStageID:    stage.ID,
		ResultValue:  "entered",
		Title:        "流程已启动",
		Snapshot: map[string]any{
			"owner_department_id": owner.DepartmentID,
			"owner_staff_id":      owner.ID,
		},
	}) == 0 {
		return fmt.Errorf("流程启动记录创建失败")
	}
	return createStageTodos(ctx, progress, stage)
}

func defaultEntryWorkflowStage(ctx context.Context) (*crmmodel.Workflow, *crmmodel.Stage) {
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"default_entry": true,
		"status":        crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	if workflow == nil {
		return nil, nil
	}
	return workflow, firstEnabledStage(ctx, workflow.ID)
}

func firstRequestedOwnerID(ownerStaffIDs []uint64) uint64 {
	if len(ownerStaffIDs) == 0 {
		return 0
	}
	return ownerStaffIDs[0]
}

func enterWorkflowStage(
	ctx context.Context,
	progress *crmmodel.CustomerStage,
	workflow *crmmodel.Workflow,
	stage *crmmodel.Stage,
	requestedOwnerID uint64,
) (*crmmodel.Staff, error) {
	if progress == nil || workflow == nil || stage == nil {
		return nil, fmt.Errorf("流程阶段不能为空")
	}
	owner, err := resolveStageOwner(ctx, stage, requestedOwnerID)
	if err != nil {
		return nil, err
	}

	previousWorkflowID := progress.WorkflowID
	previousStageID := progress.StageID
	now := time.Now()
	updated := crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"id":          progress.ID,
		"workflow_id": previousWorkflowID,
		"stage_id":    previousStageID,
		"status":      crmmodel.ProgressStatusActive,
	}, map[string]any{
		"workflow_id":         workflow.ID,
		"stage_id":            stage.ID,
		"owner_department_id": owner.DepartmentID,
		"owner_staff_id":      owner.ID,
		"status":              crmmodel.ProgressStatusActive,
		"started_at":          now,
		"updated_at":          now,
	})
	if updated == 0 {
		return nil, fmt.Errorf("资产流程已变化，请刷新后重试")
	}
	progress.WorkflowID = workflow.ID
	progress.StageID = stage.ID
	progress.OwnerDepartmentID = owner.DepartmentID
	progress.OwnerStaffID = owner.ID
	progress.Status = crmmodel.ProgressStatusActive
	progress.StartedAt = now
	progress.UpdatedAt = now
	return owner, nil
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
		if task == nil || crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
			"asset_id": progress.AssetID,
			"stage_id": stage.ID,
			"task_id":  task.ID,
		}) != nil {
			continue
		}
		departmentID, staffID, err := resolveTaskAssignee(ctx, progress, task)
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
		createdTodo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{"id": todoID})
		if createdTodo == nil {
			return fmt.Errorf("阶段待办创建失败")
		}
		if staffID > 0 {
			assignee := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID})
			if recordWorkTodoAssignment(ctx, nil, progress, createdTodo, task, 0, assignee) == 0 {
				return fmt.Errorf("任务分配记录创建失败")
			}
		}
		if task.TaskType == crmmodel.TaskTypeRule {
			ruleTodos = append(ruleTodos, struct {
				todo *crmmodel.WorkTodo
				task *crmmodel.Task
			}{
				todo: createdTodo,
				task: task,
			})
		}
	}
	for _, current := range ruleTodos {
		if !workRuleTodoReady(ctx, current.todo, current.task) {
			continue
		}
		if err := executePendingRuleTodo(ctx, current.todo, current.task); err != nil {
			return err
		}
	}
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

func firstEnabledStage(ctx context.Context, workflowID uint64) *crmmodel.Stage {
	return crmmodel.NewStageModel().Find(ctx, map[string]any{
		"workflow_id": workflowID,
		"status":      crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
}

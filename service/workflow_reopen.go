package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
	"github.com/shemic/dever/orm"
)

type WorkflowStageReopenRequest struct {
	InstanceID   uint64
	StageID      uint64
	OwnerStaffID uint64
	Reason       string
	Snapshot     map[string]any
}

func ReopenWorkflowInstanceAtStage(ctx context.Context, req WorkflowStageReopenRequest) (*crmmodel.WorkflowInstance, error) {
	var reopened *crmmodel.WorkflowInstance
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		instance, err := activeWorkflowInstance(txCtx, req.InstanceID)
		if err != nil {
			return err
		}
		workflow := crmmodel.NewWorkflowModel().Find(txCtx, map[string]any{
			"id":     instance.WorkflowID,
			"status": crmmodel.StatusEnabled,
		})
		if workflow == nil {
			return fmt.Errorf("当前流程不存在或已停用")
		}
		currentStage := crmmodel.NewStageModel().Find(txCtx, map[string]any{
			"id":          instance.StageID,
			"workflow_id": workflow.ID,
		})
		targetStage := crmmodel.NewStageModel().Find(txCtx, map[string]any{
			"id":          req.StageID,
			"workflow_id": workflow.ID,
			"status":      crmmodel.StatusEnabled,
		})
		if currentStage == nil || targetStage == nil {
			return fmt.Errorf("当前阶段或目标阶段不存在")
		}
		if !stagePrecedes(targetStage, currentStage) {
			return fmt.Errorf("目标阶段必须早于当前阶段")
		}
		reason := strings.TrimSpace(req.Reason)
		if reason == "" {
			return fmt.Errorf("请填写重新打开原因")
		}

		stageIDs := stageIDsAtOrAfter(txCtx, workflow.ID, targetStage)
		todos := crmmodel.NewWorkTodoModel().Select(txCtx, map[string]any{
			"workflow_instance_id": instance.ID,
			"stage_id":             stageIDs,
			"status": []string{
				crmmodel.WorkTodoStatusPending,
				crmmodel.WorkTodoStatusDone,
			},
		}, map[string]any{"order": "id asc"})
		canceledTodoIDs := workTodoIDs(todos)
		if len(canceledTodoIDs) > 0 {
			updated := crmmodel.NewWorkTodoModel().Update(txCtx, map[string]any{"id": canceledTodoIDs}, map[string]any{
				"status":     crmmodel.WorkTodoStatusCanceled,
				"updated_at": time.Now(),
			})
			if updated != int64(len(canceledTodoIDs)) {
				return fmt.Errorf("流程待办已变化，请重新执行")
			}
		}

		fromWorkflowID := instance.WorkflowID
		fromStageID := instance.StageID
		owner, err := enterWorkflowStage(txCtx, instance, workflow, targetStage, req.OwnerStaffID)
		if err != nil {
			return err
		}
		ownerStaffID := uint64(0)
		if owner != nil {
			ownerStaffID = owner.ID
		}
		snapshot := map[string]any{
			"reason":            reason,
			"canceled_todo_ids": canceledTodoIDs,
			"owner_staff_id":    ownerStaffID,
		}
		for key, value := range req.Snapshot {
			snapshot[key] = value
		}
		if recordWorkStageChange(txCtx, nil, instance, workStageChange{
			FromWorkflowID: fromWorkflowID,
			FromStageID:    fromStageID,
			ToWorkflowID:   workflow.ID,
			ToStageID:      targetStage.ID,
			ResultValue:    "reopened",
			Title:          "流程重新打开：" + targetStage.Name,
			Content:        reason,
			Snapshot:       snapshot,
		}) == 0 {
			return fmt.Errorf("流程重新打开记录创建失败")
		}
		if owner != nil {
			if err := createStageTodos(txCtx, instance, targetStage); err != nil {
				return err
			}
		}
		reopened = instance
		return nil
	})
	return reopened, err
}

func stagePrecedes(target, current *crmmodel.Stage) bool {
	if target == nil || current == nil {
		return false
	}
	return target.Sort < current.Sort || target.Sort == current.Sort && target.ID < current.ID
}

func stageIDsAtOrAfter(ctx context.Context, workflowID uint64, target *crmmodel.Stage) []uint64 {
	if workflowID == 0 || target == nil {
		return nil
	}
	result := []uint64{}
	for _, stage := range crmmodel.NewStageModel().Select(ctx, map[string]any{"workflow_id": workflowID}) {
		if stage != nil && (stage.Sort > target.Sort || stage.Sort == target.Sort && stage.ID >= target.ID) {
			result = append(result, stage.ID)
		}
	}
	return result
}

func workTodoIDs(todos []*crmmodel.WorkTodo) []uint64 {
	result := make([]uint64, 0, len(todos))
	for _, todo := range todos {
		if todo != nil {
			result = append(result, todo.ID)
		}
	}
	return result
}

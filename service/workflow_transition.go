package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

func CompleteAssetStage(ctx context.Context, staff *WorkStaffSession, assetID, nextOwnerStaffID uint64) (*crmmodel.CustomerStage, error) {
	var completed *crmmodel.CustomerStage
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		progress, err := activeAssetProgress(txCtx, assetID)
		if err != nil {
			return err
		}
		if !canCompleteAssetStage(staff, progress) {
			return fmt.Errorf("只有当前负责人或流程调度员可以完成阶段")
		}
		if pendingRequiredTodoCount(txCtx, progress) > 0 {
			return fmt.Errorf("必做任务尚未全部完成")
		}

		workflow, stage, err := nextWorkflowStage(txCtx, progress)
		if err != nil {
			return err
		}
		cancelPendingOptionalTodos(txCtx, progress)
		if workflow == nil || stage == nil {
			if err := completeAssetProgress(txCtx, staff, progress); err != nil {
				return err
			}
			completed = progress
			return nil
		}

		fromWorkflowID := progress.WorkflowID
		fromStageID := progress.StageID
		owner, err := enterWorkflowStage(txCtx, progress, workflow, stage, nextOwnerStaffID)
		if err != nil {
			return err
		}
		if recordWorkStageChange(txCtx, staff, progress, workStageChange{
			FromWorkflowID: fromWorkflowID,
			FromStageID:    fromStageID,
			ToWorkflowID:   workflow.ID,
			ToStageID:      stage.ID,
			ResultValue:    "entered",
			Title:          "进入阶段：" + stage.Name,
			Content:        "负责人：" + owner.Name,
			Snapshot: map[string]any{
				"owner_department_id": owner.DepartmentID,
				"owner_staff_id":      owner.ID,
			},
		}) == 0 {
			return fmt.Errorf("阶段流转记录创建失败")
		}
		if err := createStageTodos(txCtx, progress, stage); err != nil {
			return err
		}
		completed = progress
		return nil
	})
	return completed, err
}

func TerminateAssetWorkflow(ctx context.Context, staff *WorkStaffSession, assetID uint64, reason string) (*crmmodel.CustomerStage, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("请填写终止原因")
	}
	var terminated *crmmodel.CustomerStage
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		progress, err := activeAssetProgress(txCtx, assetID)
		if err != nil {
			return err
		}
		if staff == nil || staff.ID == 0 || progress.OwnerStaffID != staff.ID {
			return fmt.Errorf("只有当前负责人可以终止流程")
		}
		now := time.Now()
		crmmodel.NewWorkTodoModel().Update(txCtx, map[string]any{
			"asset_id": assetID,
			"status":   crmmodel.WorkTodoStatusPending,
		}, map[string]any{
			"status":     crmmodel.WorkTodoStatusCanceled,
			"result":     "流程已终止：" + reason,
			"updated_at": now,
		})
		if crmmodel.NewCustomerStageModel().Update(txCtx, map[string]any{
			"id":     progress.ID,
			"status": crmmodel.ProgressStatusActive,
		}, map[string]any{
			"status":            crmmodel.ProgressStatusTerminated,
			"terminated_at":     now,
			"terminated_reason": reason,
			"updated_at":        now,
		}) == 0 {
			return fmt.Errorf("流程已变化，请刷新后重试")
		}
		progress.Status = crmmodel.ProgressStatusTerminated
		progress.TerminatedAt = &now
		progress.TerminatedReason = reason
		progress.UpdatedAt = now
		if recordWorkStageChange(txCtx, staff, progress, workStageChange{
			FromWorkflowID: progress.WorkflowID,
			FromStageID:    progress.StageID,
			ResultValue:    "terminated",
			Title:          "流程已终止",
			Content:        reason,
			Snapshot:       map[string]any{"reason": reason},
		}) == 0 {
			return fmt.Errorf("流程终止记录创建失败")
		}
		terminated = progress
		return nil
	})
	return terminated, err
}

func activeAssetProgress(ctx context.Context, assetID uint64) (*crmmodel.CustomerStage, error) {
	if assetID == 0 {
		return nil, fmt.Errorf("客户资产不能为空")
	}
	progress := crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"asset_id": assetID,
		"status":   crmmodel.ProgressStatusActive,
	})
	if progress == nil {
		return nil, fmt.Errorf("资产没有进行中的流程")
	}
	return progress, nil
}

func canCompleteAssetStage(staff *WorkStaffSession, progress *crmmodel.CustomerStage) bool {
	if staff == nil || staff.ID == 0 || progress == nil {
		return false
	}
	return progress.OwnerStaffID == staff.ID || staff.CanDispatch
}

func pendingRequiredTodoCount(ctx context.Context, progress *crmmodel.CustomerStage) int64 {
	if progress == nil {
		return 0
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"asset_id": progress.AssetID,
		"stage_id": progress.StageID,
		"required": true,
		"status":   crmmodel.WorkTodoStatusPending,
	})
}

func cancelPendingOptionalTodos(ctx context.Context, progress *crmmodel.CustomerStage) {
	if progress == nil {
		return
	}
	crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{
		"asset_id": progress.AssetID,
		"stage_id": progress.StageID,
		"required": false,
		"status":   crmmodel.WorkTodoStatusPending,
	}, map[string]any{
		"status":     crmmodel.WorkTodoStatusCanceled,
		"result":     "阶段已完成",
		"updated_at": time.Now(),
	})
}

func nextWorkflowStage(ctx context.Context, progress *crmmodel.CustomerStage) (*crmmodel.Workflow, *crmmodel.Stage, error) {
	if progress == nil {
		return nil, nil, fmt.Errorf("资产流程不存在")
	}
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"id":     progress.WorkflowID,
		"status": crmmodel.StatusEnabled,
	})
	if workflow == nil {
		return nil, nil, fmt.Errorf("当前流程不存在或已停用")
	}
	if stage := nextEnabledStage(ctx, workflow.ID, progress.StageID); stage != nil {
		return workflow, stage, nil
	}
	if workflow.NextWorkflowID == 0 {
		return nil, nil, nil
	}
	nextWorkflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"id":     workflow.NextWorkflowID,
		"status": crmmodel.StatusEnabled,
	})
	if nextWorkflow == nil {
		return nil, nil, fmt.Errorf("后续流程不存在或已停用")
	}
	nextStage := firstEnabledStage(ctx, nextWorkflow.ID)
	if nextStage == nil {
		return nil, nil, fmt.Errorf("后续流程没有已启用阶段")
	}
	return nextWorkflow, nextStage, nil
}

func completeAssetProgress(ctx context.Context, staff *WorkStaffSession, progress *crmmodel.CustomerStage) error {
	now := time.Now()
	if crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"id":          progress.ID,
		"workflow_id": progress.WorkflowID,
		"stage_id":    progress.StageID,
		"status":      crmmodel.ProgressStatusActive,
	}, map[string]any{
		"status":       crmmodel.ProgressStatusCompleted,
		"completed_at": now,
		"updated_at":   now,
	}) == 0 {
		return fmt.Errorf("流程已变化，请刷新后重试")
	}
	progress.Status = crmmodel.ProgressStatusCompleted
	progress.CompletedAt = &now
	progress.UpdatedAt = now
	if recordWorkStageChange(ctx, staff, progress, workStageChange{
		FromWorkflowID: progress.WorkflowID,
		FromStageID:    progress.StageID,
		ResultValue:    "completed",
		Title:          "流程已完成",
	}) == 0 {
		return fmt.Errorf("流程完成记录创建失败")
	}
	return nil
}

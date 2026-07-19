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

type workflowSubject struct {
	LeadID            uint64
	CustomerID        uint64
	AssetID           uint64
	CustomerProductID uint64
}

func leadWorkflowSubject(leadID uint64) workflowSubject {
	return workflowSubject{LeadID: leadID}
}

func assetWorkflowSubject(customerID, assetID, customerProductID uint64) workflowSubject {
	return workflowSubject{
		CustomerID:        customerID,
		AssetID:           assetID,
		CustomerProductID: customerProductID,
	}
}

func StartLeadWorkflow(ctx context.Context, leadID uint64, ownerStaffID ...uint64) error {
	return orm.Transaction(ctx, func(txCtx context.Context) error {
		return startLeadWorkflow(txCtx, leadID, ownerStaffID...)
	})
}

func startLeadWorkflow(ctx context.Context, leadID uint64, ownerStaffID ...uint64) error {
	if leadID == 0 || crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": leadID}) == nil {
		return fmt.Errorf("线索不存在")
	}
	workflow, _ := defaultEntryWorkflowStage(ctx, crmmodel.WorkflowSubjectLead)
	if workflow == nil {
		return ErrNoAvailableWorkflow
	}
	if crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"lead_id":     leadID,
		"workflow_id": workflow.ID,
		"status":      crmmodel.ProgressStatusActive,
	}) != nil {
		return nil
	}
	_, err := startWorkflowInstance(ctx, leadWorkflowSubject(leadID), workflow.ID, firstRequestedOwnerID(ownerStaffID))
	return err
}

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
	workflow, _ := defaultEntryWorkflowStage(ctx, crmmodel.WorkflowSubjectCustomerAsset)
	if workflow == nil {
		return ErrNoAvailableWorkflow
	}
	if crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"customer_id":         customerID,
		"asset_id":            assetID,
		"customer_product_id": uint64(0),
		"workflow_id":         workflow.ID,
	}) != nil {
		return nil
	}
	_, err := startWorkflowInstance(ctx, assetWorkflowSubject(customerID, assetID, 0), workflow.ID, firstRequestedOwnerID(ownerStaffID))
	return err
}

func startWorkflowInstance(
	ctx context.Context,
	subject workflowSubject,
	workflowID uint64,
	requestedOwnerID uint64,
) (*crmmodel.WorkflowInstance, error) {
	return createWorkflowInstance(ctx, subject, workflowID, requestedOwnerID, false)
}

func restartWorkflowInstance(
	ctx context.Context,
	subject workflowSubject,
	workflowID uint64,
) (*crmmodel.WorkflowInstance, error) {
	return createWorkflowInstance(ctx, subject, workflowID, 0, true)
}

func createWorkflowInstance(
	ctx context.Context,
	subject workflowSubject,
	workflowID uint64,
	requestedOwnerID uint64,
	activeDuplicateOnly bool,
) (*crmmodel.WorkflowInstance, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("流程不能为空")
	}
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"id":     workflowID,
		"status": crmmodel.StatusEnabled,
	})
	if workflow == nil {
		return nil, fmt.Errorf("流程不存在或已停用")
	}
	if err := validateWorkflowSubject(ctx, workflow, subject); err != nil {
		return nil, err
	}
	stage := firstEnabledStage(ctx, workflow.ID)
	if stage == nil {
		return nil, fmt.Errorf("流程没有已启用阶段")
	}
	instanceModel := crmmodel.NewWorkflowInstanceModel()
	duplicateFilters := workflowSubjectInstanceFilters(subject, workflow.ID)
	if activeDuplicateOnly {
		duplicateFilters["status"] = crmmodel.ProgressStatusActive
	}
	if existing := instanceModel.Find(ctx, duplicateFilters); existing != nil {
		return existing, nil
	}
	owner, err := resolveStageOwner(ctx, stage, requestedOwnerID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	instanceID := uint64(instanceModel.Insert(ctx, map[string]any{
		"lead_id":             subject.LeadID,
		"customer_id":         subject.CustomerID,
		"asset_id":            subject.AssetID,
		"customer_product_id": subject.CustomerProductID,
		"workflow_id":         workflow.ID,
		"stage_id":            stage.ID,
		"owner_department_id": owner.DepartmentID,
		"owner_staff_id":      owner.ID,
		"status":              crmmodel.ProgressStatusActive,
		"started_at":          now,
		"terminated_reason":   "",
		"updated_at":          now,
	}))
	if instanceID == 0 {
		return nil, fmt.Errorf("流程启动失败")
	}
	instance := instanceModel.Find(ctx, map[string]any{"id": instanceID})
	if instance == nil {
		return nil, fmt.Errorf("流程启动失败")
	}
	if recordWorkStageChange(ctx, nil, instance, workStageChange{
		ToWorkflowID: workflow.ID,
		ToStageID:    stage.ID,
		ResultValue:  "entered",
		Title:        "流程已启动",
		Snapshot: map[string]any{
			"lead_id":             subject.LeadID,
			"owner_department_id": owner.DepartmentID,
			"owner_staff_id":      owner.ID,
		},
	}) == 0 {
		return nil, fmt.Errorf("流程启动记录创建失败")
	}
	if err := createStageTodos(ctx, instance, stage); err != nil {
		return nil, err
	}
	return instance, nil
}

func validateWorkflowSubject(ctx context.Context, workflow *crmmodel.Workflow, subject workflowSubject) error {
	if workflow == nil {
		return fmt.Errorf("流程不存在")
	}
	switch workflow.SubjectType {
	case crmmodel.WorkflowSubjectLead:
		if subject.LeadID == 0 || subject.CustomerID > 0 || subject.AssetID > 0 {
			return fmt.Errorf("该流程只能处理线索")
		}
		if crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": subject.LeadID}) == nil {
			return fmt.Errorf("线索不存在")
		}
	case crmmodel.WorkflowSubjectCustomerAsset:
		if subject.LeadID > 0 || subject.CustomerID == 0 || subject.AssetID == 0 {
			return fmt.Errorf("该流程只能处理客户资产")
		}
		if crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{
			"id":          subject.AssetID,
			"customer_id": subject.CustomerID,
		}) == nil {
			return fmt.Errorf("客户资产不存在")
		}
	default:
		return fmt.Errorf("流程对象配置无效")
	}
	return nil
}

func workflowSubjectInstanceFilters(subject workflowSubject, workflowID uint64) map[string]any {
	filters := map[string]any{"workflow_id": workflowID}
	if subject.LeadID > 0 {
		filters["lead_id"] = subject.LeadID
		filters["status"] = crmmodel.ProgressStatusActive
		return filters
	}
	filters["customer_id"] = subject.CustomerID
	filters["asset_id"] = subject.AssetID
	filters["customer_product_id"] = subject.CustomerProductID
	return filters
}

func defaultEntryWorkflowStage(ctx context.Context, subjectType string) (*crmmodel.Workflow, *crmmodel.Stage) {
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"subject_type":  subjectType,
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
	instance *crmmodel.WorkflowInstance,
	workflow *crmmodel.Workflow,
	stage *crmmodel.Stage,
	requestedOwnerID uint64,
) (*crmmodel.Staff, error) {
	if instance == nil || workflow == nil || stage == nil {
		return nil, fmt.Errorf("流程阶段不能为空")
	}
	owner, err := resolveStageTransitionOwner(ctx, instance, stage, requestedOwnerID)
	if err != nil {
		return nil, err
	}

	previousWorkflowID := instance.WorkflowID
	previousStageID := instance.StageID
	now := time.Now()
	updated := crmmodel.NewWorkflowInstanceModel().Update(ctx, map[string]any{
		"id":          instance.ID,
		"workflow_id": previousWorkflowID,
		"stage_id":    previousStageID,
		"status":      crmmodel.ProgressStatusActive,
	}, map[string]any{
		"workflow_id":         workflow.ID,
		"stage_id":            stage.ID,
		"owner_department_id": owner.DepartmentID,
		"owner_staff_id":      owner.ID,
		"started_at":          now,
		"updated_at":          now,
	})
	if updated == 0 {
		return nil, fmt.Errorf("流程已变化，请刷新后重试")
	}
	instance.WorkflowID = workflow.ID
	instance.StageID = stage.ID
	instance.OwnerDepartmentID = owner.DepartmentID
	instance.OwnerStaffID = owner.ID
	instance.StartedAt = now
	instance.UpdatedAt = now
	if instance.LeadID > 0 {
		crmmodel.NewLeadModel().Update(ctx, map[string]any{"id": instance.LeadID}, map[string]any{
			"owner_department_id": owner.DepartmentID,
			"owner_staff_id":      owner.ID,
			"updated_at":          now,
		})
	}
	return owner, nil
}

func createStageTodos(ctx context.Context, instance *crmmodel.WorkflowInstance, stage *crmmodel.Stage) error {
	if instance == nil || stage == nil {
		return fmt.Errorf("流程实例和阶段不能为空")
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
		createdTodo, activated, err := createOrReactivateStageTodo(ctx, instance, task, now)
		if err != nil {
			return err
		}
		if !activated {
			continue
		}
		if createdTodo.AssigneeStaffID > 0 {
			assignee := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": createdTodo.AssigneeStaffID})
			if recordWorkTodoAssignment(ctx, nil, instance, createdTodo, task, 0, assignee) == 0 {
				return fmt.Errorf("任务分配记录创建失败")
			}
		}
		if task.TaskType == crmmodel.TaskTypeRule {
			ruleTodos = append(ruleTodos, struct {
				todo *crmmodel.WorkTodo
				task *crmmodel.Task
			}{todo: createdTodo, task: task})
		}
	}
	for _, current := range ruleTodos {
		if workRuleTodoReady(ctx, current.todo, current.task) {
			if err := executePendingRuleTodo(ctx, current.todo, current.task); err != nil {
				return err
			}
		}
	}
	return nil
}

func createOrReactivateStageTodo(
	ctx context.Context,
	instance *crmmodel.WorkflowInstance,
	task *crmmodel.Task,
	now time.Time,
) (*crmmodel.WorkTodo, bool, error) {
	if instance == nil || task == nil {
		return nil, false, fmt.Errorf("流程实例和任务不能为空")
	}
	todoModel := crmmodel.NewWorkTodoModel()
	existing := todoModel.Find(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"stage_id":             task.StageID,
		"task_id":              task.ID,
	})
	if existing != nil && existing.Status != crmmodel.WorkTodoStatusCanceled {
		return existing, false, nil
	}
	departmentID, staffID, err := resolveTaskAssignee(ctx, instance, task)
	if err != nil {
		return nil, false, err
	}
	var dueAt *time.Time
	if task.DueDays > 0 {
		deadline := now.AddDate(0, 0, task.DueDays)
		dueAt = &deadline
	}
	data := map[string]any{
		"lead_id":                instance.LeadID,
		"customer_id":            instance.CustomerID,
		"asset_id":               instance.AssetID,
		"workflow_instance_id":   instance.ID,
		"customer_product_id":    instance.CustomerProductID,
		"workflow_id":            instance.WorkflowID,
		"stage_id":               task.StageID,
		"task_id":                task.ID,
		"assignee_department_id": departmentID,
		"assignee_staff_id":      staffID,
		"required":               task.Required,
		"status":                 crmmodel.WorkTodoStatusPending,
		"due_at":                 dueAt,
		"result":                 "",
		"completed_at":           nil,
		"updated_at":             now,
	}
	if existing != nil {
		if todoModel.Update(ctx, map[string]any{
			"id":     existing.ID,
			"status": crmmodel.WorkTodoStatusCanceled,
		}, data) == 0 {
			return nil, false, fmt.Errorf("已取消待办重新启用失败")
		}
		createdTodo := todoModel.Find(ctx, map[string]any{"id": existing.ID})
		if createdTodo == nil {
			return nil, false, fmt.Errorf("已取消待办重新启用后无法读取")
		}
		return createdTodo, true, nil
	}
	data["created_at"] = now
	todoID := uint64(todoModel.Insert(ctx, data))
	if todoID == 0 {
		return nil, false, fmt.Errorf("阶段待办创建失败")
	}
	createdTodo := todoModel.Find(ctx, map[string]any{"id": todoID})
	if createdTodo == nil {
		return nil, false, fmt.Errorf("阶段待办创建后无法读取")
	}
	return createdTodo, true, nil
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

package service

import (
	"context"

	crmmodel "github.com/dever-package/crm/model"
)

func workflowForSubject(ctx context.Context, workflowID uint64, subjectType string) *crmmodel.Workflow {
	filters := map[string]any{
		"subject_type": subjectType,
		"status":       crmmodel.StatusEnabled,
	}
	if workflowID > 0 {
		filters["id"] = workflowID
	} else {
		filters["default_entry"] = true
	}
	return crmmodel.NewWorkflowModel().Find(ctx, filters, map[string]any{"order": "sort asc,id asc"})
}

func canAccessWorkflow(ctx context.Context, staff *WorkStaffSession, workflow *crmmodel.Workflow) bool {
	if staff == nil || staff.ID == 0 || workflow == nil || workflow.Status != crmmodel.StatusEnabled {
		return false
	}
	if staff.CanDispatch {
		return true
	}
	stages := crmmodel.NewStageModel().Select(ctx, map[string]any{
		"workflow_id": workflow.ID,
		"status":      crmmodel.StatusEnabled,
	})
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		if stage.OwnerDepartmentID == staff.DepartmentID {
			return true
		}
		if crmmodel.NewTaskModel().Count(ctx, map[string]any{
			"stage_id":               stage.ID,
			"assignee_department_id": staff.DepartmentID,
			"status":                 crmmodel.StatusEnabled,
		}) > 0 {
			return true
		}
	}
	if crmmodel.NewWorkflowInstanceModel().Count(ctx, map[string]any{
		"workflow_id":    workflow.ID,
		"owner_staff_id": staff.ID,
	}) > 0 {
		return true
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"workflow_id":       workflow.ID,
		"assignee_staff_id": staff.ID,
	}) > 0
}

func canCreateLeadInWorkflow(ctx context.Context, staff *WorkStaffSession, workflow *crmmodel.Workflow) bool {
	if staff == nil || staff.ID == 0 || workflow == nil || workflow.SubjectType != crmmodel.WorkflowSubjectLead {
		return false
	}
	if staff.CanDispatch {
		return true
	}
	stage := firstEnabledStage(ctx, workflow.ID)
	return stage != nil && stage.OwnerDepartmentID == staff.DepartmentID
}

func canManageLeadWorkflow(staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	return staff != nil && staff.ID > 0 && instance != nil &&
		(instance.OwnerStaffID == staff.ID || staff.CanDispatch)
}

func canChangeWorkflowOwner(staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	if staff == nil || staff.ID == 0 || instance == nil || instance.Status != crmmodel.ProgressStatusActive {
		return false
	}
	if staff.CanDispatch {
		return true
	}
	return isWorkflowDepartmentLeader(staff, instance)
}

func isWorkflowDepartmentLeader(staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	return staff != nil && staff.ID > 0 && instance != nil &&
		staff.StaffType == crmmodel.StaffTypeLeader && staff.DepartmentID > 0 &&
		staff.DepartmentID == instance.OwnerDepartmentID
}

func canViewAssignedWorkflowInstance(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	if staff == nil || staff.ID == 0 || instance == nil {
		return false
	}
	if instance.OwnerStaffID == staff.ID || isWorkflowDepartmentLeader(staff, instance) {
		return true
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"assignee_staff_id":    staff.ID,
		"status":               crmmodel.WorkTodoStatusPending,
	}) > 0
}

func canViewWorkflowInstanceInScope(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	if staff == nil || instance == nil {
		return false
	}
	if staff.CanDispatch && staff.ViewAll {
		return true
	}
	return canViewAssignedWorkflowInstance(ctx, staff, instance)
}

func canViewWorkflowInstance(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	if staff == nil || staff.ID == 0 || instance == nil {
		return false
	}
	if staff.CanDispatch {
		return true
	}
	return canViewAssignedWorkflowInstance(ctx, staff, instance)
}

func workflowInstanceForLead(ctx context.Context, leadID, workflowID uint64) *crmmodel.WorkflowInstance {
	if leadID == 0 || workflowID == 0 {
		return nil
	}
	return crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"lead_id":     leadID,
		"workflow_id": workflowID,
	}, map[string]any{"order": "id desc"})
}

func activeWorkflowInstanceForLead(ctx context.Context, leadID, workflowID uint64) *crmmodel.WorkflowInstance {
	if leadID == 0 || workflowID == 0 {
		return nil
	}
	return crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
		"lead_id":     leadID,
		"workflow_id": workflowID,
		"status":      crmmodel.ProgressStatusActive,
	}, map[string]any{"order": "id desc"})
}

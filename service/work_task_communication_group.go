package service

import (
	"context"
	"fmt"

	crmmodel "github.com/dever-package/crm/model"
)

func syncWorkCommunicationGroupFromFormTask(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	task *crmmodel.Task,
	formInput *workFormInput,
	values map[string]any,
) error {
	if task != nil && task.CommunicationGroupEnabled {
		return syncWorkCommunicationGroupPayload(ctx, staff, todo, values, true)
	}
	statusValue, submitted, createdValue := workCommunicationGroupStatusValue(ctx, formInput)
	if !submitted || createdValue == "" {
		return nil
	}
	activeGroup := crmmodel.NewCommunicationGroupModel().Find(ctx, map[string]any{
		"workflow_instance_id": todo.WorkflowInstanceID,
		"status":               crmmodel.CommunicationGroupStatusActive,
	})
	if statusValue != createdValue {
		if activeGroup != nil {
			return fmt.Errorf("当前案件已有使用中的沟通群，建群状态必须选择已建群")
		}
		return nil
	}

	return syncWorkCommunicationGroupPayloadWithActiveGroup(ctx, staff, todo, values, activeGroup, false)
}

func syncWorkCommunicationGroupPayload(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	values map[string]any,
	required bool,
) error {
	activeGroup := crmmodel.NewCommunicationGroupModel().Find(ctx, map[string]any{
		"workflow_instance_id": todo.WorkflowInstanceID,
		"status":               crmmodel.CommunicationGroupStatusActive,
	})
	return syncWorkCommunicationGroupPayloadWithActiveGroup(ctx, staff, todo, values, activeGroup, required)
}

func syncWorkCommunicationGroupPayloadWithActiveGroup(
	ctx context.Context,
	staff *WorkStaffSession,
	todo *crmmodel.WorkTodo,
	values map[string]any,
	activeGroup *crmmodel.CommunicationGroup,
	required bool,
) error {
	payload := mapFromAny(values["communication_group"])
	if len(payload) == 0 {
		if activeGroup != nil && !required {
			return nil
		}
		return fmt.Errorf("请填写建群信息")
	}
	payloadGroupID := firstUint64(payload, "communication_group_id", "communicationGroupId", "id")
	if activeGroup == nil && payloadGroupID > 0 {
		return fmt.Errorf("当前案件没有可编辑的使用中沟通群")
	}
	if activeGroup != nil {
		if payloadGroupID > 0 && payloadGroupID != activeGroup.ID {
			return fmt.Errorf("沟通群不属于当前案件")
		}
		payload["communication_group_id"] = activeGroup.ID
	}
	if instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId"); instanceID > 0 && instanceID != todo.WorkflowInstanceID {
		return fmt.Errorf("沟通群不属于当前案件")
	}
	payload["workflow_instance_id"] = todo.WorkflowInstanceID
	instance, err := activeWorkflowInstance(ctx, todo.WorkflowInstanceID)
	if err != nil {
		return err
	}
	_, err = saveCommunicationGroupRecord(ctx, staff, instance, activeGroup, payload, false)
	return err
}

func workCommunicationGroupStatusValue(
	ctx context.Context,
	formInput *workFormInput,
) (statusValue string, submitted bool, createdValue string) {
	if formInput == nil {
		return "", false, ""
	}
	template, field := communicationGroupStatusTarget(ctx)
	if template == nil || field == nil {
		return "", false, ""
	}
	createdValue = communicationGroupEstablishedOptionValue(ctx, field)
	record, exists := formInput.customerDataRecords[template.ID]
	if !exists {
		return "", false, createdValue
	}
	value, submitted := record[fmt.Sprintf("%d", field.ID)]
	return inputText(value), submitted, createdValue
}

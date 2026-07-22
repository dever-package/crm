package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

type communicationGroupStaffInput struct {
	StaffID uint64
	Role    string
}

type communicationGroupInput struct {
	GroupTypeID     uint64
	Name            string
	ExternalGroupID string
	EstablishedAt   time.Time
	Summary         string
	Remark          string
	Staff           []communicationGroupStaffInput
}

func registerCommunicationGroup(
	ctx context.Context,
	instance *crmmodel.WorkflowInstance,
	createdByStaffID uint64,
	input communicationGroupInput,
) (*crmmodel.CommunicationGroup, error) {
	if instance == nil || instance.ID == 0 || instance.CustomerID == 0 {
		return nil, fmt.Errorf("案件流程不存在")
	}
	if err := validateCommunicationGroupInput(ctx, input, 0); err != nil {
		return nil, err
	}
	if crmmodel.NewCommunicationGroupModel().Count(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"status":               crmmodel.CommunicationGroupStatusActive,
	}) > 0 {
		return nil, fmt.Errorf("当前案件已有使用中的沟通群")
	}
	now := time.Now()
	establishedAt := input.EstablishedAt
	if establishedAt.IsZero() {
		establishedAt = workBeginningOfDay(now.In(scheduleLocation()))
	}
	groupID := uint64(crmmodel.NewCommunicationGroupModel().Insert(ctx, map[string]any{
		"customer_id":          instance.CustomerID,
		"asset_id":             instance.AssetID,
		"workflow_instance_id": instance.ID,
		"group_type_id":        input.GroupTypeID,
		"name":                 strings.TrimSpace(input.Name),
		"external_group_id":    strings.TrimSpace(input.ExternalGroupID),
		"status":               crmmodel.CommunicationGroupStatusActive,
		"established_at":       establishedAt,
		"dissolve_reason":      "",
		"summary":              strings.TrimSpace(input.Summary),
		"remark":               strings.TrimSpace(input.Remark),
		"created_by_staff_id":  createdByStaffID,
		"created_at":           now,
		"updated_at":           now,
	}))
	if groupID == 0 {
		return nil, fmt.Errorf("沟通群创建失败，请确认当前案件没有其他使用中的群")
	}
	if err := replaceCommunicationGroupStaff(ctx, groupID, input.Staff); err != nil {
		return nil, err
	}
	group := crmmodel.NewCommunicationGroupModel().Find(ctx, map[string]any{"id": groupID})
	if group == nil {
		return nil, fmt.Errorf("沟通群创建失败")
	}
	return group, nil
}

func reviseCommunicationGroup(
	ctx context.Context,
	group *crmmodel.CommunicationGroup,
	input communicationGroupInput,
) (*crmmodel.CommunicationGroup, error) {
	if group == nil || group.ID == 0 {
		return nil, fmt.Errorf("沟通群不存在")
	}
	if err := validateCommunicationGroupInput(ctx, input, group.GroupTypeID); err != nil {
		return nil, err
	}
	establishedAt := input.EstablishedAt
	if establishedAt.IsZero() {
		establishedAt = group.EstablishedAt
	}
	updates := map[string]any{
		"group_type_id":     input.GroupTypeID,
		"name":              strings.TrimSpace(input.Name),
		"external_group_id": strings.TrimSpace(input.ExternalGroupID),
		"established_at":    establishedAt,
		"summary":           strings.TrimSpace(input.Summary),
		"remark":            strings.TrimSpace(input.Remark),
		"updated_at":        time.Now(),
	}
	if crmmodel.NewCommunicationGroupModel().Update(ctx, map[string]any{
		"id": group.ID,
	}, updates) == 0 {
		return nil, fmt.Errorf("沟通群保存失败")
	}
	if err := replaceCommunicationGroupStaff(ctx, group.ID, input.Staff); err != nil {
		return nil, err
	}
	updated := crmmodel.NewCommunicationGroupModel().Find(ctx, map[string]any{"id": group.ID})
	if updated == nil {
		return nil, fmt.Errorf("沟通群保存失败")
	}
	return updated, nil
}

func dissolveCommunicationGroup(
	ctx context.Context,
	group *crmmodel.CommunicationGroup,
	dissolvedAt time.Time,
	reason string,
) (*crmmodel.CommunicationGroup, error) {
	if group == nil || group.ID == 0 {
		return nil, fmt.Errorf("沟通群不存在")
	}
	if group.Status != crmmodel.CommunicationGroupStatusActive {
		return nil, fmt.Errorf("沟通群已经解散")
	}
	if dissolvedAt.IsZero() {
		dissolvedAt = time.Now()
	}
	if communicationGroupDateBefore(dissolvedAt, group.EstablishedAt) {
		return nil, fmt.Errorf("解散日期不能早于建群日期")
	}
	if crmmodel.NewCommunicationGroupModel().Update(ctx, map[string]any{
		"id":     group.ID,
		"status": crmmodel.CommunicationGroupStatusActive,
	}, map[string]any{
		"status":          crmmodel.CommunicationGroupStatusDissolved,
		"dissolved_at":    dissolvedAt,
		"dissolve_reason": strings.TrimSpace(reason),
		"updated_at":      time.Now(),
	}) == 0 {
		return nil, fmt.Errorf("沟通群状态已变化，请刷新后重试")
	}
	updated := crmmodel.NewCommunicationGroupModel().Find(ctx, map[string]any{"id": group.ID})
	if updated == nil {
		return nil, fmt.Errorf("沟通群解散失败")
	}
	return updated, nil
}

func communicationGroupDateBefore(left time.Time, right time.Time) bool {
	leftDate := left.In(scheduleLocation()).Format("2006-01-02")
	rightDate := right.In(scheduleLocation()).Format("2006-01-02")
	return leftDate < rightDate
}

func validateCommunicationGroupInput(ctx context.Context, input communicationGroupInput, currentTypeID uint64) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("请填写群名称")
	}
	if input.GroupTypeID == 0 {
		return fmt.Errorf("请选择群类型")
	}
	groupType := crmmodel.NewCommunicationGroupTypeModel().Find(ctx, map[string]any{"id": input.GroupTypeID})
	if groupType == nil || (groupType.Status != crmmodel.StatusEnabled && input.GroupTypeID != currentTypeID) {
		return fmt.Errorf("沟通群类型不存在或已停用")
	}
	return nil
}

func replaceCommunicationGroupStaff(ctx context.Context, groupID uint64, inputs []communicationGroupStaffInput) error {
	if groupID == 0 {
		return fmt.Errorf("沟通群不存在")
	}
	model := crmmodel.NewCommunicationGroupStaffModel()
	existingRows := model.Select(ctx, map[string]any{"communication_group_id": groupID})
	existingByStaff := make(map[uint64]*crmmodel.CommunicationGroupStaff, len(existingRows))
	for _, row := range existingRows {
		if row != nil {
			existingByStaff[row.StaffID] = row
		}
	}
	targets := normalizeCommunicationGroupStaffInputs(inputs)
	for staffID := range targets {
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID})
		if staff == nil || (staff.Status != crmmodel.StatusEnabled && existingByStaff[staffID] == nil) {
			return fmt.Errorf("关联人员不存在或已停用")
		}
	}
	now := time.Now()
	for staffID, row := range existingByStaff {
		if _, keep := targets[staffID]; keep {
			continue
		}
		model.Delete(ctx, map[string]any{"id": row.ID})
	}
	for staffID, role := range targets {
		if row := existingByStaff[staffID]; row != nil {
			if role == "" || role == row.Role {
				continue
			}
			model.Update(ctx, map[string]any{"id": row.ID}, map[string]any{
				"role":       role,
				"updated_at": now,
			})
			continue
		}
		if role == "" {
			role = crmmodel.CommunicationGroupStaffParticipant
		}
		if model.Insert(ctx, map[string]any{
			"communication_group_id": groupID,
			"staff_id":               staffID,
			"role":                   role,
			"created_at":             now,
			"updated_at":             now,
		}) == 0 {
			return fmt.Errorf("关联人员保存失败")
		}
	}
	return nil
}

func normalizeCommunicationGroupStaffInputs(inputs []communicationGroupStaffInput) map[uint64]string {
	result := make(map[uint64]string, len(inputs))
	for _, input := range inputs {
		if input.StaffID == 0 {
			continue
		}
		role := normalizeCommunicationGroupStaffRole(input.Role)
		if current, exists := result[input.StaffID]; !exists || current == "" {
			result[input.StaffID] = role
		}
	}
	return result
}

func normalizeCommunicationGroupStaffRole(role string) string {
	switch strings.TrimSpace(role) {
	case crmmodel.CommunicationGroupStaffParticipant,
		crmmodel.CommunicationGroupStaffNPLOwner,
		crmmodel.CommunicationGroupStaffPMOwner,
		crmmodel.CommunicationGroupStaffALAOwner:
		return strings.TrimSpace(role)
	default:
		return ""
	}
}

package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
	"github.com/shemic/dever/orm"
)

const (
	workBusinessEventCommunicationGroupCreated   = "communication_group_created"
	workBusinessEventCommunicationGroupUpdated   = "communication_group_updated"
	workBusinessEventCommunicationGroupDissolved = "communication_group_dissolved"
)

func (WorkService) PeopleOptions(ctx context.Context, staff *WorkStaffSession) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	return workPeopleOptions(ctx, staff), nil
}

func (WorkService) SaveCommunicationGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	var saved *crmmodel.CommunicationGroup
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		groupID := firstUint64(payload, "communication_group_id", "communicationGroupId", "id")
		group := crmmodel.NewCommunicationGroupModel().Find(txCtx, map[string]any{"id": groupID})
		instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
		if group != nil {
			instanceID = group.WorkflowInstanceID
		}
		instance, err := manageableCommunicationGroupInstance(txCtx, staff, instanceID)
		if err != nil {
			return err
		}
		if groupID > 0 && group == nil {
			return fmt.Errorf("沟通群不存在")
		}
		if group != nil && (group.CustomerID != instance.CustomerID || group.AssetID != instance.AssetID) {
			return fmt.Errorf("沟通群不属于当前案件")
		}
		saved, err = saveCommunicationGroupRecord(txCtx, staff, instance, group, payload, true)
		return err
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"saved": true,
		"group": workCommunicationGroupResult(ctx, staff, saved),
	}, nil
}

func saveCommunicationGroupRecord(
	ctx context.Context,
	staff *WorkStaffSession,
	instance *crmmodel.WorkflowInstance,
	group *crmmodel.CommunicationGroup,
	payload map[string]any,
	syncEstablishedStatus bool,
) (*crmmodel.CommunicationGroup, error) {
	if instance == nil || instance.ID == 0 {
		return nil, fmt.Errorf("案件流程不存在")
	}
	if group != nil && (group.WorkflowInstanceID != instance.ID || group.CustomerID != instance.CustomerID || group.AssetID != instance.AssetID) {
		return nil, fmt.Errorf("沟通群不属于当前案件")
	}
	establishedAt, err := parseCommunicationGroupDate(firstPresent(payload, "established_at", "establishedAt"), "建群日期")
	if err != nil {
		return nil, err
	}
	input := communicationGroupInput{
		GroupTypeID:     firstUint64(payload, "group_type_id", "groupTypeId"),
		Name:            firstText(payload, "name", "group_name", "groupName"),
		ExternalGroupID: firstText(payload, "external_group_id", "externalGroupId"),
		EstablishedAt:   establishedAt,
		Summary:         firstText(payload, "summary"),
		Remark:          firstText(payload, "remark"),
		Staff:           workCommunicationGroupStaffInputs(firstPresent(payload, "staff_ids", "staffIds")),
	}
	resultValue := workBusinessEventCommunicationGroupCreated
	title := "新增沟通群"
	var saved *crmmodel.CommunicationGroup
	if group == nil {
		saved, err = registerCommunicationGroup(ctx, instance, staff.ID, input)
	} else {
		resultValue = workBusinessEventCommunicationGroupUpdated
		title = "修改沟通群"
		saved, err = reviseCommunicationGroup(ctx, group, input)
	}
	if err != nil {
		return nil, err
	}
	operationID := recordCommunicationGroupOperation(ctx, staff, instance, saved, resultValue, title)
	if operationID == 0 {
		return nil, fmt.Errorf("沟通群操作记录保存失败")
	}
	if syncEstablishedStatus && resultValue == workBusinessEventCommunicationGroupCreated {
		if err := syncCommunicationGroupEstablishedStatus(ctx, saved, operationID); err != nil {
			return nil, err
		}
	}
	return saved, nil
}

func (WorkService) DissolveCommunicationGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	var dissolved *crmmodel.CommunicationGroup
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		groupID := firstUint64(payload, "communication_group_id", "communicationGroupId", "id")
		group := crmmodel.NewCommunicationGroupModel().Find(txCtx, map[string]any{"id": groupID})
		if group == nil {
			return fmt.Errorf("沟通群不存在")
		}
		instance, err := manageableCommunicationGroupInstance(txCtx, staff, group.WorkflowInstanceID)
		if err != nil {
			return err
		}
		dissolvedAt, err := parseCommunicationGroupDate(firstPresent(payload, "dissolved_at", "dissolvedAt"), "解散日期")
		if err != nil {
			return err
		}
		dissolved, err = dissolveCommunicationGroup(txCtx, group, dissolvedAt, firstText(payload, "reason", "dissolve_reason", "dissolveReason"))
		if err != nil {
			return err
		}
		if recordCommunicationGroupOperation(
			txCtx,
			staff,
			instance,
			dissolved,
			workBusinessEventCommunicationGroupDissolved,
			"解散沟通群",
		) == 0 {
			return fmt.Errorf("沟通群操作记录保存失败")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"dissolved": true,
		"group":     workCommunicationGroupResult(ctx, staff, dissolved),
	}, nil
}

func manageableCommunicationGroupInstance(ctx context.Context, staff *WorkStaffSession, instanceID uint64) (*crmmodel.WorkflowInstance, error) {
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": instanceID})
	if instance == nil || instance.CustomerID == 0 {
		return nil, fmt.Errorf("案件流程不存在")
	}
	if !canManageCommunicationGroup(ctx, staff, instance) {
		return nil, fmt.Errorf("无权维护当前案件的沟通群")
	}
	return instance, nil
}

func canManageCommunicationGroup(ctx context.Context, staff *WorkStaffSession, instance *crmmodel.WorkflowInstance) bool {
	if staff == nil || staff.ID == 0 || instance == nil {
		return false
	}
	if staff.CanDispatch || instance.OwnerStaffID == staff.ID || isWorkflowDepartmentLeader(ctx, staff, instance) {
		return true
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"workflow_instance_id": instance.ID,
		"assignee_staff_id":    staff.ID,
		"status":               crmmodel.WorkTodoStatusPending,
	}) > 0
}

func workPeopleOptions(ctx context.Context, staff *WorkStaffSession) map[string]any {
	return map[string]any{
		"staff":                 workStaffOptions(ctx),
		"departments":           enabledDepartmentOptions(ctx),
		"current_staff_id":      staff.ID,
		"current_department_id": staff.DepartmentID,
	}
}

func workStaffOptions(ctx context.Context) []map[string]any {
	staffRows := crmmodel.NewStaffModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(staffRows))
	for _, staff := range staffRows {
		if staff == nil {
			continue
		}
		result = append(result, map[string]any{
			"id":            staff.ID,
			"name":          staff.Name,
			"phone":         staff.Phone,
			"department_id": staff.DepartmentID,
		})
	}
	return result
}

func workCommunicationGroupTypes(ctx context.Context) []map[string]any {
	types := crmmodel.NewCommunicationGroupTypeModel().Select(ctx, map[string]any{})
	result := make([]map[string]any, 0, len(types))
	for _, groupType := range types {
		if groupType == nil {
			continue
		}
		result = append(result, map[string]any{
			"id":     groupType.ID,
			"code":   groupType.Code,
			"name":   groupType.Name,
			"status": groupType.Status,
		})
	}
	return result
}

func workCommunicationGroupRows(
	ctx context.Context,
	staff *WorkStaffSession,
	customerID uint64,
	assetID uint64,
	workflowInstanceID uint64,
) []map[string]any {
	filters := map[string]any{"customer_id": customerID}
	if workflowInstanceID > 0 {
		filters["workflow_instance_id"] = workflowInstanceID
	} else if assetID > 0 {
		filters["asset_id"] = assetID
	}
	groups := crmmodel.NewCommunicationGroupModel().Select(ctx, filters, map[string]any{
		"order": "status asc,established_at desc,id desc",
	})
	result := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		if group != nil {
			result = append(result, workCommunicationGroupResult(ctx, staff, group))
		}
	}
	return result
}

func workCommunicationGroupResult(ctx context.Context, staff *WorkStaffSession, group *crmmodel.CommunicationGroup) map[string]any {
	if group == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":                     group.ID,
		"communication_group_id": group.ID,
		"customer_id":            group.CustomerID,
		"asset_id":               group.AssetID,
		"workflow_instance_id":   group.WorkflowInstanceID,
		"group_type_id":          group.GroupTypeID,
		"name":                   group.Name,
		"external_group_id":      group.ExternalGroupID,
		"status":                 group.Status,
		"status_name":            crmmodel.CommunicationGroupStatusName(group.Status),
		"established_at":         group.EstablishedAt,
		"dissolved_at":           group.DissolvedAt,
		"dissolve_reason":        group.DissolveReason,
		"summary":                group.Summary,
		"remark":                 group.Remark,
		"created_at":             group.CreatedAt,
		"updated_at":             group.UpdatedAt,
		"staff":                  workCommunicationGroupStaffRows(ctx, group.ID),
	}
	if groupType := crmmodel.NewCommunicationGroupTypeModel().Find(ctx, map[string]any{"id": group.GroupTypeID}); groupType != nil {
		result["group_type_name"] = groupType.Name
		result["group_type_code"] = groupType.Code
	}
	instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": group.WorkflowInstanceID})
	result["can_edit"] = canManageCommunicationGroup(ctx, staff, instance)
	return result
}

func workCommunicationGroupStaffRows(ctx context.Context, groupID uint64) []map[string]any {
	relations := crmmodel.NewCommunicationGroupStaffModel().Select(ctx, map[string]any{
		"communication_group_id": groupID,
	})
	result := make([]map[string]any, 0, len(relations))
	for _, relation := range relations {
		if relation == nil {
			continue
		}
		row := map[string]any{
			"staff_id":  relation.StaffID,
			"role":      relation.Role,
			"role_name": crmmodel.CommunicationGroupStaffRoleName(relation.Role),
		}
		if person := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": relation.StaffID}); person != nil {
			row["staff_name"] = person.Name
			row["phone"] = person.Phone
			row["department_id"] = person.DepartmentID
			row["staff_status"] = person.Status
			if department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": person.DepartmentID}); department != nil {
				row["department_name"] = department.Name
			}
		}
		result = append(result, row)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := inputText(result[i]["department_name"]) + inputText(result[i]["staff_name"])
		right := inputText(result[j]["department_name"]) + inputText(result[j]["staff_name"])
		return left < right
	})
	return result
}

func workCommunicationGroupStaffInputs(value any) []communicationGroupStaffInput {
	ids := uint64ListFromAny(value)
	result := make([]communicationGroupStaffInput, 0, len(ids))
	for _, staffID := range ids {
		result = append(result, communicationGroupStaffInput{StaffID: staffID})
	}
	return result
}

func parseCommunicationGroupDate(value any, label string) (time.Time, error) {
	if emptyWorkFieldValue(value) {
		return time.Time{}, nil
	}
	text := inputText(value)
	if parsed, err := time.ParseInLocation("2006-01-02", text, scheduleLocation()); err == nil {
		return parsed, nil
	}
	parsed, err := parseScheduleTime(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s格式错误", label)
	}
	return parsed, nil
}

func recordCommunicationGroupOperation(
	ctx context.Context,
	staff *WorkStaffSession,
	instance *crmmodel.WorkflowInstance,
	group *crmmodel.CommunicationGroup,
	resultValue string,
	title string,
) uint64 {
	if group == nil {
		return 0
	}
	groupTypeName := ""
	if groupType := crmmodel.NewCommunicationGroupTypeModel().Find(ctx, map[string]any{"id": group.GroupTypeID}); groupType != nil {
		groupTypeName = groupType.Name
	}
	staffNames := make([]string, 0)
	for _, row := range workCommunicationGroupStaffRows(ctx, group.ID) {
		if name := inputText(row["staff_name"]); name != "" {
			staffNames = append(staffNames, name)
		}
	}
	snapshot := map[string]any{
		"communication_group_id": group.ID,
		"group_name":             group.Name,
		"group_type":             groupTypeName,
		"established_at":         group.EstablishedAt,
		"dissolved_at":           group.DissolvedAt,
		"dissolve_reason":        group.DissolveReason,
		"summary":                group.Summary,
		"remark":                 group.Remark,
		"staff_names":            strings.Join(staffNames, "、"),
	}
	return recordWorkManagementOperation(ctx, staff, instance, resultValue, title+"："+group.Name, groupTypeName, snapshot)
}

func syncCommunicationGroupEstablishedStatus(ctx context.Context, group *crmmodel.CommunicationGroup, operationID uint64) error {
	if group == nil || group.CustomerID == 0 {
		return nil
	}
	template, field := communicationGroupStatusTarget(ctx)
	if template == nil || field == nil {
		return nil
	}
	optionValue := communicationGroupEstablishedOptionValue(ctx, field)
	if optionValue == "" {
		return nil
	}
	recordID := saveWorkDataRecord(
		ctx,
		workDataOwnership{CustomerID: group.CustomerID},
		template.ID,
		0,
		operationID,
		map[string]any{fmt.Sprintf("%d", field.ID): optionValue},
	)
	if recordID == 0 {
		return fmt.Errorf("建群状态同步失败")
	}
	return nil
}

func communicationGroupEstablishedOptionValue(ctx context.Context, field *crmmodel.DataField) string {
	for _, option := range workDataFieldOptionRows(ctx, field) {
		if inputText(option["name"]) == "已建群" || inputText(option["value"]) == "created" {
			return inputText(option["value"])
		}
	}
	return ""
}

func communicationGroupStatusTarget(ctx context.Context) (*crmmodel.DataTemplate, *crmmodel.DataField) {
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"field_key": "service_group_status",
		"status":    crmmodel.StatusEnabled,
	})
	if field == nil {
		return nil, nil
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":      field.DataTemplateID,
		"cate_id": crmmodel.CustomerDataTemplateCateID,
		"status":  crmmodel.StatusEnabled,
	})
	if template == nil {
		return nil, nil
	}
	return template, field
}

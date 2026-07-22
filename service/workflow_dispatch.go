package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const departmentDispatchCASRetries = 5

var departmentDispatchLocation = loadDepartmentDispatchLocation()

type workflowDispatchReference struct {
	Source             string
	LeadID             uint64
	WorkflowInstanceID uint64
	WorkTodoID         uint64
	PreviousStaffID    uint64
	OperatorStaffID    uint64
}

type weeklyDispatchSchedule map[string][][2]int

func loadDepartmentDispatchLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return location
	}
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func selectDepartmentAssignee(
	ctx context.Context,
	departmentID uint64,
	legacyLoad workflowStaffLoad,
) (*crmmodel.Staff, error) {
	if !enabledDepartment(ctx, departmentID) {
		return nil, fmt.Errorf("目标部门不存在或已停用")
	}
	settingModel := crmmodel.NewDepartmentDispatchSettingModel()
	for attempt := 0; attempt < departmentDispatchCASRetries; attempt++ {
		setting, member, staff, err := departmentDispatchCandidate(ctx, departmentID, legacyLoad)
		if err != nil {
			return nil, err
		}
		if setting == nil {
			return staff, nil
		}
		if member == nil || staff == nil {
			return nil, nil
		}
		now := time.Now()
		if settingModel.Update(ctx, map[string]any{
			"id":             setting.ID,
			"department_id":  departmentID,
			"active_pool_id": setting.ActivePoolID,
			"version":        setting.Version,
			"status":         crmmodel.StatusEnabled,
		}, map[string]any{
			"last_member_id": member.ID,
			"version":        setting.Version + 1,
			"updated_at":     now,
		}) > 0 {
			return staff, nil
		}
	}
	return nil, fmt.Errorf("派单配置正在更新，请稍后重试")
}

func selectConfiguredDepartmentAssignee(ctx context.Context, departmentID uint64) (*crmmodel.Staff, error) {
	return selectDepartmentAssignee(ctx, departmentID, nil)
}

func previewDepartmentAssignee(
	ctx context.Context,
	departmentID uint64,
	legacyLoad workflowStaffLoad,
) (*crmmodel.Staff, error) {
	if !enabledDepartment(ctx, departmentID) {
		return nil, fmt.Errorf("目标部门不存在或已停用")
	}
	_, _, staff, err := departmentDispatchCandidate(ctx, departmentID, legacyLoad)
	return staff, err
}

func departmentDispatchCandidate(
	ctx context.Context,
	departmentID uint64,
	legacyLoad workflowStaffLoad,
) (*crmmodel.DepartmentDispatchSetting, *crmmodel.DispatchPoolMember, *crmmodel.Staff, error) {
	setting := crmmodel.NewDepartmentDispatchSettingModel().Find(ctx, map[string]any{
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	})
	if setting == nil || setting.ActivePoolID == 0 {
		if legacyLoad == nil {
			return nil, nil, nil, nil
		}
		staff, err := selectLeastLoadedStaff(ctx, departmentID, legacyLoad)
		return nil, nil, staff, err
	}
	pool := crmmodel.NewDispatchPoolModel().Find(ctx, map[string]any{
		"id":            setting.ActivePoolID,
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	})
	if pool == nil {
		return setting, nil, nil, nil
	}
	members := crmmodel.NewDispatchPoolMemberModel().Select(ctx, map[string]any{
		"pool_id":       pool.ID,
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	member, staff := nextEligibleDispatchMember(ctx, members, setting.LastMemberID, time.Now())
	return setting, member, staff, nil
}

func nextEligibleDispatchMember(
	ctx context.Context,
	members []*crmmodel.DispatchPoolMember,
	lastMemberID uint64,
	now time.Time,
) (*crmmodel.DispatchPoolMember, *crmmodel.Staff) {
	if len(members) == 0 {
		return nil, nil
	}
	start := 0
	for index, member := range members {
		if member != nil && member.ID == lastMemberID {
			start = (index + 1) % len(members)
			break
		}
	}
	for offset := 0; offset < len(members); offset++ {
		member := members[(start+offset)%len(members)]
		if member == nil || !dispatchMemberAvailableAt(member.WeeklyScheduleJSON, now) {
			continue
		}
		staff := enabledStaffInDepartment(ctx, member.StaffID, member.DepartmentID)
		if staff == nil || dispatchMemberDailyLimitReached(ctx, member, now) {
			continue
		}
		return member, staff
	}
	return nil, nil
}

func dispatchMemberAvailableAt(raw string, now time.Time) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return true
	}
	var schedule weeklyDispatchSchedule
	if err := json.Unmarshal([]byte(raw), &schedule); err != nil {
		return false
	}
	local := now.In(departmentDispatchLocation)
	weekday := int(local.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	minute := local.Hour()*60 + local.Minute()
	for _, period := range schedule[fmt.Sprint(weekday)] {
		if period[0] < 0 || period[1] > 1440 || period[0] >= period[1] {
			continue
		}
		if minute >= period[0] && minute < period[1] {
			return true
		}
	}
	return false
}

func dispatchMemberDailyLimitReached(ctx context.Context, member *crmmodel.DispatchPoolMember, now time.Time) bool {
	if member == nil || member.DailyLimit <= 0 {
		return false
	}
	local := now.In(departmentDispatchLocation)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, departmentDispatchLocation)
	dayEnd := dayStart.AddDate(0, 0, 1)
	count := crmmodel.NewDispatchRecordModel().Count(ctx, map[string]any{
		"dispatch_type": crmmodel.DispatchTypeAuto,
		"department_id": member.DepartmentID,
		"staff_id":      member.StaffID,
		"created_at": map[string]any{
			">=": dayStart,
			"<":  dayEnd,
		},
	})
	return count >= int64(member.DailyLimit)
}

func recordAutomaticDispatch(
	ctx context.Context,
	departmentID uint64,
	staffID uint64,
	reference workflowDispatchReference,
) error {
	if err := recordWorkflowDispatch(ctx, crmmodel.DispatchTypeAuto, departmentID, staffID, reference); err != nil {
		return err
	}
	crmmodel.NewStaffModel().Update(ctx, map[string]any{"id": staffID}, map[string]any{
		"last_assigned_at": time.Now(),
	})
	return nil
}

func recordManualDispatch(
	ctx context.Context,
	departmentID uint64,
	staffID uint64,
	reference workflowDispatchReference,
) error {
	if strings.TrimSpace(reference.Source) == "" {
		reference.Source = crmmodel.DispatchSourceManual
	}
	return recordWorkflowDispatch(ctx, crmmodel.DispatchTypeManual, departmentID, staffID, reference)
}

func recordWorkflowDispatch(
	ctx context.Context,
	dispatchType string,
	departmentID uint64,
	staffID uint64,
	reference workflowDispatchReference,
) error {
	if departmentID == 0 || staffID == 0 {
		return fmt.Errorf("派单部门和人员不能为空")
	}
	if strings.TrimSpace(reference.Source) == "" {
		reference.Source = crmmodel.DispatchSourceTask
	}
	id := crmmodel.NewDispatchRecordModel().Insert(ctx, map[string]any{
		"dispatch_type":        dispatchType,
		"source":               reference.Source,
		"department_id":        departmentID,
		"staff_id":             staffID,
		"previous_staff_id":    reference.PreviousStaffID,
		"workflow_instance_id": reference.WorkflowInstanceID,
		"work_todo_id":         reference.WorkTodoID,
		"lead_id":              reference.LeadID,
		"operator_staff_id":    reference.OperatorStaffID,
		"created_at":           time.Now(),
	})
	if id == 0 {
		return fmt.Errorf("派单记录创建失败")
	}
	return nil
}

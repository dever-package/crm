package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

func (WorkService) DispatchConfig(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	department, departments, err := dispatchDepartmentScope(ctx, staff, firstUint64(payload, "department_id", "departmentId"))
	if err != nil {
		return nil, err
	}
	if _, _, err := ensureDepartmentDispatchSetting(ctx, department.ID); err != nil {
		return nil, err
	}
	return departmentDispatchConfigPayload(ctx, staff, department, departments), nil
}

func (WorkService) SaveDispatchConfig(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	department, departments, err := dispatchDepartmentScope(ctx, staff, firstUint64(payload, "department_id", "departmentId"))
	if err != nil {
		return nil, err
	}
	if err := saveDepartmentDispatchConfiguration(ctx, department.ID, payload, nil); err != nil {
		return nil, err
	}
	return departmentDispatchConfigAfterRetry(ctx, staff, department, departments), nil
}

func saveDepartmentDispatchConfiguration(
	ctx context.Context,
	departmentID uint64,
	payload map[string]any,
	afterSave func(context.Context) error,
) error {
	poolID := firstUint64(payload, "pool_id", "poolId")
	activePoolID := firstUint64(payload, "active_pool_id", "activePoolId")
	members, err := normalizeDispatchMemberInputs(firstPresent(payload, "members", "member_list", "memberList"))
	if err != nil {
		return err
	}
	return orm.Transaction(ctx, func(txCtx context.Context) error {
		setting, directPool, ensureErr := ensureDepartmentDispatchSetting(txCtx, departmentID)
		if ensureErr != nil {
			return ensureErr
		}
		if poolID == 0 {
			poolID = setting.ActivePoolID
			if poolID == 0 {
				poolID = directPool.ID
			}
		}
		pool := enabledDispatchPool(txCtx, departmentID, poolID)
		if pool == nil {
			return fmt.Errorf("派单池不存在或已停用")
		}
		if activePoolID == 0 {
			activePoolID = setting.ActivePoolID
			if activePoolID == 0 {
				activePoolID = directPool.ID
			}
		}
		if enabledDispatchPool(txCtx, departmentID, activePoolID) == nil {
			return fmt.Errorf("当前派单池不存在或已停用")
		}
		if err := saveDispatchPoolMembers(txCtx, departmentID, pool.ID, members); err != nil {
			return err
		}
		now := time.Now()
		lastMemberID := setting.LastMemberID
		if activePoolID != setting.ActivePoolID {
			lastMemberID = 0
		}
		if crmmodel.NewDepartmentDispatchSettingModel().Update(txCtx, map[string]any{
			"id":            setting.ID,
			"department_id": departmentID,
			"version":       setting.Version,
		}, map[string]any{
			"active_pool_id": activePoolID,
			"last_member_id": lastMemberID,
			"version":        setting.Version + 1,
			"status":         crmmodel.StatusEnabled,
			"updated_at":     now,
		}) == 0 {
			return fmt.Errorf("派单配置已变化，请刷新后重试")
		}
		crmmodel.NewDispatchPoolModel().Update(txCtx, map[string]any{"id": pool.ID}, map[string]any{"updated_at": now})
		if afterSave != nil {
			return afterSave(txCtx)
		}
		return nil
	})
}

func (WorkService) CreateDispatchGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	department, departments, err := dispatchDepartmentScope(ctx, staff, firstUint64(payload, "department_id", "departmentId"))
	if err != nil {
		return nil, err
	}
	name, err := dispatchGroupName(payload)
	if err != nil {
		return nil, err
	}
	if err := createDepartmentDispatchGroup(ctx, department.ID, name); err != nil {
		return nil, err
	}
	return departmentDispatchConfigPayload(ctx, staff, department, departments), nil
}

func createDepartmentDispatchGroup(ctx context.Context, departmentID uint64, name string) error {
	return orm.Transaction(ctx, func(txCtx context.Context) error {
		if _, _, ensureErr := ensureDepartmentDispatchSetting(txCtx, departmentID); ensureErr != nil {
			return ensureErr
		}
		if crmmodel.NewDispatchPoolModel().Count(txCtx, map[string]any{
			"department_id": departmentID,
			"name":          name,
			"status":        crmmodel.StatusEnabled,
		}) > 0 {
			return fmt.Errorf("同名派单池已存在")
		}
		now := time.Now()
		if crmmodel.NewDispatchPoolModel().Insert(txCtx, map[string]any{
			"department_id": departmentID,
			"name":          name,
			"pool_type":     crmmodel.DispatchPoolTypeGroup,
			"status":        crmmodel.StatusEnabled,
			"sort":          nextDispatchPoolSort(txCtx, departmentID),
			"created_at":    now,
			"updated_at":    now,
		}) == 0 {
			return fmt.Errorf("工作组创建失败")
		}
		return nil
	})
}

func (WorkService) RenameDispatchGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	department, departments, err := dispatchDepartmentScope(ctx, staff, firstUint64(payload, "department_id", "departmentId"))
	if err != nil {
		return nil, err
	}
	poolID := firstUint64(payload, "pool_id", "poolId")
	name, err := dispatchGroupName(payload)
	if err != nil {
		return nil, err
	}
	if poolID == 0 {
		return nil, fmt.Errorf("请选择工作组")
	}
	if err := renameDepartmentDispatchGroup(ctx, department.ID, poolID, name); err != nil {
		return nil, err
	}
	return departmentDispatchConfigPayload(ctx, staff, department, departments), nil
}

func renameDepartmentDispatchGroup(ctx context.Context, departmentID, poolID uint64, name string) error {
	pool := enabledDispatchPool(ctx, departmentID, poolID)
	if pool == nil || pool.PoolType != crmmodel.DispatchPoolTypeGroup {
		return fmt.Errorf("工作组不存在或已停用")
	}
	if crmmodel.NewDispatchPoolModel().Count(ctx, map[string]any{
		"department_id": departmentID,
		"name":          name,
		"status":        crmmodel.StatusEnabled,
		"id":            map[string]any{"!=": pool.ID},
	}) > 0 {
		return fmt.Errorf("同名派单池已存在")
	}
	if crmmodel.NewDispatchPoolModel().Update(ctx, map[string]any{
		"id":            pool.ID,
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	}, map[string]any{
		"name":       name,
		"updated_at": time.Now(),
	}) == 0 {
		return fmt.Errorf("工作组已变化，请刷新后重试")
	}
	return nil
}

func (WorkService) DeleteDispatchGroup(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	department, departments, err := dispatchDepartmentScope(ctx, staff, firstUint64(payload, "department_id", "departmentId"))
	if err != nil {
		return nil, err
	}
	poolID := firstUint64(payload, "pool_id", "poolId")
	if err := deleteDepartmentDispatchGroup(ctx, department.ID, poolID); err != nil {
		return nil, err
	}
	return departmentDispatchConfigAfterRetry(ctx, staff, department, departments), nil
}

func deleteDepartmentDispatchGroup(ctx context.Context, departmentID, poolID uint64) error {
	return orm.Transaction(ctx, func(txCtx context.Context) error {
		setting, directPool, ensureErr := ensureDepartmentDispatchSetting(txCtx, departmentID)
		if ensureErr != nil {
			return ensureErr
		}
		pool := enabledDispatchPool(txCtx, departmentID, poolID)
		if pool == nil || pool.PoolType != crmmodel.DispatchPoolTypeGroup {
			return fmt.Errorf("工作组不存在或已停用")
		}
		now := time.Now()
		if crmmodel.NewDispatchPoolModel().Update(txCtx, map[string]any{
			"id":            pool.ID,
			"department_id": departmentID,
			"status":        crmmodel.StatusEnabled,
		}, map[string]any{
			"status":     crmmodel.StatusDisabled,
			"updated_at": now,
		}) == 0 {
			return fmt.Errorf("工作组已变化，请刷新后重试")
		}
		crmmodel.NewDispatchPoolMemberModel().Update(txCtx, map[string]any{
			"pool_id": pool.ID,
			"status":  crmmodel.StatusEnabled,
		}, map[string]any{
			"status":     crmmodel.StatusDisabled,
			"updated_at": now,
		})
		if setting.ActivePoolID == pool.ID {
			if crmmodel.NewDepartmentDispatchSettingModel().Update(txCtx, map[string]any{
				"id":      setting.ID,
				"version": setting.Version,
			}, map[string]any{
				"active_pool_id": directPool.ID,
				"last_member_id": uint64(0),
				"version":        setting.Version + 1,
				"updated_at":     now,
			}) == 0 {
				return fmt.Errorf("派单配置已变化，请刷新后重试")
			}
		}
		return nil
	})
}

func dispatchDepartmentScope(
	ctx context.Context,
	staff *WorkStaffSession,
	requestedDepartmentID uint64,
) (*crmmodel.Department, []*crmmodel.Department, error) {
	if staff == nil || staff.ID == 0 {
		return nil, nil, fmt.Errorf("请先登录")
	}
	departments := dispatchManageableDepartments(ctx, staff)
	if len(departments) == 0 {
		return nil, nil, fmt.Errorf("当前账号没有派单管理权限")
	}
	if requestedDepartmentID == 0 {
		requestedDepartmentID = departments[0].ID
	}
	for _, department := range departments {
		if department != nil && department.ID == requestedDepartmentID {
			return department, departments, nil
		}
	}
	return nil, nil, fmt.Errorf("无权管理该部门派单")
}

func dispatchManageableDepartments(ctx context.Context, staff *WorkStaffSession) []*crmmodel.Department {
	if staff == nil || staff.ID == 0 {
		return nil
	}
	if staff.CanDispatch {
		return crmmodel.NewDepartmentModel().Select(ctx, map[string]any{
			"status": crmmodel.StatusEnabled,
		}, map[string]any{"order": "sort asc,id asc"})
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{
		"id":              staff.DepartmentID,
		"leader_staff_id": staff.ID,
		"status":          crmmodel.StatusEnabled,
	})
	if department == nil {
		return nil
	}
	return []*crmmodel.Department{department}
}

func ensureDepartmentDispatchSetting(
	ctx context.Context,
	departmentID uint64,
) (*crmmodel.DepartmentDispatchSetting, *crmmodel.DispatchPool, error) {
	if !enabledDepartment(ctx, departmentID) {
		return nil, nil, fmt.Errorf("目标部门不存在或已停用")
	}
	poolModel := crmmodel.NewDispatchPoolModel()
	directPool := poolModel.Find(ctx, map[string]any{
		"department_id": departmentID,
		"pool_type":     crmmodel.DispatchPoolTypeDirect,
	})
	directPoolCreated := false
	now := time.Now()
	if directPool == nil {
		poolID := uint64(poolModel.Insert(ctx, map[string]any{
			"department_id": departmentID,
			"name":          "按员工分配",
			"pool_type":     crmmodel.DispatchPoolTypeDirect,
			"status":        crmmodel.StatusEnabled,
			"sort":          10,
			"created_at":    now,
			"updated_at":    now,
		}))
		directPool = poolModel.Find(ctx, map[string]any{"id": poolID})
		directPoolCreated = directPool != nil
	} else if directPool.Status != crmmodel.StatusEnabled {
		poolModel.Update(ctx, map[string]any{"id": directPool.ID}, map[string]any{
			"status":     crmmodel.StatusEnabled,
			"updated_at": now,
		})
		directPool.Status = crmmodel.StatusEnabled
	}
	if directPool == nil {
		return nil, nil, fmt.Errorf("默认派单池创建失败")
	}
	settingModel := crmmodel.NewDepartmentDispatchSettingModel()
	setting := settingModel.Find(ctx, map[string]any{"department_id": departmentID})
	settingCreated := false
	if setting == nil {
		settingID := uint64(settingModel.Insert(ctx, map[string]any{
			"department_id":  departmentID,
			"active_pool_id": directPool.ID,
			"last_member_id": uint64(0),
			"version":        uint64(1),
			"status":         crmmodel.StatusEnabled,
			"created_at":     now,
			"updated_at":     now,
		}))
		setting = settingModel.Find(ctx, map[string]any{"id": settingID})
		settingCreated = setting != nil
	}
	if setting == nil {
		return nil, nil, fmt.Errorf("部门派单配置创建失败")
	}
	if setting.ActivePoolID == 0 || enabledDispatchPool(ctx, departmentID, setting.ActivePoolID) == nil {
		if settingModel.Update(ctx, map[string]any{
			"id":      setting.ID,
			"version": setting.Version,
		}, map[string]any{
			"active_pool_id": directPool.ID,
			"last_member_id": uint64(0),
			"version":        setting.Version + 1,
			"status":         crmmodel.StatusEnabled,
			"updated_at":     now,
		}) == 0 {
			return nil, nil, fmt.Errorf("部门派单配置已变化，请刷新后重试")
		}
		setting.ActivePoolID = directPool.ID
		setting.LastMemberID = 0
		setting.Version++
		setting.Status = crmmodel.StatusEnabled
		setting.UpdatedAt = now
	}
	if shouldSeedInitialDispatchMembers(ctx, setting, directPool, directPoolCreated || settingCreated) {
		if err := seedInitialDispatchPoolMembers(ctx, departmentID, directPool.ID); err != nil {
			return nil, nil, err
		}
	}
	return setting, directPool, nil
}

func shouldSeedInitialDispatchMembers(
	ctx context.Context,
	setting *crmmodel.DepartmentDispatchSetting,
	directPool *crmmodel.DispatchPool,
	created bool,
) bool {
	if setting == nil || directPool == nil {
		return false
	}
	if crmmodel.NewDispatchPoolMemberModel().Count(ctx, map[string]any{"pool_id": directPool.ID}) > 0 {
		return false
	}
	if created {
		return true
	}
	return setting.Version <= 1 && setting.LastMemberID == 0 && directPool.UpdatedAt.Equal(directPool.CreatedAt)
}

func seedInitialDispatchPoolMembers(ctx context.Context, departmentID, poolID uint64) error {
	if departmentID == 0 || poolID == 0 {
		return fmt.Errorf("默认派单池不存在")
	}
	memberModel := crmmodel.NewDispatchPoolMemberModel()
	now := time.Now()
	for index, staff := range enabledDepartmentStaff(ctx, departmentID) {
		if staff == nil {
			continue
		}
		if memberModel.Insert(ctx, map[string]any{
			"pool_id":              poolID,
			"department_id":        departmentID,
			"staff_id":             staff.ID,
			"weekly_schedule_json": crmmodel.DefaultDispatchScheduleJSON,
			"daily_limit":          0,
			"status":               crmmodel.StatusEnabled,
			"sort":                 (index + 1) * 10,
			"created_at":           now,
			"updated_at":           now,
		}) == 0 {
			return fmt.Errorf("默认派单成员创建失败")
		}
	}
	return nil
}

func ensureDepartmentDispatchDefaults(ctx context.Context) error {
	crmmodel.NewDispatchRecordModel().Count(ctx, map[string]any{})
	for _, department := range crmmodel.NewDepartmentModel().Select(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"}) {
		if department == nil {
			continue
		}
		if err := orm.Transaction(ctx, func(txCtx context.Context) error {
			_, _, ensureErr := ensureDepartmentDispatchSetting(txCtx, department.ID)
			return ensureErr
		}); err != nil {
			return fmt.Errorf("初始化%s派单配置失败：%w", department.Name, err)
		}
	}
	return nil
}

func enabledDispatchPool(ctx context.Context, departmentID, poolID uint64) *crmmodel.DispatchPool {
	if departmentID == 0 || poolID == 0 {
		return nil
	}
	return crmmodel.NewDispatchPoolModel().Find(ctx, map[string]any{
		"id":            poolID,
		"department_id": departmentID,
		"status":        crmmodel.StatusEnabled,
	})
}

type dispatchMemberInput struct {
	StaffID            uint64
	DailyLimit         int
	Status             int16
	Sort               int
	WeeklyScheduleJSON string
}

func normalizeDispatchMemberInputs(value any) ([]dispatchMemberInput, error) {
	rows := mapListFromAny(value)
	result := make([]dispatchMemberInput, 0, len(rows))
	seen := map[uint64]bool{}
	for index, row := range rows {
		staffID := firstUint64(row, "staff_id", "staffId", "id")
		if staffID == 0 {
			return nil, fmt.Errorf("派单成员不能为空")
		}
		if seen[staffID] {
			return nil, fmt.Errorf("同一员工不能重复加入派单池")
		}
		seen[staffID] = true
		dailyLimit := inputInt(firstPresent(row, "daily_limit", "dailyLimit"))
		if dailyLimit < 0 || dailyLimit > 100000 {
			return nil, fmt.Errorf("每日上限必须在0到100000之间")
		}
		status := crmmodel.StatusEnabled
		if rawStatus, exists := row["status"]; exists && inputInt(rawStatus) == int(crmmodel.StatusDisabled) {
			status = crmmodel.StatusDisabled
		}
		if enabled, exists := row["enabled"]; exists && !booleanFromAny(enabled) {
			status = crmmodel.StatusDisabled
		}
		scheduleJSON, err := normalizeDispatchSchedule(firstPresent(row, "weekly_schedule", "weeklySchedule", "weekly_schedule_json", "weeklyScheduleJson"))
		if err != nil {
			return nil, fmt.Errorf("员工派单时间无效：%w", err)
		}
		result = append(result, dispatchMemberInput{
			StaffID:            staffID,
			DailyLimit:         dailyLimit,
			Status:             status,
			Sort:               (index + 1) * 10,
			WeeklyScheduleJSON: scheduleJSON,
		})
	}
	return result, nil
}

func normalizeDispatchSchedule(value any) (string, error) {
	if value == nil || strings.TrimSpace(inputText(value)) == "" {
		return crmmodel.DefaultDispatchScheduleJSON, nil
	}
	var raw []byte
	if text, ok := value.(string); ok {
		raw = []byte(strings.TrimSpace(text))
	} else {
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		raw = encoded
	}
	var schedule weeklyDispatchSchedule
	if err := json.Unmarshal(raw, &schedule); err != nil {
		return "", fmt.Errorf("工作时间格式错误")
	}
	if len(schedule) == 0 {
		return crmmodel.DefaultDispatchScheduleJSON, nil
	}
	normalized := weeklyDispatchSchedule{}
	for day := 1; day <= 7; day++ {
		key := fmt.Sprint(day)
		periods := append([][2]int(nil), schedule[key]...)
		sort.Slice(periods, func(left, right int) bool {
			return periods[left][0] < periods[right][0]
		})
		for index, period := range periods {
			if period[0] < 0 || period[1] > 1440 || period[0] >= period[1] {
				return "", fmt.Errorf("星期%d存在无效时段", day)
			}
			if index > 0 && periods[index-1][1] > period[0] {
				return "", fmt.Errorf("星期%d的工作时段不能重叠", day)
			}
		}
		normalized[key] = periods
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func saveDispatchPoolMembers(
	ctx context.Context,
	departmentID uint64,
	poolID uint64,
	members []dispatchMemberInput,
) error {
	model := crmmodel.NewDispatchPoolMemberModel()
	existingRows := model.Select(ctx, map[string]any{"pool_id": poolID})
	existingByStaff := make(map[uint64]*crmmodel.DispatchPoolMember, len(existingRows))
	for _, row := range existingRows {
		if row != nil {
			existingByStaff[row.StaffID] = row
		}
	}
	now := time.Now()
	seen := map[uint64]bool{}
	for _, member := range members {
		if enabledStaffInDepartment(ctx, member.StaffID, departmentID) == nil {
			return fmt.Errorf("所选员工不属于当前部门或已停用")
		}
		seen[member.StaffID] = true
		data := map[string]any{
			"department_id":        departmentID,
			"weekly_schedule_json": member.WeeklyScheduleJSON,
			"daily_limit":          member.DailyLimit,
			"status":               member.Status,
			"sort":                 member.Sort,
			"updated_at":           now,
		}
		if current := existingByStaff[member.StaffID]; current != nil {
			if model.Update(ctx, map[string]any{"id": current.ID, "pool_id": poolID}, data) == 0 {
				return fmt.Errorf("派单成员已变化，请刷新后重试")
			}
			continue
		}
		data["pool_id"] = poolID
		data["staff_id"] = member.StaffID
		data["created_at"] = now
		if model.Insert(ctx, data) == 0 {
			return fmt.Errorf("派单成员保存失败")
		}
	}
	for _, current := range existingRows {
		if current == nil || seen[current.StaffID] {
			continue
		}
		model.Delete(ctx, map[string]any{"id": current.ID, "pool_id": poolID})
	}
	return nil
}

func nextDispatchPoolSort(ctx context.Context, departmentID uint64) int {
	rows := crmmodel.NewDispatchPoolModel().Select(ctx, map[string]any{"department_id": departmentID})
	maxSort := 10
	for _, row := range rows {
		if row != nil && row.Sort > maxSort {
			maxSort = row.Sort
		}
	}
	return maxSort + 10
}

func departmentDispatchConfigPayload(
	ctx context.Context,
	staff *WorkStaffSession,
	department *crmmodel.Department,
	departments []*crmmodel.Department,
) map[string]any {
	setting := crmmodel.NewDepartmentDispatchSettingModel().Find(ctx, map[string]any{"department_id": department.ID})
	todayAutoCounts := dispatchDepartmentAutoCountsToday(ctx, department.ID, time.Now())
	departmentRows := make([]map[string]any, 0, len(departments))
	for _, row := range departments {
		if row != nil {
			departmentRows = append(departmentRows, map[string]any{"id": row.ID, "name": row.Name, "code": row.Code})
		}
	}
	staffRows := enabledDepartmentStaff(ctx, department.ID)
	staffPayload := make([]map[string]any, 0, len(staffRows))
	for _, row := range staffRows {
		if row != nil {
			staffPayload = append(staffPayload, map[string]any{
				"id":               row.ID,
				"name":             row.Name,
				"phone":            row.Phone,
				"staff_type":       row.StaffType,
				"today_auto_count": todayAutoCounts[row.ID],
			})
		}
	}
	pools := crmmodel.NewDispatchPoolModel().Select(ctx, map[string]any{
		"department_id": department.ID,
		"status":        crmmodel.StatusEnabled,
	}, map[string]any{"order": "sort asc,id asc"})
	poolPayload := make([]map[string]any, 0, len(pools))
	for _, pool := range pools {
		if pool == nil {
			continue
		}
		members := crmmodel.NewDispatchPoolMemberModel().Select(ctx, map[string]any{
			"pool_id": pool.ID,
		}, map[string]any{"order": "sort asc,id asc"})
		memberPayload := make([]map[string]any, 0, len(members))
		for _, member := range members {
			if member == nil {
				continue
			}
			if enabledStaffInDepartment(ctx, member.StaffID, department.ID) == nil {
				continue
			}
			var schedule weeklyDispatchSchedule
			if err := json.Unmarshal([]byte(member.WeeklyScheduleJSON), &schedule); err != nil {
				schedule = weeklyDispatchSchedule{}
			}
			memberPayload = append(memberPayload, map[string]any{
				"id":                   member.ID,
				"staff_id":             member.StaffID,
				"daily_limit":          member.DailyLimit,
				"status":               member.Status,
				"sort":                 member.Sort,
				"weekly_schedule":      schedule,
				"weekly_schedule_json": member.WeeklyScheduleJSON,
				"today_auto_count":     todayAutoCounts[member.StaffID],
			})
		}
		poolPayload = append(poolPayload, map[string]any{
			"id":          pool.ID,
			"name":        pool.Name,
			"pool_type":   pool.PoolType,
			"is_active":   setting != nil && setting.ActivePoolID == pool.ID,
			"member_list": memberPayload,
		})
	}
	activePoolID := uint64(0)
	version := uint64(0)
	if setting != nil {
		activePoolID = setting.ActivePoolID
		version = setting.Version
	}
	return map[string]any{
		"can_manage":      true,
		"is_global":       staff.CanDispatch,
		"department_id":   department.ID,
		"department_name": department.Name,
		"departments":     departmentRows,
		"staff":           staffPayload,
		"active_pool_id":  activePoolID,
		"version":         version,
		"pools":           poolPayload,
		"pending":         pendingDepartmentDispatchRows(ctx, department.ID),
		"pending_count":   pendingDepartmentDispatchCount(ctx, department.ID),
	}
}

func departmentDispatchConfigAfterRetry(
	ctx context.Context,
	staff *WorkStaffSession,
	department *crmmodel.Department,
	departments []*crmmodel.Department,
) map[string]any {
	summary, retryErr := RetryPendingDepartmentDispatch(ctx, department.ID)
	result := departmentDispatchConfigPayload(ctx, staff, department, departments)
	result["retry"] = summary
	if retryErr != nil {
		result["retry_warning"] = retryErr.Error()
	}
	return result
}

func dispatchDepartmentAutoCountsToday(ctx context.Context, departmentID uint64, now time.Time) map[uint64]int64 {
	local := now.In(departmentDispatchLocation)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, departmentDispatchLocation)
	rows := crmmodel.NewDispatchRecordModel().Select(ctx, map[string]any{
		"dispatch_type": crmmodel.DispatchTypeAuto,
		"department_id": departmentID,
		"created_at":    map[string]any{">=": dayStart, "<": dayStart.AddDate(0, 0, 1)},
	})
	counts := make(map[uint64]int64)
	for _, row := range rows {
		if row != nil && row.StaffID > 0 {
			counts[row.StaffID]++
		}
	}
	return counts
}

func pendingDepartmentDispatchRows(ctx context.Context, departmentID uint64) []map[string]any {
	rows := make([]map[string]any, 0)
	reservedInstanceIDs := reservedLeadDispatchInstanceIDs(ctx, departmentID)
	for _, instance := range crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{
		"owner_department_id": departmentID,
		"owner_staff_id":      uint64(0),
		"status":              crmmodel.ProgressStatusActive,
	}, map[string]any{"order": "started_at asc,id asc"}) {
		if instance == nil || reservedInstanceIDs[instance.ID] {
			continue
		}
		row := pendingDispatchSubjectRow(ctx, instance)
		row["kind"] = "stage"
		row["id"] = instance.ID
		row["workflow_instance_id"] = instance.ID
		row["stage_id"] = instance.StageID
		row["created_at"] = instance.StartedAt
		if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": instance.StageID}); stage != nil {
			row["title"] = stage.Name
		}
		rows = append(rows, row)
	}
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"assignee_department_id": departmentID,
		"assignee_staff_id":      uint64(0),
		"status":                 crmmodel.WorkTodoStatusPending,
	}, map[string]any{"order": "created_at asc,id asc"}) {
		if todo == nil {
			continue
		}
		instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": todo.WorkflowInstanceID})
		if instance == nil || instance.Status != crmmodel.ProgressStatusActive || instance.StageID != todo.StageID {
			continue
		}
		row := pendingDispatchSubjectRow(ctx, instance)
		row["kind"] = "task"
		row["id"] = todo.ID
		row["todo_id"] = todo.ID
		row["workflow_instance_id"] = todo.WorkflowInstanceID
		row["stage_id"] = todo.StageID
		row["created_at"] = todo.CreatedAt
		if task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID}); task != nil {
			row["title"] = task.Name
			row["task_id"] = task.ID
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(left, right int) bool {
		leftTime, _ := rows[left]["created_at"].(time.Time)
		rightTime, _ := rows[right]["created_at"].(time.Time)
		return leftTime.Before(rightTime)
	})
	return rows
}

func pendingDispatchSubjectRow(ctx context.Context, instance *crmmodel.WorkflowInstance) map[string]any {
	row := map[string]any{
		"lead_id":      instance.LeadID,
		"customer_id":  instance.CustomerID,
		"asset_id":     instance.AssetID,
		"subject_name": "",
		"subject_no":   "",
	}
	if instance.LeadID > 0 {
		if lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": instance.LeadID}); lead != nil {
			row["subject_name"] = lead.Name
			row["subject_no"] = lead.Code
		}
		return row
	}
	if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": instance.CustomerID}); customer != nil {
		row["subject_name"] = customer.Name
		row["subject_no"] = customer.Code
	}
	if asset := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": instance.AssetID}); asset != nil {
		row["asset_name"] = asset.AssetName
		row["asset_no"] = asset.AssetNo
	}
	return row
}

func workDispatchNavigation(ctx context.Context, staff *WorkStaffSession) map[string]any {
	scopes := manageableLeadDispatchScopes(ctx, staff)
	if len(scopes) == 0 {
		return map[string]any{"enabled": false, "pending_count": 0}
	}
	pendingCount := 0
	for _, scope := range scopes {
		if scope != nil {
			pendingCount += pendingLeadDispatchCount(ctx, scope.Workflow.ID)
		}
	}
	return map[string]any{
		"enabled":       true,
		"pending_count": pendingCount,
	}
}

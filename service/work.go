package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	deverjwt "github.com/shemic/dever/auth/jwt"
	"github.com/shemic/dever/config"

	crmmodel "my/package/crm/model"
	frontservice "my/package/front/service"
	fronteval "my/package/front/service/eval"
)

const (
	workSiteKey              = "work"
	workAuthProvider         = "crm_work"
	workResultSuccess        = "success"
	workCustomerModePending  = "pending"
	workCustomerModeDone     = "done"
	maxWorkAutoTriggerDepth  = 5
	workTransitionModeAll    = "all"
	workTransitionModeAny    = "any"
	workTransitionScriptPass = "pass"
)

type WorkService struct{}

type WorkStaffSession struct {
	ID           uint64
	Name         string
	Phone        string
	DepartmentID uint64
}

type workExecutionRuntime struct {
	depth int
	seen  map[string]bool
}

func NewWorkService() WorkService {
	return WorkService{}
}

func (WorkService) Login(ctx context.Context, payload map[string]any) (map[string]any, error) {
	phone := firstText(payload, "phone", "account")
	password := firstText(payload, "password")
	if phone == "" || password == "" {
		return nil, fmt.Errorf("手机号和密码不能为空")
	}
	staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"phone":  phone,
		"status": crmmodel.StatusEnabled,
	})
	if staff == nil || !VerifyCRMStaffPassword(staff.Password, password) {
		return nil, fmt.Errorf("手机号或密码错误")
	}
	expiredAt := time.Now().Add(7 * 24 * time.Hour)
	token, err := createWorkToken(staff, expiredAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"token": token,
		"user":  workStaffPayload(staff, expiredAt),
	}, nil
}

func (WorkService) Me(_ context.Context, staff *WorkStaffSession) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	return map[string]any{"user": staff}, nil
}

func (WorkService) Customers(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	customers := workCustomersByMode(ctx, staff, firstText(payload, "mode"))
	if hasWorkCustomerStructuredFilter(payload) {
		customers = filterWorkCustomersByFields(customers, payload)
	}
	keyword := firstText(payload, "keyword")
	if keyword != "" {
		customers = filterWorkCustomers(customers, keyword)
	}
	return map[string]any{
		"list":  customers,
		"total": len(customers),
	}, nil
}

func (WorkService) Operations(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	filter := map[string]any{}
	if customerID := firstUint64(payload, "customer_id", "customerId"); customerID > 0 {
		if !canViewWorkCustomer(ctx, staff, customerID) {
			return nil, fmt.Errorf("无权查看该客户")
		}
		filter["customer_id"] = customerID
	} else {
		filter["operator_staff_id"] = staff.ID
	}
	if assetID := firstUint64(payload, "asset_id", "assetId"); assetID > 0 {
		filter["asset_id"] = assetID
	}
	if booleanFromAny(payload["mine"]) {
		filter["operator_staff_id"] = staff.ID
	}
	rows := crmmodel.NewOperationLogModel().SelectMap(ctx, filter)
	if keyword := firstText(payload, "keyword"); keyword != "" {
		rows = filterWorkOperations(rows, keyword)
	}
	enrichWorkOperationRows(ctx, staff, rows)
	return map[string]any{
		"list":  rows,
		"total": len(rows),
	}, nil
}

func (WorkService) Bookings(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	rows := crmmodel.NewPublicResourceBookingModel().SelectMap(ctx, map[string]any{
		"booker_staff_id": staff.ID,
	})
	for _, row := range rows {
		enrichWorkBookingRow(ctx, row)
	}
	if keyword := firstText(payload, "keyword"); keyword != "" {
		rows = filterWorkBookings(rows, keyword)
	}
	return map[string]any{
		"list":  rows,
		"total": len(rows),
	}, nil
}

func (WorkService) Tasks(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID ...uint64) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	currentAssetID := firstOptionalUint64(assetID)
	if customerID == 0 {
		tasks := workDepartmentTasks(ctx, staff)
		return map[string]any{
			"list":  tasks,
			"total": len(tasks),
		}, nil
	}
	state := ensureCurrentWorkCustomerStage(ctx, staff, customerID, currentAssetID)
	stageCode := ""
	if state != nil {
		stageCode = state.CurrentStageCode
	}
	tasks := workAvailableTasks(ctx, staff, state)
	if currentAssetID > 0 {
		tasks = workAssetRowTasks(tasks)
	} else {
		tasks = workCustomerRowTasks(tasks)
	}
	return map[string]any{
		"list":       tasks,
		"total":      len(tasks),
		"stage_code": stageCode,
	}, nil
}

func (WorkService) Options(ctx context.Context, staff *WorkStaffSession) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	return map[string]any{
		"departments": workDepartmentOptions(ctx),
		"staffs":      workStaffOptions(ctx),
	}, nil
}

func (WorkService) Execute(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	return executeWorkTask(ctx, staff, payload, newWorkExecutionRuntime())
}

func executeWorkTask(ctx context.Context, staff *WorkStaffSession, payload map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	taskID := firstUint64(payload, "task_id", "taskId")
	if taskID == 0 {
		return nil, fmt.Errorf("任务不能为空")
	}
	customerID := firstUint64(payload, "customer_id", "customerId")
	assetID := firstUint64(payload, "asset_id", "assetId")
	if assetID > 0 && !workCustomerOwnsAsset(ctx, customerID, assetID) {
		return nil, fmt.Errorf("客户资产不存在")
	}
	task := workAllowedTask(ctx, staff, taskID, customerID, assetID)
	if task == nil {
		return nil, fmt.Errorf("当前人员无权执行该任务")
	}
	if !beginWorkTaskExecution(runtime, customerID, assetID, task.ID) {
		return map[string]any{"customer_id": customerID, "skipped": true}, nil
	}
	defer endWorkTaskExecution(runtime, customerID, assetID, task.ID)
	switch task.TaskType {
	case crmmodel.TaskTypeCreate:
		if customerID > 0 {
			return nil, fmt.Errorf("创建资料任务不能在已有客户上执行")
		}
		return executeCreateCustomerTask(ctx, staff, task, mapFromAny(payload["values"]), runtime)
	case crmmodel.TaskTypeForm:
		return executeFormTask(ctx, staff, task, customerID, assetID, mapFromAny(payload["values"]), runtime)
	case crmmodel.TaskTypeAssign:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		return executeAssignCustomerTask(ctx, staff, task, customerID, assetID, workActionValues(payload), runtime)
	case crmmodel.TaskTypeDecision:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		return executeDecisionCustomerTask(ctx, staff, task, customerID, assetID, mapFromAny(payload["values"]), runtime)
	case crmmodel.TaskTypeBooking:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		return executeBookingCustomerTask(ctx, staff, task, customerID, assetID, mapFromAny(payload["values"]), runtime)
	default:
		return nil, fmt.Errorf("该任务动作暂未接入工作台")
	}
}

func executeFormTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if customerID == 0 {
		return nil, fmt.Errorf("客户不能为空")
	}
	return executeEditFormTask(ctx, staff, task, customerID, assetID, values, runtime)
}

func newWorkExecutionRuntime() *workExecutionRuntime {
	return &workExecutionRuntime{seen: map[string]bool{}}
}

func beginWorkTaskExecution(runtime *workExecutionRuntime, customerID uint64, assetID uint64, taskID uint64) bool {
	if runtime == nil {
		return true
	}
	if runtime.seen == nil {
		runtime.seen = map[string]bool{}
	}
	if runtime.depth >= maxWorkAutoTriggerDepth {
		return false
	}
	key := fmt.Sprintf("%d:%d:%d", customerID, assetID, taskID)
	if runtime.seen[key] {
		return false
	}
	runtime.seen[key] = true
	runtime.depth++
	return true
}

func endWorkTaskExecution(runtime *workExecutionRuntime, customerID uint64, assetID uint64, taskID uint64) {
	if runtime == nil {
		return
	}
	if runtime.depth > 0 {
		runtime.depth--
	}
}

func CurrentWorkStaff(ctx context.Context) *WorkStaffSession {
	claims := deverjwt.Claims(ctx)
	staffID := inputUint64(claims["staff_id"])
	if staffID == 0 {
		staffID = inputUint64(claims["uid"])
	}
	if staffID == 0 {
		return nil
	}
	staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"id":     staffID,
		"status": crmmodel.StatusEnabled,
	})
	if staff == nil {
		return nil
	}
	return &WorkStaffSession{
		ID:           staff.ID,
		Name:         staff.Name,
		Phone:        staff.Phone,
		DepartmentID: staff.DepartmentID,
	}
}

func VerifyCRMStaffPassword(stored string, password string) bool {
	if inputText(stored) == "" || inputText(password) == "" {
		return false
	}
	hashed := frontservice.HashPlainPassword(password)
	return stored == hashed || stored == password
}

func createWorkToken(staff *crmmodel.Staff, expiredAt time.Time) (string, error) {
	cfg, err := config.Load("")
	if err != nil {
		return "", fmt.Errorf("读取配置失败")
	}
	signer, err := deverjwt.ResolveSigner(cfg.Auth, "user", "default")
	if err != nil {
		return "", fmt.Errorf("JWT密钥未配置")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":           fmt.Sprintf("%d", staff.ID),
		"staff_id":      fmt.Sprintf("%d", staff.ID),
		"department_id": fmt.Sprintf("%d", staff.DepartmentID),
		"site":          workSiteKey,
		"scope":         workAuthProvider,
		"exp":           expiredAt.Unix(),
		"iat":           time.Now().Unix(),
	})
	return token.SignedString([]byte(signer.Secret))
}

func workStaffPayload(staff *crmmodel.Staff, expiredAt time.Time) map[string]any {
	return map[string]any{
		"id":            staff.ID,
		"name":          staff.Name,
		"phone":         staff.Phone,
		"department_id": staff.DepartmentID,
		"exp":           expiredAt.UnixMilli(),
	}
}

func visibleWorkCustomers(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	members := crmmodel.NewCustomerMemberModel().Select(ctx, map[string]any{
		"staff_id": staff.ID,
		"status":   crmmodel.StatusEnabled,
	})
	seen := map[uint64]bool{}
	rows := make([]map[string]any, 0, len(members))
	for _, member := range members {
		if member == nil || member.CustomerID == 0 || seen[member.CustomerID] || !member.CanView {
			continue
		}
		rows = appendVisibleWorkCustomer(ctx, staff, rows, seen, member.CustomerID)
	}
	if staff.DepartmentID > 0 {
		departmentMembers := crmmodel.NewCustomerMemberModel().Select(ctx, map[string]any{
			"department_id": staff.DepartmentID,
			"status":        crmmodel.StatusEnabled,
		})
		for _, member := range departmentMembers {
			if member == nil || member.CustomerID == 0 || seen[member.CustomerID] || !member.CanView {
				continue
			}
			rows = appendVisibleWorkCustomer(ctx, staff, rows, seen, member.CustomerID)
		}
	}
	created := crmmodel.NewCustomerModel().Select(ctx, map[string]any{"created_by_staff_id": staff.ID})
	for _, customer := range created {
		if customer == nil || seen[customer.ID] {
			continue
		}
		rows = appendVisibleWorkCustomer(ctx, staff, rows, seen, customer.ID)
	}
	for _, state := range crmmodel.NewCustomerStageModel().Select(ctx, map[string]any{"current_staff_id": staff.ID}) {
		if state == nil || state.CustomerID == 0 || seen[state.CustomerID] {
			continue
		}
		rows = appendVisibleWorkCustomer(ctx, staff, rows, seen, state.CustomerID)
	}
	if staff.DepartmentID > 0 {
		for _, state := range crmmodel.NewCustomerStageModel().Select(ctx, map[string]any{"current_department_id": staff.DepartmentID}) {
			if state == nil || state.CustomerID == 0 || seen[state.CustomerID] {
				continue
			}
			rows = appendVisibleWorkCustomer(ctx, staff, rows, seen, state.CustomerID)
		}
	}
	return rows
}

func pendingWorkCustomers(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	return workRowsWithPendingTasks(visibleWorkCustomers(ctx, staff))
}

func workCustomersByMode(ctx context.Context, staff *WorkStaffSession, mode string) []map[string]any {
	switch normalizeWorkCustomerMode(mode) {
	case workCustomerModeDone:
		return doneWorkCustomers(ctx, staff)
	default:
		return pendingWorkCustomers(ctx, staff)
	}
}

func normalizeWorkCustomerMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case workCustomerModeDone:
		return workCustomerModeDone
	default:
		return workCustomerModePending
	}
}

type doneWorkCustomerTarget struct {
	customerID uint64
	assetIDs   []uint64
}

func doneWorkCustomers(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	targets := doneWorkCustomerTargets(ctx, staff)
	rows := make([]map[string]any, 0, len(targets))
	for _, target := range targets {
		if row := doneWorkCustomerRow(ctx, staff, target.customerID, target.assetIDs); len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func doneWorkCustomerTargets(ctx context.Context, staff *WorkStaffSession) []doneWorkCustomerTarget {
	rows := crmmodel.NewOperationLogModel().SelectMap(ctx, map[string]any{
		"operator_staff_id": staff.ID,
	})
	targetIndexes := map[uint64]int{}
	seenAssetIDs := map[uint64]map[uint64]bool{}
	targets := make([]doneWorkCustomerTarget, 0, len(rows))
	for _, row := range rows {
		customerID := inputUint64(row["customer_id"])
		if customerID == 0 {
			continue
		}
		index, exists := targetIndexes[customerID]
		if !exists {
			index = len(targets)
			targetIndexes[customerID] = index
			targets = append(targets, doneWorkCustomerTarget{customerID: customerID})
		}
		assetID := inputUint64(row["asset_id"])
		if assetID == 0 {
			continue
		}
		if seenAssetIDs[customerID] == nil {
			seenAssetIDs[customerID] = map[uint64]bool{}
		}
		if seenAssetIDs[customerID][assetID] {
			continue
		}
		seenAssetIDs[customerID][assetID] = true
		targets[index].assetIDs = append(targets[index].assetIDs, assetID)
	}
	return targets
}

func doneWorkCustomerRow(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetIDs []uint64) map[string]any {
	if customerID == 0 {
		return map[string]any{}
	}
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	if len(customer) == 0 {
		return map[string]any{}
	}
	customer["data_values"] = workCustomerFormValues(ctx, customerID, 0, customer)
	customer["data_value_labels"] = workDataValueLabels(ctx, mapFromAny(customer["data_values"]))
	customer["assets"] = doneWorkAssetRows(ctx, staff, customerID, assetIDs)
	if state := currentWorkCustomerStage(ctx, customerID, 0); state != nil {
		customer["state.id"] = state.ID
		customer["state.current_stage_code"] = state.CurrentStageCode
		customer["state.current_department_id"] = state.CurrentDepartmentID
		customer["state.current_staff_id"] = state.CurrentStaffID
		customer["stage_code"] = state.CurrentStageCode
		customer["stage_name"] = workStageName(ctx, state.CurrentStageCode)
	}
	customer["row_tasks"] = []map[string]any{}
	customer["tasks"] = []map[string]any{}
	customer["edit_tasks"] = []map[string]any{}
	enrichWorkCustomerRow(ctx, customer)
	return customer
}

func doneWorkAssetRows(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetIDs []uint64) []map[string]any {
	if customerID == 0 || len(assetIDs) == 0 {
		return []map[string]any{}
	}
	rows := make([]map[string]any, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		if assetID == 0 {
			continue
		}
		if asset := doneWorkAssetRow(ctx, staff, customerID, assetID); len(asset) > 0 {
			rows = append(rows, asset)
		}
	}
	return rows
}

func doneWorkAssetRow(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) map[string]any {
	asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
		"id":          assetID,
		"customer_id": customerID,
	})
	if len(asset) == 0 {
		return map[string]any{}
	}
	asset["data_values"] = workAssetFormValues(ctx, customerID, assetID, asset)
	asset["data_value_labels"] = workDataValueLabels(ctx, mapFromAny(asset["data_values"]))
	asset["asset_status_name"] = workAssetStatusName(ctx, inputUint64(asset["asset_status_id"]))
	if state := currentWorkCustomerStage(ctx, customerID, assetID); state != nil {
		asset["state.id"] = state.ID
		asset["state.current_stage_code"] = state.CurrentStageCode
		asset["state.current_department_id"] = state.CurrentDepartmentID
		asset["state.current_staff_id"] = state.CurrentStaffID
		asset["stage_code"] = state.CurrentStageCode
		asset["stage_name"] = workStageName(ctx, state.CurrentStageCode)
	}
	asset["row_tasks"] = []map[string]any{}
	asset["tasks"] = []map[string]any{}
	return asset
}

func workRowsWithPendingTasks(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if pendingRow, ok := workRowWithPendingTasks(row); ok {
			result = append(result, pendingRow)
		}
	}
	return result
}

func workRowWithPendingTasks(row map[string]any) (map[string]any, bool) {
	if len(row) == 0 {
		return nil, false
	}
	rowTasks := mapListFromAny(row["row_tasks"])
	assets := mapListFromAny(row["assets"])
	pendingAssets := workAssetsWithPendingTasks(assets)
	if len(rowTasks) == 0 && len(pendingAssets) == 0 {
		return nil, false
	}
	pendingRow := copyMap(row)
	if len(assets) > 0 {
		pendingRow["assets"] = pendingAssets
	}
	return pendingRow, true
}

func workAssetsWithPendingTasks(assets []map[string]any) []map[string]any {
	if len(assets) == 0 {
		return assets
	}
	result := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		if len(mapListFromAny(asset["row_tasks"])) > 0 {
			result = append(result, asset)
		}
	}
	return result
}

func canViewWorkCustomer(ctx context.Context, staff *WorkStaffSession, customerID uint64) bool {
	if staff == nil || staff.ID == 0 || customerID == 0 {
		return false
	}
	if crmmodel.NewOperationLogModel().Find(ctx, map[string]any{
		"customer_id":       customerID,
		"operator_staff_id": staff.ID,
	}) != nil {
		return true
	}
	if canOperateCurrentState(staff, currentWorkCustomerStage(ctx, customerID, 0)) {
		return true
	}
	if crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"customer_id":      customerID,
		"current_staff_id": staff.ID,
	}) != nil {
		return true
	}
	if staff.DepartmentID > 0 {
		if crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
			"customer_id":           customerID,
			"current_department_id": staff.DepartmentID,
		}) != nil {
			return true
		}
	}
	if crmmodel.NewCustomerMemberModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"staff_id":    staff.ID,
		"status":      crmmodel.StatusEnabled,
		"can_view":    true,
	}) != nil {
		return true
	}
	if staff.DepartmentID > 0 {
		if crmmodel.NewCustomerMemberModel().Find(ctx, map[string]any{
			"customer_id":   customerID,
			"department_id": staff.DepartmentID,
			"status":        crmmodel.StatusEnabled,
			"can_view":      true,
		}) != nil {
			return true
		}
	}
	if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}); customer != nil {
		return customer.CreatedByStaffID == staff.ID
	}
	return false
}

func canViewWorkAsset(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) bool {
	if staff == nil || customerID == 0 || assetID == 0 {
		return false
	}
	if canOperateCurrentState(staff, currentWorkCustomerStage(ctx, customerID, assetID)) {
		return true
	}
	if crmmodel.NewCustomerMemberModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    0,
		"staff_id":    staff.ID,
		"status":      crmmodel.StatusEnabled,
		"can_view":    true,
	}) != nil {
		return true
	}
	if crmmodel.NewCustomerMemberModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"staff_id":    staff.ID,
		"status":      crmmodel.StatusEnabled,
		"can_view":    true,
	}) != nil {
		return true
	}
	if staff.DepartmentID > 0 {
		if crmmodel.NewCustomerMemberModel().Find(ctx, map[string]any{
			"customer_id":   customerID,
			"asset_id":      0,
			"department_id": staff.DepartmentID,
			"status":        crmmodel.StatusEnabled,
			"can_view":      true,
		}) != nil {
			return true
		}
		if crmmodel.NewCustomerMemberModel().Find(ctx, map[string]any{
			"customer_id":   customerID,
			"asset_id":      assetID,
			"department_id": staff.DepartmentID,
			"status":        crmmodel.StatusEnabled,
			"can_view":      true,
		}) != nil {
			return true
		}
	}
	return false
}

func appendVisibleWorkCustomer(ctx context.Context, staff *WorkStaffSession, rows []map[string]any, seen map[uint64]bool, customerID uint64) []map[string]any {
	if customerID == 0 || seen[customerID] {
		return rows
	}
	if row := workCustomerRow(ctx, staff, customerID); len(row) > 0 {
		rows = append(rows, row)
		seen[customerID] = true
	}
	return rows
}

func workCustomerRow(ctx context.Context, staff *WorkStaffSession, customerID uint64) map[string]any {
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	if len(customer) == 0 {
		return map[string]any{}
	}
	customer["data_values"] = workCustomerFormValues(ctx, customerID, 0, customer)
	customer["data_value_labels"] = workDataValueLabels(ctx, mapFromAny(customer["data_values"]))
	customer["assets"] = workAssetRows(ctx, staff, customerID)
	state := ensureCurrentWorkCustomerStage(ctx, staff, customerID, 0)
	if state != nil {
		customer["state.id"] = state.ID
		customer["state.current_stage_code"] = state.CurrentStageCode
		customer["state.current_department_id"] = state.CurrentDepartmentID
		customer["state.current_staff_id"] = state.CurrentStaffID
		customer["stage_code"] = state.CurrentStageCode
		customer["stage_name"] = workStageName(ctx, state.CurrentStageCode)
		tasks := workAvailableTasks(ctx, staff, state)
		customer["row_tasks"] = workCustomerRowTasks(tasks)
		customer["todo_summary"] = workTodoSummary(customer, tasks)
	}
	enrichWorkCustomerRow(ctx, customer)
	return customer
}

func workAssetRows(ctx context.Context, staff *WorkStaffSession, customerID uint64) []map[string]any {
	if customerID == 0 {
		return []map[string]any{}
	}
	assets := crmmodel.NewCustomerAssetModel().SelectMap(ctx, map[string]any{"customer_id": customerID})
	rows := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		assetID := inputUint64(asset["id"])
		if assetID == 0 {
			continue
		}
		if !canViewWorkAsset(ctx, staff, customerID, assetID) {
			continue
		}
		asset["data_values"] = workAssetFormValues(ctx, customerID, assetID, asset)
		asset["data_value_labels"] = workDataValueLabels(ctx, mapFromAny(asset["data_values"]))
		asset["asset_status_name"] = workAssetStatusName(ctx, inputUint64(asset["asset_status_id"]))
		state := ensureCurrentWorkCustomerStage(ctx, staff, customerID, assetID)
		if state != nil {
			asset["state.id"] = state.ID
			asset["state.current_stage_code"] = state.CurrentStageCode
			asset["state.current_department_id"] = state.CurrentDepartmentID
			asset["state.current_staff_id"] = state.CurrentStaffID
			asset["stage_code"] = state.CurrentStageCode
			asset["stage_name"] = workStageName(ctx, state.CurrentStageCode)
			asset["row_tasks"] = workAssetRowTasks(workAvailableTasks(ctx, staff, state))
		}
		rows = append(rows, asset)
	}
	return rows
}

func enrichWorkCustomerRow(ctx context.Context, customer map[string]any) {
	code := inputText(customer["code"])
	if code != "" {
		customer["code_display"] = customerCodePrefixForWork(ctx) + code
	}
	customer["source_name"] = workCustomerSourceName(ctx, inputUint64(customer["source_id"]))
	customer["channel_name"] = workCustomerChannelName(ctx, inputUint64(customer["channel_id"]))
	customer["level_name"] = workCustomerLevelName(ctx, inputUint64(customer["level_id"]))
	customer["gender_name"] = workGenderName(inputText(customer["gender"]))
}

func workCustomerSourceName(ctx context.Context, id uint64) string {
	if id == 0 {
		return ""
	}
	if source := crmmodel.NewCustomerSourceModel().Find(ctx, map[string]any{"id": id}); source != nil {
		return source.Name
	}
	return ""
}

func workCustomerChannelName(ctx context.Context, id uint64) string {
	if id == 0 {
		return ""
	}
	if channel := crmmodel.NewCustomerChannelModel().Find(ctx, map[string]any{"id": id}); channel != nil {
		return channel.Name
	}
	return ""
}

func workCustomerLevelName(ctx context.Context, id uint64) string {
	if id == 0 {
		return ""
	}
	if level := crmmodel.NewCustomerLevelModel().Find(ctx, map[string]any{"id": id}); level != nil {
		return level.Name
	}
	return ""
}

func customerCodePrefixForWork(ctx context.Context) string {
	config := crmmodel.NewBasicConfigModel().Find(ctx, map[string]any{"id": crmmodel.DefaultBasicConfigID})
	if config == nil {
		return crmmodel.DefaultBasicConfig().CustomerCodePrefix
	}
	return config.CustomerCodePrefix
}

func filterWorkCustomers(rows []map[string]any, keyword string) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		text := inputText(row["code"]) + " " + inputText(row["code_display"]) + " " + inputText(row["name"]) + " " + inputText(row["phone"]) + " " + inputText(row["wechat"])
		for _, asset := range mapListFromAny(row["assets"]) {
			text += " " + inputText(asset["asset_no"]) + " " + inputText(asset["asset_name"]) + " " + inputText(asset["stage_code"]) + " " + inputText(asset["stage_name"])
		}
		if containsFold(text, keyword) {
			result = append(result, row)
		}
	}
	return result
}

func hasWorkCustomerStructuredFilter(payload map[string]any) bool {
	return firstText(payload, "customer_no", "customerNo") != "" ||
		firstText(payload, "customer_name", "customerName") != "" ||
		firstText(payload, "phone") != "" ||
		firstText(payload, "wechat") != "" ||
		firstText(payload, "asset_no", "assetNo") != "" ||
		firstText(payload, "status") != ""
}

func filterWorkCustomersByFields(rows []map[string]any, payload map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if !workCustomerMatchesFields(row, payload) {
			continue
		}
		assets := mapListFromAny(row["assets"])
		filteredAssets := filterWorkAssetsByFields(assets, payload)
		if workFilterRequiresAsset(payload) && len(filteredAssets) == 0 {
			continue
		}
		if len(assets) == 0 && !workCustomerStatusMatches(row, payload) {
			continue
		}
		if len(assets) > 0 && firstText(payload, "status") != "" && len(filteredAssets) == 0 {
			continue
		}
		next := copyMap(row)
		next["assets"] = filteredAssets
		result = append(result, next)
	}
	return result
}

func workCustomerMatchesFields(row map[string]any, payload map[string]any) bool {
	customerNo := firstText(payload, "customer_no", "customerNo")
	if customerNo != "" && !containsFold(inputText(row["code"])+" "+inputText(row["code_display"]), customerNo) {
		return false
	}
	customerName := firstText(payload, "customer_name", "customerName")
	if customerName != "" && !containsFold(inputText(row["name"]), customerName) {
		return false
	}
	phone := firstText(payload, "phone")
	if phone != "" && !containsFold(inputText(row["phone"]), phone) {
		return false
	}
	wechat := firstText(payload, "wechat")
	if wechat != "" && !containsFold(inputText(row["wechat"]), wechat) {
		return false
	}
	return true
}

func workCustomerStatusMatches(row map[string]any, payload map[string]any) bool {
	status := firstText(payload, "status")
	return status == "" || containsFold(inputText(row["stage_code"])+" "+inputText(row["stage_name"]), status)
}

func filterWorkAssetsByFields(assets []map[string]any, payload map[string]any) []map[string]any {
	assetNo := firstText(payload, "asset_no", "assetNo")
	status := firstText(payload, "status")
	result := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		if assetNo != "" && !containsFold(inputText(asset["asset_no"]), assetNo) {
			continue
		}
		if status != "" && !containsFold(inputText(asset["stage_code"])+" "+inputText(asset["stage_name"]), status) {
			continue
		}
		result = append(result, asset)
	}
	return result
}

func workFilterRequiresAsset(payload map[string]any) bool {
	return firstText(payload, "asset_no", "assetNo") != ""
}

func filterWorkOperations(rows []map[string]any, keyword string) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		text := inputText(row["title"]) + " " + inputText(row["content"]) + " " + inputText(row["stage_code"]) + " " + inputText(row["result_value"])
		if containsFold(text, keyword) {
			result = append(result, row)
		}
	}
	return result
}

func enrichWorkOperationRows(ctx context.Context, staff *WorkStaffSession, rows []map[string]any) {
	for _, row := range rows {
		enrichWorkOperationRow(ctx, staff, row)
	}
}

func enrichWorkOperationRow(ctx context.Context, staff *WorkStaffSession, row map[string]any) {
	if row == nil {
		return
	}
	if task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": inputUint64(row["task_id"])}); task != nil {
		row["task.name"] = task.Name
		row["task.task_type"] = task.TaskType
	}
	if asset := crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": inputUint64(row["asset_id"])}); asset != nil {
		row["asset.asset_no"] = asset.AssetNo
		row["asset.asset_name"] = asset.AssetName
	}
	if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": inputUint64(row["operator_staff_id"])}); staff != nil {
		row["operator_staff.name"] = staff.Name
		row["operator_staff.phone"] = staff.Phone
	}
	if department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": inputUint64(row["operator_department_id"])}); department != nil {
		row["operator_department.name"] = department.Name
	}
	row["operator_is_current"] = staff != nil && staff.ID > 0 && inputUint64(row["operator_staff_id"]) == staff.ID
	summaryItems := workOperationSummaryItems(ctx, row)
	row["summary"] = workOperationSummary(row, summaryItems)
	row["summary_items"] = summaryItems
}

func workOperationSummary(row map[string]any, items []map[string]any) string {
	taskType := inputText(row["task_type"])
	switch taskType {
	case crmmodel.TaskTypeAssign:
		departmentName := ""
		staffName := ""
		for _, item := range items {
			switch inputText(item["label"]) {
			case "分配部门":
				departmentName = inputText(item["value"])
			case "分配人员":
				staffName = inputText(item["value"])
			}
		}
		if staffName != "" && departmentName != "" {
			return fmt.Sprintf("分配给 %s / %s", departmentName, staffName)
		}
		if staffName != "" {
			return fmt.Sprintf("分配给 %s", staffName)
		}
		if departmentName != "" {
			return fmt.Sprintf("分配给 %s", departmentName)
		}
		return "完成分配"
	case crmmodel.TaskTypeDecision:
		if result := inputText(row["result_value"]); result != "" && result != workResultSuccess {
			return "决策结果：" + result
		}
		return "完成决策"
	case crmmodel.TaskTypeBooking:
		return "完成资源预定"
	case crmmodel.TaskTypeCreate:
		if len(items) > 0 {
			return fmt.Sprintf("收集线索，填写 %d 项资料", len(items))
		}
		return "收集线索"
	default:
		if len(items) > 0 {
			return fmt.Sprintf("补充 %d 项资料", len(items))
		}
		return "完成任务"
	}
}

func workOperationSummaryItems(ctx context.Context, row map[string]any) []map[string]any {
	values := mapFromAny(row["data_snapshot_json"])
	if len(values) == 0 {
		return []map[string]any{}
	}
	items := make([]map[string]any, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		value := values[key]
		text := inputText(value)
		if text == "" {
			continue
		}
		label, displayValue := workOperationSnapshotItem(ctx, key, value)
		if label == "" || displayValue == "" {
			continue
		}
		items = append(items, map[string]any{
			"key":   key,
			"label": label,
			"value": displayValue,
		})
	}
	return items
}

func workOperationSnapshotItem(ctx context.Context, key string, value any) (string, string) {
	switch key {
	case "department_id", "departmentId":
		return "分配部门", workDepartmentName(ctx, inputUint64(value))
	case "staff_id", "staffId":
		return "分配人员", workStaffName(ctx, inputUint64(value))
	case "result_value", "resultValue":
		return "决策结果", inputText(value)
	case "resource_id", "resourceId", "booking:resource_id", "main:resource_id":
		return "预定资源", workPublicResourceName(ctx, inputUint64(value))
	case "start_at", "startAt", "booking:start_at", "main:start_at":
		return "开始时间", inputText(value)
	case "end_at", "endAt", "booking:end_at", "main:end_at":
		return "结束时间", inputText(value)
	case "remark", "booking:remark", "main:remark":
		return "备注", inputText(value)
	}
	if strings.HasPrefix(key, "main:") {
		field := strings.TrimPrefix(key, "main:")
		return workMainFieldLabel(field), workMainFieldDisplayValue(ctx, field, value)
	}
	if strings.HasPrefix(key, "data:") {
		fieldID := inputUint64(strings.TrimPrefix(key, "data:"))
		if fieldID == 0 {
			return "", ""
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
		if field == nil {
			return key, inputText(value)
		}
		return field.Name, workDataFieldDisplayValue(ctx, fieldID, value)
	}
	return key, inputText(value)
}

func workDataFieldDisplayValue(ctx context.Context, fieldID uint64, value any) string {
	text := inputText(value)
	if fieldID == 0 || text == "" {
		return text
	}
	if option := crmmodel.NewDataFieldOptionModel().Find(ctx, map[string]any{
		"data_field_id": fieldID,
		"value":         text,
	}); option != nil {
		return option.Name
	}
	return text
}

func workMainFieldLabel(field string) string {
	switch field {
	case "name":
		return "姓名"
	case "phone":
		return "手机号"
	case "wechat":
		return "微信"
	case "id_card":
		return "身份证号"
	case "gender":
		return "性别"
	case "source_id":
		return "来源"
	case "channel_id":
		return "渠道"
	case "level_id":
		return "客户等级"
	case "asset_name":
		return "资产名称"
	case "asset_no":
		return "资产编号"
	case "asset_status_id":
		return "资产状态"
	case "remark":
		return "备注"
	default:
		return field
	}
}

func workMainFieldDisplayValue(ctx context.Context, field string, value any) string {
	switch field {
	case "gender":
		return workGenderName(inputText(value))
	case "source_id":
		return workCustomerSourceName(ctx, inputUint64(value))
	case "channel_id":
		return workCustomerChannelName(ctx, inputUint64(value))
	case "level_id":
		return workCustomerLevelName(ctx, inputUint64(value))
	case "asset_status_id":
		return workAssetStatusName(ctx, inputUint64(value))
	default:
		return inputText(value)
	}
}

func filterWorkBookings(rows []map[string]any, keyword string) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		text := inputText(row["title"]) + " " + inputText(row["remark"]) + " " + inputText(row["resource.name"]) + " " + inputText(row["customer.name"])
		if containsFold(text, keyword) {
			result = append(result, row)
		}
	}
	return result
}

func workDepartmentTasks(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	if staff == nil || staff.DepartmentID == 0 {
		return []map[string]any{}
	}
	stages := crmmodel.NewStageModel().Select(ctx, map[string]any{
		"owner_department_id": staff.DepartmentID,
		"status":              crmmodel.StatusEnabled,
	})
	stageIDs := map[uint64]bool{}
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		stageIDs[stage.ID] = true
	}
	if len(stageIDs) == 0 {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(stageIDs))
	for _, task := range crmmodel.NewTaskModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}) {
		stageID := inputUint64(task["stage_id"])
		if stageID == 0 || !stageIDs[stageID] {
			continue
		}
		if inputText(task["task_type"]) != crmmodel.TaskTypeCreate {
			continue
		}
		if workTaskTriggerType(task) != crmmodel.TaskTriggerManual {
			continue
		}
		attachWorkTaskForm(ctx, task)
		if !workTaskCreatesCustomerFromDepartment(task) {
			continue
		}
		result = append(result, task)
	}
	return result
}

func workTaskCreatesCustomerFromDepartment(task map[string]any) bool {
	return inputText(task["task_type"]) == crmmodel.TaskTypeCreate && !workTaskFormHasAssetFields(task)
}

func workAvailableTasks(ctx context.Context, staff *WorkStaffSession, state *crmmodel.CustomerStage) []map[string]any {
	if staff == nil || state == nil || !canOperateCurrentState(staff, state) {
		return []map[string]any{}
	}
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
		"code":   state.CurrentStageCode,
		"status": crmmodel.StatusEnabled,
	})
	if stage == nil {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0)
	for _, task := range crmmodel.NewTaskModel().SelectMap(ctx, map[string]any{"stage_id": stage.ID, "status": crmmodel.StatusEnabled}) {
		if len(task) == 0 {
			continue
		}
		if workTaskTriggerType(task) != crmmodel.TaskTriggerManual {
			continue
		}
		attachWorkTaskForm(ctx, task)
		result = append(result, task)
	}
	return result
}

func workCustomerRowTasks(tasks []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		if workTaskCanRunOnCustomerRow(task) {
			result = append(result, task)
		}
	}
	return result
}

func workAssetRowTasks(tasks []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		if workTaskCanRunOnAssetRow(task) {
			result = append(result, task)
		}
	}
	return result
}

func workTaskCanRunOnCustomerRow(task map[string]any) bool {
	return inputText(task["task_type"]) != crmmodel.TaskTypeCreate
}

func workTaskCanRunOnAssetRow(task map[string]any) bool {
	return inputText(task["task_type"]) != crmmodel.TaskTypeCreate
}

func workTaskIsCustomerCreateForm(task map[string]any) bool {
	return inputText(task["task_type"]) == crmmodel.TaskTypeCreate &&
		!workTaskFormHasAssetFields(task)
}

func workTaskIsAssetCreateForm(task map[string]any) bool {
	return inputText(task["task_type"]) == crmmodel.TaskTypeCreate &&
		workTaskFormHasAssetFields(task)
}

func workTaskFormHasAssetFields(task map[string]any) bool {
	form := mapFromAny(task["form"])
	for _, field := range mapListFromAny(form["fields"]) {
		if workMapFormFieldIsAsset(field) {
			return true
		}
	}
	return false
}

func workMapFormFieldIsAsset(field map[string]any) bool {
	if inputUint64(field["data_template_cate_id"]) == crmmodel.CustomerAssetDataTemplateCateID {
		return true
	}
	switch inputText(field["main_field"]) {
	case "asset_name", "asset_status_id", "remark":
		return true
	default:
		return false
	}
}

func workTaskTriggerType(task map[string]any) string {
	triggerType := inputText(task["trigger_type"])
	if triggerType == "" {
		return crmmodel.TaskTriggerManual
	}
	return triggerType
}

func workTodoSummary(customer map[string]any, tasks []map[string]any) map[string]any {
	missing := make([]map[string]any, 0)
	for _, task := range tasks {
		for _, field := range workTaskRequiredFields(task) {
			if !emptyWorkFieldValue(workCustomerValueForTaskField(customer, field)) {
				continue
			}
			missing = append(missing, map[string]any{
				"task_id":       inputUint64(task["id"]),
				"task_name":     inputText(task["name"]),
				"field":         inputText(field["main_field"]),
				"data_field_id": inputUint64(field["data_field_id"]),
				"name":          inputText(field["name"]),
			})
		}
	}
	return map[string]any{
		"available_task_count":    len(tasks),
		"missing_required_count":  len(missing),
		"missing_required_fields": missing,
	}
}

func workTaskRequiredFields(task map[string]any) []map[string]any {
	form := mapFromAny(task["form"])
	fields := mapsFromAny(form["fields"])
	result := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		if booleanFromAny(field["required"]) {
			result = append(result, field)
		}
	}
	return result
}

func workCustomerValueForTaskField(customer map[string]any, field map[string]any) any {
	if mainField := inputText(field["main_field"]); mainField != "" {
		return customer[mainField]
	}
	if fieldID := inputUint64(field["data_field_id"]); fieldID > 0 {
		return mapFromAny(customer["data_values"])[fmt.Sprintf("data:%d", fieldID)]
	}
	return nil
}

func workStageName(ctx context.Context, stageCode string) string {
	if stageCode == "" {
		return ""
	}
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
		"code":   stageCode,
		"status": crmmodel.StatusEnabled,
	})
	if stage == nil {
		return stageCode
	}
	return stage.Name
}

func workAssetStatusName(ctx context.Context, statusID uint64) string {
	if statusID == 0 {
		return ""
	}
	status := crmmodel.NewAssetStatusModel().Find(ctx, map[string]any{
		"id":     statusID,
		"status": crmmodel.StatusEnabled,
	})
	if status == nil {
		return ""
	}
	return status.Name
}

func workCustomerOwnsAsset(ctx context.Context, customerID uint64, assetID uint64) bool {
	if customerID == 0 || assetID == 0 {
		return false
	}
	return crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{
		"id":          assetID,
		"customer_id": customerID,
	}) != nil
}

func workGenderName(gender string) string {
	switch gender {
	case "male":
		return "男"
	case "female":
		return "女"
	case "unknown", "":
		return "未知"
	default:
		return gender
	}
}

func workCustomerDataValues(ctx context.Context, customerID uint64, assetID uint64) map[string]any {
	values := map[string]any{}
	records := crmmodel.NewDataRecordModel().Select(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"status":      crmmodel.StatusEnabled,
	})
	for _, record := range records {
		if record == nil {
			continue
		}
		for fieldID, value := range mapFromAny(record.RecordJSON) {
			if fieldID != "" {
				values["data:"+fieldID] = value
			}
		}
	}
	return values
}

func workCustomerFormValues(ctx context.Context, customerID uint64, assetID uint64, customer map[string]any) map[string]any {
	values := workCustomerDataValues(ctx, customerID, assetID)
	for _, field := range workCustomerMainFormFields() {
		if value, exists := customer[field]; exists {
			values["main:"+field] = value
		}
	}
	return values
}

func workDataValueLabels(ctx context.Context, values map[string]any) map[string]string {
	labels := map[string]string{}
	for key := range values {
		if !strings.HasPrefix(key, "data:") {
			continue
		}
		fieldID := inputUint64(strings.TrimPrefix(key, "data:"))
		if fieldID == 0 {
			continue
		}
		if field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"id":     fieldID,
			"status": crmmodel.StatusEnabled,
		}); field != nil {
			labels[key] = field.Name
		}
	}
	return labels
}

func workAssetFormValues(ctx context.Context, customerID uint64, assetID uint64, asset map[string]any) map[string]any {
	values := workCustomerDataValues(ctx, customerID, assetID)
	for _, field := range workAssetMainFormFields() {
		if value, exists := asset[field]; exists {
			values["main:"+field] = value
		}
	}
	return values
}

func workCustomerMainFormFields() []string {
	return []string{
		"name",
		"phone",
		"id_card",
		"wechat",
		"gender",
		"source_id",
		"channel_id",
		"level_id",
		"tags",
		"remark",
	}
}

func workAssetMainFormFields() []string {
	return []string{
		"asset_name",
		"asset_status_id",
		"remark",
	}
}

func workAllowedTask(ctx context.Context, staff *WorkStaffSession, taskID uint64, customerID uint64, assetID uint64) *crmmodel.Task {
	data, err := (WorkService{}).Tasks(ctx, staff, customerID, assetID)
	if err != nil {
		return nil
	}
	rows, _ := data["list"].([]map[string]any)
	for _, row := range rows {
		if inputUint64(row["id"]) == taskID {
			return crmmodel.NewTaskModel().Find(ctx, map[string]any{
				"id":     taskID,
				"status": crmmodel.StatusEnabled,
			})
		}
	}
	return nil
}

func attachWorkTaskForm(ctx context.Context, task map[string]any) {
	attachWorkTaskConfig(ctx, task)
	formID := inputUint64(task["form_id"])
	if formID == 0 {
		return
	}
	form := crmmodel.NewFormModel().FindMap(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
	if len(form) == 0 {
		return
	}
	fields := crmmodel.NewFormFieldModel().SelectMap(ctx, map[string]any{
		"form_id": formID,
		"status":  crmmodel.StatusEnabled,
	})
	for _, field := range fields {
		attachWorkFormFieldOptions(ctx, field)
	}
	form["fields"] = fields
	task["form"] = form
}

func attachWorkTaskConfig(ctx context.Context, task map[string]any) {
	switch inputText(task["task_type"]) {
	case crmmodel.TaskTypeAssign:
		config := mapFromAny(task["config_json"])
		assignMode := normalizeWorkAssignMode(inputText(config["assign_mode"]))
		task["assign_mode"] = assignMode
		task["assign_department_ids"] = uint64ListFromAny(config["assign_department_ids"])
	case crmmodel.TaskTypeDecision:
		task["result_schema"] = decisionResultSchemaRows(inputText(task["result_schema_json"]))
	case crmmodel.TaskTypeBooking:
		config := mapFromAny(task["config_json"])
		resourceCateID := inputUint64(config["resource_cate_id"])
		if resourceCateID == 0 {
			resourceCateID = crmmodel.DefaultResourceCateID
		}
		task["booking_resource_cate_id"] = resourceCateID
		task["booking_need_confirm"] = booleanFromAny(config["need_confirm"])
		task["form"] = workBookingForm(ctx, resourceCateID)
	}
}

func attachWorkFormFieldOptions(ctx context.Context, field map[string]any) {
	if fieldID := inputUint64(field["data_field_id"]); fieldID > 0 {
		dataField := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
		if dataField != nil {
			field["field_type"] = dataField.FieldType
			field["default_value"] = dataField.DefaultValue
			field["options"] = crmmodel.NewDataFieldOptionModel().SelectMap(ctx, map[string]any{"data_field_id": fieldID})
		}
		return
	}
	mainField := inputText(field["main_field"])
	field["field_type"] = mainFieldInputType(mainField)
	field["options"] = mainFieldOptions(ctx, mainField)
}

func workBookingForm(ctx context.Context, resourceCateID uint64) map[string]any {
	return map[string]any{
		"id":   0,
		"name": "资源预定",
		"fields": []map[string]any{
			{
				"id":            "booking-resource",
				"name":          "资源",
				"main_field":    "resource_id",
				"field_type":    "select",
				"required":      true,
				"default_value": "",
				"options":       workResourceOptions(ctx, resourceCateID),
			},
			{
				"id":            "booking-start-at",
				"name":          "开始时间",
				"main_field":    "start_at",
				"field_type":    "datetime",
				"required":      true,
				"default_value": "",
				"options":       []map[string]any{},
			},
			{
				"id":            "booking-end-at",
				"name":          "结束时间",
				"main_field":    "end_at",
				"field_type":    "datetime",
				"required":      true,
				"default_value": "",
				"options":       []map[string]any{},
			},
			{
				"id":            "booking-title",
				"name":          "用途",
				"main_field":    "title",
				"field_type":    "text",
				"required":      true,
				"default_value": "",
				"options":       []map[string]any{},
			},
			{
				"id":            "booking-remark",
				"name":          "备注",
				"main_field":    "remark",
				"field_type":    "textarea",
				"required":      false,
				"default_value": "",
				"options":       []map[string]any{},
			},
		},
	}
}

func workResourceOptions(ctx context.Context, resourceCateID uint64) []map[string]any {
	filter := map[string]any{"status": crmmodel.StatusEnabled}
	if resourceCateID > 0 {
		filter["resource_cate_id"] = resourceCateID
	}
	resources := crmmodel.NewPublicResourceModel().Select(ctx, filter)
	options := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		label := resource.Name
		if resource.Location != "" {
			label = label + "（" + resource.Location + "）"
		}
		options = append(options, map[string]any{
			"id":    resource.ID,
			"name":  label,
			"value": resource.ID,
		})
	}
	return options
}

func executeCreateCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	formInput, err := collectWorkFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	customerRecord := defaultWorkCustomerRecord(staff)
	for key, value := range formInput.customerFields {
		customerRecord[key] = value
	}
	if duplicateField := duplicatedWorkCustomerField(ctx, customerRecord); duplicateField != "" {
		return nil, fmt.Errorf("%s", duplicateWorkCustomerFieldMessage(duplicateField))
	}
	if inputText(customerRecord["code"]) == "" {
		code, err := crmmodel.GenerateUniqueCustomerCode(ctx)
		if err != nil {
			return nil, err
		}
		customerRecord["code"] = code
	}
	customerID := uint64(crmmodel.NewCustomerModel().Insert(ctx, customerRecord))
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, 0, values)
	for templateID, record := range formInput.customerDataRecords {
		saveWorkDataRecord(ctx, customerID, 0, templateID, task.ID, operationID, record)
	}
	insertWorkCustomerMember(ctx, staff, customerID)
	insertWorkCustomerStage(ctx, staff, customerID, 0, operationID, task.ID)
	applyWorkStageTransition(ctx, staff, customerID, 0, currentWorkCustomerStage(ctx, customerID, 0), task, operationID, workResultSuccess)
	runWorkAutoTriggers(ctx, staff, customerID, 0, task, workResultSuccess, runtime)
	return map[string]any{
		"customer_id": customerID,
		"saved":       true,
	}, nil
}

func executeEditFormTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	formInput, err := collectWorkFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	if formInputHasAssetValues(formInput) && assetID == 0 {
		assetID, err = createWorkCustomerAsset(ctx, customerID, formInput)
		if err != nil {
			return nil, err
		}
		fromState = currentWorkCustomerStage(ctx, customerID, assetID)
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, assetID, values)
	saveWorkFormDataRecords(ctx, customerID, assetID, task.ID, operationID, formInput)
	applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, workResultSuccess)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, runtime)
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"saved":       true,
	}, nil
}

func executeAssignCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	formInput, err := collectOptionalWorkFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	config := mapFromAny(task.ConfigJSON)
	assignMode := normalizeWorkAssignMode(inputText(config["assign_mode"]))
	targetDepartmentID, targetStaffID, err := resolveWorkAssignTarget(ctx, assignMode, uint64ListFromAny(config["assign_department_ids"]), values)
	if err != nil {
		return nil, err
	}
	createdAsset := false
	if assetID == 0 && formInputHasAssetValues(formInput) {
		assetID, err = createWorkCustomerAsset(ctx, customerID, formInput)
		if err != nil {
			return nil, err
		}
		createdAsset = true
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	logValues := copyMap(values)
	logValues["department_id"] = targetDepartmentID
	logValues["staff_id"] = targetStaffID
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, assetID, logValues)
	saveWorkFormDataRecords(ctx, customerID, assetID, task.ID, operationID, formInput)
	if createdAsset {
		insertWorkCustomerStage(ctx, staff, customerID, assetID, operationID, task.ID)
		fromState = currentWorkCustomerStage(ctx, customerID, assetID)
	}
	updateWorkCustomerOwner(ctx, customerID, assetID, targetDepartmentID, targetStaffID, operationID)
	upsertWorkAssigneeMember(ctx, customerID, assetID, targetDepartmentID, targetStaffID)
	applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, workResultSuccess, targetDepartmentID, targetStaffID)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, runtime)
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"saved":       true,
	}, nil
}

func executeDecisionCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	resultValue, resultPayload, err := resolveWorkDecisionResult(ctx, staff, task, customerID, assetID, fromState, values)
	if err != nil {
		return nil, err
	}
	if resultValue == "" {
		return nil, fmt.Errorf("请选择决策结果")
	}
	logValues := values
	if len(resultPayload) > 0 {
		logValues = map[string]any{}
		for key, value := range values {
			logValues[key] = value
		}
		logValues["decision_result"] = resultPayload
	}
	operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, logValues, resultValue)
	applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, resultValue, runtime)
	return map[string]any{
		"customer_id":  customerID,
		"result_value": resultValue,
		"saved":        true,
	}, nil
}

func executeBookingCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	bookingInput, err := collectWorkBookingInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{
		"id":     bookingInput.resourceID,
		"status": crmmodel.StatusEnabled,
	})
	if resource == nil {
		return nil, fmt.Errorf("公共资源不存在或已停用")
	}
	if resource.ResourceCateID != bookingInput.resourceCateID {
		return nil, fmt.Errorf("该资源不属于当前任务配置的资源分类")
	}
	if err := ensureWorkBookingTimeAvailable(ctx, 0, bookingInput.resourceID, bookingInput.startAt, bookingInput.endAt); err != nil {
		return nil, err
	}
	bookingStatus := crmmodel.ResourceBookingStatusReserved
	if bookingInput.needConfirm || resource.NeedConfirm {
		bookingStatus = crmmodel.ResourceBookingStatusPending
	}
	resultValue := workResultSuccess
	if bookingStatus == crmmodel.ResourceBookingStatusPending {
		resultValue = crmmodel.ResourceBookingStatusPending
	}
	operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, values, resultValue)
	bookingID := insertWorkResourceBooking(ctx, staff, task, customerID, assetID, operationID, fromState, bookingInput, bookingStatus)
	applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, resultValue, runtime)
	return map[string]any{
		"customer_id":    customerID,
		"booking_id":     bookingID,
		"booking_status": bookingStatus,
		"result_value":   resultValue,
		"saved":          true,
	}, nil
}

type workBookingInput struct {
	resourceCateID uint64
	resourceID     uint64
	startAt        time.Time
	endAt          time.Time
	title          string
	remark         string
	needConfirm    bool
}

func collectWorkBookingInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workBookingInput, error) {
	config := mapFromAny(task.ConfigJSON)
	resourceCateID := inputUint64(config["resource_cate_id"])
	if resourceCateID == 0 {
		resourceCateID = crmmodel.DefaultResourceCateID
	}
	resourceID := firstUint64(values, "resource_id", "resourceId", "booking:resource_id", "main:resource_id")
	if resourceID == 0 {
		return nil, fmt.Errorf("请选择资源")
	}
	startAt, err := parseWorkBookingDateTime(firstText(values, "start_at", "startAt", "booking:start_at", "main:start_at"))
	if err != nil {
		return nil, fmt.Errorf("开始时间格式错误")
	}
	endAt, err := parseWorkBookingDateTime(firstText(values, "end_at", "endAt", "booking:end_at", "main:end_at"))
	if err != nil {
		return nil, fmt.Errorf("结束时间格式错误")
	}
	if !endAt.After(startAt) {
		return nil, fmt.Errorf("结束时间必须晚于开始时间")
	}
	title := firstText(values, "title", "booking:title", "main:title")
	if title == "" {
		return nil, fmt.Errorf("用途不能为空")
	}
	if crmmodel.NewPublicResourceCateModel().Find(ctx, map[string]any{"id": resourceCateID, "status": crmmodel.StatusEnabled}) == nil {
		return nil, fmt.Errorf("资源分类不存在或已停用")
	}
	return &workBookingInput{
		resourceCateID: resourceCateID,
		resourceID:     resourceID,
		startAt:        startAt,
		endAt:          endAt,
		title:          title,
		remark:         firstText(values, "remark", "booking:remark", "main:remark"),
		needConfirm:    booleanFromAny(config["need_confirm"]),
	}, nil
}

func ensureWorkBookingTimeAvailable(ctx context.Context, currentID uint64, resourceID uint64, startAt time.Time, endAt time.Time) error {
	for _, booking := range crmmodel.NewPublicResourceBookingModel().Select(ctx, map[string]any{"resource_id": resourceID}) {
		if booking == nil || booking.ID == currentID || workBookingInactiveStatus(booking.BookingStatus) {
			continue
		}
		if startAt.Before(booking.EndAt) && endAt.After(booking.StartAt) {
			return fmt.Errorf("该资源在所选时间已被预定")
		}
	}
	return nil
}

func workBookingInactiveStatus(status string) bool {
	return status == crmmodel.ResourceBookingStatusCanceled || status == crmmodel.ResourceBookingStatusRejected
}

func parseWorkBookingDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime")
}

func enrichWorkBookingRow(ctx context.Context, row map[string]any) {
	resourceID := inputUint64(row["resource_id"])
	if resourceID > 0 {
		resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{"id": resourceID})
		if resource != nil {
			row["resource.name"] = resource.Name
			row["resource.location"] = resource.Location
		}
	}
	customerID := inputUint64(row["customer_id"])
	if customerID > 0 {
		customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
		if customer != nil {
			row["customer.name"] = customer.Name
			row["customer.phone"] = customer.Phone
		}
	}
	row["booking_status_name"] = workBookingStatusName(inputText(row["booking_status"]))
}

func workBookingStatusName(status string) string {
	switch status {
	case crmmodel.ResourceBookingStatusPending:
		return "待确认"
	case crmmodel.ResourceBookingStatusReserved:
		return "已预定"
	case crmmodel.ResourceBookingStatusCanceled:
		return "已取消"
	case crmmodel.ResourceBookingStatusRejected:
		return "已拒绝"
	case crmmodel.ResourceBookingStatusDone:
		return "已完成"
	default:
		return status
	}
}

func insertWorkResourceBooking(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, operationID uint64, state *crmmodel.CustomerStage, input *workBookingInput, bookingStatus string) uint64 {
	now := time.Now()
	statusCode := ""
	if state != nil {
		statusCode = state.CurrentStageCode
	}
	return uint64(crmmodel.NewPublicResourceBookingModel().Insert(ctx, map[string]any{
		"resource_id":          input.resourceID,
		"customer_id":          customerID,
		"asset_id":             assetID,
		"task_id":              task.ID,
		"operation_log_id":     operationID,
		"stage_code":           statusCode,
		"booking_status":       bookingStatus,
		"title":                input.title,
		"remark":               input.remark,
		"start_at":             input.startAt,
		"end_at":               input.endAt,
		"booker_staff_id":      staff.ID,
		"booker_department_id": staff.DepartmentID,
		"created_at":           now,
		"updated_at":           now,
	}))
}

func resolveWorkAssignTarget(ctx context.Context, assignMode string, allowedDepartmentIDs []uint64, values map[string]any) (uint64, uint64, error) {
	departmentID := firstUint64(values, "department_id", "departmentId")
	staffID := firstUint64(values, "staff_id", "staffId")
	if departmentID == 0 {
		return 0, 0, fmt.Errorf("请选择部门")
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled})
	if department == nil {
		return 0, 0, fmt.Errorf("部门不存在或已停用")
	}
	if len(allowedDepartmentIDs) > 0 && !uint64SetContains(allowedDepartmentIDs, departmentID) {
		return 0, 0, fmt.Errorf("该部门不在当前任务可选范围内")
	}
	switch assignMode {
	case crmmodel.TaskAssignModeDepartment:
		leaderStaffID := workDepartmentLeaderStaffID(ctx, department)
		if leaderStaffID == 0 {
			return 0, 0, fmt.Errorf("该部门未配置负责人，无法自动派单")
		}
		return departmentID, leaderStaffID, nil
	default:
		if staffID == 0 {
			leaderStaffID := workDepartmentLeaderStaffID(ctx, department)
			if leaderStaffID == 0 {
				return 0, 0, fmt.Errorf("该部门未配置负责人，无法自动派单")
			}
			return departmentID, leaderStaffID, nil
		}
		targetStaff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
		if targetStaff == nil {
			return 0, 0, fmt.Errorf("人员不存在或已停用")
		}
		if targetStaff.DepartmentID != departmentID {
			return 0, 0, fmt.Errorf("人员不属于所选部门")
		}
		return departmentID, staffID, nil
	}
}

func normalizeWorkAssignMode(assignMode string) string {
	if strings.TrimSpace(assignMode) == crmmodel.TaskAssignModeDepartment {
		return crmmodel.TaskAssignModeDepartment
	}
	return crmmodel.TaskAssignModeStaff
}

func workDepartmentLeaderStaffID(ctx context.Context, department *crmmodel.Department) uint64 {
	if department == nil || department.ID == 0 {
		return 0
	}
	if department.LeaderStaffID > 0 {
		if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
			"id":            department.LeaderStaffID,
			"department_id": department.ID,
			"status":        crmmodel.StatusEnabled,
		}); staff != nil {
			return staff.ID
		}
	}
	if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"department_id": department.ID,
		"staff_type":    crmmodel.StaffTypeLeader,
		"status":        crmmodel.StatusEnabled,
	}); staff != nil {
		return staff.ID
	}
	return 0
}

type workFormInput struct {
	customerFields      map[string]any
	assetFields         map[string]any
	customerDataRecords map[uint64]map[string]any
	assetDataRecords    map[uint64]map[string]any
}

func collectWorkFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	form := crmmodel.NewFormModel().Find(ctx, map[string]any{"id": task.FormID, "status": crmmodel.StatusEnabled})
	if form == nil {
		return nil, fmt.Errorf("任务未配置有效资料模板")
	}
	fields := crmmodel.NewFormFieldModel().Select(ctx, map[string]any{"form_id": form.ID, "status": crmmodel.StatusEnabled})
	if len(fields) == 0 {
		return nil, fmt.Errorf("资料模板未配置字段")
	}
	result := &workFormInput{
		customerFields:      map[string]any{},
		assetFields:         map[string]any{},
		customerDataRecords: map[uint64]map[string]any{},
		assetDataRecords:    map[uint64]map[string]any{},
	}
	for _, field := range fields {
		if field == nil {
			continue
		}
		value := values[workFieldInputKey(field)]
		if field.Required && emptyWorkFieldValue(value) {
			return nil, fmt.Errorf("%s不能为空", field.Name)
		}
		if field.MainField != "" {
			if isWorkAssetMainField(field) {
				applyWorkAssetMainField(result.assetFields, field.MainField, value)
				continue
			}
			applyWorkCustomerMainField(result.customerFields, field.MainField, value)
			continue
		}
		if field.DataTemplateID > 0 && field.DataFieldID > 0 {
			records := result.customerDataRecords
			if isWorkAssetTemplateField(field) {
				records = result.assetDataRecords
			}
			if records[field.DataTemplateID] == nil {
				records[field.DataTemplateID] = map[string]any{}
			}
			records[field.DataTemplateID][fmt.Sprintf("%d", field.DataFieldID)] = value
		}
	}
	return result, nil
}

func collectOptionalWorkFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	if task == nil || task.FormID == 0 {
		return emptyWorkFormInput(), nil
	}
	return collectWorkFormInput(ctx, task, values)
}

func emptyWorkFormInput() *workFormInput {
	return &workFormInput{
		customerFields:      map[string]any{},
		assetFields:         map[string]any{},
		customerDataRecords: map[uint64]map[string]any{},
		assetDataRecords:    map[uint64]map[string]any{},
	}
}

func formInputHasAssetValues(formInput *workFormInput) bool {
	return formInput != nil && (len(formInput.assetFields) > 0 || len(formInput.assetDataRecords) > 0)
}

func saveWorkFormInput(ctx context.Context, customerID uint64, assetID uint64, formInput *workFormInput) error {
	if formInput == nil {
		return nil
	}
	if len(formInput.customerFields) > 0 {
		formInput.customerFields["updated_at"] = time.Now()
		crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customerID}, formInput.customerFields)
	}
	if len(formInput.assetFields) > 0 {
		if assetID == 0 {
			return fmt.Errorf("客户资产不能为空")
		}
		formInput.assetFields["updated_at"] = time.Now()
		crmmodel.NewCustomerAssetModel().Update(ctx, map[string]any{"id": assetID, "customer_id": customerID}, formInput.assetFields)
	}
	return nil
}

func createWorkCustomerAsset(ctx context.Context, customerID uint64, formInput *workFormInput) (uint64, error) {
	assetRecord := defaultWorkAssetRecord(customerID)
	assetSeq, assetNo, err := nextWorkCustomerAssetIdentity(ctx, customerID)
	if err != nil {
		return 0, err
	}
	assetRecord["asset_seq"] = assetSeq
	assetRecord["asset_no"] = assetNo
	if formInput != nil {
		for key, value := range formInput.assetFields {
			assetRecord[key] = value
		}
		formInput.assetFields = map[string]any{}
	}
	return uint64(crmmodel.NewCustomerAssetModel().Insert(ctx, assetRecord)), nil
}

func nextWorkCustomerAssetIdentity(ctx context.Context, customerID uint64) (uint64, string, error) {
	customerCode, err := ensureWorkCustomerCode(ctx, customerID)
	if err != nil {
		return 0, "", err
	}
	model := crmmodel.NewCustomerAssetModel()
	assets := model.Select(ctx, map[string]any{"customer_id": customerID})
	seq := uint64(len(assets) + 1)
	prefix := customerCodePrefixForWork(ctx)
	for i := 0; i < 200; i++ {
		assetNo := fmt.Sprintf("%s%s-%d", prefix, customerCode, seq)
		if model.Find(ctx, map[string]any{"asset_no": assetNo}) == nil {
			return seq, assetNo, nil
		}
		seq++
	}
	return 0, "", fmt.Errorf("资产编号生成失败，请重试")
}

func ensureWorkCustomerCode(ctx context.Context, customerID uint64) (string, error) {
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	if customer == nil {
		return "", fmt.Errorf("客户不存在")
	}
	code := strings.TrimSpace(customer.Code)
	if code != "" {
		return code, nil
	}
	code, err := crmmodel.GenerateUniqueCustomerCode(ctx)
	if err != nil {
		return "", err
	}
	crmmodel.NewCustomerModel().Update(ctx, map[string]any{"id": customerID}, map[string]any{
		"code":       code,
		"updated_at": time.Now(),
	})
	return code, nil
}

func saveWorkFormDataRecords(ctx context.Context, customerID uint64, assetID uint64, taskID uint64, operationID uint64, formInput *workFormInput) {
	if formInput == nil {
		return
	}
	for templateID, record := range formInput.customerDataRecords {
		saveWorkDataRecord(ctx, customerID, 0, templateID, taskID, operationID, record)
	}
	if assetID == 0 {
		return
	}
	for templateID, record := range formInput.assetDataRecords {
		saveWorkDataRecord(ctx, customerID, assetID, templateID, taskID, operationID, record)
	}
}

func defaultWorkCustomerRecord(staff *WorkStaffSession) map[string]any {
	now := time.Now()
	return map[string]any{
		"code":                "",
		"name":                "",
		"phone":               "",
		"wechat":              "",
		"id_card":             "",
		"gender":              "unknown",
		"source_id":           crmmodel.DefaultCustomerSourceID,
		"channel_id":          crmmodel.DefaultCustomerChannelID,
		"level_id":            crmmodel.DefaultCustomerLevelID,
		"tags":                "",
		"remark":              "",
		"created_by_staff_id": staff.ID,
		"created_at":          now,
		"updated_at":          now,
	}
}

func defaultWorkAssetRecord(customerID uint64) map[string]any {
	now := time.Now()
	return map[string]any{
		"asset_no":        defaultAssetNo(),
		"asset_name":      "",
		"customer_id":     customerID,
		"asset_status_id": crmmodel.DefaultAssetStatusID,
		"remark":          "",
		"created_at":      now,
		"updated_at":      now,
	}
}

func duplicatedWorkCustomerField(ctx context.Context, record map[string]any) string {
	model := crmmodel.NewCustomerModel()
	for _, field := range []string{"phone", "wechat", "id_card"} {
		value := inputText(record[field])
		if value == "" {
			continue
		}
		if customer := model.Find(ctx, map[string]any{field: value}); customer != nil {
			return field
		}
	}
	return ""
}

func duplicateWorkCustomerFieldMessage(field string) string {
	switch field {
	case "phone":
		return "手机号已存在。"
	case "wechat":
		return "微信已存在。"
	case "id_card":
		return "身份证号已存在。"
	default:
		return "线索已存在。"
	}
}

func applyWorkCustomerMainField(record map[string]any, field string, value any) {
	switch field {
	case "name", "phone", "wechat", "id_card", "gender", "tags", "remark":
		record[field] = inputText(value)
	case "source_id", "channel_id", "level_id":
		if id := inputUint64(value); id > 0 {
			record[field] = id
		}
	default:
		if field != "" {
			record[field] = value
		}
	}
}

func applyWorkAssetMainField(record map[string]any, field string, value any) {
	switch field {
	case "asset_name", "remark":
		record[field] = inputText(value)
	case "asset_status_id":
		if id := inputUint64(value); id > 0 {
			record[field] = id
		}
	default:
		if field != "" {
			record[field] = value
		}
	}
}

func isWorkAssetMainField(field *crmmodel.FormField) bool {
	if field == nil {
		return false
	}
	if isWorkAssetTemplateField(field) {
		return true
	}
	switch field.MainField {
	case "asset_name", "asset_status_id", "remark":
		return true
	default:
		return false
	}
}

func isWorkAssetTemplateField(field *crmmodel.FormField) bool {
	return field != nil && field.DataTemplateCateID == crmmodel.CustomerAssetDataTemplateCateID
}

func runWorkAutoTriggers(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, sourceTask *crmmodel.Task, resultValue string, runtime *workExecutionRuntime) {
	if staff == nil || sourceTask == nil || customerID == 0 || runtime == nil || runtime.depth >= maxWorkAutoTriggerDepth {
		return
	}
	for _, task := range workAfterTaskTriggers(ctx, sourceTask.ID) {
		executeAutoWorkTask(ctx, staff, customerID, assetID, task, runtime)
	}
	state := currentWorkCustomerStage(ctx, customerID, assetID)
	if state == nil {
		return
	}
	for _, task := range workStageEnterTriggers(ctx, state) {
		executeAutoWorkTask(ctx, staff, customerID, assetID, task, runtime)
	}
}

func executeAutoWorkTask(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, task *crmmodel.Task, runtime *workExecutionRuntime) {
	if task == nil || task.TaskType != crmmodel.TaskTypeDecision || task.ScriptID == 0 {
		return
	}
	if !beginWorkTaskExecution(runtime, customerID, assetID, task.ID) {
		return
	}
	defer endWorkTaskExecution(runtime, customerID, assetID, task.ID)
	_, _ = executeDecisionCustomerTask(ctx, staff, task, customerID, assetID, map[string]any{}, runtime)
}

func workAfterTaskTriggers(ctx context.Context, taskID uint64) []*crmmodel.Task {
	if taskID == 0 {
		return nil
	}
	return crmmodel.NewTaskModel().Select(ctx, map[string]any{
		"trigger_type":    crmmodel.TaskTriggerAfterTask,
		"trigger_task_id": taskID,
		"status":          crmmodel.StatusEnabled,
	})
}

func workStageEnterTriggers(ctx context.Context, state *crmmodel.CustomerStage) []*crmmodel.Task {
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
		"code":   state.CurrentStageCode,
		"status": crmmodel.StatusEnabled,
	})
	if stage == nil {
		return nil
	}
	return crmmodel.NewTaskModel().Select(ctx, map[string]any{
		"stage_id":     stage.ID,
		"trigger_type": crmmodel.TaskTriggerStageEnter,
		"status":       crmmodel.StatusEnabled,
	})
}

func resolveWorkDecisionResult(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any) (string, map[string]any, error) {
	if task.ScriptID > 0 {
		return executeWorkDecisionScript(ctx, staff, task, customerID, assetID, state, values)
	}
	resultValue := firstText(values, "result_value", "resultValue", "decision_result", "decisionResult")
	if resultValue == "" {
		resultValue = firstAvailableDecisionResult(task)
	}
	return resultValue, map[string]any{"result_value": resultValue, "source": "manual"}, nil
}

func executeWorkDecisionScript(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any) (string, map[string]any, error) {
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": task.ScriptID, "status": crmmodel.StatusEnabled})
	if script == nil {
		return "", nil, fmt.Errorf("决策脚本不存在或已停用")
	}
	timeout := fronteval.DefaultTimeout
	if script.TimeoutMS > 0 {
		timeout = time.Duration(script.TimeoutMS) * time.Millisecond
	}
	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: script.Language,
		Script:   script.Script,
		Entry:    script.Entry,
		Timeout:  timeout,
		Input:    workDecisionInput(ctx, staff, task, customerID, assetID, state, values),
		Config:   mapFromAny(task.ConfigJSON),
	})
	if err != nil {
		return "", nil, err
	}
	payload := normalizeDecisionResultPayload(result.Value)
	resultValue := inputText(payload["result_value"])
	if resultValue == "" {
		resultValue = inputText(payload["value"])
	}
	if resultValue == "" {
		resultValue = inputText(result.Value)
	}
	payload["result_value"] = resultValue
	payload["duration_ms"] = result.DurationMS
	payload["source"] = "script"
	return resultValue, payload, nil
}

func firstAvailableDecisionResult(task *crmmodel.Task) string {
	for _, row := range decisionResultSchemaRows(task.ResultSchemaJSON) {
		if value := inputText(row["result_value"]); value != "" {
			return value
		}
	}
	return ""
}

func decisionResultSchemaRows(raw string) []map[string]any {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &rows); err == nil {
		return rows
	}
	var generic []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &generic); err != nil {
		return nil
	}
	result := make([]map[string]any, 0, len(generic))
	for _, item := range generic {
		if row, ok := item.(map[string]any); ok {
			result = append(result, row)
		}
	}
	return result
}

func normalizeDecisionResultPayload(value any) map[string]any {
	if row := mapFromAny(value); len(row) > 0 {
		return row
	}
	return map[string]any{"value": value}
}

func workDecisionInput(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any) map[string]any {
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	asset := map[string]any{}
	if assetID > 0 {
		asset = crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{"id": assetID})
	}
	return map[string]any{
		"staff": map[string]any{
			"id":            staff.ID,
			"name":          staff.Name,
			"phone":         staff.Phone,
			"department_id": staff.DepartmentID,
		},
		"task": map[string]any{
			"id":        task.ID,
			"name":      task.Name,
			"task_type": task.TaskType,
		},
		"customer":    customer,
		"asset":       asset,
		"state":       workStateSnapshot(state),
		"data_values": workCustomerFormValues(ctx, customerID, assetID, customer),
		"values":      values,
	}
}

func workStateSnapshot(state *crmmodel.CustomerStage) map[string]any {
	if state == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":                    state.ID,
		"customer_id":           state.CustomerID,
		"asset_id":              state.AssetID,
		"current_stage_code":    state.CurrentStageCode,
		"current_department_id": state.CurrentDepartmentID,
		"current_staff_id":      state.CurrentStaffID,
		"last_operation_log_id": state.LastOperationLogID,
	}
}

func insertWorkOperationLog(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any) uint64 {
	return insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, currentWorkCustomerStage(ctx, customerID, assetID), values, workResultSuccess)
}

func insertWorkOperationLogWithResult(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any, resultValue string) uint64 {
	now := time.Now()
	statusCode := ""
	if state != nil {
		statusCode = state.CurrentStageCode
	}
	if resultValue == "" {
		resultValue = workResultSuccess
	}
	operationID := uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":            customerID,
		"asset_id":               assetID,
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"stage_code":             statusCode,
		"result_value":           resultValue,
		"title":                  task.Name,
		"content":                "",
		"data_snapshot_json":     jsonText(values),
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"created_at":             now,
	}))
	syncWorkTaskStatEvent(ctx, staff, task, customerID, assetID, statusCode, operationID, resultValue, now)
	return operationID
}

func saveWorkDataRecord(ctx context.Context, customerID uint64, assetID uint64, templateID uint64, taskID uint64, operationID uint64, record map[string]any) {
	now := time.Now()
	recordJSON := jsonText(record)
	data := map[string]any{
		"customer_id":      customerID,
		"asset_id":         assetID,
		"data_template_id": templateID,
		"task_id":          taskID,
		"operation_log_id": operationID,
		"record_json":      recordJSON,
		"summary":          "",
		"status":           crmmodel.StatusEnabled,
		"sort":             100,
		"updated_at":       now,
	}
	model := crmmodel.NewDataRecordModel()
	existing := model.Find(ctx, map[string]any{
		"customer_id":      customerID,
		"asset_id":         assetID,
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	})
	if existing != nil {
		merged := mapFromAny(existing.RecordJSON)
		for key, value := range record {
			merged[key] = value
		}
		data["record_json"] = jsonText(merged)
		model.Update(ctx, map[string]any{"id": existing.ID}, data)
		syncWorkStatFieldValues(ctx, customerID, assetID, templateID, taskID, operationID, record, now)
		return
	}
	data["created_at"] = now
	model.Insert(ctx, data)
	syncWorkStatFieldValues(ctx, customerID, assetID, templateID, taskID, operationID, record, now)
}

func syncWorkStatFieldValues(ctx context.Context, customerID uint64, assetID uint64, templateID uint64, taskID uint64, operationID uint64, record map[string]any, changedAt time.Time) {
	defer func() {
		_ = recover()
	}()
	if customerID == 0 || templateID == 0 || len(record) == 0 {
		return
	}
	model := crmmodel.NewStatFieldValueModel()
	for fieldIDText, value := range record {
		fieldID := inputUint64(fieldIDText)
		if fieldID == 0 {
			continue
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"id":               fieldID,
			"data_template_id": templateID,
			"stat_enabled":     true,
			"status":           crmmodel.StatusEnabled,
		})
		if field == nil || strings.TrimSpace(field.FieldKey) == "" {
			continue
		}
		data := workStatFieldValueRecord(customerID, assetID, templateID, taskID, operationID, field, value, changedAt)
		existing := model.Find(ctx, map[string]any{
			"customer_id":   customerID,
			"asset_id":      assetID,
			"data_field_id": field.ID,
		})
		if existing != nil {
			model.Update(ctx, map[string]any{"id": existing.ID}, data)
			continue
		}
		data["created_at"] = changedAt
		model.Insert(ctx, data)
	}
}

func workStatFieldValueRecord(customerID uint64, assetID uint64, templateID uint64, taskID uint64, operationID uint64, field *crmmodel.DataField, value any, changedAt time.Time) map[string]any {
	valueText := inputText(value)
	if emptyWorkFieldValue(value) {
		valueText = ""
	}
	return map[string]any{
		"customer_id":      customerID,
		"asset_id":         assetID,
		"data_template_id": templateID,
		"data_field_id":    field.ID,
		"field_key":        field.FieldKey,
		"field_name":       field.Name,
		"field_type":       field.FieldType,
		"stat_type":        normalizeWorkStatType(field.StatType),
		"stat_group":       field.StatGroup,
		"value_text":       valueText,
		"value_number":     workStatNumberValue(field, value),
		"value_date":       workStatDateValue(field, value),
		"value_bool":       booleanFromAny(value),
		"value_json":       workStatJSONValue(value),
		"source":           crmmodel.StatValueSourceForm,
		"task_id":          taskID,
		"operation_log_id": operationID,
		"status":           crmmodel.StatusEnabled,
		"updated_at":       changedAt,
	}
}

func normalizeWorkStatType(statType string) string {
	switch strings.TrimSpace(statType) {
	case crmmodel.DataFieldStatTypeMetric,
		crmmodel.DataFieldStatTypeAmount,
		crmmodel.DataFieldStatTypeTime,
		crmmodel.DataFieldStatTypeStatus,
		crmmodel.DataFieldStatTypeText:
		return strings.TrimSpace(statType)
	default:
		return crmmodel.DataFieldStatTypeDimension
	}
}

func workStatNumberValue(field *crmmodel.DataField, value any) float64 {
	if field == nil {
		return 0
	}
	switch field.StatType {
	case crmmodel.DataFieldStatTypeMetric, crmmodel.DataFieldStatTypeAmount:
		return numericValue(value)
	}
	switch field.FieldType {
	case "number", "money":
		return numericValue(value)
	default:
		return 0
	}
}

func workStatDateValue(field *crmmodel.DataField, value any) time.Time {
	if field == nil {
		return time.Time{}
	}
	if field.StatType != crmmodel.DataFieldStatTypeTime && field.FieldType != "date" && field.FieldType != "datetime" {
		return time.Time{}
	}
	text := inputText(value)
	if text == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func workStatJSONValue(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func syncWorkTaskStatEvent(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, stageCode string, operationID uint64, resultValue string, eventAt time.Time) {
	defer func() {
		_ = recover()
	}()
	if staff == nil || task == nil || customerID == 0 || operationID == 0 {
		return
	}
	eventKey := fmt.Sprintf("task:%d:%s", task.ID, resultValue)
	insertWorkStatEvent(ctx, map[string]any{
		"event_type":             crmmodel.StatEventTypeTask,
		"event_key":              eventKey,
		"customer_id":            customerID,
		"asset_id":               assetID,
		"stage_code":             stageCode,
		"from_stage_code":        "",
		"to_stage_code":          "",
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"result_value":           resultValue,
		"operation_log_id":       operationID,
		"transition_log_id":      0,
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"event_at":               eventAt,
		"created_at":             eventAt,
	})
}

func syncWorkTransitionStatEvent(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, fromStageCode string, toStageCode string, operationID uint64, transitionLogID uint64, resultValue string, eventAt time.Time) {
	defer func() {
		_ = recover()
	}()
	if staff == nil || task == nil || customerID == 0 || operationID == 0 || transitionLogID == 0 {
		return
	}
	eventKey := fmt.Sprintf("transition:%s:%s:%d:%s", fromStageCode, toStageCode, task.ID, resultValue)
	insertWorkStatEvent(ctx, map[string]any{
		"event_type":             crmmodel.StatEventTypeTransition,
		"event_key":              eventKey,
		"customer_id":            customerID,
		"asset_id":               assetID,
		"stage_code":             toStageCode,
		"from_stage_code":        fromStageCode,
		"to_stage_code":          toStageCode,
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"result_value":           resultValue,
		"operation_log_id":       operationID,
		"transition_log_id":      transitionLogID,
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"event_at":               eventAt,
		"created_at":             eventAt,
	})
}

func insertWorkStatEvent(ctx context.Context, record map[string]any) {
	model := crmmodel.NewStatEventModel()
	if existing := model.Find(ctx, map[string]any{
		"event_key":         record["event_key"],
		"operation_log_id":  record["operation_log_id"],
		"transition_log_id": record["transition_log_id"],
	}); existing != nil {
		return
	}
	model.Insert(ctx, record)
}

func insertWorkCustomerMember(ctx context.Context, staff *WorkStaffSession, customerID uint64) {
	crmmodel.NewCustomerMemberModel().Insert(ctx, map[string]any{
		"customer_id":   customerID,
		"asset_id":      0,
		"department_id": staff.DepartmentID,
		"staff_id":      staff.ID,
		"relation_type": crmmodel.MemberRelationCreator,
		"can_view":      true,
		"status":        crmmodel.StatusEnabled,
		"created_at":    time.Now(),
	})
}

func upsertWorkAssigneeMember(ctx context.Context, customerID uint64, assetID uint64, departmentID uint64, staffID uint64) {
	if customerID == 0 || (departmentID == 0 && staffID == 0) {
		return
	}
	model := crmmodel.NewCustomerMemberModel()
	existing := model.Find(ctx, map[string]any{
		"customer_id":   customerID,
		"asset_id":      assetID,
		"relation_type": crmmodel.MemberRelationAssignee,
		"status":        crmmodel.StatusEnabled,
	})
	record := map[string]any{
		"department_id": departmentID,
		"staff_id":      staffID,
		"can_view":      true,
	}
	if existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, record)
		return
	}
	record["customer_id"] = customerID
	record["asset_id"] = assetID
	record["relation_type"] = crmmodel.MemberRelationAssignee
	record["status"] = crmmodel.StatusEnabled
	record["created_at"] = time.Now()
	model.Insert(ctx, record)
}

func insertWorkCustomerStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, operationID uint64, taskID uint64) {
	stage := workInitialStageForTask(ctx, staff, taskID)
	stageCode := ""
	if stage != nil {
		stageCode = stage.Code
	}
	now := time.Now()
	model := crmmodel.NewCustomerStageModel()
	if existing := model.Find(ctx, map[string]any{"customer_id": customerID, "asset_id": assetID}); existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, map[string]any{
			"last_operation_log_id": operationID,
			"last_operated_at":      now,
			"updated_at":            now,
		})
		return
	}
	record := map[string]any{
		"customer_id":            customerID,
		"asset_id":               assetID,
		"current_stage_code":     stageCode,
		"current_department_id":  staff.DepartmentID,
		"current_staff_id":       0,
		"last_operation_log_id":  operationID,
		"last_transition_log_id": 0,
		"last_operated_at":       now,
		"context_json":           "{}",
		"created_at":             now,
		"updated_at":             now,
	}
	model.Insert(ctx, record)
}

func workInitialStageForTask(ctx context.Context, staff *WorkStaffSession, taskID uint64) *crmmodel.Stage {
	if taskID > 0 {
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": taskID, "status": crmmodel.StatusEnabled})
		if task != nil && task.StageID > 0 {
			if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": task.StageID, "status": crmmodel.StatusEnabled}); stage != nil {
				return stage
			}
		}
	}
	if staff == nil || staff.DepartmentID == 0 {
		return nil
	}
	return crmmodel.NewStageModel().Find(ctx, map[string]any{
		"owner_department_id": staff.DepartmentID,
		"status":              crmmodel.StatusEnabled,
	})
}

func currentWorkCustomerStage(ctx context.Context, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	return crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	})
}

func ensureCurrentWorkCustomerStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	state := crmmodel.NewCustomerStageModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	})
	if state != nil || customerID == 0 {
		return state
	}
	return ensureWorkCustomerStage(ctx, staff, customerID, assetID)
}

func ensureWorkCustomerStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) *crmmodel.CustomerStage {
	customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID})
	if customer == nil {
		return nil
	}
	stage := firstEnabledWorkStage(ctx)
	if stage == nil {
		return nil
	}
	departmentID := initialWorkCustomerDepartmentID(ctx, staff, stage, customer)
	now := time.Now()
	record := map[string]any{
		"customer_id":            customerID,
		"asset_id":               assetID,
		"current_stage_code":     stage.Code,
		"current_department_id":  departmentID,
		"current_staff_id":       uint64(0),
		"last_operation_log_id":  uint64(0),
		"last_transition_log_id": uint64(0),
		"last_operated_at":       now,
		"context_json":           "{}",
		"created_at":             now,
		"updated_at":             now,
	}
	model := crmmodel.NewCustomerStageModel()
	model.Insert(ctx, record)
	return model.Find(ctx, map[string]any{"customer_id": customerID, "asset_id": assetID})
}

func firstEnabledWorkStage(ctx context.Context) *crmmodel.Stage {
	return crmmodel.NewStageModel().Find(
		ctx,
		map[string]any{"status": crmmodel.StatusEnabled},
		map[string]any{"order": "sort asc, id asc"},
	)
}

func initialWorkCustomerDepartmentID(ctx context.Context, staff *WorkStaffSession, stage *crmmodel.Stage, customer *crmmodel.Customer) uint64 {
	if stage != nil && stage.OwnerDepartmentID > 0 {
		return stage.OwnerDepartmentID
	}
	if staff != nil && staff.DepartmentID > 0 {
		return staff.DepartmentID
	}
	if customer != nil && customer.CreatedByStaffID > 0 {
		if creator := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": customer.CreatedByStaffID, "status": crmmodel.StatusEnabled}); creator != nil {
			return creator.DepartmentID
		}
	}
	return 0
}

func updateWorkCustomerOwner(ctx context.Context, customerID uint64, assetID uint64, departmentID uint64, staffID uint64, operationID uint64) {
	now := time.Now()
	crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}, map[string]any{
		"current_department_id": departmentID,
		"current_staff_id":      staffID,
		"last_operation_log_id": operationID,
		"last_operated_at":      now,
		"updated_at":            now,
	})
}

func updateWorkCustomerStageOperation(ctx context.Context, customerID uint64, assetID uint64, operationID uint64) {
	now := time.Now()
	crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}, map[string]any{
		"last_operation_log_id": operationID,
		"last_operated_at":      now,
		"updated_at":            now,
	})
}

func applyWorkStageTransition(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, operationID uint64, resultValue string) {
	applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue, 0, 0)
}

func applyWorkStageTransitionWithOwner(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, operationID uint64, resultValue string, assignedDepartmentID uint64, assignedStaffID uint64) {
	if fromState == nil || task == nil {
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return
	}
	transition := findWorkStageTransition(ctx, staff, customerID, fromState, task, resultValue)
	if transition == nil {
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return
	}
	departmentID, staffID := transitionOwner(ctx, staff, fromState, transition, assignedDepartmentID, assignedStaffID)
	now := time.Now()
	transitionLogID := uint64(crmmodel.NewStageTransitionLogModel().Insert(ctx, map[string]any{
		"customer_id":       customerID,
		"asset_id":          assetID,
		"from_stage_code":   fromState.CurrentStageCode,
		"to_stage_code":     transition.ToStageCode,
		"task_id":           task.ID,
		"result_value":      resultValue,
		"operation_log_id":  operationID,
		"operator_staff_id": staff.ID,
		"created_at":        now,
	}))
	crmmodel.NewCustomerStageModel().Update(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
	}, map[string]any{
		"current_stage_code":     transition.ToStageCode,
		"current_department_id":  departmentID,
		"current_staff_id":       staffID,
		"last_operation_log_id":  operationID,
		"last_transition_log_id": transitionLogID,
		"last_operated_at":       now,
		"updated_at":             now,
	})
	if departmentID > 0 || staffID > 0 {
		upsertWorkAssigneeMember(ctx, customerID, assetID, departmentID, staffID)
	}
	syncWorkTransitionStatEvent(ctx, staff, task, customerID, assetID, fromState.CurrentStageCode, transition.ToStageCode, operationID, transitionLogID, resultValue, now)
}

func findWorkStageTransition(ctx context.Context, staff *WorkStaffSession, customerID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, resultValue string) *crmmodel.StageTransition {
	if fromState == nil || task == nil {
		return nil
	}
	transitions := crmmodel.NewStageTransitionModel().Select(ctx, map[string]any{
		"from_stage_code": fromState.CurrentStageCode,
		"task_id":         task.ID,
		"status":          crmmodel.StatusEnabled,
	})
	var fallback *crmmodel.StageTransition
	for _, transition := range transitions {
		if transition == nil {
			continue
		}
		if transition.ResultValue == resultValue && workTransitionMatches(ctx, staff, customerID, fromState, task, transition, resultValue) {
			return transition
		}
		if transition.ResultValue == "" && fallback == nil && workTransitionMatches(ctx, staff, customerID, fromState, task, transition, resultValue) {
			fallback = transition
		}
	}
	if fallback != nil {
		return fallback
	}
	return defaultWorkTaskTransition(ctx, fromState, task)
}

func defaultWorkTaskTransition(ctx context.Context, fromState *crmmodel.CustomerStage, task *crmmodel.Task) *crmmodel.StageTransition {
	nextStageCode := inputText(mapFromAny(task.ConfigJSON)["next_stage_code"])
	if nextStageCode == "" || nextStageCode == fromState.CurrentStageCode {
		return nil
	}
	if crmmodel.NewStageModel().Find(ctx, map[string]any{"code": nextStageCode, "status": crmmodel.StatusEnabled}) == nil {
		return nil
	}
	ownerMode := crmmodel.StageOwnerKeep
	if task.TaskType == crmmodel.TaskTypeAssign {
		ownerMode = crmmodel.StageOwnerAssign
	}
	return &crmmodel.StageTransition{
		FromStageCode: fromState.CurrentStageCode,
		TaskID:        task.ID,
		ToStageCode:   nextStageCode,
		OwnerMode:     ownerMode,
		Status:        crmmodel.StatusEnabled,
	}
}

func workTransitionMatches(ctx context.Context, staff *WorkStaffSession, customerID uint64, state *crmmodel.CustomerStage, task *crmmodel.Task, transition *crmmodel.StageTransition, resultValue string) bool {
	input := workTransitionInput(ctx, staff, task, transition, customerID, state, resultValue)
	if !workTransitionConditionMatches(transition.ConditionJSON, input) {
		return false
	}
	return workTransitionScriptMatches(ctx, transition, input)
}

func workTransitionInput(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, transition *crmmodel.StageTransition, customerID uint64, state *crmmodel.CustomerStage, resultValue string) map[string]any {
	assetID := uint64(0)
	if state != nil {
		assetID = state.AssetID
	}
	input := workDecisionInput(ctx, staff, task, customerID, assetID, state, map[string]any{"result_value": resultValue})
	input["transition"] = map[string]any{
		"id":              transition.ID,
		"from_stage_code": transition.FromStageCode,
		"to_stage_code":   transition.ToStageCode,
		"result_value":    transition.ResultValue,
	}
	input["result_value"] = resultValue
	return input
}

func workTransitionConditionMatches(raw string, input map[string]any) bool {
	mode, rows := workTransitionConditionRows(raw)
	if len(rows) == 0 {
		return true
	}
	matched := 0
	for _, row := range rows {
		if workTransitionConditionRowMatches(row, input) {
			matched++
			if mode == workTransitionModeAny {
				return true
			}
		} else if mode == workTransitionModeAll {
			return false
		}
	}
	return matched == len(rows)
}

func workTransitionConditionRows(raw string) (string, []map[string]any) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "[]" {
		return workTransitionModeAll, nil
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(raw), &object); err == nil {
		mode := inputText(object["mode"])
		if mode == "" {
			mode = workTransitionModeAll
		}
		if rows := mapsFromAny(object[workTransitionModeAny]); len(rows) > 0 {
			return workTransitionModeAny, rows
		}
		if rows := mapsFromAny(object[workTransitionModeAll]); len(rows) > 0 {
			return workTransitionModeAll, rows
		}
		if rows := mapsFromAny(object["conditions"]); len(rows) > 0 {
			return mode, rows
		}
		if inputText(object["field"]) != "" || inputText(object["path"]) != "" {
			return mode, []map[string]any{object}
		}
		return mode, nil
	}
	return workTransitionModeAll, mapsFromJSON(raw)
}

func workTransitionConditionRowMatches(row map[string]any, input map[string]any) bool {
	path := firstText(row, "field", "path")
	if path == "" {
		return true
	}
	actual := valueByPath(input, path)
	expected := firstPresent(row, "value", "values", "expected")
	operator := firstText(row, "operator", "op")
	if operator == "" {
		operator = "equals"
	}
	switch operator {
	case "equals", "eq", "=", "==":
		return valuesEqual(actual, expected)
	case "notEquals", "ne", "!=", "<>":
		return !valuesEqual(actual, expected)
	case "in":
		return valueInList(actual, expected)
	case "notIn":
		return !valueInList(actual, expected)
	case "empty":
		return emptyWorkFieldValue(actual)
	case "notEmpty":
		return !emptyWorkFieldValue(actual)
	case "contains":
		return strings.Contains(inputText(actual), inputText(expected))
	default:
		return valuesEqual(actual, expected)
	}
}

func workTransitionScriptMatches(ctx context.Context, transition *crmmodel.StageTransition, input map[string]any) bool {
	if transition.ScriptID == 0 {
		return true
	}
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": transition.ScriptID, "status": crmmodel.StatusEnabled})
	if script == nil {
		return false
	}
	timeout := fronteval.DefaultTimeout
	if script.TimeoutMS > 0 {
		timeout = time.Duration(script.TimeoutMS) * time.Millisecond
	}
	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: script.Language,
		Script:   script.Script,
		Entry:    script.Entry,
		Timeout:  timeout,
		Input:    input,
		Config:   map[string]any{},
	})
	if err != nil {
		return false
	}
	return workTransitionScriptResultPassed(result.Value)
}

func workTransitionScriptResultPassed(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case map[string]any:
		for _, key := range []string{"matched", "pass", "ok", "success"} {
			if booleanFromAny(typed[key]) {
				return true
			}
		}
		return inputText(typed["result"]) == workTransitionScriptPass
	default:
		return booleanFromAny(value) || inputText(value) == workTransitionScriptPass
	}
}

func transitionOwner(ctx context.Context, staff *WorkStaffSession, fromState *crmmodel.CustomerStage, transition *crmmodel.StageTransition, assignedDepartmentID uint64, assignedStaffID uint64) (uint64, uint64) {
	switch transition.OwnerMode {
	case crmmodel.StageOwnerFixedDepartment:
		return transition.ToDepartmentID, 0
	case crmmodel.StageOwnerFixedStaff:
		if targetStaff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": transition.ToStaffID, "status": crmmodel.StatusEnabled}); targetStaff != nil {
			departmentID := transition.ToDepartmentID
			if departmentID == 0 {
				departmentID = targetStaff.DepartmentID
			}
			return departmentID, targetStaff.ID
		}
	case crmmodel.StageOwnerCreator:
		if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": fromState.CustomerID}); customer != nil {
			if creator := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": customer.CreatedByStaffID, "status": crmmodel.StatusEnabled}); creator != nil {
				return creator.DepartmentID, creator.ID
			}
		}
	case crmmodel.StageOwnerAssign:
		if assignedStaffID > 0 || assignedDepartmentID > 0 {
			return assignedDepartmentID, assignedStaffID
		}
		if transition.ToStaffID > 0 || transition.ToDepartmentID > 0 {
			return transition.ToDepartmentID, transition.ToStaffID
		}
		return fromState.CurrentDepartmentID, fromState.CurrentStaffID
	case crmmodel.StageOwnerKeep:
		return fromState.CurrentDepartmentID, fromState.CurrentStaffID
	}
	if transition.ToStaffID > 0 || transition.ToDepartmentID > 0 {
		return transition.ToDepartmentID, transition.ToStaffID
	}
	if staff != nil {
		return staff.DepartmentID, staff.ID
	}
	return fromState.CurrentDepartmentID, fromState.CurrentStaffID
}

func workActionValues(payload map[string]any) map[string]any {
	values := mapFromAny(payload["values"])
	for _, key := range []string{"department_id", "departmentId", "staff_id", "staffId"} {
		if _, exists := values[key]; !exists && payload[key] != nil {
			values[key] = payload[key]
		}
	}
	return values
}

func workDepartmentOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewDepartmentModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := inputUint64(row["id"])
		name := inputText(row["name"])
		if id == 0 || name == "" {
			continue
		}
		result = append(result, map[string]any{"id": id, "name": name})
	}
	return result
}

func workStaffOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewStaffModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := inputUint64(row["id"])
		name := inputText(row["name"])
		if id == 0 || name == "" {
			continue
		}
		result = append(result, map[string]any{
			"id":            id,
			"name":          name,
			"phone":         inputText(row["phone"]),
			"department_id": inputUint64(row["department_id"]),
		})
	}
	return result
}

func workFieldInputKey(field *crmmodel.FormField) string {
	if field.MainField != "" {
		return "main:" + field.MainField
	}
	if field.DataFieldID > 0 {
		return fmt.Sprintf("data:%d", field.DataFieldID)
	}
	return fmt.Sprintf("field:%d", field.ID)
}

func mainFieldInputType(field string) string {
	switch field {
	case "remark":
		return "textarea"
	case "gender":
		return "radio"
	case "source_id", "channel_id", "level_id", "asset_status_id":
		return "select"
	default:
		return "text"
	}
}

func mainFieldOptions(ctx context.Context, field string) []map[string]any {
	switch field {
	case "gender":
		return []map[string]any{
			{"id": "male", "name": "男", "value": "male"},
			{"id": "female", "name": "女", "value": "female"},
			{"id": "unknown", "name": "未知", "value": "unknown"},
		}
	case "source_id":
		return namedWorkOptions(crmmodel.NewCustomerSourceModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}))
	case "channel_id":
		return namedWorkOptions(crmmodel.NewCustomerChannelModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}))
	case "level_id":
		return namedWorkOptions(crmmodel.NewCustomerLevelModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}))
	case "asset_status_id":
		return namedWorkOptions(crmmodel.NewAssetStatusModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled}))
	default:
		return []map[string]any{}
	}
}

func namedWorkOptions(rows []map[string]any) []map[string]any {
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := inputUint64(row["id"])
		name := inputText(row["name"])
		if id == 0 || name == "" {
			continue
		}
		options = append(options, map[string]any{
			"id":    id,
			"name":  name,
			"value": fmt.Sprintf("%d", id),
		})
	}
	return options
}

func workDepartmentName(ctx context.Context, id uint64) string {
	if id == 0 {
		return ""
	}
	if department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": id}); department != nil {
		return department.Name
	}
	return ""
}

func workStaffName(ctx context.Context, id uint64) string {
	if id == 0 {
		return ""
	}
	if staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": id}); staff != nil {
		if staff.Phone != "" {
			return fmt.Sprintf("%s（%s）", staff.Name, staff.Phone)
		}
		return staff.Name
	}
	return ""
}

func workPublicResourceName(ctx context.Context, id uint64) string {
	if id == 0 {
		return ""
	}
	if resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{"id": id}); resource != nil {
		if resource.Location != "" {
			return fmt.Sprintf("%s（%s）", resource.Name, resource.Location)
		}
		return resource.Name
	}
	return ""
}

func emptyWorkFieldValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return inputText(typed) == ""
	case []any:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return false
	}
}

func canOperateCurrentState(staff *WorkStaffSession, state *crmmodel.CustomerStage) bool {
	if staff == nil || state == nil {
		return false
	}
	if state.CurrentStaffID > 0 {
		return state.CurrentStaffID == staff.ID
	}
	if state.CurrentDepartmentID > 0 && staff.DepartmentID > 0 {
		return state.CurrentDepartmentID == staff.DepartmentID
	}
	return state.CurrentDepartmentID == 0
}

func uint64ListFromJSON(raw string) []uint64 {
	return uint64ListFromAny(raw)
}

func uint64ListFromAny(value any) []uint64 {
	values := stringListFromJSON(value)
	result := make([]uint64, 0, len(values))
	seen := map[uint64]bool{}
	for _, value := range values {
		id := inputUint64(value)
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func uint64SetContains(values []uint64, target uint64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func mapsFromJSON(raw string) []map[string]any {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &rows); err == nil {
		return rows
	}
	var generic []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &generic); err == nil {
		return mapsFromAny(generic)
	}
	return nil
}

func mapsFromAny(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		result := make([]map[string]any, 0, len(rows))
		for _, item := range rows {
			if row := mapFromAny(item); len(row) > 0 {
				result = append(result, row)
			}
		}
		return result
	case string:
		return mapsFromJSON(rows)
	default:
		return nil
	}
}

func valueByPath(input map[string]any, path string) any {
	current := any(input)
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		row := mapFromAny(current)
		if len(row) == 0 {
			return nil
		}
		current = row[part]
	}
	return current
}

func firstPresent(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			return value
		}
	}
	return nil
}

func valuesEqual(left any, right any) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	switch right.(type) {
	case []any, []string:
		return valueInList(left, right)
	}
	return inputText(left) == inputText(right)
}

func valueInList(value any, list any) bool {
	switch values := list.(type) {
	case []any:
		for _, item := range values {
			if valuesEqual(value, item) {
				return true
			}
		}
	case []string:
		for _, item := range values {
			if inputText(value) == item {
				return true
			}
		}
	case string:
		for _, item := range stringListFromJSON(values) {
			if inputText(value) == item {
				return true
			}
		}
	default:
		return inputText(value) == inputText(list)
	}
	return false
}

func booleanFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case uint64:
		return typed != 0
	case float64:
		return typed != 0
	default:
		text := strings.ToLower(inputText(value))
		return text == "true" || text == "1" || text == "yes" || text == "pass" || text == "success"
	}
}

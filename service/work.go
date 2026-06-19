package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	deverjwt "github.com/shemic/dever/auth/jwt"
	"github.com/shemic/dever/config"
	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
	frontservice "github.com/dever-package/front/service"
	uploadrepo "github.com/dever-package/front/service/upload/repository"
)

const (
	workSiteKey              = "work"
	workAuthProvider         = "crm_work"
	feishuAPIBase            = "https://open.feishu.cn/open-apis"
	feishuRequestTimeout     = 10 * time.Second
	workResultSuccess        = "success"
	workResultProgress       = "progress"
	workResultAutoFailed     = "auto_failed"
	workCustomerModeAll      = "all"
	workCustomerModePending  = "pending"
	workCustomerModeDone     = "done"
	maxWorkAutoTriggerDepth  = 5
	workTransitionModeAll    = "all"
	workTransitionModeAny    = "any"
	workTransitionScriptPass = "pass"
	workSubmitModeComplete   = "complete"
	workSubmitModeProgress   = "progress"
)

type WorkService struct{}

type WorkStaffSession struct {
	ID           uint64
	Name         string
	Phone        string
	FeishuOpenID string
	DepartmentID uint64
}

type workExecutionRuntime struct {
	depth int
	seen  map[string]bool
}

type feishuAppAccessTokenResponse struct {
	Code           int    `json:"code"`
	Msg            string `json:"msg"`
	AppAccessToken string `json:"app_access_token"`
}

type feishuAccessTokenResponse struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data feishuIdentityResult `json:"data"`
}

type feishuIdentityResult struct {
	OpenID    string `json:"open_id"`
	UnionID   string `json:"union_id"`
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	EnName    string `json:"en_name"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
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
	staff, err := findUniqueEnabledStaffByField(ctx, "phone", phone, "手机号")
	if err != nil {
		return nil, err
	}
	if staff == nil || !verifyCRMStaffPassword(ctx, staff, password) {
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

func (WorkService) FeishuConfig(ctx context.Context) (map[string]any, error) {
	config := currentWorkFeishuConfig(ctx)
	return map[string]any{
		"enabled": config.FeishuAppID != "" && config.FeishuAppSecret != "",
		"app_id":  config.FeishuAppID,
		"appId":   config.FeishuAppID,
	}, nil
}

func (WorkService) FeishuLogin(ctx context.Context, payload map[string]any) (map[string]any, error) {
	code := firstText(payload, "code")
	if code == "" {
		return nil, fmt.Errorf("飞书授权码不能为空")
	}
	identity, err := fetchWorkFeishuIdentity(ctx, code)
	if err != nil {
		return nil, err
	}
	openID := strings.TrimSpace(identity.OpenID)
	if openID == "" {
		return nil, fmt.Errorf("飞书未返回 open_id")
	}
	staff, err := findUniqueEnabledStaffByField(ctx, "feishu_open_id", openID, "飞书 OpenID")
	if err != nil {
		return nil, err
	}
	if staff == nil {
		return nil, fmt.Errorf("飞书账号未绑定人员，请管理员在人员管理中配置 OpenID：%s", openID)
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

func currentWorkFeishuConfig(ctx context.Context) crmmodel.BasicConfig {
	return CurrentBasicConfig(ctx)
}

func fetchWorkFeishuIdentity(ctx context.Context, code string) (feishuIdentityResult, error) {
	appAccessToken, err := fetchWorkFeishuAppAccessToken(ctx)
	if err != nil {
		return feishuIdentityResult{}, err
	}
	var payload feishuAccessTokenResponse
	if err := postFeishuJSON(ctx, "/authen/v1/access_token", map[string]any{
		"grant_type": "authorization_code",
		"code":       code,
	}, appAccessToken, &payload); err != nil {
		return feishuIdentityResult{}, err
	}
	if payload.Code != 0 {
		return feishuIdentityResult{}, fmt.Errorf("飞书身份换取失败：%s", fallbackFeishuMessage(payload.Msg))
	}
	if strings.TrimSpace(payload.Data.OpenID) == "" {
		return feishuIdentityResult{}, fmt.Errorf("飞书未返回用户身份信息")
	}
	return payload.Data, nil
}

func fetchWorkFeishuAppAccessToken(ctx context.Context) (string, error) {
	config := currentWorkFeishuConfig(ctx)
	appID := strings.TrimSpace(config.FeishuAppID)
	appSecret := strings.TrimSpace(config.FeishuAppSecret)
	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("请先配置飞书 AppID 和 AppSecret")
	}
	var payload feishuAppAccessTokenResponse
	if err := postFeishuJSON(ctx, "/auth/v3/app_access_token/internal", map[string]any{
		"app_id":     appID,
		"app_secret": appSecret,
	}, "", &payload); err != nil {
		return "", err
	}
	if payload.Code != 0 {
		return "", fmt.Errorf("飞书 app_access_token 获取失败：%s", fallbackFeishuMessage(payload.Msg))
	}
	if strings.TrimSpace(payload.AppAccessToken) == "" {
		return "", fmt.Errorf("飞书未返回 app_access_token")
	}
	return payload.AppAccessToken, nil
}

func postFeishuJSON(ctx context.Context, path string, body any, bearerToken string, target any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	requestCtx, cancel := context.WithTimeout(ctx, feishuRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, feishuAPIBase+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if strings.TrimSpace(bearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("飞书接口请求失败：%w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取飞书接口响应失败：%w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("飞书接口请求失败：%d", resp.StatusCode)
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("解析飞书接口响应失败：%w", err)
	}
	return nil
}

func fallbackFeishuMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "未知错误"
	}
	return message
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
	tasks = append(tasks, workPendingTodoTasks(ctx, staff, customerID, currentAssetID)...)
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
		"forms":       workFormOptions(ctx),
	}, nil
}

func (WorkService) Execute(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	var result map[string]any
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		var executeErr error
		result, executeErr = executeWorkTask(txCtx, staff, payload, newWorkExecutionRuntime())
		return executeErr
	})
	if err != nil {
		return nil, err
	}
	return result, nil
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
		FeishuOpenID: staff.FeishuOpenID,
		DepartmentID: staff.DepartmentID,
	}
}

func VerifyCRMStaffPassword(stored string, password string) bool {
	return frontservice.VerifyPassword(stored, password)
}

func findUniqueEnabledStaffByField(ctx context.Context, field string, value string, label string) (*crmmodel.Staff, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	rows := crmmodel.NewStaffModel().Select(ctx, map[string]any{
		field:    value,
		"status": crmmodel.StatusEnabled,
	})
	if len(rows) > 1 {
		return nil, fmt.Errorf("%s 绑定了多个启用人员，请先在后台处理重复数据", label)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func verifyCRMStaffPassword(ctx context.Context, staff *crmmodel.Staff, password string) bool {
	if staff == nil || inputText(password) == "" {
		return false
	}
	if !frontservice.VerifyPassword(staff.Password, password) {
		return false
	}
	if frontservice.PasswordNeedsUpgrade(staff.Password) {
		if hashed, err := frontservice.HashPassword(password); err == nil {
			crmmodel.NewStaffModel().Update(ctx, map[string]any{"id": staff.ID}, map[string]any{
				"password": hashed,
			})
		}
	}
	return true
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
		"id":             staff.ID,
		"name":           staff.Name,
		"phone":          staff.Phone,
		"feishu_open_id": staff.FeishuOpenID,
		"department_id":  staff.DepartmentID,
		"exp":            expiredAt.UnixMilli(),
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
	case workCustomerModeAll:
		return allWorkCustomers(ctx, staff)
	case workCustomerModeDone:
		return doneWorkCustomers(ctx, staff)
	default:
		return pendingWorkCustomers(ctx, staff)
	}
}

func normalizeWorkCustomerMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case workCustomerModeAll:
		return workCustomerModeAll
	case workCustomerModeDone:
		return workCustomerModeDone
	default:
		return workCustomerModePending
	}
}

func allWorkCustomers(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	rows := visibleWorkCustomers(ctx, staff)
	seen := map[uint64]bool{}
	for _, row := range rows {
		if customerID := inputUint64(row["id"]); customerID > 0 {
			seen[customerID] = true
		}
	}
	for _, target := range doneWorkCustomerTargets(ctx, staff) {
		if target.customerID == 0 || seen[target.customerID] {
			continue
		}
		if row := doneWorkCustomerRow(ctx, staff, target.customerID, target.assetIDs); len(row) > 0 {
			rows = append(rows, row)
			seen[target.customerID] = true
		}
	}
	sortWorkRowsByPendingTasks(rows)
	return rows
}

func sortWorkRowsByPendingTasks(rows []map[string]any) {
	sort.SliceStable(rows, func(i, j int) bool {
		leftPending := workRowHasPendingTasks(rows[i])
		rightPending := workRowHasPendingTasks(rows[j])
		if leftPending != rightPending {
			return leftPending
		}
		return false
	})
}

func workRowHasPendingTasks(row map[string]any) bool {
	if len(mapListFromAny(row["row_tasks"])) > 0 {
		return true
	}
	for _, asset := range mapListFromAny(row["assets"]) {
		if len(mapListFromAny(asset["row_tasks"])) > 0 {
			return true
		}
	}
	return false
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
	customer["row_tasks"] = workPendingTodoTasks(ctx, staff, customerID, 0)
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
	asset["row_tasks"] = workPendingTodoTasks(ctx, staff, customerID, assetID)
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
	if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}); customer != nil && customer.CreatedByStaffID == staff.ID {
		return true
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
		tasks = append(tasks, workPendingTodoTasks(ctx, staff, customerID, 0)...)
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
			tasks := workAvailableTasks(ctx, staff, state)
			tasks = append(tasks, workPendingTodoTasks(ctx, staff, customerID, assetID)...)
			asset["row_tasks"] = workAssetRowTasks(tasks)
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
	if inputText(row["result_value"]) == workResultProgress {
		return "保存进度"
	}
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
	case crmmodel.TaskTypeCollaborate:
		values := mapFromAny(row["data_snapshot_json"])
		if todoName := inputText(values["todo_name"]); todoName != "" {
			return "完成" + todoName
		}
		for _, item := range items {
			if inputText(item["label"]) == "协作待办数" {
				return fmt.Sprintf("生成 %s 个协作待办", inputText(item["value"]))
			}
		}
		return "协作任务"
	case crmmodel.TaskTypeDecision:
		if result := inputText(row["result_value"]); result != "" && result != workResultSuccess {
			return "决策：" + result
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
		label, displayValue, meta := workOperationSnapshotItem(ctx, key, value)
		if label == "" || displayValue == "" {
			continue
		}
		item := map[string]any{
			"key":   key,
			"label": label,
			"value": displayValue,
		}
		for metaKey, metaValue := range meta {
			item[metaKey] = metaValue
		}
		items = append(items, item)
	}
	return items
}

func workOperationSnapshotItem(ctx context.Context, key string, value any) (string, string, map[string]any) {
	switch key {
	case "department_id", "departmentId":
		return "分配部门", workDepartmentName(ctx, inputUint64(value)), nil
	case "staff_id", "staffId":
		return "分配人员", workStaffName(ctx, inputUint64(value)), nil
	case "result_value", "resultValue":
		return "决策结果", inputText(value), nil
	case "resource_id", "resourceId", "booking:resource_id", "main:resource_id":
		return "预定资源", workPublicResourceName(ctx, inputUint64(value)), nil
	case "start_at", "startAt", "booking:start_at", "main:start_at":
		return "开始时间", inputText(value), nil
	case "end_at", "endAt", "booking:end_at", "main:end_at":
		return "结束时间", inputText(value), nil
	case "remark", "booking:remark", "main:remark":
		return "备注", inputText(value), nil
	case "todo_name":
		return "协作待办", inputText(value), nil
	case "todo_id", "todoId":
		return "", "", nil
	case "todo_count":
		return "协作待办数", inputText(value), nil
	}
	if strings.HasPrefix(key, "main:") {
		field := strings.TrimPrefix(key, "main:")
		return workMainFieldLabel(field), workMainFieldDisplayValue(ctx, field, value), nil
	}
	if strings.HasPrefix(key, "data:") {
		fieldID := inputUint64(strings.TrimPrefix(key, "data:"))
		if fieldID == 0 {
			return "", "", nil
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
		if field == nil {
			return key, inputText(value), nil
		}
		displayValue, meta := workDataFieldDisplayValue(ctx, field, value)
		return field.Name, displayValue, meta
	}
	return key, inputText(value), nil
}

func workDataFieldDisplayValue(ctx context.Context, field *crmmodel.DataField, value any) (string, map[string]any) {
	text := inputText(value)
	if field == nil || field.ID == 0 || text == "" {
		return text, nil
	}
	if workIsAttachmentFieldType(field.FieldType) {
		files := workUploadFilePayloads(ctx, uint64ListFromAny(value))
		if len(files) == 0 {
			return text, nil
		}
		return fmt.Sprintf("%d 个附件", len(files)), map[string]any{
			"value_type": "files",
			"files":      files,
		}
	}
	if option := crmmodel.NewDataFieldOptionModel().Find(ctx, map[string]any{
		"data_field_id": field.ID,
		"value":         text,
	}); option != nil {
		return option.Name, nil
	}
	return text, nil
}

func workIsAttachmentFieldType(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "attachment", "file", "image":
		return true
	default:
		return false
	}
}

func workUploadFilePayloads(ctx context.Context, ids []uint64) []map[string]any {
	if len(ids) == 0 {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		file, err := uploadrepo.FindUploadFile(ctx, id)
		if err != nil || file.ID == 0 {
			continue
		}
		result = append(result, uploadrepo.BuildUploadFilePayload(file))
	}
	return result
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
		if inputText(task["task_type"]) == crmmodel.TaskTypeCollaborate && workTaskAlreadyOperated(ctx, state.CustomerID, state.AssetID, inputUint64(task["id"])) {
			continue
		}
		if !workTaskVisibleForState(ctx, task, state) {
			continue
		}
		attachWorkTaskForm(ctx, task)
		result = append(result, task)
	}
	return result
}

func workTaskVisibleForState(ctx context.Context, task map[string]any, state *crmmodel.CustomerStage) bool {
	visibleWhen := mapFromAny(mapFromAny(task["config_json"])["visible_when"])
	fieldID := inputUint64(visibleWhen["data_field_id"])
	if fieldID == 0 {
		return true
	}
	if state == nil || state.CustomerID == 0 {
		return false
	}
	assetID := state.AssetID
	if workTaskVisibleDataTemplateCateID(ctx, fieldID) == crmmodel.CustomerDataTemplateCateID {
		assetID = 0
	}
	value := crmmodel.NewStatFieldValueModel().Find(ctx, map[string]any{
		"customer_id":   state.CustomerID,
		"asset_id":      assetID,
		"data_field_id": fieldID,
		"status":        crmmodel.StatusEnabled,
	})
	actual := any(nil)
	if value != nil {
		actual = value.ValueText
	}
	expected := firstPresent(visibleWhen, "value", "values", "expected")
	operator := firstText(visibleWhen, "operator", "op")
	if operator == "" {
		operator = "equals"
	}
	return workConditionValueMatches(actual, expected, operator)
}

func workTaskVisibleDataTemplateCateID(ctx context.Context, fieldID uint64) uint64 {
	if fieldID == 0 {
		return 0
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":     fieldID,
		"status": crmmodel.StatusEnabled,
	})
	if field == nil {
		return 0
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{
		"id":     field.DataTemplateID,
		"status": crmmodel.StatusEnabled,
	})
	if template == nil {
		return 0
	}
	return template.CateID
}

func workTaskAlreadyOperated(ctx context.Context, customerID uint64, assetID uint64, taskID uint64) bool {
	if customerID == 0 || taskID == 0 {
		return false
	}
	return crmmodel.NewOperationLogModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"task_id":     taskID,
	}) != nil
}

func workPendingTodoTasks(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) []map[string]any {
	if staff == nil || customerID == 0 {
		return []map[string]any{}
	}
	rows := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"status":      crmmodel.WorkTodoStatusPending,
	})
	result := make([]map[string]any, 0, len(rows))
	for _, todo := range rows {
		if todo == nil || !canOperateWorkTodo(staff, todo) {
			continue
		}
		task := workTodoTask(ctx, todo)
		if len(task) == 0 {
			continue
		}
		result = append(result, task)
	}
	sort.SliceStable(result, func(i, j int) bool {
		leftSort := inputInt(result[i]["todo_sort"])
		rightSort := inputInt(result[j]["todo_sort"])
		if leftSort != rightSort {
			return leftSort < rightSort
		}
		return inputUint64(result[i]["todo_id"]) < inputUint64(result[j]["todo_id"])
	})
	return result
}

func workTodoTask(ctx context.Context, todo *crmmodel.WorkTodo) map[string]any {
	if todo == nil || todo.SourceTaskID == 0 {
		return map[string]any{}
	}
	task := crmmodel.NewTaskModel().FindMap(ctx, map[string]any{
		"id":     todo.SourceTaskID,
		"status": crmmodel.StatusEnabled,
	})
	if len(task) == 0 {
		return map[string]any{}
	}
	task["task_type"] = crmmodel.TaskTypeCollaborate
	task["task_name"] = todo.SubTaskName
	task["name"] = todo.SubTaskName
	task["todo_id"] = todo.ID
	task["todo_status"] = todo.Status
	task["todo_required"] = todo.Required
	task["todo_sort"] = todo.Sort
	task["completion_mode"] = normalizeWorkTaskCompletionMode(todo.CompletionMode)
	task["assigned_at"] = todo.AssignedAt
	task["assignee_department_id"] = todo.AssigneeDepartmentID
	task["assignee_staff_id"] = todo.AssigneeStaffID
	if todo.FormID > 0 {
		task["form_id"] = todo.FormID
	} else {
		task["form_id"] = uint64(0)
	}
	attachWorkTaskForm(ctx, task)
	return task
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
	case "asset_name", "asset_status_id":
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

func workCustomerFieldValues(ctx context.Context, customerID uint64) map[string]any {
	return workDataRecordFieldValues(ctx, customerID, 0)
}

func workAssetFieldValues(ctx context.Context, customerID uint64, assetID uint64) map[string]any {
	return workDataRecordFieldValues(ctx, customerID, assetID)
}

func workDataRecordFieldValues(ctx context.Context, customerID uint64, assetID uint64) map[string]any {
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
		fields := workDataTemplateFieldsByID(ctx, record.DataTemplateID)
		for rawFieldID, value := range mapFromAny(record.RecordJSON) {
			field := fields[inputUint64(rawFieldID)]
			if field == nil || strings.TrimSpace(field.FieldKey) == "" {
				continue
			}
			values[field.FieldKey] = value
		}
	}
	return values
}

func workDataTemplateFieldsByID(ctx context.Context, templateID uint64) map[uint64]*crmmodel.DataField {
	result := map[uint64]*crmmodel.DataField{}
	if templateID == 0 {
		return result
	}
	for _, field := range crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	}) {
		if field == nil {
			continue
		}
		result[field.ID] = field
	}
	return result
}

func workDecisionAssets(ctx context.Context, customerID uint64) []map[string]any {
	assets := crmmodel.NewCustomerAssetModel().SelectMap(ctx, map[string]any{"customer_id": customerID})
	result := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		assetID := inputUint64(asset["id"])
		if assetID == 0 {
			continue
		}
		asset["fields"] = workAssetFieldValues(ctx, customerID, assetID)
		result = append(result, asset)
	}
	return result
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

func workAllowedTask(ctx context.Context, staff *WorkStaffSession, taskID uint64, customerID uint64, assetID uint64, todoID uint64) *crmmodel.Task {
	if todoID > 0 {
		return workAllowedTodoTask(ctx, staff, taskID, customerID, assetID, todoID)
	}
	data, err := (WorkService{}).Tasks(ctx, staff, customerID, assetID)
	if err != nil {
		return nil
	}
	rows, _ := data["list"].([]map[string]any)
	for _, row := range rows {
		if inputUint64(row["todo_id"]) == 0 && inputUint64(row["id"]) == taskID {
			return crmmodel.NewTaskModel().Find(ctx, map[string]any{
				"id":     taskID,
				"status": crmmodel.StatusEnabled,
			})
		}
	}
	return nil
}

func workAllowedTodoTask(ctx context.Context, staff *WorkStaffSession, taskID uint64, customerID uint64, assetID uint64, todoID uint64) *crmmodel.Task {
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil || todo.SourceTaskID != taskID || todo.CustomerID != customerID || todo.AssetID != assetID {
		return nil
	}
	if !canOperateWorkTodo(staff, todo) {
		return nil
	}
	return crmmodel.NewTaskModel().Find(ctx, map[string]any{
		"id":     taskID,
		"status": crmmodel.StatusEnabled,
	})
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
	case crmmodel.TaskTypeForm:
		config := mapFromAny(task["config_json"])
		task["completion_mode"] = normalizeWorkTaskCompletionMode(inputText(config["completion_mode"]))
	case crmmodel.TaskTypeCollaborate:
		config := mapFromAny(task["config_json"])
		task["collaboration_items"] = mapsFromAny(config["collaboration_items"])
		task["collaboration_complete_mode"] = normalizeWorkCollaborationCompleteMode(inputText(config["collaboration_complete_mode"]))
		if inputUint64(task["todo_id"]) == 0 {
			task["completion_mode"] = crmmodel.TaskCompletionSubmit
		}
	case crmmodel.TaskTypeDecision:
		config := mapFromAny(task["config_json"])
		task["decision_result_field_id"] = inputUint64(config["decision_result_field_id"])
		if workTaskTriggerType(task) == crmmodel.TaskTriggerManual {
			task["form"] = workDecisionForm(ctx, inputUint64(config["decision_result_field_id"]))
		}
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

func workDecisionForm(ctx context.Context, fieldID uint64) map[string]any {
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":           fieldID,
		"stat_enabled": true,
		"status":       crmmodel.StatusEnabled,
	})
	if field == nil {
		return map[string]any{"id": 0, "name": "决策", "fields": []map[string]any{}}
	}
	return map[string]any{
		"id":   0,
		"name": "决策",
		"fields": []map[string]any{
			{
				"id":         "decision-result",
				"field":      "decision_result",
				"field_key":  "decision_result",
				"name":       "决策结果",
				"field_type": field.FieldType,
				"required":   true,
				"options":    workDataFieldOptions(ctx, field.ID),
			},
		},
	}
}

func workDataFieldOptions(ctx context.Context, fieldID uint64) []map[string]any {
	rows := crmmodel.NewDataFieldOptionModel().SelectMap(ctx, map[string]any{
		"data_field_id": fieldID,
	}, map[string]any{
		"order": "main.sort asc, main.id asc",
	})
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		value := inputText(row["value"])
		if value == "" {
			continue
		}
		name := inputText(row["name"])
		if name == "" {
			name = value
		}
		options = append(options, map[string]any{
			"id":    value,
			"name":  name,
			"value": value,
		})
	}
	return options
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

func workActionValues(payload map[string]any) map[string]any {
	values := mapFromAny(payload["values"])
	for _, key := range []string{"department_id", "departmentId", "staff_id", "staffId", "todo_id", "todoId", "collaboration_targets", "collaborationTargets", "submit_mode", "submitMode"} {
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

func workFormOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewFormModel().SelectMap(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	return namedWorkOptions(rows)
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

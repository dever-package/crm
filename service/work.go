package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	deverjwt "github.com/shemic/dever/auth/jwt"
	"github.com/shemic/dever/config"
	"github.com/shemic/dever/orm"

	agentmodel "my/package/bot/model/agent"
	agentservice "my/package/bot/service/agent"
	crmmodel "my/package/crm/model"
	frontservice "my/package/front/service"
	fronteval "my/package/front/service/eval"
	uploadrepo "my/package/front/service/upload/repository"
)

const (
	workSiteKey              = "work"
	workAuthProvider         = "crm_work"
	feishuAPIBase            = "https://open.feishu.cn/open-apis"
	feishuRequestTimeout     = 10 * time.Second
	workResultSuccess        = "success"
	workResultAutoFailed     = "auto_failed"
	workCustomerModeAll      = "all"
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
	staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{
		"feishu_open_id": openID,
		"status":         crmmodel.StatusEnabled,
	})
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

func (WorkService) AIFill(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
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
	formID, err := workAIFillFormID(ctx, staff, task, customerID, assetID, payload)
	if err != nil {
		return nil, err
	}
	fields := workAIFieldsForForm(ctx, formID)
	if len(fields) == 0 {
		return nil, fmt.Errorf("当前任务没有可 AI 填写的字段")
	}
	result, err := agentservice.NewService().RunInternal(ctx, agentservice.InternalRunRequest{
		AgentID: agentmodel.FrontAssistantAgentID,
		Input:   workAIFillInput(ctx, staff, task, formID, fields, customerID, assetID, payload),
		Options: map[string]any{
			"max_steps": 1,
		},
	})
	if err != nil {
		return nil, err
	}
	action := extractWorkAIFillFrontActionFromOutput(result.Output, result.Summary)
	values := sanitizeWorkAIFillValues(fields, extractWorkAIFillValues(result.Output, result.Summary, action))
	return map[string]any{
		"values":       values,
		"summary":      inputText(firstPresent(action, "summary", "text")),
		"request_id":   result.RequestID,
		"run_id":       result.RunID,
		"filled_count": len(values),
	}, nil
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
	case crmmodel.TaskTypeCollaborate:
		if customerID == 0 {
			return nil, fmt.Errorf("客户不能为空")
		}
		return executeCollaborateCustomerTask(ctx, staff, task, customerID, assetID, workActionValues(payload), runtime)
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

type workAIFillField struct {
	Key       string
	Name      string
	FieldType string
	Required  bool
	Options   []map[string]any
}

func workAIFillFormID(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, payload map[string]any) (uint64, error) {
	if task == nil {
		return 0, nil
	}
	todoID := firstUint64(payload, "todo_id", "todoId")
	if todoID == 0 {
		return task.FormID, nil
	}
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return 0, fmt.Errorf("协作待办不存在或已完成")
	}
	if todo.CustomerID != customerID || todo.AssetID != assetID {
		return 0, fmt.Errorf("协作待办不属于当前客户资产")
	}
	if !canOperateWorkTodo(staff, todo) {
		return 0, fmt.Errorf("当前人员无权完成该协作待办")
	}
	return todo.FormID, nil
}

func workAIFieldsForForm(ctx context.Context, formID uint64) []workAIFillField {
	if formID == 0 {
		return nil
	}
	form := crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
	if form == nil {
		return nil
	}
	rows := crmmodel.NewFormFieldModel().Select(ctx, map[string]any{"form_id": form.ID, "status": crmmodel.StatusEnabled})
	fields := make([]workAIFillField, 0, len(rows))
	for _, field := range rows {
		if field == nil || field.Readonly {
			continue
		}
		row := map[string]any{
			"main_field":    field.MainField,
			"data_field_id": field.DataFieldID,
		}
		attachWorkFormFieldOptions(ctx, row)
		fieldType := inputText(row["field_type"])
		if workAIFillFieldIsUpload(fieldType) {
			continue
		}
		key := workFieldInputKey(field)
		if key == "" {
			continue
		}
		fields = append(fields, workAIFillField{
			Key:       key,
			Name:      field.Name,
			FieldType: fieldType,
			Required:  field.Required,
			Options:   mapsFromAny(row["options"]),
		})
	}
	return fields
}

func workAIFillFieldIsUpload(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "attachment", "file", "image":
		return true
	default:
		return false
	}
}

func workAIFillInput(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, formID uint64, fields []workAIFillField, customerID uint64, assetID uint64, payload map[string]any) map[string]any {
	context := workAIFillContext(ctx, staff, task, formID, fields, customerID, assetID, payload)
	return map[string]any{
		"text": "请根据 CRM 工作台上下文补全当前任务表单。只输出 ```front-action fenced JSON，type 必须是 patch_form，values 只包含 fields 中存在的 key；不要保存、提交或执行任何业务动作。",
		"task": map[string]any{
			"type":                 "fill_current_form",
			"allowed_action_types": []string{"patch_form"},
			"empty_only":           true,
			"instruction":          "只补全空字段；已有值不覆盖；没有把握的字段不要填写；下拉字段必须返回选项 id；数字和金额只返回数字；日期返回 YYYY-MM-DD；时间返回 YYYY-MM-DD HH:mm:ss；布尔返回 true 或 false。",
		},
		"page_context": context,
	}
}

func workAIFillContext(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, formID uint64, fields []workAIFillField, customerID uint64, assetID uint64, payload map[string]any) map[string]any {
	customer := map[string]any{}
	if customerID > 0 {
		customer = crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
		customer["fields"] = workCustomerFieldValues(ctx, customerID)
	}
	return map[string]any{
		"surface": "crm_workbench_task_form",
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
			"form_id":   formID,
		},
		"fields":         workAIFillFieldPayloads(fields),
		"current_values": mapFromAny(payload["values"]),
		"customer":       customer,
		"assets":         workDecisionAssets(ctx, customerID),
		"current": map[string]any{
			"customer_id": customerID,
			"asset_id":    assetID,
			"stage_code":  workCurrentStageCode(currentWorkCustomerStage(ctx, customerID, assetID)),
		},
		"recent_operations": workAIFillRecentOperations(ctx, customerID, assetID, 10),
	}
}

func workAIFillFieldPayloads(fields []workAIFillField) []map[string]any {
	rows := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, map[string]any{
			"key":        field.Key,
			"name":       field.Name,
			"field_type": field.FieldType,
			"required":   field.Required,
			"options":    workAIFillOptionPayloads(field.Options),
		})
	}
	return rows
}

func workAIFillOptionPayloads(options []map[string]any) []map[string]any {
	rows := make([]map[string]any, 0, len(options))
	for _, option := range options {
		id := inputText(firstPresent(option, "id", "value"))
		name := firstText(option, "name", "label", "value")
		if id == "" || name == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"id":    id,
			"name":  name,
			"value": inputText(firstPresent(option, "value", "id")),
		})
	}
	return rows
}

func workAIFillRecentOperations(ctx context.Context, customerID uint64, assetID uint64, limit int) []map[string]any {
	if customerID == 0 || limit <= 0 {
		return nil
	}
	filter := map[string]any{"customer_id": customerID}
	if assetID > 0 {
		filter["asset_id"] = assetID
	}
	rows := crmmodel.NewOperationLogModel().SelectMap(ctx, filter)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"title":      inputText(row["title"]),
			"task_type":  inputText(row["task_type"]),
			"stage_code": inputText(row["stage_code"]),
			"result":     inputText(row["result_value"]),
			"data":       mapFromAny(row["data_snapshot_json"]),
			"created_at": inputText(row["created_at"]),
		})
	}
	return result
}

func extractWorkAIFillValues(output map[string]any, summary string, action map[string]any) map[string]any {
	for _, source := range []any{
		output["values"],
		mapFromAny(output["content"])["values"],
		action["values"],
	} {
		if values := mapFromAny(source); len(values) > 0 {
			return values
		}
	}
	return map[string]any{}
}

func extractWorkAIFillFrontActionFromOutput(output map[string]any, summary string) map[string]any {
	for _, text := range []string{inputText(output["text"]), summary} {
		body, ok := extractWorkJSONFence(text, "front-action")
		if !ok {
			continue
		}
		action := mapFromAny(body)
		if strings.TrimSpace(inputText(action["type"])) == "patch_form" {
			return action
		}
	}
	return map[string]any{}
}

func extractWorkJSONFence(text string, lang string) (string, bool) {
	search := "```" + lang
	start := strings.Index(text, search)
	if start < 0 {
		return "", false
	}
	bodyStart := start + len(search)
	for bodyStart < len(text) && (text[bodyStart] == ' ' || text[bodyStart] == '\t' || text[bodyStart] == '\r' || text[bodyStart] == '\n') {
		bodyStart++
	}
	end := strings.Index(text[bodyStart:], "```")
	if end < 0 {
		return strings.TrimSpace(text[bodyStart:]), true
	}
	return strings.TrimSpace(text[bodyStart : bodyStart+end]), true
}

func sanitizeWorkAIFillValues(fields []workAIFillField, raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	allowed := map[string]workAIFillField{}
	for _, field := range fields {
		allowed[field.Key] = field
	}
	result := map[string]any{}
	for key, value := range raw {
		field, ok := allowed[strings.TrimSpace(key)]
		if !ok || emptyWorkFieldValue(value) {
			continue
		}
		if normalized, ok := sanitizeWorkAIFillValue(field, value); ok {
			result[field.Key] = normalized
		}
	}
	return result
}

func sanitizeWorkAIFillValue(field workAIFillField, value any) (any, bool) {
	if len(field.Options) == 0 {
		if workAIFillFieldRequiresOptions(field.FieldType) {
			return nil, false
		}
		return sanitizeWorkAIFillScalarValue(field, value)
	}
	optionIDs := make([]string, 0)
	seen := map[string]bool{}
	for _, optionValue := range workAIFillOptionInputValues(value) {
		optionID, ok := matchWorkAIFillOption(field.Options, optionValue)
		if !ok || seen[optionID] {
			continue
		}
		optionIDs = append(optionIDs, optionID)
		seen[optionID] = true
	}
	if len(optionIDs) == 0 {
		return nil, false
	}
	if workAIFillFieldIsMultiple(field.FieldType) {
		return optionIDs, true
	}
	return optionIDs[0], true
}

func workAIFillFieldRequiresOptions(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "radio", "checkbox", "select", "multi_select", "multiple_select":
		return true
	default:
		return false
	}
}

func sanitizeWorkAIFillScalarValue(field workAIFillField, value any) (any, bool) {
	switch strings.ToLower(strings.TrimSpace(field.FieldType)) {
	case "number", "money":
		return sanitizeWorkAIFillNumber(value)
	case "date":
		return sanitizeWorkAIFillDate(value)
	case "datetime":
		return sanitizeWorkAIFillDateTime(value)
	case "boolean":
		return sanitizeWorkAIFillBoolean(value)
	default:
		text := inputText(value)
		if text == "" {
			return nil, false
		}
		return text, true
	}
}

func sanitizeWorkAIFillNumber(value any) (any, bool) {
	switch typed := value.(type) {
	case int:
		return strconv.FormatInt(int64(typed), 10), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case uint64:
		return strconv.FormatUint(typed, 10), true
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case json.Number:
		return sanitizeWorkAIFillNumberText(typed.String())
	default:
		return sanitizeWorkAIFillNumberText(inputText(value))
	}
}

func sanitizeWorkAIFillNumberText(text string) (any, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "，", "")
	text = strings.TrimSpace(strings.Trim(text, "￥$¥ "))
	if text == "" {
		return nil, false
	}
	if _, err := strconv.ParseFloat(text, 64); err != nil {
		return nil, false
	}
	return text, true
}

func sanitizeWorkAIFillDate(value any) (any, bool) {
	parsed, ok := parseWorkAIFillTime(value)
	if !ok {
		return nil, false
	}
	return parsed.Format("2006-01-02"), true
}

func sanitizeWorkAIFillDateTime(value any) (any, bool) {
	parsed, ok := parseWorkAIFillTime(value)
	if !ok {
		return nil, false
	}
	return parsed.Format("2006-01-02 15:04:05"), true
}

func parseWorkAIFillTime(value any) (time.Time, bool) {
	if typed, ok := value.(time.Time); ok && !typed.IsZero() {
		return typed, true
	}
	text := normalizeWorkAIFillTimeText(inputText(value))
	if text == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006-1-2 15:04",
		"2006-1-2 15:04:05",
		"2006-1-2",
	} {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func normalizeWorkAIFillTimeText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "/", "-")
	text = strings.ReplaceAll(text, "年", "-")
	text = strings.ReplaceAll(text, "月", "-")
	text = strings.ReplaceAll(text, "日", "")
	return strings.TrimSpace(text)
}

func sanitizeWorkAIFillBoolean(value any) (any, bool) {
	switch typed := value.(type) {
	case bool:
		return strconv.FormatBool(typed), true
	case int:
		return strconv.FormatBool(typed != 0), true
	case int64:
		return strconv.FormatBool(typed != 0), true
	case uint64:
		return strconv.FormatBool(typed != 0), true
	case float64:
		return strconv.FormatBool(typed != 0), true
	}
	switch strings.ToLower(inputText(value)) {
	case "true", "1", "yes", "y", "on", "是", "对", "通过", "成功":
		return "true", true
	case "false", "0", "no", "n", "off", "否", "不", "未通过", "失败":
		return "false", true
	default:
		return nil, false
	}
}

func workAIFillFieldIsMultiple(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "checkbox", "multi_select", "multiple_select":
		return true
	default:
		return false
	}
}

func workAIFillOptionInputValues(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		values := make([]any, 0, len(typed))
		for _, item := range typed {
			values = append(values, item)
		}
		return values
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		var arrayValues []any
		if strings.HasPrefix(text, "[") && json.Unmarshal([]byte(text), &arrayValues) == nil {
			return arrayValues
		}
		if strings.Contains(text, ",") || strings.Contains(text, "，") {
			parts := strings.FieldsFunc(text, func(r rune) bool {
				return r == ',' || r == '，'
			})
			values := make([]any, 0, len(parts))
			for _, part := range parts {
				if part = strings.TrimSpace(part); part != "" {
					values = append(values, part)
				}
			}
			return values
		}
		return []any{text}
	default:
		if emptyWorkFieldValue(value) {
			return nil
		}
		return []any{value}
	}
}

func matchWorkAIFillOption(options []map[string]any, value any) (string, bool) {
	text := inputText(value)
	if text == "" {
		return "", false
	}
	for _, option := range options {
		candidates := []string{
			inputText(option["id"]),
			inputText(option["value"]),
			inputText(option["name"]),
			inputText(option["label"]),
		}
		for _, candidate := range candidates {
			if candidate != "" && strings.EqualFold(candidate, text) {
				id := inputText(option["id"])
				if id == "" {
					id = inputText(option["value"])
				}
				return id, id != ""
			}
		}
	}
	return "", false
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
	form := workFormPayload(ctx, formID)
	if len(form) == 0 {
		return
	}
	task["form"] = form
}

func workFormPayload(ctx context.Context, formID uint64) map[string]any {
	if formID == 0 {
		return nil
	}
	form := crmmodel.NewFormModel().FindMap(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
	if len(form) == 0 {
		return nil
	}
	fields := crmmodel.NewFormFieldModel().SelectMap(ctx, map[string]any{
		"form_id": formID,
		"status":  crmmodel.StatusEnabled,
	})
	for _, field := range fields {
		attachWorkFormFieldOptions(ctx, field)
	}
	form["fields"] = fields
	return form
}

func attachWorkTaskConfig(ctx context.Context, task map[string]any) {
	switch inputText(task["task_type"]) {
	case crmmodel.TaskTypeAssign:
		config := mapFromAny(task["config_json"])
		assignMode := normalizeWorkAssignMode(inputText(config["assign_mode"]))
		task["assign_mode"] = assignMode
		task["assign_department_ids"] = uint64ListFromAny(config["assign_department_ids"])
	case crmmodel.TaskTypeCollaborate:
		config := mapFromAny(task["config_json"])
		items := mapsFromAny(config["collaboration_items"])
		attachWorkCollaborationItemForms(ctx, items)
		task["collaboration_items"] = items
		task["collaboration_complete_mode"] = normalizeWorkCollaborationCompleteMode(inputText(config["collaboration_complete_mode"]))
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

func attachWorkCollaborationItemForms(ctx context.Context, items []map[string]any) {
	for _, item := range items {
		form := workFormPayload(ctx, inputUint64(item["form_id"]))
		if len(form) == 0 {
			continue
		}
		item["form"] = form
		item["fields"] = form["fields"]
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

func executeCreateCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	formInput, err := collectWorkCreateFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	customerRecord := defaultWorkCustomerRecord(staff)
	for key, value := range formInput.customerFields {
		customerRecord[key] = value
	}
	if err := validateWorkCustomerContact(customerRecord); err != nil {
		return nil, err
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
	fromState := currentWorkCustomerStage(ctx, customerID, 0)
	transitionStageCode := applyWorkStageTransition(ctx, staff, customerID, 0, fromState, task, operationID, workResultSuccess)
	runWorkAutoTriggers(ctx, staff, customerID, 0, task, workResultSuccess, workEnteredStageCode(true, fromState, transitionStageCode), runtime)
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
	var createdAsset bool
	assetID, createdAsset, err = ensureWorkFormAsset(ctx, customerID, assetID, formInput)
	if err != nil {
		return nil, err
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	resultValue := workFormTaskResultValue(task, formInput)
	operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, values, resultValue)
	saveWorkFormDataRecords(ctx, customerID, assetID, task.ID, operationID, formInput)
	fromState = ensureCreatedWorkAssetStage(ctx, staff, customerID, assetID, operationID, task.ID, createdAsset, fromState)
	transitionStageCode := applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, resultValue, workEnteredStageCode(createdAsset, fromState, transitionStageCode), runtime)
	return map[string]any{
		"customer_id":  customerID,
		"asset_id":     assetID,
		"result_value": resultValue,
		"saved":        true,
	}, nil
}

func executeAssignCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
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
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	var createdAsset bool
	assetID, createdAsset, err = ensureWorkFormAsset(ctx, customerID, assetID, formInput)
	if err != nil {
		return nil, err
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	logValues := copyMap(values)
	logValues["department_id"] = targetDepartmentID
	logValues["staff_id"] = targetStaffID
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, assetID, logValues)
	saveWorkFormDataRecords(ctx, customerID, assetID, task.ID, operationID, formInput)
	fromState = ensureCreatedWorkAssetStage(ctx, staff, customerID, assetID, operationID, task.ID, createdAsset, fromState)
	updateWorkCustomerOwner(ctx, customerID, assetID, targetDepartmentID, targetStaffID, operationID)
	upsertWorkAssigneeMember(ctx, customerID, assetID, targetDepartmentID, targetStaffID)
	transitionStageCode := applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, workResultSuccess, targetDepartmentID, targetStaffID)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, workEnteredStageCode(createdAsset, fromState, transitionStageCode), runtime)
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"saved":       true,
	}, nil
}

func executeCollaborateCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	todoID := firstUint64(values, "todo_id", "todoId")
	if todoID > 0 {
		return completeWorkTodo(ctx, staff, task, customerID, assetID, todoID, values, runtime)
	}

	targets, err := workCollaborationTargets(ctx, task, values)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("请配置协作子任务")
	}
	formInput, err := collectOptionalWorkFormInput(ctx, task, values)
	if err != nil {
		return nil, err
	}
	collaborationFormInput, err := collectWorkCollaborationFormInput(ctx, targets, values)
	if err != nil {
		return nil, err
	}
	formInput = mergeWorkFormInput(formInput, collaborationFormInput)
	assetID, createdAsset, err := ensureWorkFormAsset(ctx, customerID, assetID, formInput)
	if err != nil {
		return nil, err
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	logValues := copyMap(values)
	logValues["todo_count"] = len(targets)
	operationID := insertWorkOperationLog(ctx, staff, task, customerID, assetID, logValues)
	saveWorkFormDataRecords(ctx, customerID, assetID, task.ID, operationID, formInput)
	fromState := ensureCreatedWorkAssetStage(ctx, staff, customerID, assetID, operationID, task.ID, createdAsset, nil)
	todos := createWorkCollaborationTodos(ctx, staff, task, customerID, assetID, operationID, targets)
	updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, workEnteredStageCode(createdAsset, fromState, ""), runtime)
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"todo_count":  len(todos),
		"saved":       true,
	}, nil
}

func executeDecisionCustomerTask(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	if crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) == nil {
		return nil, fmt.Errorf("客户不存在")
	}
	fromState := currentWorkCustomerStage(ctx, customerID, assetID)
	result, err := resolveWorkDecisionResult(ctx, staff, task, customerID, assetID, fromState, values)
	if err != nil {
		return nil, err
	}
	resultTarget, err := resolveWorkDecisionResultTarget(ctx, customerID, assetID, task, result.Value)
	if err != nil {
		return nil, err
	}
	logValues := map[string]any{"decision_result": result}
	operationID := insertWorkOperationLogWithResult(ctx, staff, task, customerID, assetID, fromState, logValues, result.Value)
	if err := writeWorkDecisionResult(ctx, customerID, task, operationID, resultTarget, result.Value); err != nil {
		return nil, err
	}
	transitionStageCode := applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, result.Value)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, result.Value, transitionStageCode, runtime)
	return map[string]any{
		"customer_id":  customerID,
		"result_value": result.Value,
		"reason":       result.Reason,
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
	transitionStageCode := applyWorkStageTransition(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue)
	runWorkAutoTriggers(ctx, staff, customerID, assetID, task, resultValue, transitionStageCode, runtime)
	return map[string]any{
		"customer_id":    customerID,
		"booking_id":     bookingID,
		"booking_status": bookingStatus,
		"result_value":   resultValue,
		"saved":          true,
	}, nil
}

type workDecisionResultTarget struct {
	field      *crmmodel.DataField
	writeAsset uint64
}

func resolveWorkDecisionResultTarget(ctx context.Context, customerID uint64, assetID uint64, task *crmmodel.Task, resultValue string) (workDecisionResultTarget, error) {
	fieldID := inputUint64(mapFromAny(task.ConfigJSON)["decision_result_field_id"])
	if fieldID == 0 {
		return workDecisionResultTarget{}, fmt.Errorf("决策任务未配置结果写入字段")
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
		"id":           fieldID,
		"stat_enabled": true,
		"status":       crmmodel.StatusEnabled,
	})
	if field == nil {
		return workDecisionResultTarget{}, fmt.Errorf("结果写入字段不存在、未开启条件字段或已停用")
	}
	if strings.TrimSpace(field.FieldKey) == "" {
		return workDecisionResultTarget{}, fmt.Errorf("结果写入字段必须配置字段编码")
	}
	if !workDecisionResultFieldHasOptions(field.FieldType) {
		return workDecisionResultTarget{}, fmt.Errorf("结果写入字段必须是单选或下拉字段")
	}
	if !workDecisionFieldOptionExists(ctx, field.ID, resultValue) {
		return workDecisionResultTarget{}, fmt.Errorf("自动决策结果 %s 不属于结果写入字段的可选项", resultValue)
	}
	writeAssetID := assetID
	templateCateID := workDataFieldTemplateCateID(ctx, field)
	if templateCateID == crmmodel.CustomerDataTemplateCateID {
		writeAssetID = 0
	} else if templateCateID == crmmodel.CustomerAssetDataTemplateCateID && writeAssetID == 0 {
		return workDecisionResultTarget{}, fmt.Errorf("资产级结果写入字段需要当前客户资产")
	}
	return workDecisionResultTarget{field: field, writeAsset: writeAssetID}, nil
}

func writeWorkDecisionResult(ctx context.Context, customerID uint64, task *crmmodel.Task, operationID uint64, target workDecisionResultTarget, resultValue string) error {
	if target.field == nil {
		return fmt.Errorf("决策任务未配置结果写入字段")
	}
	saveWorkDataRecord(ctx, customerID, target.writeAsset, target.field.DataTemplateID, task.ID, operationID, map[string]any{
		fmt.Sprintf("%d", target.field.ID): resultValue,
	})
	return nil
}

func workDecisionResultFieldHasOptions(fieldType string) bool {
	switch strings.TrimSpace(fieldType) {
	case "radio", "select":
		return true
	default:
		return false
	}
}

func workDecisionFieldOptionExists(ctx context.Context, fieldID uint64, value string) bool {
	if fieldID == 0 || strings.TrimSpace(value) == "" {
		return false
	}
	return crmmodel.NewDataFieldOptionModel().Find(ctx, map[string]any{
		"data_field_id": fieldID,
		"value":         strings.TrimSpace(value),
	}) != nil
}

func workDataFieldTemplateCateID(ctx context.Context, field *crmmodel.DataField) uint64 {
	if field == nil || field.DataTemplateID == 0 {
		return 0
	}
	template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": field.DataTemplateID})
	if template == nil {
		return 0
	}
	return template.CateID
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

type workCollaborationTodoTarget struct {
	Key          string
	Name         string
	DepartmentID uint64
	StaffID      uint64
	FormID       uint64
	Required     bool
	Sort         int
}

func createWorkCollaborationTodos(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, operationID uint64, targets []workCollaborationTodoTarget) []uint64 {
	now := time.Now()
	assignedAt := now
	if operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{"id": operationID}); operation != nil && !operation.CreatedAt.IsZero() {
		assignedAt = operation.CreatedAt
	}
	ids := make([]uint64, 0, len(targets))
	model := crmmodel.NewWorkTodoModel()
	for _, target := range targets {
		record := map[string]any{
			"customer_id":                customerID,
			"asset_id":                   assetID,
			"source_task_id":             task.ID,
			"parent_operation_log_id":    operationID,
			"sub_task_name":              target.Name,
			"form_id":                    target.FormID,
			"assignee_department_id":     target.DepartmentID,
			"assignee_staff_id":          target.StaffID,
			"required":                   target.Required,
			"sort":                       target.Sort,
			"status":                     crmmodel.WorkTodoStatusPending,
			"assigned_at":                assignedAt,
			"completed_operation_log_id": uint64(0),
			"created_by_staff_id":        staff.ID,
			"created_at":                 now,
			"updated_at":                 now,
		}
		id := uint64(model.Insert(ctx, record))
		ids = append(ids, id)
		upsertWorkTodoMember(ctx, customerID, assetID, target.DepartmentID, target.StaffID)
	}
	return ids
}

func workCollaborationTargets(ctx context.Context, task *crmmodel.Task, values map[string]any) ([]workCollaborationTodoTarget, error) {
	configTargets := configuredWorkCollaborationTargets(task)
	rawTargets := configTargets
	if len(rawTargets) > 0 {
		rawTargets = mergeSubmittedWorkCollaborationStaff(rawTargets, mapsFromAny(firstWorkValue(values, "collaboration_targets", "collaborationTargets")))
	} else {
		rawTargets = mapsFromAny(firstWorkValue(values, "collaboration_targets", "collaborationTargets"))
	}
	result := make([]workCollaborationTodoTarget, 0, len(rawTargets))
	seen := map[string]bool{}
	for _, raw := range rawTargets {
		if workCollaborationTargetIsBlank(raw) {
			continue
		}
		target, err := normalizeWorkCollaborationTarget(ctx, raw)
		if err != nil {
			return nil, err
		}
		if target.DepartmentID == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%d:%d", target.Name, target.DepartmentID, target.StaffID)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, target)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Sort != result[j].Sort {
			return result[i].Sort < result[j].Sort
		}
		if result[i].DepartmentID != result[j].DepartmentID {
			return result[i].DepartmentID < result[j].DepartmentID
		}
		return result[i].StaffID < result[j].StaffID
	})
	return result, nil
}

func configuredWorkCollaborationTargets(task *crmmodel.Task) []map[string]any {
	if task == nil {
		return nil
	}
	config := mapFromAny(task.ConfigJSON)
	return mapsFromAny(config["collaboration_items"])
}

func mergeSubmittedWorkCollaborationStaff(configTargets []map[string]any, submittedTargets []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(configTargets))
	submittedByKey := submittedWorkCollaborationTargetsByKey(submittedTargets)
	for index, configTarget := range configTargets {
		target := copyMap(configTarget)
		if inputUint64(firstWorkValue(configTarget, "staff_id", "assignee_staff_id")) == 0 {
			submittedTarget := submittedWorkCollaborationTarget(configTarget, submittedByKey, submittedTargets, index)
			submittedStaffID := inputUint64(firstWorkValue(submittedTarget, "staff_id", "assignee_staff_id"))
			if submittedStaffID > 0 {
				target["staff_id"] = submittedStaffID
			}
		}
		result = append(result, target)
	}
	return result
}

func submittedWorkCollaborationTargetsByKey(targets []map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any, len(targets))
	for _, target := range targets {
		key := inputText(firstWorkValue(target, "key", "target_key", "targetKey"))
		if key != "" {
			result[key] = target
		}
	}
	return result
}

func submittedWorkCollaborationTarget(configTarget map[string]any, submittedByKey map[string]map[string]any, submittedTargets []map[string]any, index int) map[string]any {
	key := inputText(firstWorkValue(configTarget, "key", "target_key", "targetKey"))
	if key != "" {
		if submittedTarget, ok := submittedByKey[key]; ok {
			return submittedTarget
		}
	}
	if index < len(submittedTargets) {
		return submittedTargets[index]
	}
	return nil
}

func workCollaborationTargetIsBlank(raw map[string]any) bool {
	return inputUint64(firstWorkValue(raw, "department_id", "assignee_department_id")) == 0 &&
		inputUint64(firstWorkValue(raw, "staff_id", "assignee_staff_id")) == 0
}

func normalizeWorkCollaborationTarget(ctx context.Context, raw map[string]any) (workCollaborationTodoTarget, error) {
	target := workCollaborationTodoTarget{
		Key:          inputText(firstWorkValue(raw, "key", "target_key", "targetKey")),
		Name:         inputText(firstWorkValue(raw, "name", "task_name", "sub_task_name")),
		DepartmentID: inputUint64(firstWorkValue(raw, "department_id", "assignee_department_id")),
		StaffID:      inputUint64(firstWorkValue(raw, "staff_id", "assignee_staff_id")),
		FormID:       inputUint64(firstWorkValue(raw, "form_id")),
		Required:     true,
		Sort:         int(inputUint64(firstWorkValue(raw, "sort"))),
	}
	if value, exists := raw["required"]; exists {
		target.Required = booleanFromAny(value)
	}
	if target.Name == "" {
		target.Name = "协作子任务"
	}
	if target.DepartmentID == 0 {
		return target, fmt.Errorf("协作子任务目标部门不能为空")
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": target.DepartmentID, "status": crmmodel.StatusEnabled})
	if department == nil {
		return target, fmt.Errorf("协作子任务目标部门不存在或已停用")
	}
	if target.StaffID == 0 {
		return target, fmt.Errorf("协作子任务处理人员不能为空")
	}
	staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": target.StaffID, "status": crmmodel.StatusEnabled})
	if staff == nil {
		return target, fmt.Errorf("协作子任务处理人员不存在或已停用")
	}
	if staff.DepartmentID != target.DepartmentID {
		return target, fmt.Errorf("协作子任务处理人员不属于目标部门")
	}
	if target.FormID > 0 && crmmodel.NewFormModel().Find(ctx, map[string]any{"id": target.FormID, "status": crmmodel.StatusEnabled}) == nil {
		return target, fmt.Errorf("协作子任务资料模板不存在或已停用")
	}
	return target, nil
}

func collectWorkCollaborationFormInput(ctx context.Context, targets []workCollaborationTodoTarget, values map[string]any) (*workFormInput, error) {
	result := emptyWorkFormInput()
	for _, target := range targets {
		if target.FormID == 0 {
			continue
		}
		formValues := workCollaborationTargetFormValues(values, target)
		if len(formValues) == 0 {
			continue
		}
		formInput, err := collectOptionalWorkFormInputForForm(ctx, target.FormID, formValues)
		if err != nil {
			return nil, err
		}
		result = mergeWorkFormInput(result, formInput)
	}
	return result, nil
}

func workCollaborationTargetFormValues(values map[string]any, target workCollaborationTodoTarget) map[string]any {
	prefix := workCollaborationTargetFormPrefix(target)
	if prefix == "" {
		return nil
	}
	result := map[string]any{}
	for key, value := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		fieldKey := strings.TrimPrefix(key, prefix)
		if fieldKey == "" {
			continue
		}
		result[fieldKey] = value
	}
	return result
}

func workCollaborationTargetFormPrefix(target workCollaborationTodoTarget) string {
	key := target.Key
	if key == "" {
		key = workCollaborationTargetFallbackKey(target)
	}
	if key == "" {
		return ""
	}
	return "collaboration_form:" + key + ":"
}

func workCollaborationTargetFallbackKey(target workCollaborationTodoTarget) string {
	if target.Name == "" && target.DepartmentID == 0 && target.FormID == 0 && target.Sort == 0 {
		return ""
	}
	return strings.Join([]string{
		strconv.Itoa(target.Sort),
		strconv.FormatUint(target.DepartmentID, 10),
		strconv.FormatUint(target.FormID, 10),
		target.Name,
	}, ":")
}

func firstWorkValue(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, exists := row[key]; exists {
			return value
		}
	}
	return nil
}

func completeWorkTodo(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, todoID uint64, values map[string]any, runtime *workExecutionRuntime) (map[string]any, error) {
	todo := crmmodel.NewWorkTodoModel().Find(ctx, map[string]any{
		"id":     todoID,
		"status": crmmodel.WorkTodoStatusPending,
	})
	if todo == nil {
		return nil, fmt.Errorf("协作待办不存在或已完成")
	}
	if todo.CustomerID != customerID || todo.AssetID != assetID {
		return nil, fmt.Errorf("协作待办不属于当前客户资产")
	}
	if !canOperateWorkTodo(staff, todo) {
		return nil, fmt.Errorf("当前人员无权完成该协作待办")
	}
	formInput := emptyWorkFormInput()
	if todo.FormID > 0 {
		todoFormInput, err := collectWorkFormInputForForm(ctx, todo.FormID, values)
		if err != nil {
			return nil, err
		}
		formInput = mergeWorkFormInput(formInput, todoFormInput)
	}
	if err := saveWorkFormInput(ctx, customerID, assetID, formInput); err != nil {
		return nil, err
	}
	logValues := copyMap(values)
	logValues["todo_id"] = todo.ID
	logValues["todo_name"] = todo.SubTaskName
	operationID := insertWorkOperationLogWithTitle(ctx, staff, task, customerID, assetID, logValues, todo.SubTaskName)
	saveWorkFormDataRecords(ctx, customerID, assetID, task.ID, operationID, formInput)
	now := time.Now()
	crmmodel.NewWorkTodoModel().Update(ctx, map[string]any{"id": todo.ID}, map[string]any{
		"status":                     crmmodel.WorkTodoStatusDone,
		"completed_at":               now,
		"completed_operation_log_id": operationID,
		"updated_at":                 now,
	})
	if workCollaborationShouldFlow(ctx, task, todo) {
		fromState := currentWorkCustomerStage(ctx, customerID, assetID)
		transitionStageCode := applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, workResultSuccess, 0, 0)
		runWorkAutoTriggers(ctx, staff, customerID, assetID, task, workResultSuccess, transitionStageCode, runtime)
	}
	return map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"todo_id":     todo.ID,
		"saved":       true,
	}, nil
}

func canOperateWorkTodo(staff *WorkStaffSession, todo *crmmodel.WorkTodo) bool {
	if staff == nil || todo == nil {
		return false
	}
	if todo.AssigneeStaffID > 0 {
		return todo.AssigneeStaffID == staff.ID
	}
	return todo.AssigneeDepartmentID > 0 && todo.AssigneeDepartmentID == staff.DepartmentID
}

func workCollaborationShouldFlow(ctx context.Context, task *crmmodel.Task, todo *crmmodel.WorkTodo) bool {
	if task == nil || todo == nil || todo.ParentOperationLogID == 0 || workCollaborationAlreadyFlowed(ctx, task, todo.ParentOperationLogID) {
		return false
	}
	mode := normalizeWorkCollaborationCompleteMode(inputText(mapFromAny(task.ConfigJSON)["collaboration_complete_mode"]))
	switch mode {
	case crmmodel.CollaborationCompleteManual:
		return false
	case crmmodel.CollaborationCompleteAny:
		return workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{"status": crmmodel.WorkTodoStatusDone}) == 1
	default:
		requiredTotal := workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{"required": true})
		if requiredTotal == 0 {
			return workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{"status": crmmodel.WorkTodoStatusDone}) == 1
		}
		if !todo.Required {
			return false
		}
		return workTodoCount(ctx, todo.ParentOperationLogID, map[string]any{
			"required": true,
			"status":   crmmodel.WorkTodoStatusPending,
		}) == 0
	}
}

func workTodoCount(ctx context.Context, parentOperationID uint64, filter map[string]any) int64 {
	query := map[string]any{"parent_operation_log_id": parentOperationID}
	for key, value := range filter {
		query[key] = value
	}
	return crmmodel.NewWorkTodoModel().Count(ctx, query)
}

func workCollaborationAlreadyFlowed(ctx context.Context, task *crmmodel.Task, parentOperationID uint64) bool {
	if task == nil || parentOperationID == 0 {
		return false
	}
	if crmmodel.NewStageTransitionLogModel().Find(ctx, map[string]any{
		"task_id":          task.ID,
		"operation_log_id": parentOperationID,
	}) != nil {
		return true
	}
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"parent_operation_log_id": parentOperationID,
	}) {
		if todo == nil || todo.CompletedOperationLogID == 0 {
			continue
		}
		if crmmodel.NewStageTransitionLogModel().Find(ctx, map[string]any{
			"task_id":          task.ID,
			"operation_log_id": todo.CompletedOperationLogID,
		}) != nil {
			return true
		}
	}
	return false
}

func normalizeWorkCollaborationCompleteMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case crmmodel.CollaborationCompleteAny:
		return crmmodel.CollaborationCompleteAny
	case crmmodel.CollaborationCompleteManual:
		return crmmodel.CollaborationCompleteManual
	default:
		return crmmodel.CollaborationCompleteAll
	}
}

func upsertWorkTodoMember(ctx context.Context, customerID uint64, assetID uint64, departmentID uint64, staffID uint64) {
	if customerID == 0 || (departmentID == 0 && staffID == 0) {
		return
	}
	model := crmmodel.NewCustomerMemberModel()
	filter := map[string]any{
		"customer_id":   customerID,
		"asset_id":      assetID,
		"department_id": departmentID,
		"staff_id":      staffID,
		"relation_type": crmmodel.MemberRelationParticipant,
		"status":        crmmodel.StatusEnabled,
	}
	if existing := model.Find(ctx, filter); existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, map[string]any{"can_view": true})
		return
	}
	record := copyMap(filter)
	record["can_view"] = true
	record["created_at"] = time.Now()
	model.Insert(ctx, record)
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

type workFormInputOptions struct {
	allowEmptyCustomerContactFields bool
	optionalValuesOnly              bool
}

func collectWorkFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不能为空")
	}
	return collectWorkFormInputByFormID(ctx, task.FormID, values)
}

func collectWorkCreateFormInput(ctx context.Context, task *crmmodel.Task, values map[string]any) (*workFormInput, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不能为空")
	}
	return collectWorkFormInputByFormID(ctx, task.FormID, values, workFormInputOptions{
		allowEmptyCustomerContactFields: true,
	})
}

func collectWorkFormInputByFormID(ctx context.Context, formID uint64, values map[string]any, options ...workFormInputOptions) (*workFormInput, error) {
	if formID == 0 {
		return nil, fmt.Errorf("任务未配置有效资料模板")
	}
	form := crmmodel.NewFormModel().Find(ctx, map[string]any{"id": formID, "status": crmmodel.StatusEnabled})
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
	inputOptions := workFormInputOptions{}
	if len(options) > 0 {
		inputOptions = options[0]
	}
	for _, field := range fields {
		if field == nil {
			continue
		}
		key := workFieldInputKey(field)
		value, exists := values[key]
		if inputOptions.optionalValuesOnly && (!exists || emptyWorkFieldValue(value)) {
			continue
		}
		if field.Required && emptyWorkFieldValue(value) {
			if !inputOptions.allowEmptyCustomerContactFields || !isWorkCustomerContactField(field) {
				return nil, fmt.Errorf("%s不能为空", field.Name)
			}
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

func collectWorkFormInputForForm(ctx context.Context, formID uint64, values map[string]any) (*workFormInput, error) {
	return collectWorkFormInputByFormID(ctx, formID, values)
}

func collectOptionalWorkFormInputForForm(ctx context.Context, formID uint64, values map[string]any) (*workFormInput, error) {
	return collectWorkFormInputByFormID(ctx, formID, values, workFormInputOptions{
		optionalValuesOnly: true,
	})
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

func mergeWorkFormInput(target *workFormInput, source *workFormInput) *workFormInput {
	if target == nil {
		target = emptyWorkFormInput()
	}
	if source == nil {
		return target
	}
	for key, value := range source.customerFields {
		target.customerFields[key] = value
	}
	for key, value := range source.assetFields {
		target.assetFields[key] = value
	}
	mergeWorkFormRecordMap(target.customerDataRecords, source.customerDataRecords)
	mergeWorkFormRecordMap(target.assetDataRecords, source.assetDataRecords)
	return target
}

func mergeWorkFormRecordMap(target map[uint64]map[string]any, source map[uint64]map[string]any) {
	for templateID, record := range source {
		if target[templateID] == nil {
			target[templateID] = map[string]any{}
		}
		for key, value := range record {
			target[templateID][key] = value
		}
	}
}

func workFormTaskResultValue(task *crmmodel.Task, formInput *workFormInput) string {
	config := mapFromAny(task.ConfigJSON)
	fieldID := inputUint64(config["result_field_id"])
	if fieldID == 0 {
		return workResultSuccess
	}
	actual := workFormInputDataFieldValue(formInput, fieldID)
	if emptyWorkFieldValue(actual) {
		return "empty"
	}
	if workFormResultAllowed(actual, config["success_values"]) {
		return workResultSuccess
	}
	return inputText(actual)
}

func workFormInputDataFieldValue(formInput *workFormInput, fieldID uint64) any {
	if formInput == nil || fieldID == 0 {
		return nil
	}
	fieldKey := fmt.Sprintf("%d", fieldID)
	for _, records := range []map[uint64]map[string]any{formInput.customerDataRecords, formInput.assetDataRecords} {
		for _, record := range records {
			if value, exists := record[fieldKey]; exists {
				return value
			}
		}
	}
	return nil
}

func workFormResultAllowed(actual any, expected any) bool {
	allowed := stringListFromAny(expected)
	if len(allowed) == 0 {
		return inputText(actual) == workResultSuccess
	}
	for _, value := range allowed {
		if valuesEqual(actual, value) {
			return true
		}
	}
	return false
}

func formInputHasAssetValues(formInput *workFormInput) bool {
	return formInput != nil && (len(formInput.assetFields) > 0 || len(formInput.assetDataRecords) > 0)
}

func ensureWorkFormAsset(ctx context.Context, customerID uint64, assetID uint64, formInput *workFormInput) (uint64, bool, error) {
	if assetID > 0 || !formInputHasAssetValues(formInput) {
		return assetID, false, nil
	}
	createdAssetID, err := createWorkCustomerAsset(ctx, customerID, formInput)
	if err != nil {
		return 0, false, err
	}
	return createdAssetID, true, nil
}

func ensureCreatedWorkAssetStage(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, operationID uint64, taskID uint64, createdAsset bool, fallback *crmmodel.CustomerStage) *crmmodel.CustomerStage {
	if !createdAsset {
		return fallback
	}
	insertWorkCustomerStage(ctx, staff, customerID, assetID, operationID, taskID)
	if state := currentWorkCustomerStage(ctx, customerID, assetID); state != nil {
		return state
	}
	return fallback
}

func workEnteredStageCode(createdStage bool, state *crmmodel.CustomerStage, transitionStageCode string) string {
	if transitionStageCode != "" {
		return transitionStageCode
	}
	if createdStage && state != nil {
		return state.CurrentStageCode
	}
	return ""
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

func validateWorkCustomerContact(record map[string]any) error {
	if inputText(record["phone"]) != "" || inputText(record["wechat"]) != "" {
		return nil
	}
	return fmt.Errorf("手机号和微信号至少填写一个")
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
	case "asset_name", "asset_status_id":
		return true
	default:
		return false
	}
}

func isWorkAssetTemplateField(field *crmmodel.FormField) bool {
	return field != nil && field.DataTemplateCateID == crmmodel.CustomerAssetDataTemplateCateID
}

func isWorkCustomerContactField(field *crmmodel.FormField) bool {
	if field == nil || isWorkAssetTemplateField(field) {
		return false
	}
	switch field.MainField {
	case "phone", "wechat":
		return true
	default:
		return false
	}
}

func runWorkAutoTriggers(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, sourceTask *crmmodel.Task, resultValue string, enteredStageCode string, runtime *workExecutionRuntime) {
	if staff == nil || sourceTask == nil || customerID == 0 || runtime == nil || runtime.depth >= maxWorkAutoTriggerDepth {
		return
	}
	for _, task := range workAfterTaskTriggers(ctx, sourceTask.ID) {
		executeAutoWorkTask(ctx, staff, customerID, assetID, task, runtime)
	}
	if enteredStageCode == "" {
		return
	}
	for _, task := range workStageEnterTriggers(ctx, enteredStageCode) {
		executeAutoWorkTask(ctx, staff, customerID, assetID, task, runtime)
	}
}

func executeAutoWorkTask(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, task *crmmodel.Task, runtime *workExecutionRuntime) {
	if task == nil || !crmmodel.TaskTypeSupportsAutoTrigger(task.TaskType) {
		return
	}
	if workAutoTaskAlreadySucceeded(ctx, customerID, assetID, task.ID) {
		return
	}
	if !beginWorkTaskExecution(runtime, customerID, assetID, task.ID) {
		return
	}
	defer endWorkTaskExecution(runtime, customerID, assetID, task.ID)
	if _, err := executeAutoWorkTaskByType(ctx, staff, customerID, assetID, task, runtime); err != nil {
		insertWorkAutoTaskFailureLog(ctx, staff, task, customerID, assetID, err)
	}
}

func workAutoTaskAlreadySucceeded(ctx context.Context, customerID uint64, assetID uint64, taskID uint64) bool {
	if customerID == 0 || taskID == 0 {
		return false
	}
	operation := crmmodel.NewOperationLogModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"asset_id":    assetID,
		"task_id":     taskID,
	})
	return operation != nil && operation.ResultValue != workResultAutoFailed
}

func executeAutoWorkTaskByType(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, task *crmmodel.Task, runtime *workExecutionRuntime) (map[string]any, error) {
	switch task.TaskType {
	case crmmodel.TaskTypeDecision:
		if task.ScriptID == 0 {
			return nil, fmt.Errorf("自动决策任务必须配置脚本规则")
		}
		return executeDecisionCustomerTask(ctx, staff, task, customerID, assetID, map[string]any{}, runtime)
	case crmmodel.TaskTypeAssign:
		values, err := workAutoAssignValues(ctx, task)
		if err != nil {
			return nil, err
		}
		return executeAssignCustomerTask(ctx, staff, task, customerID, assetID, values, runtime)
	case crmmodel.TaskTypeCollaborate:
		return executeCollaborateCustomerTask(ctx, staff, task, customerID, assetID, map[string]any{}, runtime)
	default:
		return nil, fmt.Errorf("任务动作不支持自动触发")
	}
}

func workAutoAssignValues(ctx context.Context, task *crmmodel.Task) (map[string]any, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不存在")
	}
	config := mapFromAny(task.ConfigJSON)
	departmentID := inputUint64(config["auto_assign_department_id"])
	staffID := inputUint64(config["auto_assign_staff_id"])
	if normalizeWorkAssignMode(inputText(config["assign_mode"])) == crmmodel.TaskAssignModeDepartment {
		staffID = 0
	}
	if departmentID == 0 {
		return nil, fmt.Errorf("自动分配任务必须配置自动分配部门")
	}
	department := crmmodel.NewDepartmentModel().Find(ctx, map[string]any{"id": departmentID, "status": crmmodel.StatusEnabled})
	if department == nil {
		return nil, fmt.Errorf("自动分配部门不存在或已停用")
	}
	allowedDepartmentIDs := uint64ListFromAny(config["assign_department_ids"])
	if len(allowedDepartmentIDs) > 0 && !uint64SetContains(allowedDepartmentIDs, departmentID) {
		return nil, fmt.Errorf("自动分配部门不在当前任务可选范围内")
	}
	if staffID > 0 {
		staff := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staffID, "status": crmmodel.StatusEnabled})
		if staff == nil {
			return nil, fmt.Errorf("自动分配人员不存在或已停用")
		}
		if staff.DepartmentID != departmentID {
			return nil, fmt.Errorf("自动分配人员不属于自动分配部门")
		}
	}
	return map[string]any{
		"department_id": departmentID,
		"staff_id":      staffID,
	}, nil
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

func workStageEnterTriggers(ctx context.Context, stageCode string) []*crmmodel.Task {
	if stageCode == "" {
		return nil
	}
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
		"code":   stageCode,
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

type workDecisionResult struct {
	Value      string `json:"value"`
	Reason     string `json:"reason,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

func resolveWorkDecisionResult(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any) (workDecisionResult, error) {
	if task != nil && task.TriggerType == crmmodel.TaskTriggerManual {
		return resolveManualWorkDecisionResult(values)
	}
	if task == nil || task.ScriptID == 0 {
		return workDecisionResult{}, fmt.Errorf("自动决策任务必须配置脚本规则")
	}
	return executeWorkDecisionScript(ctx, staff, task, customerID, assetID, state)
}

func resolveManualWorkDecisionResult(values map[string]any) (workDecisionResult, error) {
	resultValue := firstText(values, "decision_result", "result_value", "value")
	if resultValue == "" {
		return workDecisionResult{}, fmt.Errorf("请选择决策结果")
	}
	return workDecisionResult{
		Value:  resultValue,
		Reason: firstText(values, "decision_reason", "reason"),
	}, nil
}

func executeWorkDecisionScript(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage) (workDecisionResult, error) {
	script := crmmodel.NewRuleScriptModel().Find(ctx, map[string]any{"id": task.ScriptID, "status": crmmodel.StatusEnabled})
	if script == nil {
		return workDecisionResult{}, fmt.Errorf("自动决策脚本不存在或已停用")
	}
	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: fronteval.LanguageJavaScript,
		Script:   script.Script,
		Entry:    fronteval.DefaultEntry,
		Input:    workDecisionInput(ctx, staff, task, customerID, assetID, state),
		Config:   mapFromAny(task.ConfigJSON),
	})
	if err != nil {
		return workDecisionResult{}, err
	}
	return normalizeWorkDecisionResult(result.Value, result.DurationMS)
}

func normalizeWorkDecisionResult(value any, durationMS int64) (workDecisionResult, error) {
	payload := mapFromAny(value)
	resultValue := inputText(payload["value"])
	if resultValue == "" {
		return workDecisionResult{}, fmt.Errorf("自动决策脚本必须返回 { value: \"T几\" }")
	}
	return workDecisionResult{
		Value:      resultValue,
		Reason:     inputText(payload["reason"]),
		DurationMS: durationMS,
	}, nil
}

func workDecisionInput(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage) map[string]any {
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	customer["fields"] = workCustomerFieldValues(ctx, customerID)
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
		"customer": customer,
		"assets":   workDecisionAssets(ctx, customerID),
		"current": map[string]any{
			"stage_code": workCurrentStageCode(state),
			"asset_id":   assetID,
			"asset":      workDecisionCurrentAsset(ctx, customerID, assetID),
		},
	}
}

func workDecisionCurrentAsset(ctx context.Context, customerID uint64, assetID uint64) map[string]any {
	if customerID == 0 || assetID == 0 {
		return map[string]any{}
	}
	asset := crmmodel.NewCustomerAssetModel().FindMap(ctx, map[string]any{
		"id":          assetID,
		"customer_id": customerID,
	})
	if len(asset) == 0 {
		return map[string]any{}
	}
	asset["fields"] = workAssetFieldValues(ctx, customerID, assetID)
	return asset
}

func workCurrentStageCode(state *crmmodel.CustomerStage) string {
	if state == nil {
		return ""
	}
	return state.CurrentStageCode
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

func insertWorkOperationLogWithTitle(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, values map[string]any, title string) uint64 {
	if title == "" || task == nil {
		return insertWorkOperationLog(ctx, staff, task, customerID, assetID, values)
	}
	return insertWorkOperationLogRecord(ctx, staff, task, customerID, assetID, currentWorkCustomerStage(ctx, customerID, assetID), values, workResultSuccess, title)
}

func insertWorkOperationLogWithResult(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any, resultValue string) uint64 {
	title := ""
	if task != nil {
		title = task.Name
	}
	return insertWorkOperationLogRecord(ctx, staff, task, customerID, assetID, state, values, resultValue, title)
}

func insertWorkOperationLogRecord(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, state *crmmodel.CustomerStage, values map[string]any, resultValue string, title string) uint64 {
	now := time.Now()
	statusCode := ""
	if state != nil {
		statusCode = state.CurrentStageCode
	}
	if resultValue == "" {
		resultValue = workResultSuccess
	}
	if task == nil {
		return 0
	}
	if title == "" {
		title = task.Name
	}
	operationID := uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":            customerID,
		"asset_id":               assetID,
		"task_id":                task.ID,
		"task_type":              task.TaskType,
		"stage_code":             statusCode,
		"result_value":           resultValue,
		"title":                  title,
		"content":                "",
		"data_snapshot_json":     jsonText(values),
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"created_at":             now,
	}))
	syncWorkTaskStatEvent(ctx, staff, task, customerID, assetID, statusCode, operationID, resultValue, now)
	return operationID
}

func insertWorkAutoTaskFailureLog(ctx context.Context, staff *WorkStaffSession, task *crmmodel.Task, customerID uint64, assetID uint64, cause error) uint64 {
	if staff == nil || task == nil || cause == nil {
		return 0
	}
	state := currentWorkCustomerStage(ctx, customerID, assetID)
	stageCode := ""
	if state != nil {
		stageCode = state.CurrentStageCode
	}
	now := time.Now()
	message := cause.Error()
	return uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":  customerID,
		"asset_id":     assetID,
		"task_id":      task.ID,
		"task_type":    task.TaskType,
		"stage_code":   stageCode,
		"result_value": workResultAutoFailed,
		"title":        fmt.Sprintf("自动任务失败：%s", task.Name),
		"content":      message,
		"data_snapshot_json": jsonText(map[string]any{
			"error":        message,
			"task_type":    task.TaskType,
			"trigger_type": task.TriggerType,
			"script_id":    task.ScriptID,
		}),
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"created_at":             now,
	}))
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

func applyWorkStageTransition(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, operationID uint64, resultValue string) string {
	return applyWorkStageTransitionWithOwner(ctx, staff, customerID, assetID, fromState, task, operationID, resultValue, 0, 0)
}

func applyWorkStageTransitionWithOwner(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, fromState *crmmodel.CustomerStage, task *crmmodel.Task, operationID uint64, resultValue string, assignedDepartmentID uint64, assignedStaffID uint64) string {
	if fromState == nil || task == nil {
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return ""
	}
	transition := findWorkStageTransition(ctx, staff, customerID, fromState, task, resultValue)
	if transition == nil {
		updateWorkCustomerStageOperation(ctx, customerID, assetID, operationID)
		return ""
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
	if transition.ToStageCode == fromState.CurrentStageCode {
		return ""
	}
	return transition.ToStageCode
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
	if len(transitions) > 0 || workTaskResultRequiresExplicitTransition(task, resultValue) {
		return nil
	}
	return defaultWorkTaskTransition(ctx, fromState, task)
}

func workTaskResultRequiresExplicitTransition(task *crmmodel.Task, resultValue string) bool {
	if task == nil {
		return false
	}
	if task.TaskType == crmmodel.TaskTypeDecision {
		return true
	}
	config := mapFromAny(task.ConfigJSON)
	return inputUint64(config["result_field_id"]) > 0 && resultValue != workResultSuccess
}

func defaultWorkTaskTransition(ctx context.Context, fromState *crmmodel.CustomerStage, task *crmmodel.Task) *crmmodel.StageTransition {
	if task.TaskType == crmmodel.TaskTypeDecision {
		return nil
	}
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
	input := workDecisionInput(ctx, staff, task, customerID, assetID, state)
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
	return workConditionValueMatches(actual, expected, operator)
}

func workConditionValueMatches(actual any, expected any, operator string) bool {
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
	result, err := fronteval.Run(ctx, fronteval.Request{
		Language: fronteval.LanguageJavaScript,
		Script:   script.Script,
		Entry:    fronteval.DefaultEntry,
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
	for _, key := range []string{"department_id", "departmentId", "staff_id", "staffId", "todo_id", "todoId", "collaboration_targets", "collaborationTargets"} {
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

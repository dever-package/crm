package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
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
	workSiteKey                    = "work"
	workAuthProvider               = "crm_work"
	feishuAPIBase                  = "https://open.feishu.cn/open-apis"
	feishuRequestTimeout           = 10 * time.Second
	workResultProgress             = "progress"
	workBusinessEventLeadCreated   = "lead_created"
	workBusinessEventLeadConverted = "lead_converted"
	workCustomerModeAll            = "all"
	workCustomerModePending        = "pending"
	workCustomerModeProcessed      = "processed"
	workCustomerModeDone           = "done"
	workSubmitModeComplete         = "complete"
	workSubmitModeProgress         = "progress"
	workCustomerDefaultPageSize    = 10
)

type WorkService struct{}

type WorkStaffSession struct {
	ID           uint64
	Name         string
	Phone        string
	FeishuOpenID string
	DepartmentID uint64
	StaffType    string
	CanDispatch  bool
	ViewAll      bool
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
		staff, err = bindWorkStaffFeishuIdentity(ctx, identity)
		if err != nil {
			return nil, err
		}
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

func bindWorkStaffFeishuIdentity(ctx context.Context, identity feishuIdentityResult) (*crmmodel.Staff, error) {
	openID := strings.TrimSpace(identity.OpenID)
	mobile := normalizeWorkFeishuMobile(identity.Mobile)
	if mobile == "" {
		return nil, fmt.Errorf("飞书未返回手机号，无法自动关联人员，请检查飞书应用手机号权限")
	}
	staff, err := findUniqueEnabledStaffByField(ctx, "phone", mobile, "手机号")
	if err != nil {
		return nil, err
	}
	if staff == nil {
		return nil, fmt.Errorf("未找到手机号为 %s 的启用人员，请管理员先完善人员资料", mobile)
	}
	if boundOpenID := strings.TrimSpace(staff.FeishuOpenID); boundOpenID != "" {
		if boundOpenID == openID {
			return staff, nil
		}
		return nil, fmt.Errorf("手机号为 %s 的人员已绑定其他飞书账号", mobile)
	}
	if crmmodel.NewStaffModel().Update(ctx, map[string]any{
		"id":             staff.ID,
		"feishu_open_id": "",
	}, map[string]any{"feishu_open_id": openID}) == 0 {
		current := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": staff.ID})
		if current == nil || strings.TrimSpace(current.FeishuOpenID) != openID {
			return nil, fmt.Errorf("飞书账号自动绑定失败，请稍后重试")
		}
		return current, nil
	}
	staff.FeishuOpenID = openID
	return staff, nil
}

func normalizeWorkFeishuMobile(mobile string) string {
	mobile = strings.TrimSpace(mobile)
	mobile = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(mobile)
	switch {
	case strings.HasPrefix(mobile, "+86"):
		mobile = strings.TrimPrefix(mobile, "+86")
	case strings.HasPrefix(mobile, "0086"):
		mobile = strings.TrimPrefix(mobile, "0086")
	case strings.HasPrefix(mobile, "86") && len(mobile) == 13:
		mobile = strings.TrimPrefix(mobile, "86")
	}
	return mobile
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
	return map[string]any{"user": workStaffSessionPayload(staff)}, nil
}

func (WorkService) Customers(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	if workflowID := firstUint64(payload, "workflow_id", "workflowId"); workflowID > 0 {
		return workflowCustomerList(ctx, staff, workflowID, payload)
	}
	scope := normalizeWorkScope(staff, firstText(payload, "scope"))
	staff = workStaffWithScope(staff, scope)
	mode := normalizeWorkCustomerMode(firstText(payload, "mode"))
	stageOptions := workStageOptions(ctx)
	if !hasWorkCustomerListFilter(payload) {
		snapshot := newWorkCustomerListSnapshot(ctx, staff)
		list, page, pageSize, total := paginatedWorkCustomersFromSnapshot(ctx, staff, mode, payload, snapshot)
		return map[string]any{
			"list":          list,
			"total":         total,
			"page":          page,
			"page_size":     pageSize,
			"mode_counts":   workCustomerModeCountsFromSnapshot(snapshot, mode, total),
			"stage_options": stageOptions,
			"scope":         scope,
			"can_dispatch":  staff.CanDispatch,
		}, nil
	}
	customers := workCustomersByMode(ctx, staff, mode)
	modeCounts := workCustomerModeCounts(ctx, staff, mode, customers)
	if hasWorkCustomerStructuredFilter(payload) {
		customers = filterWorkCustomersByFields(customers, payload)
	}
	keyword := firstText(payload, "keyword")
	if keyword != "" {
		customers = filterWorkCustomers(customers, keyword)
	}
	if hasWorkCustomerWorkFilter(payload) {
		customers = filterWorkCustomersByWorkFilters(customers, payload)
	}
	list, page, pageSize, total := paginateWorkCustomerRows(customers, payload)
	list = workCustomerListRows(list)
	return map[string]any{
		"list":          list,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"mode_counts":   modeCounts,
		"stage_options": stageOptions,
		"scope":         scope,
		"can_dispatch":  staff.CanDispatch,
	}, nil
}

func workStageOptions(ctx context.Context, workflowIDs ...uint64) []map[string]any {
	filters := map[string]any{"status": crmmodel.StatusEnabled}
	if len(workflowIDs) > 0 && workflowIDs[0] > 0 {
		filters["workflow_id"] = workflowIDs[0]
	}
	rows := crmmodel.NewStageModel().Select(ctx, filters)
	result := make([]map[string]any, 0, len(rows))
	for _, stage := range rows {
		if stage == nil || stage.ID == 0 || strings.TrimSpace(stage.Name) == "" {
			continue
		}
		value := fmt.Sprintf("%d", stage.ID)
		result = append(result, map[string]any{
			"id":    value,
			"value": stage.Name,
			"code":  value,
		})
	}
	return result
}

func paginateWorkCustomerRows(rows []map[string]any, payload map[string]any) ([]map[string]any, int, int, int) {
	total := len(rows)
	page, pageSize, start, end := workCustomerPageBounds(total, payload)
	if start >= total {
		return []map[string]any{}, page, pageSize, total
	}
	return rows[start:end], page, pageSize, total
}

func workCustomerPageBounds(total int, payload map[string]any) (int, int, int, int) {
	page := inputInt(payload["page"])
	if page <= 0 {
		page = 1
	}
	pageSize := inputInt(firstPresent(payload, "page_size", "pageSize", "limit"))
	if pageSize <= 0 {
		pageSize = workCustomerDefaultPageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return page, pageSize, start, end
}

func (WorkService) Summary(ctx context.Context, staff *WorkStaffSession) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	operationRows := workStaffOperationRows(ctx, staff.ID)
	summary := workSummarySnapshot(ctx, staff)
	recentOperations := workRecentOperationRows(operationRows, 8)
	enrichWorkOperationRows(ctx, staff, recentOperations)
	return map[string]any{
		"metrics":           workSummaryMetricRows(ctx, summary),
		"trend":             workSummaryTrendRows(ctx, staff, 14),
		"stage_breakdown":   workSummaryStageRows(ctx, summary.Targets),
		"task_breakdown":    workSummaryTaskRows(ctx, summary.Targets),
		"recent_operations": recentOperations,
		"generated_at":      time.Now(),
	}, nil
}

func (WorkService) CustomerDetail(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	viewStaff := staff
	if staff.CanDispatch {
		viewStaff = workStaffWithScope(staff, "all")
	}
	customerID := firstUint64(payload, "customer_id", "customerId")
	if customerID == 0 {
		return nil, fmt.Errorf("请选择客户")
	}
	if !canViewWorkCustomer(ctx, viewStaff, customerID) {
		return nil, fmt.Errorf("无权查看该客户")
	}
	customer := workCustomerRow(ctx, viewStaff, customerID)
	if len(customer) == 0 {
		return nil, fmt.Errorf("客户不存在")
	}
	sourceLead := crmmodel.NewLeadModel().Find(ctx, map[string]any{
		"customer_id": customerID,
		"status":      crmmodel.LeadStatusConverted,
	}, map[string]any{"order": "converted_at desc,id desc"})
	if sourceLead != nil {
		customer["source_lead"] = workLeadRow(ctx, sourceLead)
	}

	assetID := firstUint64(payload, "asset_id", "assetId")
	asset := map[string]any(nil)
	var detailInstance *crmmodel.WorkflowInstance
	if assetID > 0 {
		if !canViewWorkAsset(ctx, viewStaff, customerID, assetID) {
			return nil, fmt.Errorf("无权查看该资产")
		}
		asset = workCustomerRowAsset(customer, assetID)
		if len(asset) == 0 {
			return nil, fmt.Errorf("资产不存在或不可见")
		}
		if instanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId"); instanceID > 0 {
			detailInstance = crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": instanceID})
			if detailInstance == nil || detailInstance.CustomerID != customerID || detailInstance.AssetID != assetID {
				return nil, fmt.Errorf("流程实例不存在或不属于该资产")
			}
			if !canViewWorkflowInstance(ctx, viewStaff, detailInstance) {
				return nil, fmt.Errorf("无权查看该流程记录")
			}
			attachWorkStageFields(ctx, asset, detailInstance)
		}
	}

	operationPayload := map[string]any{"customer_id": customerID}
	if assetID > 0 {
		operationPayload["asset_id"] = assetID
	}
	operations, err := (WorkService{}).Operations(ctx, viewStaff, operationPayload)
	if err != nil {
		return nil, err
	}
	detailSections := []map[string]any{}
	if sourceLead != nil {
		detailSections = append(detailSections, workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetLead,
			crmmodel.LeadDataTemplateCateID,
			workLeadDataValues(sourceLead),
		)...)
	}
	detailSections = append(detailSections, workDataDetailSections(
		ctx,
		crmmodel.DataTemplateTargetCustomer,
		crmmodel.CustomerDataTemplateCateID,
		mapFromAny(customer["data_values"]),
	)...)
	if len(asset) > 0 {
		detailSections = append(detailSections, workDataDetailSections(
			ctx,
			crmmodel.DataTemplateTargetCustomerAsset,
			crmmodel.CustomerAssetDataTemplateCateID,
			mapFromAny(asset["data_values"]),
		)...)
	}
	result := map[string]any{
		"customer":           customer,
		"asset":              asset,
		"operations":         operations["list"],
		"todos":              operations["todos"],
		"detail_sections":    detailSections,
		"customer_products":  []map[string]any{},
		"workflow_instances": []map[string]any{},
	}
	if assetID > 0 {
		customerProducts := mapListFromAny(asset["customer_products"])
		result["customer_products"] = customerProducts
		workflowInstances := make([]map[string]any, 0, len(customerProducts)+1)
		if detailInstance != nil {
			detailFlow := workFlowDetail(ctx, staff, detailInstance.ID)
			if detailInstance.CustomerProductID > 0 {
				detailFlow["flow_role"] = "product"
			} else {
				detailFlow["flow_role"] = "entry"
			}
			result["flow"] = detailFlow
		}
		if instance := currentWorkEntryInstance(ctx, customerID, assetID); instance != nil {
			entryFlow := workFlowDetail(ctx, staff, instance.ID)
			entryFlow["flow_role"] = "entry"
			if detailInstance == nil {
				result["flow"] = entryFlow
			}
			workflowInstances = append(workflowInstances, entryFlow)
		} else if detailInstance == nil {
			result["flow"] = map[string]any{
				"customer_id": customerID,
				"asset_id":    assetID,
				"status":      "not_started",
				"tasks":       []map[string]any{},
			}
		}
		for _, customerProduct := range customerProducts {
			flow := mapFromAny(customerProduct["flow"])
			if len(flow) == 0 {
				continue
			}
			flow["flow_role"] = "product"
			flow["product_id"] = customerProduct["product_id"]
			flow["product_name"] = customerProduct["product_name"]
			flow["customer_product_id"] = customerProduct["customer_product_id"]
			workflowInstances = append(workflowInstances, flow)
		}
		result["workflow_instances"] = workflowInstances
	}
	return result, nil
}

func (WorkService) Operations(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	filter := map[string]any{}
	customerID := firstUint64(payload, "customer_id", "customerId")
	workflowInstanceID := firstUint64(payload, "workflow_instance_id", "workflowInstanceId")
	var workflowInstance *crmmodel.WorkflowInstance
	if customerID > 0 {
		if !canViewWorkCustomer(ctx, staff, customerID) {
			return nil, fmt.Errorf("无权查看该客户")
		}
		filter["customer_id"] = customerID
	}
	if workflowInstanceID > 0 {
		workflowInstance = crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": workflowInstanceID})
		if workflowInstance == nil || !canViewWorkflowInstance(ctx, staff, workflowInstance) {
			return nil, fmt.Errorf("无权查看该流程记录")
		}
		filter["workflow_instance_id"] = workflowInstanceID
	}
	if customerID == 0 && workflowInstanceID == 0 {
		filter["operator_staff_id"] = staff.ID
	}
	if assetID := firstUint64(payload, "asset_id", "assetId"); assetID > 0 {
		filter["asset_id"] = assetID
	}
	mineOnly := booleanFromAny(payload["mine"])
	if mineOnly && (workflowInstance == nil || workflowInstance.LeadID == 0) {
		filter["operator_staff_id"] = staff.ID
	}
	rows := crmmodel.NewOperationLogModel().SelectMap(ctx, filter)
	rows = workBusinessOperationRows(ctx, rows)
	if mineOnly && workflowInstance != nil && workflowInstance.LeadID > 0 {
		rows = workOperationRowsByOperator(rows, staff.ID)
	}
	if keyword := firstText(payload, "keyword"); keyword != "" {
		rows = filterWorkOperations(rows, keyword)
	}
	enrichWorkOperationRows(ctx, staff, rows)
	todos := []map[string]any{}
	if customerID > 0 {
		todos = workTodoRows(ctx, staff, customerID, firstUint64(payload, "asset_id", "assetId"))
	}
	return map[string]any{
		"list":  rows,
		"total": len(rows),
		"todos": todos,
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
	if customerID > 0 && !canViewWorkCustomer(ctx, staff, customerID) {
		return nil, fmt.Errorf("无权查看该客户")
	}
	if currentAssetID > 0 && !canViewWorkAsset(ctx, staff, customerID, currentAssetID) {
		return nil, fmt.Errorf("无权查看该资产")
	}
	tasks := queryWorkTodoTasks(ctx, staff, customerID, currentAssetID, true)
	result := map[string]any{
		"list":  tasks,
		"total": len(tasks),
	}
	if progress := currentWorkTargetInstance(ctx, staff, customerID, currentAssetID); progress != nil {
		result["workflow_instance_id"] = progress.ID
		result["workflow_id"] = progress.WorkflowID
		result["stage_id"] = progress.StageID
		result["progress_status"] = progress.Status
		if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": progress.WorkflowID}); workflow != nil {
			result["workflow_name"] = workflow.Name
		}
		if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": progress.StageID}); stage != nil {
			result["stage_name"] = stage.Name
		}
	}
	return result, nil
}

func (WorkService) Execute(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	var result map[string]any
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		var executeErr error
		result, executeErr = executeWorkTask(txCtx, staff, payload)
		if executeErr != nil {
			return executeErr
		}
		if inputText(result["result_value"]) == workResultProgress {
			return nil
		}
		if booleanFromAny(result["kept_pending"]) {
			return nil
		}
		if booleanFromAny(result["workflow_terminated"]) {
			return nil
		}
		nextOwnerStaffID := firstUint64(payload, "next_owner_staff_id", "nextOwnerStaffId")
		advanced, err := advanceWorkflowAfterTaskCompletion(txCtx, staff, result, nextOwnerStaffID)
		if err != nil {
			return err
		}
		if advanced != nil {
			result["workflow_id"] = advanced.WorkflowID
			result["stage_id"] = advanced.StageID
			result["workflow_status"] = advanced.Status
			result["stage_advanced"] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func advanceWorkflowAfterTaskCompletion(
	ctx context.Context,
	staff *WorkStaffSession,
	result map[string]any,
	nextOwnerStaffID uint64,
) (*crmmodel.WorkflowInstance, error) {
	instance, err := activeWorkflowInstance(ctx, inputUint64(result["workflow_instance_id"]))
	if err != nil {
		return nil, err
	}
	if instance.StageID != inputUint64(result["stage_id"]) || pendingRequiredTodoCount(ctx, instance) > 0 {
		return nil, nil
	}
	if instance.LeadID > 0 {
		nextTarget, err := nextWorkflowAssignmentTarget(ctx, instance)
		if err != nil {
			return nil, err
		}
		if nextTarget.CrossObject {
			if nextOwnerStaffID == 0 && stageAssignmentMode(nextTarget.Stage) == crmmodel.StageAssignmentManual {
				return nil, fmt.Errorf("请选择%s阶段负责人", nextTarget.Stage.Name)
			}
			lead := crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": instance.LeadID})
			if lead == nil {
				return nil, fmt.Errorf("线索不存在")
			}
			converted, err := convertWorkLead(ctx, staff, lead, instance, instance.WorkflowID, nextOwnerStaffID)
			if err != nil {
				return nil, err
			}
			for key, value := range converted {
				result[key] = value
			}
			return workflowInstanceByID(ctx, instance.ID)
		}
	}
	return completeWorkflowStage(ctx, staff, instance, nextOwnerStaffID)
}

func CurrentWorkStaff(ctx context.Context) *WorkStaffSession {
	claims := deverjwt.Claims(ctx)
	if inputText(claims["site"]) != workSiteKey && inputText(claims["scope"]) != workAuthProvider {
		return nil
	}
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
		StaffType:    staff.StaffType,
		CanDispatch:  staff.CanDispatch,
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
		"staff_type":     staff.StaffType,
		"can_dispatch":   staff.CanDispatch,
		"exp":            expiredAt.UnixMilli(),
	}
}

func workStaffSessionPayload(staff *WorkStaffSession) map[string]any {
	if staff == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":             staff.ID,
		"name":           staff.Name,
		"phone":          staff.Phone,
		"feishu_open_id": staff.FeishuOpenID,
		"department_id":  staff.DepartmentID,
		"staff_type":     staff.StaffType,
		"can_dispatch":   staff.CanDispatch,
	}
}

func normalizeWorkScope(staff *WorkStaffSession, scope string) string {
	if staff != nil && staff.CanDispatch && strings.TrimSpace(scope) == "all" {
		return "all"
	}
	return "mine"
}

func workStaffWithScope(staff *WorkStaffSession, scope string) *WorkStaffSession {
	if staff == nil {
		return nil
	}
	copy := *staff
	copy.ViewAll = staff.CanDispatch && scope == "all"
	return &copy
}

func visibleWorkCustomers(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	customerIDs := visibleWorkCustomerIDs(ctx, staff)
	rows := make([]map[string]any, 0, len(customerIDs))
	seen := map[uint64]bool{}
	for _, customerID := range customerIDs {
		rows = appendVisibleWorkCustomer(ctx, staff, rows, seen, customerID)
	}
	return rows
}

func visibleWorkCustomerIDs(ctx context.Context, staff *WorkStaffSession) []uint64 {
	if staff == nil || staff.ID == 0 {
		return []uint64{}
	}
	seen := map[uint64]bool{}
	rows := make([]uint64, 0)
	appendID := func(customerID uint64) {
		if customerID == 0 || seen[customerID] {
			return
		}
		seen[customerID] = true
		rows = append(rows, customerID)
	}
	if staff.CanDispatch && staff.ViewAll {
		for _, customer := range crmmodel.NewCustomerModel().Select(ctx, map[string]any{}) {
			if customer != nil {
				appendID(customer.ID)
			}
		}
		return rows
	}
	members := crmmodel.NewCustomerMemberModel().Select(ctx, map[string]any{
		"staff_id": staff.ID,
		"status":   crmmodel.StatusEnabled,
	})
	for _, member := range members {
		if member == nil || !member.CanView {
			continue
		}
		appendID(member.CustomerID)
	}
	if staff.DepartmentID > 0 {
		departmentMembers := crmmodel.NewCustomerMemberModel().Select(ctx, map[string]any{
			"department_id": staff.DepartmentID,
			"status":        crmmodel.StatusEnabled,
		})
		for _, member := range departmentMembers {
			if member == nil || !member.CanView {
				continue
			}
			appendID(member.CustomerID)
		}
	}
	created := crmmodel.NewCustomerModel().Select(ctx, map[string]any{"created_by_staff_id": staff.ID})
	for _, customer := range created {
		if customer == nil {
			continue
		}
		appendID(customer.ID)
	}
	for _, state := range crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{"owner_staff_id": staff.ID}) {
		if state == nil {
			continue
		}
		appendID(state.CustomerID)
	}
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{"status": crmmodel.WorkTodoStatusPending}) {
		if todo != nil && canOperateWorkTodo(staff, todo) {
			appendID(todo.CustomerID)
		}
	}
	for _, operation := range crmmodel.NewOperationLogModel().Select(ctx, map[string]any{"operator_staff_id": staff.ID}) {
		if operation != nil {
			appendID(operation.CustomerID)
		}
	}
	return rows
}

func workCustomersByMode(ctx context.Context, staff *WorkStaffSession, mode string) []map[string]any {
	switch normalizeWorkCustomerMode(mode) {
	case workCustomerModeAll:
		return allWorkCustomers(ctx, staff)
	case workCustomerModeDone:
		return doneWorkCustomers(ctx, staff)
	case workCustomerModeProcessed:
		return processedWorkCustomers(ctx, staff)
	default:
		return pendingWorkCustomers(ctx, staff)
	}
}

type workCustomerListTarget struct {
	customerID      uint64
	assetIDs        []uint64
	doneOnly        bool
	processed       map[uint64]workProcessedOperation
	latestProcessed *workProcessedOperation
}

type workCustomerListSnapshot struct {
	visibleCustomerIDs []uint64
	doneTargets        []doneWorkCustomerTarget
	pendingTargets     []workCustomerListTarget
	processedTargets   []workCustomerListTarget
}

func newWorkCustomerListSnapshot(ctx context.Context, staff *WorkStaffSession) workCustomerListSnapshot {
	visibleCustomerIDs := visibleWorkCustomerIDs(ctx, staff)
	pendingTargets := workPendingTargetsForCustomerIDs(ctx, staff, visibleCustomerIDs)
	return workCustomerListSnapshot{
		visibleCustomerIDs: visibleCustomerIDs,
		doneTargets:        doneWorkCustomerTargets(ctx, staff),
		pendingTargets:     pendingWorkCustomerListTargetsFromTargets(pendingTargets),
		processedTargets:   processedWorkCustomerListTargets(ctx, staff),
	}
}

func paginatedWorkCustomersFromSnapshot(ctx context.Context, staff *WorkStaffSession, mode string, payload map[string]any, snapshot workCustomerListSnapshot) ([]map[string]any, int, int, int) {
	targets := workCustomerListTargetsFromSnapshot(snapshot, mode)
	pageTargets, page, pageSize, total := paginateWorkCustomerListTargets(targets, payload)
	return workCustomerRowsForListTargets(ctx, staff, mode, pageTargets), page, pageSize, total
}

func workCustomerListTargetsFromSnapshot(snapshot workCustomerListSnapshot, mode string) []workCustomerListTarget {
	switch normalizeWorkCustomerMode(mode) {
	case workCustomerModeAll:
		return allWorkCustomerListTargetsFromSnapshot(snapshot)
	case workCustomerModeDone:
		return doneWorkCustomerListTargetsFromTargets(snapshot.doneTargets)
	case workCustomerModeProcessed:
		return snapshot.processedTargets
	default:
		return snapshot.pendingTargets
	}
}

func paginateWorkCustomerListTargets(targets []workCustomerListTarget, payload map[string]any) ([]workCustomerListTarget, int, int, int) {
	total := len(targets)
	page, pageSize, start, end := workCustomerPageBounds(total, payload)
	if start >= total {
		return []workCustomerListTarget{}, page, pageSize, total
	}
	return targets[start:end], page, pageSize, total
}

func workCustomerRowsForListTargets(ctx context.Context, staff *WorkStaffSession, mode string, targets []workCustomerListTarget) []map[string]any {
	return newWorkCustomerListRowBuilder(ctx, staff).rows(mode, targets)
}

type workCustomerListRowBuilder struct {
	ctx              context.Context
	staff            *WorkStaffSession
	stageNames       map[uint64]string
	assetStatusNames map[uint64]string
	codePrefix       string
	codePrefixLoaded bool
}

func newWorkCustomerListRowBuilder(ctx context.Context, staff *WorkStaffSession) *workCustomerListRowBuilder {
	return &workCustomerListRowBuilder{
		ctx:              ctx,
		staff:            staff,
		stageNames:       map[uint64]string{},
		assetStatusNames: map[uint64]string{},
	}
}

func (builder *workCustomerListRowBuilder) rows(mode string, targets []workCustomerListTarget) []map[string]any {
	mode = normalizeWorkCustomerMode(mode)
	rows := make([]map[string]any, 0, len(targets))
	for _, target := range targets {
		row := builder.rowForTarget(mode, target)
		if len(row) == 0 {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func (builder *workCustomerListRowBuilder) rowForTarget(mode string, target workCustomerListTarget) map[string]any {
	if target.customerID == 0 {
		return map[string]any{}
	}
	if mode == workCustomerModeDone || target.doneOnly {
		return builder.doneCustomerRow(target.customerID, target.assetIDs)
	}
	if mode == workCustomerModeProcessed {
		return builder.processedCustomerRow(target)
	}
	row := builder.activeCustomerRow(target.customerID)
	if mode == workCustomerModePending {
		pendingRow, ok := workRowWithPendingTasks(row)
		if !ok {
			return map[string]any{}
		}
		return pendingRow
	}
	return row
}

func (builder *workCustomerListRowBuilder) activeCustomerRow(customerID uint64) map[string]any {
	customer := builder.customerBaseRow(customerID)
	if len(customer) == 0 {
		return map[string]any{}
	}
	customer["assets"] = builder.activeAssetRows(customerID)
	return workCustomerListRow(customer)
}

func (builder *workCustomerListRowBuilder) doneCustomerRow(customerID uint64, assetIDs []uint64) map[string]any {
	customer := builder.customerBaseRow(customerID)
	if len(customer) == 0 {
		return map[string]any{}
	}
	displayAssetIDs := uniqueUint64Values(assetIDs)
	if len(displayAssetIDs) == 0 {
		displayAssetIDs = workSummaryVisibleAssetIDs(builder.ctx, builder.staff, customerID)
	}
	customer["assets"] = builder.doneAssetRows(customerID, displayAssetIDs)
	return workCustomerListRow(customer)
}

func (builder *workCustomerListRowBuilder) customerBaseRow(customerID uint64) map[string]any {
	if customerID == 0 {
		return map[string]any{}
	}
	customer := crmmodel.NewCustomerModel().FindMap(builder.ctx, map[string]any{"id": customerID}, map[string]any{
		"field": "id,code,name,phone,wechat,gender,source_id,channel_id,level_id,created_at",
	})
	if len(customer) == 0 {
		return map[string]any{}
	}
	if code := inputText(customer["code"]); code != "" {
		customer["code_display"] = builder.customerCodePrefix() + code
	}
	customer["gender_name"] = workGenderName(inputText(customer["gender"]))
	return customer
}

func (builder *workCustomerListRowBuilder) customerCodePrefix() string {
	if !builder.codePrefixLoaded {
		builder.codePrefix = customerCodePrefixForWork(builder.ctx)
		builder.codePrefixLoaded = true
	}
	return builder.codePrefix
}

func (builder *workCustomerListRowBuilder) activeAssetRows(customerID uint64) []map[string]any {
	if customerID == 0 {
		return []map[string]any{}
	}
	assets := crmmodel.NewCustomerAssetModel().SelectMap(builder.ctx, map[string]any{"customer_id": customerID}, map[string]any{
		"field": "id,customer_id,asset_no,asset_name,asset_status_id,remark,created_at",
	})
	rows := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		assetID := inputUint64(asset["id"])
		if assetID == 0 || !canViewWorkAsset(builder.ctx, builder.staff, customerID, assetID) {
			continue
		}
		asset["asset_status_name"] = builder.assetStatusName(inputUint64(asset["asset_status_id"]))
		asset["customer_products"] = workCustomerProductRows(builder.ctx, builder.staff, customerID, assetID)
		if state := currentWorkTargetInstance(builder.ctx, builder.staff, customerID, assetID); state != nil {
			builder.attachStageFields(asset, state)
			asset["row_tasks"] = workPendingTodoRowTasks(builder.ctx, builder.staff, customerID, assetID)
		}
		rows = append(rows, workAssetListRow(asset))
	}
	return rows
}

func (builder *workCustomerListRowBuilder) doneAssetRows(customerID uint64, assetIDs []uint64) []map[string]any {
	if customerID == 0 || len(assetIDs) == 0 {
		return []map[string]any{}
	}
	rows := make([]map[string]any, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		asset := builder.doneAssetRow(customerID, assetID)
		if len(asset) > 0 {
			rows = append(rows, asset)
		}
	}
	return rows
}

func (builder *workCustomerListRowBuilder) doneAssetRow(customerID uint64, assetID uint64) map[string]any {
	if customerID == 0 || assetID == 0 {
		return map[string]any{}
	}
	asset := crmmodel.NewCustomerAssetModel().FindMap(builder.ctx, map[string]any{
		"id":          assetID,
		"customer_id": customerID,
	}, map[string]any{
		"field": "id,customer_id,asset_no,asset_name,asset_status_id,remark,created_at",
	})
	if len(asset) == 0 {
		return map[string]any{}
	}
	asset["asset_status_name"] = builder.assetStatusName(inputUint64(asset["asset_status_id"]))
	asset["customer_products"] = workCustomerProductRows(builder.ctx, builder.staff, customerID, assetID)
	if state := currentWorkTargetInstance(builder.ctx, builder.staff, customerID, assetID); state != nil {
		builder.attachStageFields(asset, state)
	}
	asset["row_tasks"] = workPendingTodoRowTasks(builder.ctx, builder.staff, customerID, assetID)
	return workAssetListRow(asset)
}

func (builder *workCustomerListRowBuilder) attachStageFields(target map[string]any, state *crmmodel.WorkflowInstance) {
	if target == nil || state == nil {
		return
	}
	stageEnteredAt := workStageEnteredAt(builder.ctx, state)
	target["state.id"] = state.ID
	target["state.workflow_id"] = state.WorkflowID
	target["state.stage_id"] = state.StageID
	target["state.owner_department_id"] = state.OwnerDepartmentID
	target["state.owner_staff_id"] = state.OwnerStaffID
	target["workflow_id"] = state.WorkflowID
	target["workflow_instance_id"] = state.ID
	target["stage_id"] = state.StageID
	target["stage_code"] = fmt.Sprintf("%d", state.StageID)
	target["stage_name"] = builder.stageName(state.StageID)
	target["progress_status"] = state.Status
	target["stage_entered_at"] = stageEnteredAt
	target["stage_days"] = workStageDwellDays(stageEnteredAt)
	target["last_operated_at"] = state.UpdatedAt
}

func (builder *workCustomerListRowBuilder) stageName(stageID uint64) string {
	if stageID == 0 {
		return ""
	}
	if name, exists := builder.stageNames[stageID]; exists {
		return name
	}
	name := workStageName(builder.ctx, stageID)
	builder.stageNames[stageID] = name
	return name
}

func (builder *workCustomerListRowBuilder) assetStatusName(statusID uint64) string {
	if statusID == 0 {
		return ""
	}
	if name, exists := builder.assetStatusNames[statusID]; exists {
		return name
	}
	name := workAssetStatusName(builder.ctx, statusID)
	builder.assetStatusNames[statusID] = name
	return name
}

var workCustomerListFields = []string{
	"id",
	"customer_id",
	"customer_no",
	"code_display",
	"code",
	"no",
	"name",
	"customer_name",
	"phone",
	"mobile",
	"wechat",
	"gender_name",
	"source_name",
	"source",
	"channel_name",
	"channel",
	"level_name",
	"customer_level",
	"status_name",
	"stage_name",
	"stage_code",
	"status_code",
	"current_stage_name",
	"current_status_name",
	"stage_entered_at",
	"stage_days",
	"last_operated_at",
	"created_at",
	"create_time",
	"processed_task_name",
	"processed_result",
	"processed_content",
	"processed_at",
}

var workAssetListFields = []string{
	"id",
	"asset_id",
	"customer_id",
	"asset_no",
	"asset_code",
	"code",
	"name",
	"asset_name",
	"asset_status_id",
	"asset_status_name",
	"workflow_instance_id",
	"workflow_id",
	"progress_status",
	"status_name",
	"stage_name",
	"stage_code",
	"status_code",
	"current_stage_name",
	"current_status_name",
	"stage_entered_at",
	"stage_days",
	"last_operated_at",
	"remark",
	"processed_task_name",
	"processed_result",
	"processed_content",
	"processed_at",
}

func workCustomerListRows(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if compact := workCustomerListRow(row); len(compact) > 0 {
			result = append(result, compact)
		}
	}
	return result
}

func workCustomerListRow(row map[string]any) map[string]any {
	if len(row) == 0 {
		return map[string]any{}
	}
	result := pickWorkListFields(row, workCustomerListFields)
	if tasks := workListRowTasks(row); len(tasks) > 0 {
		result["row_tasks"] = tasks
	}
	if assets := workAssetListRows(mapListFromAny(row["assets"])); len(assets) > 0 {
		result["assets"] = assets
	}
	return result
}

func workAssetListRows(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if compact := workAssetListRow(row); len(compact) > 0 {
			result = append(result, compact)
		}
	}
	return result
}

func workAssetListRow(row map[string]any) map[string]any {
	if len(row) == 0 {
		return map[string]any{}
	}
	result := pickWorkListFields(row, workAssetListFields)
	if flow := mapFromAny(row["flow"]); len(flow) > 0 {
		result["flow"] = flow
	}
	if tasks := workListRowTasks(row); len(tasks) > 0 {
		result["row_tasks"] = tasks
	}
	if customerProducts := workCustomerProductListRows(mapListFromAny(row["customer_products"])); len(customerProducts) > 0 {
		result["customer_products"] = customerProducts
	}
	return result
}

var workCustomerProductListFields = []string{
	"id",
	"customer_product_id",
	"product_id",
	"product_name",
	"product_code",
	"workflow_instance_id",
	"workflow_name",
	"stage_name",
	"owner_staff_name",
	"status",
	"created_at",
	"updated_at",
}

func workCustomerProductListRows(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if compact := pickWorkListFields(row, workCustomerProductListFields); len(compact) > 0 {
			if flow := mapFromAny(row["flow"]); len(flow) > 0 {
				compact["flow"] = flow
			}
			result = append(result, compact)
		}
	}
	return result
}

func pickWorkListFields(row map[string]any, fields []string) map[string]any {
	result := map[string]any{}
	for _, field := range fields {
		if value, exists := row[field]; exists {
			result[field] = value
		}
	}
	return result
}

func workListRowTasks(row map[string]any) []map[string]any {
	for _, field := range []string{"row_tasks", "edit_tasks", "tasks"} {
		tasks := mapListFromAny(row[field])
		if len(tasks) > 0 {
			return workListTaskRows(tasks)
		}
	}
	return nil
}

var workTaskListFields = []string{
	"id",
	"name",
	"task_name",
	"task_type",
	"form_id",
	"workflow_instance_id",
	"customer_product_id",
	"product_options",
	"selected_product_ids",
	"todo_id",
	"todo_status",
	"status_name",
	"todo_required",
	"todo_sort",
	"due_at",
	"result",
	"can_operate",
	"unassigned",
	"assigned_at",
	"assignee_department_id",
	"assignee_department_name",
	"assignee_staff_id",
	"assignee_staff_name",
	"opinion_requirement",
	"reject_submit_form",
}

func workListTaskRows(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if compact := pickWorkListFields(row, workTaskListFields); len(compact) > 0 {
			result = append(result, compact)
		}
	}
	return result
}

func allWorkCustomerListTargetsFromSnapshot(snapshot workCustomerListSnapshot) []workCustomerListTarget {
	seen := map[uint64]bool{}
	targets := make([]workCustomerListTarget, 0)
	for _, customerID := range snapshot.visibleCustomerIDs {
		if customerID == 0 || seen[customerID] {
			continue
		}
		seen[customerID] = true
		targets = append(targets, workCustomerListTarget{customerID: customerID})
	}
	for _, target := range snapshot.doneTargets {
		if target.customerID == 0 || seen[target.customerID] {
			continue
		}
		seen[target.customerID] = true
		targets = append(targets, workCustomerListTarget{
			customerID: target.customerID,
			assetIDs:   target.assetIDs,
			doneOnly:   true,
		})
	}
	sortWorkCustomerListTargetsByPendingTargets(targets, snapshot.pendingTargets)
	return targets
}

func doneWorkCustomerListTargetsFromTargets(doneTargets []doneWorkCustomerTarget) []workCustomerListTarget {
	targets := make([]workCustomerListTarget, 0, len(doneTargets))
	for _, target := range doneTargets {
		if target.customerID == 0 {
			continue
		}
		targets = append(targets, workCustomerListTarget{
			customerID: target.customerID,
			assetIDs:   target.assetIDs,
			doneOnly:   true,
		})
	}
	return targets
}

func pendingWorkCustomerListTargets(ctx context.Context, staff *WorkStaffSession) []workCustomerListTarget {
	return pendingWorkCustomerListTargetsFromTargets(workPendingTargets(ctx, staff))
}

func pendingWorkCustomerListTargetsFromTargets(pendingTargets []workPendingTarget) []workCustomerListTarget {
	seen := map[uint64]bool{}
	targets := make([]workCustomerListTarget, 0, len(pendingTargets))
	for _, target := range pendingTargets {
		if target.customerID == 0 || seen[target.customerID] {
			continue
		}
		seen[target.customerID] = true
		targets = append(targets, workCustomerListTarget{customerID: target.customerID})
	}
	return targets
}

func sortWorkCustomerListTargetsByPendingTargets(targets []workCustomerListTarget, pendingTargets []workCustomerListTarget) {
	if len(targets) == 0 {
		return
	}
	pendingCustomerIDs := map[uint64]bool{}
	for _, target := range pendingTargets {
		if target.customerID > 0 {
			pendingCustomerIDs[target.customerID] = true
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		leftPending := pendingCustomerIDs[targets[i].customerID]
		rightPending := pendingCustomerIDs[targets[j].customerID]
		if leftPending != rightPending {
			return leftPending
		}
		return false
	})
}

type workPendingTarget struct {
	customerID uint64
	assetID    uint64
}

func pendingWorkCustomers(ctx context.Context, staff *WorkStaffSession) []map[string]any {
	targets := workPendingTargets(ctx, staff)
	if len(targets) == 0 {
		return []map[string]any{}
	}
	return workPendingCustomerRows(ctx, staff, targets)
}

func workPendingTargets(ctx context.Context, staff *WorkStaffSession) []workPendingTarget {
	return workPendingTargetsForCustomerIDs(ctx, staff, visibleWorkCustomerIDs(ctx, staff))
}

func workPendingTargetsForCustomerIDs(ctx context.Context, staff *WorkStaffSession, customerIDs []uint64) []workPendingTarget {
	targets := make([]workPendingTarget, 0, len(customerIDs))
	for _, customerID := range customerIDs {
		if customerID == 0 {
			continue
		}
		customerProgress := currentWorkTargetInstance(ctx, staff, customerID, 0)
		if canManageWorkProgress(staff, customerProgress) || len(workPendingTargetTasks(ctx, staff, customerID, 0)) > 0 {
			targets = append(targets, workPendingTarget{customerID: customerID})
		}
		for _, assetID := range workSummaryVisibleAssetIDs(ctx, staff, customerID) {
			progress := currentWorkTargetInstance(ctx, staff, customerID, assetID)
			if !canManageWorkProgress(staff, progress) && len(workPendingTargetTasks(ctx, staff, customerID, assetID)) == 0 {
				continue
			}
			targets = append(targets, workPendingTarget{
				customerID: customerID,
				assetID:    assetID,
			})
		}
	}
	return targets
}

func canManageWorkProgress(staff *WorkStaffSession, progress *crmmodel.WorkflowInstance) bool {
	if staff == nil || progress == nil || progress.Status != crmmodel.ProgressStatusActive {
		return false
	}
	return progress.OwnerStaffID == staff.ID || staff.CanDispatch && staff.ViewAll
}

func workPendingTargetTasks(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) []map[string]any {
	state := currentWorkTargetInstance(ctx, staff, customerID, assetID)
	if state == nil {
		return []map[string]any{}
	}
	return workPendingTodoRowTasks(ctx, staff, customerID, assetID)
}

func workPendingCustomerRows(ctx context.Context, staff *WorkStaffSession, targets []workPendingTarget) []map[string]any {
	seen := map[uint64]bool{}
	rows := make([]map[string]any, 0, len(targets))
	for _, target := range targets {
		if target.customerID == 0 || seen[target.customerID] {
			continue
		}
		seen[target.customerID] = true
		row := workCustomerRow(ctx, staff, target.customerID)
		if pendingRow, ok := workRowWithPendingTasks(row); ok {
			rows = append(rows, pendingRow)
		}
	}
	return rows
}

func workCustomerModeCounts(ctx context.Context, staff *WorkStaffSession, currentMode string, currentRows []map[string]any) map[string]int {
	currentMode = normalizeWorkCustomerMode(currentMode)
	doneTargets := doneWorkCustomerTargets(ctx, staff)
	return map[string]int{
		workCustomerModePending:   workCustomerModeCount(ctx, staff, currentMode, currentRows, doneTargets, workCustomerModePending),
		workCustomerModeProcessed: workCustomerModeCount(ctx, staff, currentMode, currentRows, doneTargets, workCustomerModeProcessed),
		workCustomerModeDone:      workCustomerModeCount(ctx, staff, currentMode, currentRows, doneTargets, workCustomerModeDone),
		workCustomerModeAll:       workCustomerModeCount(ctx, staff, currentMode, currentRows, doneTargets, workCustomerModeAll),
	}
}

func workCustomerModeCountsFromSnapshot(snapshot workCustomerListSnapshot, currentMode string, currentTotal int) map[string]int {
	currentMode = normalizeWorkCustomerMode(currentMode)
	return map[string]int{
		workCustomerModePending:   workCustomerModeCountFromSnapshot(snapshot, currentMode, currentTotal, workCustomerModePending),
		workCustomerModeProcessed: workCustomerModeCountFromSnapshot(snapshot, currentMode, currentTotal, workCustomerModeProcessed),
		workCustomerModeDone:      workCustomerModeCountFromSnapshot(snapshot, currentMode, currentTotal, workCustomerModeDone),
		workCustomerModeAll:       workCustomerModeCountFromSnapshot(snapshot, currentMode, currentTotal, workCustomerModeAll),
	}
}

func workCustomerModeCountFromSnapshot(snapshot workCustomerListSnapshot, currentMode string, currentTotal int, targetMode string) int {
	if currentMode == targetMode {
		return currentTotal
	}
	switch targetMode {
	case workCustomerModeAll:
		return workCustomerListSnapshotAllCount(snapshot)
	case workCustomerModeDone:
		return len(snapshot.doneTargets)
	case workCustomerModeProcessed:
		return len(snapshot.processedTargets)
	case workCustomerModePending:
		return len(snapshot.pendingTargets)
	default:
		return 0
	}
}

func workCustomerListSnapshotAllCount(snapshot workCustomerListSnapshot) int {
	seen := map[uint64]bool{}
	for _, customerID := range snapshot.visibleCustomerIDs {
		if customerID > 0 {
			seen[customerID] = true
		}
	}
	for _, target := range snapshot.doneTargets {
		if target.customerID > 0 {
			seen[target.customerID] = true
		}
	}
	return len(seen)
}

func workCustomerModeCount(ctx context.Context, staff *WorkStaffSession, currentMode string, currentRows []map[string]any, doneTargets []doneWorkCustomerTarget, targetMode string) int {
	if currentMode == targetMode {
		return len(currentRows)
	}
	switch targetMode {
	case workCustomerModeAll:
		return workAllCustomerCount(ctx, staff, doneTargets)
	case workCustomerModeDone:
		return len(doneTargets)
	case workCustomerModeProcessed:
		return len(processedWorkCustomerListTargets(ctx, staff))
	case workCustomerModePending:
		if currentMode == workCustomerModeAll {
			return countWorkRowsWithPendingTasks(currentRows)
		}
		return len(pendingWorkCustomerListTargets(ctx, staff))
	default:
		return 0
	}
}

func workAllCustomerCount(ctx context.Context, staff *WorkStaffSession, doneTargets []doneWorkCustomerTarget) int {
	seen := map[uint64]bool{}
	for _, customerID := range visibleWorkCustomerIDs(ctx, staff) {
		if customerID > 0 {
			seen[customerID] = true
		}
	}
	for _, target := range doneTargets {
		if target.customerID > 0 {
			seen[target.customerID] = true
		}
	}
	return len(seen)
}

func countWorkRowsWithPendingTasks(rows []map[string]any) int {
	count := 0
	for _, row := range rows {
		if workRowHasPendingTasks(row) {
			count++
		}
	}
	return count
}

func normalizeWorkCustomerMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case workCustomerModeAll:
		return workCustomerModeAll
	case workCustomerModeDone:
		return workCustomerModeDone
	case workCustomerModeProcessed:
		return workCustomerModeProcessed
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
	if len(mapListFromAny(row["row_tasks"])) > 0 || inputText(row["progress_status"]) == crmmodel.ProgressStatusActive {
		return true
	}
	for _, asset := range mapListFromAny(row["assets"]) {
		if len(mapListFromAny(asset["row_tasks"])) > 0 || inputText(asset["progress_status"]) == crmmodel.ProgressStatusActive {
			return true
		}
	}
	return false
}

type workSummaryTarget struct {
	WorkflowID uint64
	StageID    uint64
	StageName  string
	Tasks      []map[string]any
}

type workSummaryData struct {
	PendingTaskCount     int
	OverdueTaskCount     int
	CompletedTodayCount  int
	Targets              []workSummaryTarget
	PendingWorkflowIDs   map[uint64]struct{}
	OverdueWorkflowIDs   map[uint64]struct{}
	CompletedWorkflowIDs map[uint64]struct{}
}

func workSummarySnapshot(ctx context.Context, staff *WorkStaffSession) workSummaryData {
	workload := currentWorkPersonalWorkload(ctx, staff)
	now := time.Now()
	summary := workSummaryData{
		PendingWorkflowIDs:   map[uint64]struct{}{},
		OverdueWorkflowIDs:   map[uint64]struct{}{},
		CompletedWorkflowIDs: map[uint64]struct{}{},
	}
	for _, instance := range workload.instances {
		if instance == nil {
			continue
		}
		todos := workload.pendingTodosByInstance[instance.ID]
		summary.PendingTaskCount += len(todos)
		summary.PendingWorkflowIDs[instance.WorkflowID] = struct{}{}
		for _, todo := range todos {
			if todo != nil && todo.DueAt != nil && todo.DueAt.Before(now) {
				summary.OverdueTaskCount++
				summary.OverdueWorkflowIDs[instance.WorkflowID] = struct{}{}
			}
		}
		summary.Targets = append(summary.Targets, workSummaryTarget{
			WorkflowID: instance.WorkflowID,
			StageID:    instance.StageID,
			StageName:  workStageName(ctx, instance.StageID),
			Tasks:      workload.pendingTaskRows(ctx, staff, instance.ID),
		})
	}
	today := workBeginningOfDay(now)
	tomorrow := today.AddDate(0, 0, 1)
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{
		"assignee_staff_id": staff.ID,
		"status":            crmmodel.WorkTodoStatusDone,
	}) {
		if todo == nil || todo.CompletedAt == nil || todo.CompletedAt.Before(today) || !todo.CompletedAt.Before(tomorrow) {
			continue
		}
		task := crmmodel.NewTaskModel().Find(ctx, map[string]any{"id": todo.TaskID})
		if task != nil && task.TaskType != crmmodel.TaskTypeRule {
			summary.CompletedTodayCount++
			summary.CompletedWorkflowIDs[todo.WorkflowID] = struct{}{}
		}
	}
	return summary
}

func workSummaryVisibleAssetIDs(ctx context.Context, staff *WorkStaffSession, customerID uint64) []uint64 {
	if customerID == 0 {
		return []uint64{}
	}
	rows := crmmodel.NewCustomerAssetModel().SelectMap(ctx, map[string]any{"customer_id": customerID})
	assetIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		assetID := inputUint64(row["id"])
		if assetID == 0 || !canViewWorkAsset(ctx, staff, customerID, assetID) {
			continue
		}
		assetIDs = append(assetIDs, assetID)
	}
	return assetIDs
}

func workSummaryMetricRows(ctx context.Context, summary workSummaryData) []map[string]any {
	return []map[string]any{
		workSummaryMetric(ctx, "pending_tasks", "我的待办", summary.PendingTaskCount, "当前分配给我的未完成任务", summary.PendingWorkflowIDs, workCustomerModePending, "personalPending", nil),
		workSummaryMetric(ctx, "pending_targets", "处理中流程", len(summary.Targets), "当前包含我的待办任务的流程", summary.PendingWorkflowIDs, workCustomerModePending, "personalPending", nil),
		workSummaryMetric(ctx, "completed_today", "今日完成", summary.CompletedTodayCount, "今天由我完成的人工任务", summary.CompletedWorkflowIDs, workCustomerModeAll, "completedToday", nil),
		workSummaryMetric(ctx, "overdue_tasks", "超期待办", summary.OverdueTaskCount, "已超过办理期限的我的待办", summary.OverdueWorkflowIDs, workCustomerModePending, "overdue", nil),
	}
}

func workSummaryMetric(
	ctx context.Context,
	key string,
	name string,
	value int,
	description string,
	workflowIDs map[uint64]struct{},
	mode string,
	quickFilter string,
	filters map[string]string,
) map[string]any {
	row := map[string]any{
		"key":         key,
		"name":        name,
		"value":       value,
		"description": description,
	}
	if path := workSummaryDrilldownPath(ctx, workflowIDs, mode, quickFilter, filters); path != "" {
		row["drilldown_path"] = path
	}
	return row
}

func workSummaryStageRows(ctx context.Context, targets []workSummaryTarget) []map[string]any {
	counts := map[string]int{}
	names := map[string]string{}
	workflowIDs := map[string]map[uint64]struct{}{}
	for _, target := range targets {
		key := "_empty"
		name := "未进入阶段"
		if target.StageID > 0 {
			key = fmt.Sprintf("%d", target.StageID)
			name = target.StageName
			if name == "" {
				name = key
			}
		}
		counts[key]++
		names[key] = name
		workSummaryAddWorkflowID(workflowIDs, key, target.WorkflowID)
	}
	return workSummaryBreakdownRows(ctx, counts, names, workflowIDs, len(targets), workCustomerModePending, "personalPending", "stage_filter")
}

func workSummaryTaskRows(ctx context.Context, targets []workSummaryTarget) []map[string]any {
	counts := map[string]int{}
	names := map[string]string{}
	workflowIDs := map[string]map[uint64]struct{}{}
	total := 0
	for _, target := range targets {
		for _, task := range target.Tasks {
			key := inputText(task["task_type"])
			if key == "" {
				key = "_unknown"
			}
			counts[key]++
			names[key] = WorkTaskTypeName(key)
			workSummaryAddWorkflowID(workflowIDs, key, target.WorkflowID)
			total++
		}
	}
	return workSummaryBreakdownRows(ctx, counts, names, workflowIDs, total, workCustomerModePending, "personalPending", "task_filter")
}

func workSummaryBreakdownRows(
	ctx context.Context,
	counts map[string]int,
	names map[string]string,
	workflowIDs map[string]map[uint64]struct{},
	total int,
	mode string,
	quickFilter string,
	filterName string,
) []map[string]any {
	rows := make([]map[string]any, 0, len(counts))
	for key, count := range counts {
		percent := 0
		if total > 0 {
			percent = int(float64(count) / float64(total) * 100)
		}
		row := map[string]any{
			"key":     key,
			"name":    names[key],
			"count":   count,
			"percent": percent,
		}
		if path := workSummaryDrilldownPath(ctx, workflowIDs[key], mode, quickFilter, map[string]string{filterName: key}); path != "" {
			row["drilldown_path"] = path
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left := inputUint64(rows[i]["count"])
		right := inputUint64(rows[j]["count"])
		if left != right {
			return left > right
		}
		return inputText(rows[i]["name"]) < inputText(rows[j]["name"])
	})
	return rows
}

func workSummaryAddWorkflowID(target map[string]map[uint64]struct{}, key string, workflowID uint64) {
	if workflowID == 0 {
		return
	}
	if target[key] == nil {
		target[key] = map[uint64]struct{}{}
	}
	target[key][workflowID] = struct{}{}
}

func workSummaryDrilldownPath(
	ctx context.Context,
	workflowIDs map[uint64]struct{},
	mode string,
	quickFilter string,
	filters map[string]string,
) string {
	if len(workflowIDs) != 1 {
		return ""
	}
	workflowID := uint64(0)
	for id := range workflowIDs {
		workflowID = id
	}
	workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{
		"id":     workflowID,
		"status": crmmodel.StatusEnabled,
	})
	if workflow == nil {
		return ""
	}
	path := workflowNavigationPath(workflow)
	values := url.Values{}
	if mode != "" {
		values.Set("mode", mode)
	}
	if quickFilter != "" {
		values.Set("quick_filter", quickFilter)
	}
	for key, value := range filters {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			values.Set(key, value)
		}
	}
	if len(values) > 0 {
		path += "&" + values.Encode()
	}
	return path
}

func WorkTaskTypeName(taskType string) string {
	switch taskType {
	case crmmodel.TaskTypeTodo:
		return "普通事项"
	case crmmodel.TaskTypeForm:
		return "填写资料"
	case crmmodel.TaskTypeApproval:
		return "审核"
	case crmmodel.TaskTypeRule:
		return "自动核验"
	default:
		return "其他任务"
	}
}

func workSummaryTrendRows(ctx context.Context, staff *WorkStaffSession, days int) []map[string]any {
	if staff == nil || staff.ID == 0 {
		return []map[string]any{}
	}
	if days <= 0 {
		days = 14
	}
	today := workBeginningOfDay(time.Now())
	end := today.AddDate(0, 0, 1)
	start := today.AddDate(0, 0, -days+1)
	rows := make([]map[string]any, 0, days)
	indexes := map[string]int{}
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		indexes[key] = i
		rows = append(rows, map[string]any{
			"date":             key,
			"label":            day.Format("01-02"),
			"task_count":       0,
			"transition_count": 0,
		})
	}
	for _, event := range crmmodel.NewStatEventModel().Select(ctx, map[string]any{"operator_staff_id": staff.ID}) {
		if event == nil || event.EventAt.Before(start) || !event.EventAt.Before(end) {
			continue
		}
		index, exists := indexes[workBeginningOfDay(event.EventAt).Format("2006-01-02")]
		if !exists {
			continue
		}
		switch event.EventType {
		case crmmodel.StatEventTypeTask:
			if event.ResultValue == workResultProgress || event.TaskType == crmmodel.TaskTypeRule {
				continue
			}
			rows[index]["task_count"] = inputInt(rows[index]["task_count"]) + 1
		case crmmodel.StatEventTypeTransition:
			rows[index]["transition_count"] = inputInt(rows[index]["transition_count"]) + 1
		}
	}
	return rows
}

func workTimeValue(value any) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case *time.Time:
		if typed == nil {
			return time.Time{}
		}
		return *typed
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return time.Time{}
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
			if parsed, err := time.Parse(layout, text); err == nil {
				return parsed
			}
		}
	}
	return time.Time{}
}

func workBeginningOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
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
	if staff == nil || staff.ID == 0 {
		return []doneWorkCustomerTarget{}
	}
	targetIndexes := map[uint64]int{}
	targets := make([]doneWorkCustomerTarget, 0)
	for _, status := range []string{crmmodel.ProgressStatusCompleted, crmmodel.ProgressStatusTerminated} {
		for _, progress := range crmmodel.NewWorkflowInstanceModel().Select(ctx, map[string]any{"status": status}) {
			if progress == nil || progress.CustomerID == 0 || progress.AssetID == 0 {
				continue
			}
			if !canViewWorkAsset(ctx, staff, progress.CustomerID, progress.AssetID) {
				continue
			}
			index, exists := targetIndexes[progress.CustomerID]
			if !exists {
				index = len(targets)
				targetIndexes[progress.CustomerID] = index
				targets = append(targets, doneWorkCustomerTarget{customerID: progress.CustomerID})
			}
			targets[index].assetIDs = append(targets[index].assetIDs, progress.AssetID)
		}
	}
	return targets
}

func workStaffOperationRows(ctx context.Context, staffID uint64) []map[string]any {
	if staffID == 0 {
		return []map[string]any{}
	}
	return crmmodel.NewOperationLogModel().SelectMap(ctx, map[string]any{
		"operator_staff_id": staffID,
	})
}

func workRecentOperationRows(rows []map[string]any, limit int) []map[string]any {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func uniqueUint64Values(values []uint64) []uint64 {
	seen := map[uint64]bool{}
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func doneWorkCustomerRow(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetIDs []uint64) map[string]any {
	if customerID == 0 {
		return map[string]any{}
	}
	customer := crmmodel.NewCustomerModel().FindMap(ctx, map[string]any{"id": customerID})
	if len(customer) == 0 {
		return map[string]any{}
	}
	attachWorkEntityDataValues(ctx, customer, workCustomerFormValues(ctx, customerID, 0, customer), crmmodel.CustomerDataTemplateCateID)
	attachWorkCustomerTagIDs(ctx, customer, customerID)
	displayAssetIDs := uniqueUint64Values(assetIDs)
	if len(displayAssetIDs) == 0 {
		displayAssetIDs = workSummaryVisibleAssetIDs(ctx, staff, customerID)
	}
	customer["assets"] = doneWorkAssetRows(ctx, staff, customerID, displayAssetIDs)
	if state := currentWorkTargetInstance(ctx, staff, customerID, 0); state != nil {
		attachWorkStageFields(ctx, customer, state)
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
	attachWorkEntityDataValues(ctx, asset, workAssetFormValues(ctx, customerID, assetID, asset), crmmodel.CustomerAssetDataTemplateCateID)
	asset["asset_status_name"] = workAssetStatusName(ctx, inputUint64(asset["asset_status_id"]))
	asset["customer_products"] = workCustomerProductRows(ctx, staff, customerID, assetID)
	if state := currentWorkTargetInstance(ctx, staff, customerID, assetID); state != nil {
		attachWorkStageFields(ctx, asset, state)
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
	if len(rowTasks) == 0 && inputText(row["progress_status"]) != crmmodel.ProgressStatusActive && len(pendingAssets) == 0 {
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
		if len(mapListFromAny(asset["row_tasks"])) > 0 || inputText(asset["progress_status"]) == crmmodel.ProgressStatusActive {
			result = append(result, asset)
		}
	}
	return result
}

func canViewWorkCustomer(ctx context.Context, staff *WorkStaffSession, customerID uint64) bool {
	if staff == nil || staff.ID == 0 || customerID == 0 {
		return false
	}
	if staff.CanDispatch && staff.ViewAll {
		return crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}) != nil
	}
	if canViewWorkWorkflowSubject(ctx, staff, customerID, 0) {
		return true
	}
	if crmmodel.NewOperationLogModel().Find(ctx, map[string]any{
		"customer_id":       customerID,
		"operator_staff_id": staff.ID,
	}) != nil {
		return true
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
	if staff.CanDispatch && staff.ViewAll {
		return crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{
			"id":          assetID,
			"customer_id": customerID,
		}) != nil
	}
	if canViewWorkWorkflowSubject(ctx, staff, customerID, assetID) {
		return true
	}
	if crmmodel.NewOperationLogModel().Find(ctx, map[string]any{
		"customer_id":       customerID,
		"asset_id":          assetID,
		"operator_staff_id": staff.ID,
	}) != nil {
		return true
	}
	if customer := crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": customerID}); customer != nil && customer.CreatedByStaffID == staff.ID {
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

func canViewWorkWorkflowSubject(ctx context.Context, staff *WorkStaffSession, customerID, assetID uint64) bool {
	filters := map[string]any{"customer_id": customerID}
	if assetID > 0 {
		filters["asset_id"] = assetID
	}
	for _, instance := range crmmodel.NewWorkflowInstanceModel().Select(ctx, filters) {
		if canViewWorkflowInstanceInScope(ctx, staff, instance) {
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
	attachWorkEntityDataValues(ctx, customer, workCustomerFormValues(ctx, customerID, 0, customer), crmmodel.CustomerDataTemplateCateID)
	attachWorkCustomerTagIDs(ctx, customer, customerID)
	customer["assets"] = workAssetRows(ctx, staff, customerID)
	enrichWorkCustomerRow(ctx, customer)
	return customer
}

func workCustomerRowAsset(customer map[string]any, assetID uint64) map[string]any {
	if assetID == 0 {
		return map[string]any{}
	}
	for _, asset := range mapListFromAny(customer["assets"]) {
		if inputUint64(asset["id"]) == assetID {
			return asset
		}
	}
	return map[string]any{}
}

func attachWorkCustomerTagIDs(ctx context.Context, customer map[string]any, customerID uint64) {
	if len(customer) == 0 || customerID == 0 {
		return
	}
	if tagIDs := CustomerTagIDs(ctx, customerID); len(tagIDs) > 0 {
		customer["tag_ids"] = tagIDs
	}
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
		attachWorkEntityDataValues(ctx, asset, workAssetFormValues(ctx, customerID, assetID, asset), crmmodel.CustomerAssetDataTemplateCateID)
		asset["asset_status_name"] = workAssetStatusName(ctx, inputUint64(asset["asset_status_id"]))
		asset["customer_products"] = workCustomerProductRows(ctx, staff, customerID, assetID)
		state := currentWorkTargetInstance(ctx, staff, customerID, assetID)
		if state != nil {
			attachWorkStageFields(ctx, asset, state)
			asset["row_tasks"] = workPendingTodoTasks(ctx, staff, customerID, assetID)
		}
		rows = append(rows, asset)
	}
	return rows
}

func workCustomerProductRows(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) []map[string]any {
	if customerID == 0 {
		return []map[string]any{}
	}
	filter := map[string]any{
		"customer_id": customerID,
	}
	if assetID > 0 {
		filter["asset_id"] = assetID
	}
	rows := crmmodel.NewCustomerProductModel().Select(ctx, filter, map[string]any{"order": "id desc"})
	result := make([]map[string]any, 0, len(rows))
	for _, customerProduct := range rows {
		if customerProduct == nil {
			continue
		}
		product := crmmodel.NewProductModel().Find(ctx, map[string]any{"id": customerProduct.ProductID})
		row := map[string]any{
			"id":                  customerProduct.ID,
			"customer_product_id": customerProduct.ID,
			"customer_id":         customerProduct.CustomerID,
			"asset_id":            customerProduct.AssetID,
			"product_id":          customerProduct.ProductID,
			"status":              customerProduct.Status,
			"status_name":         crmmodel.CustomerProductStatusName(customerProduct.Status),
			"created_at":          customerProduct.CreatedAt,
			"updated_at":          customerProduct.UpdatedAt,
		}
		if product != nil {
			row["product_name"] = product.Name
			row["product_code"] = product.Code
			row["service_workflow_id"] = product.ServiceWorkflowID
		}
		instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{
			"customer_product_id": customerProduct.ID,
		}, map[string]any{"order": "id desc"})
		if instance != nil {
			row["workflow_instance_id"] = instance.ID
			row["workflow_id"] = instance.WorkflowID
			row["stage_id"] = instance.StageID
			row["flow"] = workFlowDetail(ctx, staff, instance.ID)
			if workflow := crmmodel.NewWorkflowModel().Find(ctx, map[string]any{"id": instance.WorkflowID}); workflow != nil {
				row["workflow_name"] = workflow.Name
			}
			if stage := crmmodel.NewStageModel().Find(ctx, map[string]any{"id": instance.StageID}); stage != nil {
				row["stage_name"] = stage.Name
			}
			if owner := crmmodel.NewStaffModel().Find(ctx, map[string]any{"id": instance.OwnerStaffID}); owner != nil {
				row["owner_staff_name"] = owner.Name
			}
		}
		result = append(result, row)
	}
	return result
}

func attachWorkStageFields(ctx context.Context, target map[string]any, state *crmmodel.WorkflowInstance) {
	if target == nil || state == nil {
		return
	}
	stageEnteredAt := workStageEnteredAt(ctx, state)
	target["state.id"] = state.ID
	target["state.workflow_id"] = state.WorkflowID
	target["state.stage_id"] = state.StageID
	target["state.owner_department_id"] = state.OwnerDepartmentID
	target["state.owner_staff_id"] = state.OwnerStaffID
	target["workflow_id"] = state.WorkflowID
	target["stage_id"] = state.StageID
	target["stage_code"] = fmt.Sprintf("%d", state.StageID)
	target["stage_name"] = workStageName(ctx, state.StageID)
	target["progress_status"] = state.Status
	target["stage_entered_at"] = stageEnteredAt
	target["stage_days"] = workStageDwellDays(stageEnteredAt)
	target["last_operated_at"] = state.UpdatedAt
}

func workStageEnteredAt(_ context.Context, state *crmmodel.WorkflowInstance) time.Time {
	if state == nil {
		return time.Time{}
	}
	if !state.StartedAt.IsZero() {
		return state.StartedAt
	}
	return state.UpdatedAt
}

func workStageDwellDays(start time.Time) int {
	if start.IsZero() {
		return 0
	}
	days := int(time.Since(start).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func enrichWorkCustomerRow(ctx context.Context, customer map[string]any) {
	if code := customerCodeDisplayForWork(ctx, inputText(customer["code"])); code != "" {
		customer["code_display"] = code
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

func customerCodeDisplayForWork(ctx context.Context, code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	return customerCodePrefixForWork(ctx) + code
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

func hasWorkCustomerListFilter(payload map[string]any) bool {
	return hasWorkCustomerStructuredFilter(payload) ||
		firstText(payload, "keyword") != "" ||
		hasWorkCustomerWorkFilter(payload)
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

func hasWorkCustomerWorkFilter(payload map[string]any) bool {
	quickFilter := firstText(payload, "quick_filter", "quickFilter")
	return (quickFilter != "" && quickFilter != "all") ||
		firstText(payload, "stage_filter", "stage") != "" ||
		firstText(payload, "task_filter", "task") != ""
}

func filterWorkCustomersByWorkFilters(rows []map[string]any, payload map[string]any) []map[string]any {
	quickFilter := firstText(payload, "quick_filter", "quickFilter")
	stageFilter := firstText(payload, "stage_filter", "stage")
	taskFilter := firstText(payload, "task_filter", "task")
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if !workRowMatchesTargetFilters(row, quickFilter, stageFilter, taskFilter) {
			continue
		}
		assets := mapListFromAny(row["assets"])
		filteredAssets := filterWorkAssetsByWorkFilters(assets, quickFilter, stageFilter, taskFilter)
		next := copyMap(row)
		if len(assets) > 0 {
			next["assets"] = filteredAssets
		}
		result = append(result, next)
	}
	return result
}

func filterWorkAssetsByWorkFilters(assets []map[string]any, quickFilter string, stageFilter string, taskFilter string) []map[string]any {
	if len(assets) == 0 {
		return assets
	}
	result := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		if workTargetMatchesFilters(asset, quickFilter, stageFilter, taskFilter, true) {
			result = append(result, asset)
		}
	}
	return result
}

func workRowMatchesTargetFilters(row map[string]any, quickFilter string, stageFilter string, taskFilter string) bool {
	assets := mapListFromAny(row["assets"])
	if len(assets) == 0 {
		return workTargetMatchesFilters(row, quickFilter, stageFilter, taskFilter, false)
	}
	for _, asset := range assets {
		if workTargetMatchesFilters(asset, quickFilter, stageFilter, taskFilter, true) {
			return true
		}
	}
	return false
}

func workTargetMatchesFilters(target map[string]any, quickFilter string, stageFilter string, taskFilter string, hasAsset bool) bool {
	if !workTargetMatchesQuickFilter(target, quickFilter, hasAsset) {
		return false
	}
	if stageFilter != "" && !workTargetStageMatches(target, stageFilter) {
		return false
	}
	if taskFilter != "" && !workTargetTaskMatches(target, taskFilter) {
		return false
	}
	return true
}

func workTargetMatchesQuickFilter(target map[string]any, quickFilter string, hasAsset bool) bool {
	switch quickFilter {
	case "", "all":
		return true
	case "hasTasks":
		return len(mapListFromAny(target["row_tasks"])) > 0
	case "missingAsset":
		return !hasAsset
	case "approval":
		return workTargetHasTaskType(target, crmmodel.TaskTypeApproval)
	case "archived":
		return strings.Contains(workTargetStageText(target), "归档")
	case "personalPending", "overdue", "completedToday":
		// These filters are resolved against todo timestamps before row rendering.
		return true
	default:
		return true
	}
}

func workTargetStageMatches(target map[string]any, stageFilter string) bool {
	if stageFilter == "" {
		return true
	}
	return inputText(target["stage_code"]) == stageFilter || inputText(target["stage_name"]) == stageFilter
}

func workTargetTaskMatches(target map[string]any, taskFilter string) bool {
	if taskFilter == "" {
		return true
	}
	for _, task := range mapListFromAny(target["row_tasks"]) {
		if workTaskFilterKey(task) == taskFilter {
			return true
		}
	}
	return false
}

func workTargetHasTaskType(target map[string]any, taskType string) bool {
	for _, task := range mapListFromAny(target["row_tasks"]) {
		if inputText(task["task_type"]) == taskType || inputText(task["task_action"]) == taskType || inputText(task["action_type"]) == taskType {
			return true
		}
	}
	return false
}

func workTargetStageText(target map[string]any) string {
	return inputText(target["stage_name"]) + " " + inputText(target["current_stage_name"]) + " " + inputText(target["stage_code"])
}

func workTaskFilterKey(task map[string]any) string {
	for _, key := range []string{"task_type", "task_action", "action_type", "id"} {
		value := inputText(task[key])
		if value != "" {
			return value
		}
	}
	return ""
}

func workCustomerFilterOptions(rows []map[string]any) map[string]any {
	stages := map[string]string{}
	tasks := map[string]string{}
	for _, row := range rows {
		collectWorkTargetFilterOptions(row, stages, tasks)
		for _, asset := range mapListFromAny(row["assets"]) {
			collectWorkTargetFilterOptions(asset, stages, tasks)
		}
	}
	return map[string]any{
		"stages": workFilterOptionRows(stages),
		"tasks":  workFilterOptionRows(tasks),
	}
}

func collectWorkTargetFilterOptions(target map[string]any, stages map[string]string, tasks map[string]string) {
	stageKey := inputText(target["stage_code"])
	stageLabel := inputText(target["stage_name"])
	if stageKey != "" && stageLabel == "" {
		stageLabel = stageKey
	}
	if stageKey != "" && stageLabel != "" {
		stages[stageKey] = stageLabel
	}
	for _, task := range mapListFromAny(target["row_tasks"]) {
		taskKey := workTaskFilterKey(task)
		if taskKey == "" {
			continue
		}
		tasks[taskKey] = workTaskDisplayName(task)
	}
}

func workFilterOptionRows(options map[string]string) []map[string]any {
	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return options[keys[i]] < options[keys[j]]
	})
	rows := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, map[string]any{
			"value": key,
			"label": options[key],
		})
	}
	return rows
}

func workTaskDisplayName(task map[string]any) string {
	for _, key := range []string{"task_name", "name"} {
		value := inputText(task[key])
		if value != "" {
			return value
		}
	}
	return WorkTaskTypeName(inputText(task["task_type"]))
}

func filterWorkOperations(rows []map[string]any, keyword string) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		text := inputText(row["title"]) + " " + inputText(row["content"]) + " " + inputText(row["result_value"])
		if containsFold(text, keyword) {
			result = append(result, row)
		}
	}
	return result
}

func workOperationRowsByOperator(rows []map[string]any, staffID uint64) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if inputUint64(row["operator_staff_id"]) == staffID {
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
	if stageName := workStageName(ctx, firstUint64(row, "stage_id", "stageId", "stage_code")); stageName != "" {
		row["stage_name"] = stageName
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
	row["result_value_name"] = workOperationResultDisplayValue(ctx, row, row["result_value"])
	summaryItems := workOperationSummaryItems(ctx, row)
	row["summary"] = workOperationSummary(ctx, row, summaryItems)
	row["summary_items"] = summaryItems
}

func workOperationSummary(_ context.Context, row map[string]any, items []map[string]any) string {
	taskType := inputText(row["task_type"])
	resultValue := inputText(row["result_value"])
	if resultValue == workResultProgress {
		return "保存进度"
	}
	switch taskType {
	case crmmodel.TaskTypeTodo:
		return "完成事项"
	case crmmodel.TaskTypeForm:
		if resultValue == crmmodel.MeetingArrivalNoShow {
			return "记录未到访"
		}
		isChangeSnapshot := inputText(mapFromAny(row["data_snapshot_json"])["snapshot_type"]) == workFormChangeSnapshotType
		if len(items) > 0 {
			if isChangeSnapshot {
				return fmt.Sprintf("变更 %d 项资料", len(items))
			}
			return fmt.Sprintf("补充 %d 项资料", len(items))
		}
		if isChangeSnapshot {
			return "本次未修改资料"
		}
		return "提交资料"
	case crmmodel.TaskTypeApproval:
		if resultValue == "rejected" {
			return "审核驳回"
		}
		return "审核通过"
	case crmmodel.TaskTypeRule:
		if resultValue == "failed" {
			return "自动核验未通过"
		}
		return "自动核验通过"
	default:
		return inputText(row["title"])
	}
}

func workOperationSummaryItems(ctx context.Context, row map[string]any) []map[string]any {
	values := mapFromAny(row["data_snapshot_json"])
	if len(values) == 0 {
		return []map[string]any{}
	}
	if inputText(values["snapshot_type"]) == workFormChangeSnapshotType {
		return workOperationChangeSummaryItems(ctx, row, mapListFromAny(values["changes"]))
	}
	if inputText(row["result_value"]) == workBusinessEventLeadConverted {
		if items := mapListFromAny(values["summary_items"]); len(items) > 0 {
			return items
		}
		return workLeadConversionSummaryItems(
			ctx,
			firstUint64(values, "lead_id", "leadId"),
			firstUint64(values, "customer_id", "customerId"),
			firstUint64(values, "asset_id", "assetId"),
		)
	}
	items := make([]map[string]any, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		value := values[key]
		if inputText(value) == "" {
			continue
		}
		label, displayValue, meta := workOperationSnapshotItem(ctx, row, key, value)
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

func workOperationChangeSummaryItems(ctx context.Context, row map[string]any, changes []map[string]any) []map[string]any {
	items := make([]map[string]any, 0, len(changes))
	for _, change := range changes {
		key := firstText(change, "key")
		if key == "" {
			continue
		}
		before := firstPresent(change, "before")
		after := firstPresent(change, "after")
		beforeLabel, beforeValue, beforeMeta := workOperationSnapshotItem(ctx, row, key, before)
		afterLabel, afterValue, afterMeta := workOperationSnapshotItem(ctx, row, key, after)
		label := afterLabel
		if label == "" {
			label = beforeLabel
		}
		if label == "" {
			continue
		}
		changeType := workOperationChangeType(before, after)
		if changeType == "cleared" {
			afterValue = "已清空"
		}
		if afterValue == "" {
			continue
		}
		item := map[string]any{
			"key":         key,
			"label":       label,
			"value":       afterValue,
			"change_type": changeType,
		}
		if beforeValue != "" {
			item["previous_value"] = beforeValue
		}
		for metaKey, metaValue := range workOperationChangeMeta(afterMeta, beforeMeta, changeType == "cleared") {
			item[metaKey] = metaValue
		}
		items = append(items, item)
	}
	return items
}

func workOperationChangeType(before any, after any) string {
	if emptyWorkFieldValue(before) {
		return "added"
	}
	if emptyWorkFieldValue(after) {
		return "cleared"
	}
	return "updated"
}

func workOperationChangeMeta(after map[string]any, before map[string]any, cleared bool) map[string]any {
	result := map[string]any{}
	if !cleared {
		for key, value := range after {
			result[key] = value
		}
	}
	for _, key := range []string{"group_id", "group_label"} {
		if result[key] == nil && before[key] != nil {
			result[key] = before[key]
		}
	}
	return result
}

func workOperationSnapshotItem(ctx context.Context, row map[string]any, key string, value any) (string, string, map[string]any) {
	if workOperationInternalSnapshotKey(key) {
		return "", "", nil
	}
	switch key {
	case workMeetingStartFieldKey:
		return "预约时间", inputText(value), nil
	case workMeetingDurationFieldKey:
		return "会议时长（分钟）", inputText(value), nil
	case workMeetingResourceFieldKey:
		resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{"id": inputUint64(value)})
		if resource == nil {
			return "会议室", inputText(value), nil
		}
		return "会议室", resource.Name, nil
	case workMeetingArrivalFieldKey:
		switch inputText(value) {
		case crmmodel.MeetingArrivalArrived:
			return "到访结果", "已到访", nil
		case crmmodel.MeetingArrivalNoShow:
			return "到访结果", "未到访", nil
		}
		return "到访结果", "待确认", nil
	case workMeetingNoShowReasonKey:
		return "未到访原因", inputText(value), nil
	case "result_value", "resultValue":
		return "处理结果", workOperationResultDisplayValue(ctx, row, value), nil
	case "result":
		return "处理结果", inputText(value), nil
	case "opinion", "approval_opinion", "approvalOpinion":
		return "审核意见", inputText(value), nil
	case "approval", "approval_result", "approvalResult":
		return "审核结果", workOperationResultDisplayValue(ctx, row, value), nil
	case "remark":
		return "备注", inputText(value), nil
	}
	if strings.HasPrefix(key, "main:") {
		field := strings.TrimPrefix(key, "main:")
		return workMainFieldLabel(field), workMainFieldDisplayValue(ctx, field, value), nil
	}
	if strings.HasPrefix(key, "data:") {
		return workOperationDataFieldSnapshotItem(ctx, strings.TrimPrefix(key, "data:"), value)
	}
	if strings.HasPrefix(key, "field:") {
		return workOperationFormFieldSnapshotItem(ctx, strings.TrimPrefix(key, "field:"), value)
	}
	return "", "", nil
}

func workOperationResultDisplayValue(_ context.Context, _ map[string]any, value any) string {
	resultValue := inputText(value)
	if resultValue == "" {
		return ""
	}
	if resultName := WorkOperationResultName(resultValue); resultName != "" {
		return resultName
	}
	return resultValue
}

func WorkOperationResultName(value string) string {
	switch value {
	case workResultProgress:
		return "保存进度"
	case "completed":
		return "已完成"
	case "submitted":
		return "已提交"
	case "approved":
		return "审核通过"
	case "rejected":
		return "审核驳回"
	case "passed":
		return "核验通过"
	case "failed":
		return "核验未通过"
	case "canceled":
		return "已取消"
	case crmmodel.MeetingArrivalNoShow:
		return "未到访"
	case "entered":
		return "进入阶段"
	case workBusinessEventLeadConverted:
		return "已转化"
	default:
		return ""
	}
}

func workOperationInternalSnapshotKey(key string) bool {
	switch key {
	case "todo_id", "todoId", "submit_mode", "submitMode", "workflow_instance_id", "workflowInstanceId", "raw_result", "duration_ms", "summary_items", "summaryItems", "snapshot_type", "changes":
		return true
	default:
		return false
	}
}

func workOperationDataFieldSnapshotItem(ctx context.Context, fieldIDText string, value any) (string, string, map[string]any) {
	fieldID := inputUint64(fieldIDText)
	if fieldID == 0 {
		return "", "", nil
	}
	field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
	if field == nil {
		return "", "", nil
	}
	displayValue, meta := workDataFieldDisplayValue(ctx, field, value)
	if groupMeta := workDataFieldGroupSummaryMeta(ctx, field); len(groupMeta) > 0 {
		if meta == nil {
			meta = map[string]any{}
		}
		for key, value := range groupMeta {
			meta[key] = value
		}
	}
	return field.Name, displayValue, meta
}

func workDataFieldGroupSummaryMeta(ctx context.Context, field *crmmodel.DataField) map[string]any {
	if field == nil || field.ParentFieldID == 0 {
		return nil
	}
	parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": field.ParentFieldID})
	if parent == nil {
		return nil
	}
	label := strings.TrimSpace(parent.Name)
	if label == "" {
		label = strings.TrimSpace(parent.FieldKey)
	}
	if label == "" {
		return nil
	}
	return map[string]any{
		"group_id":    fmt.Sprintf("data:%d", parent.ID),
		"group_label": label,
	}
}

func workOperationFormFieldSnapshotItem(ctx context.Context, fieldIDText string, value any) (string, string, map[string]any) {
	fieldID := inputUint64(fieldIDText)
	if fieldID == 0 {
		return "", "", nil
	}
	field := crmmodel.NewFormFieldModel().Find(ctx, map[string]any{"id": fieldID})
	if field == nil {
		return "", "", nil
	}
	label := strings.TrimSpace(field.Name)
	displayValue := inputText(value)
	var meta map[string]any
	if field.DataFieldID > 0 {
		dataLabel, dataValue, dataMeta := workOperationDataFieldSnapshotItem(ctx, fmt.Sprintf("%d", field.DataFieldID), value)
		if label == "" {
			label = dataLabel
		}
		if dataValue == "" {
			return "", "", nil
		}
		displayValue = dataValue
		meta = dataMeta
	} else if field.MainField != "" {
		if label == "" {
			label = workMainFieldLabel(field.MainField)
		}
		displayValue = workMainFieldDisplayValue(ctx, field.MainField, value)
	}
	if label == "" || displayValue == "" {
		return "", "", nil
	}
	return label, displayValue, meta
}

func workDataFieldDisplayValue(ctx context.Context, field *crmmodel.DataField, value any) (string, map[string]any) {
	if field == nil || field.ID == 0 {
		return inputText(value), nil
	}
	if workIsAttachmentFieldType(field.FieldType) {
		fileIDs := uint64ListFromAny(value)
		if len(fileIDs) == 0 {
			return "", nil
		}
		files := workUploadFilePayloads(ctx, fileIDs)
		if len(files) == 0 {
			return "", nil
		}
		return fmt.Sprintf("%d 个附件", len(files)), map[string]any{
			"value_type": "files",
			"files":      files,
		}
	}
	text := inputText(value)
	if text == "" {
		return text, nil
	}
	if label := workDataFieldOptionDisplayValue(ctx, field, value); label != "" {
		return label, nil
	}
	return text, nil
}

func workDataFieldOptionDisplayValue(ctx context.Context, field *crmmodel.DataField, value any) string {
	labelMap := workDataFieldOptionLabelMap(ctx, field)
	if len(labelMap) == 0 {
		return ""
	}
	values := stringListFromAny(value)
	if len(values) == 0 {
		if text := inputText(value); text != "" {
			values = []string{text}
		}
	}
	labels := make([]string, 0, len(values))
	for _, optionValue := range values {
		label := labelMap[optionValue]
		if label == "" {
			label = optionValue
		}
		labels = append(labels, label)
	}
	return strings.Join(labels, "、")
}

func workDataFieldOptionLabelMap(ctx context.Context, field *crmmodel.DataField) map[string]string {
	rows := workDataFieldOptionRows(ctx, field)
	labels := make(map[string]string, len(rows))
	for _, row := range rows {
		value := inputText(row["value"])
		if value == "" {
			continue
		}
		label := inputText(row["name"])
		if label == "" {
			label = value
		}
		labels[value] = label
	}
	return labels
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
	case "external_id":
		return "外部线索ID"
	case "city":
		return "城市"
	case "initial_need":
		return "初始诉求"
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
	case "tags":
		return "标签"
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

func workPendingTodoTasks(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) []map[string]any {
	return workPendingTodoTasksWithForm(ctx, staff, customerID, assetID, true)
}

func workPendingTodoRowTasks(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) []map[string]any {
	return workPendingTodoTasksWithForm(ctx, staff, customerID, assetID, false)
}

func workPendingTodoTasksWithForm(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64, withForm bool) []map[string]any {
	return queryWorkTodoTasks(ctx, staff, customerID, assetID, withForm)
}

func workTodoRows(ctx context.Context, staff *WorkStaffSession, customerID uint64, assetID uint64) []map[string]any {
	return queryWorkTodoRows(ctx, staff, customerID, assetID, false)
}

func workTodoStatusName(status string) string {
	switch status {
	case crmmodel.WorkTodoStatusPending:
		return "待处理"
	case crmmodel.WorkTodoStatusDone:
		return "已完成"
	case crmmodel.WorkTodoStatusCanceled:
		return "已取消"
	default:
		return status
	}
}

func workStageName(ctx context.Context, stageID uint64) string {
	if stageID == 0 {
		return ""
	}
	stage := crmmodel.NewStageModel().Find(ctx, map[string]any{
		"id": stageID,
	})
	if stage == nil {
		return ""
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
		"customer_id":          customerID,
		"asset_id":             assetID,
		"workflow_instance_id": uint64(0),
		"status":               crmmodel.StatusEnabled,
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
		"customer_id":          customerID,
		"asset_id":             assetID,
		"workflow_instance_id": uint64(0),
		"status":               crmmodel.StatusEnabled,
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
		if field == nil || field.FieldType == "group" {
			continue
		}
		result[field.ID] = field
	}
	return result
}

func workRuleAssets(ctx context.Context, customerID uint64) []map[string]any {
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

func attachWorkEntityDataValues(ctx context.Context, row map[string]any, values map[string]any, templateCateID uint64) {
	row["data_values"] = values
	row["data_file_values"] = workDataFileValues(ctx, values)
	row["data_value_labels"] = workDataValueLabels(ctx, values)
	row["data_completeness"] = workDataCompletenessTemplates(ctx, templateCateID, values)
}

func workDataFileValues(ctx context.Context, values map[string]any) map[string]any {
	filesByField := map[string]any{}
	for key, value := range values {
		if !strings.HasPrefix(key, "data:") {
			continue
		}
		fieldID := inputUint64(strings.TrimPrefix(key, "data:"))
		if fieldID == 0 {
			continue
		}
		field := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"id":     fieldID,
			"status": crmmodel.StatusEnabled,
		})
		if field == nil || !workIsAttachmentFieldType(field.FieldType) {
			continue
		}
		if files := workUploadFilePayloads(ctx, uint64ListFromAny(value)); len(files) > 0 {
			filesByField[key] = files
		}
	}
	return filesByField
}

func workDataDetailSections(ctx context.Context, targetType string, cateID uint64, values map[string]any) []map[string]any {
	if cateID == 0 {
		return []map[string]any{}
	}
	templates := crmmodel.NewDataTemplateModel().Select(ctx, map[string]any{
		"cate_id": cateID,
		"status":  crmmodel.StatusEnabled,
	})
	result := make([]map[string]any, 0, len(templates))
	for _, template := range templates {
		if template == nil || template.DisplayMode == crmmodel.DataTemplateDisplayHidden {
			continue
		}
		fields := workDataDetailFields(ctx, template.ID, values)
		if len(fields) == 0 {
			continue
		}
		filled := 0
		for _, field := range fields {
			if !booleanFromAny(field["empty"]) {
				filled++
			}
		}
		if template.DisplayMode == crmmodel.DataTemplateDisplayFilled && filled == 0 {
			continue
		}
		percent := int(math.Round(float64(filled) / float64(len(fields)) * 100))
		result = append(result, map[string]any{
			"id":          fmt.Sprintf("%s:%d", targetType, template.ID),
			"name":        template.Name,
			"target_type": targetType,
			"template_id": template.ID,
			"filled":      filled,
			"total":       len(fields),
			"percent":     percent,
			"fields":      fields,
		})
	}
	return result
}

func workDataDetailFields(ctx context.Context, templateID uint64, values map[string]any) []map[string]any {
	fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	})
	parentNames := workDataCompletenessParentNames(ctx, fields)
	result := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		if field == nil || field.FieldType == "group" {
			continue
		}
		key := fmt.Sprintf("data:%d", field.ID)
		rawValue := values[key]
		empty := emptyWorkFieldValue(rawValue)
		displayValue := ""
		meta := map[string]any{}
		if !empty {
			displayValue, meta = workDataFieldDisplayValue(ctx, field, rawValue)
			if displayValue == "" {
				empty = true
			}
		}
		item := map[string]any{
			"key":        key,
			"label":      field.Name,
			"value":      displayValue,
			"value_type": "text",
			"empty":      empty,
		}
		if group := parentNames[field.ParentFieldID]; group != "" {
			item["group"] = group
		}
		for metaKey, metaValue := range meta {
			item[metaKey] = metaValue
		}
		result = append(result, item)
	}
	return result
}

func workDataCompletenessTemplates(ctx context.Context, templateCateID uint64, values map[string]any) []map[string]any {
	if templateCateID == 0 {
		return []map[string]any{}
	}
	templates := crmmodel.NewDataTemplateModel().Select(ctx, map[string]any{
		"cate_id": templateCateID,
		"status":  crmmodel.StatusEnabled,
	})
	result := make([]map[string]any, 0, len(templates))
	for _, template := range templates {
		if template == nil {
			continue
		}
		summary := workDataCompletenessTemplate(ctx, template, values)
		if inputInt(summary["total"]) == 0 {
			continue
		}
		result = append(result, summary)
	}
	return result
}

func workDataCompletenessTemplate(ctx context.Context, template *crmmodel.DataTemplate, values map[string]any) map[string]any {
	fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": template.ID,
		"status":           crmmodel.StatusEnabled,
	})
	total := 0
	filled := 0
	missing := make([]string, 0)
	parentNames := workDataCompletenessParentNames(ctx, fields)
	probeFields := elevenDimensionProbeFields(fields, parentNames)
	isProbe := isElevenDimensionProbeTemplate(probeFields)
	for _, field := range fields {
		if field == nil || field.FieldType == "group" {
			continue
		}
		if isProbe {
			if _, ok := probeFields[field.ID]; !ok {
				continue
			}
		}
		total++
		value := values[fmt.Sprintf("data:%d", field.ID)]
		if !emptyWorkFieldValue(value) {
			filled++
			continue
		}
		missing = append(missing, workDataCompletenessFieldLabel(field, parentNames))
	}
	percent := 0
	if total > 0 {
		percent = int(math.Round(float64(filled) / float64(total) * 100))
	}
	return map[string]any{
		"template_id":   template.ID,
		"template_name": template.Name,
		"name":          template.Name,
		"total":         total,
		"filled":        filled,
		"percent":       percent,
		"missing":       missing,
		"is_probe":      isProbe,
	}
}

func workDataCompletenessParentNames(ctx context.Context, fields []*crmmodel.DataField) map[uint64]string {
	parentIDs := map[uint64]bool{}
	for _, field := range fields {
		if field != nil && field.ParentFieldID > 0 {
			parentIDs[field.ParentFieldID] = true
		}
	}
	result := map[uint64]string{}
	for parentID := range parentIDs {
		if parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": parentID}); parent != nil {
			result[parentID] = parent.Name
		}
	}
	return result
}

func workDataCompletenessFieldLabel(field *crmmodel.DataField, parentNames map[uint64]string) string {
	if field == nil {
		return ""
	}
	if field.ParentFieldID == 0 {
		return field.Name
	}
	if parentName := strings.TrimSpace(parentNames[field.ParentFieldID]); parentName != "" {
		return parentName + "/" + field.Name
	}
	return field.Name
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

func attachWorkTaskForm(ctx context.Context, task map[string]any) {
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
	groupCache := map[uint64]*crmmodel.DataField{}
	for _, field := range fields {
		attachWorkFormFieldOptions(ctx, field, groupCache)
	}
	if booleanFromAny(task["meeting_enabled"]) {
		fields = workMeetingTaskFormFields(ctx, task, fields)
	}
	if booleanFromAny(task["customer_follow_enabled"]) {
		fields = append(fields, workCustomerFollowFormFields()...)
	}
	form["fields"] = fields
	task["form"] = form
}

func workDataFieldOptionsForField(ctx context.Context, field *crmmodel.DataField) []map[string]any {
	if field == nil || field.ID == 0 {
		return []map[string]any{}
	}
	rows := workDataFieldOptionRows(ctx, field)
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

func workDataFieldOptionRows(ctx context.Context, field *crmmodel.DataField) []map[string]any {
	if field == nil || field.ID == 0 {
		return []map[string]any{}
	}
	if field.OptionSetID > 0 {
		return crmmodel.NewOptionSetItemModel().SelectMap(ctx, map[string]any{
			"option_set_id": field.OptionSetID,
			"status":        crmmodel.StatusEnabled,
		}, map[string]any{
			"field": "main.name, main.value, main.sort",
			"order": "main.sort asc, main.id asc",
		})
	}
	return crmmodel.NewDataFieldOptionModel().SelectMap(ctx, map[string]any{
		"data_field_id": field.ID,
	}, map[string]any{
		"field": "main.name, main.value, main.sort",
		"order": "main.sort asc, main.id asc",
	})
}

func attachWorkFormFieldOptions(ctx context.Context, field map[string]any, groupCache map[uint64]*crmmodel.DataField) {
	if fieldID := inputUint64(field["data_field_id"]); fieldID > 0 {
		dataField := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": fieldID})
		if dataField != nil {
			field["field_key"] = dataField.FieldKey
			field["field_type"] = dataField.FieldType
			field["default_value"] = dataField.DefaultValue
			attachWorkFormFieldGroup(ctx, field, dataField, groupCache)
			if dataField.FieldType == "group" {
				field["options"] = []map[string]any{}
				field["children"] = workDataFieldChildFormFields(ctx, dataField, field)
				return
			}
			field["options"] = workDataFieldOptionsForField(ctx, dataField)
		}
		return
	}
	mainField := inputText(field["main_field"])
	field["field_type"] = mainFieldInputType(mainField)
	field["options"] = mainFieldOptions(ctx, mainField)
}

func attachWorkFormFieldGroup(ctx context.Context, target map[string]any, field *crmmodel.DataField, cache map[uint64]*crmmodel.DataField) {
	if target == nil || field == nil || field.ParentFieldID == 0 || cache == nil {
		return
	}
	group, exists := cache[field.ParentFieldID]
	if !exists {
		group = crmmodel.NewDataFieldModel().Find(ctx, map[string]any{
			"id":               field.ParentFieldID,
			"data_template_id": field.DataTemplateID,
			"field_type":       "group",
			"status":           crmmodel.StatusEnabled,
		})
		cache[field.ParentFieldID] = group
	}
	if group == nil {
		return
	}
	target["group_id"] = group.ID
	target["group_key"] = group.FieldKey
	target["group_label"] = group.Name
}

func workDataFieldChildFormFields(ctx context.Context, group *crmmodel.DataField, formField map[string]any) []map[string]any {
	if group == nil || group.ID == 0 {
		return []map[string]any{}
	}
	children := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": group.DataTemplateID,
		"parent_field_id":  group.ID,
		"status":           crmmodel.StatusEnabled,
	})
	result := make([]map[string]any, 0, len(children))
	cateID := inputUint64(formField["data_template_cate_id"])
	if cateID == 0 {
		cateID = workDataFieldTemplateCateID(ctx, group)
	}
	for _, child := range children {
		if child == nil || child.FieldType == "group" {
			continue
		}
		row := map[string]any{
			"id":                    child.ID,
			"name":                  child.Name,
			"field_key":             child.FieldKey,
			"field_type":            child.FieldType,
			"default_value":         child.DefaultValue,
			"data_template_id":      child.DataTemplateID,
			"data_template_cate_id": cateID,
			"data_field_id":         child.ID,
			"parent_field_id":       group.ID,
			"required":              booleanFromAny(formField["required"]),
			"readonly":              booleanFromAny(formField["readonly"]),
			"visible_when_field_id": inputUint64(formField["visible_when_field_id"]),
			"visible_when_operator": inputText(formField["visible_when_operator"]),
			"visible_when_value":    inputText(formField["visible_when_value"]),
			"sort":                  child.Sort,
			"options":               workDataFieldOptionsForField(ctx, child),
		}
		result = append(result, row)
	}
	return result
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

func workActionValues(payload map[string]any) map[string]any {
	values := mapFromAny(payload["values"])
	for _, key := range []string{"todo_id", "todoId", "workflow_instance_id", "workflowInstanceId", "product_ids", "productIds", "submit_mode", "submitMode", "result", "opinion", "approval"} {
		if _, exists := values[key]; !exists && payload[key] != nil {
			values[key] = payload[key]
		}
	}
	return values
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
	case "remark", "initial_need":
		return "textarea"
	case "gender":
		return "radio"
	case "tags":
		return "customer_tags"
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
	case "tags":
		return CustomerTagOptions(ctx)
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

func workPublicResourceOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewPublicResourceModel().SelectMap(ctx, map[string]any{
		"status": crmmodel.StatusEnabled,
	}, map[string]any{
		"field": "main.id, main.name, main.location, main.sort",
		"order": "main.sort asc, main.id asc",
	})
	options := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := inputUint64(row["id"])
		name := inputText(row["name"])
		if id == 0 || name == "" {
			continue
		}
		label := name
		if location := inputText(row["location"]); location != "" {
			label += "（" + location + "）"
		}
		options = append(options, map[string]any{
			"id":    id,
			"name":  label,
			"label": label,
			"value": fmt.Sprintf("%d", id),
		})
	}
	return options
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

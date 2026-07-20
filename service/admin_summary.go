package service

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const adminSummaryTrendDays = 14
const adminSummaryMaxRangeDays = 366

type AdminSummaryService struct{}

type AdminSummaryQuery struct {
	Mode         string
	WorkflowID   uint64
	DepartmentID uint64
	StaffID      uint64
	DateFrom     string
	DateTo       string
}

type adminSummaryRange struct {
	Start time.Time
	End   time.Time
}

type adminSummaryStageInfo struct {
	ID         uint64
	WorkflowID uint64
	Name       string
	Sort       int
}

type adminSummaryTaskInfo struct {
	ID      uint64
	StageID uint64
	Type    string
	FormID  uint64
}

type adminSummaryStaffStat struct {
	ID                  uint64
	Name                string
	DepartmentName      string
	CompletedTaskCount  int
	TransitionCount     int
	PendingTaskCount    int
	OnTimeEligibleCount int
	OnTimeCount         int
	DurationSeconds     float64
	DurationSampleCount int
	LastActiveAt        time.Time
}

type adminSummaryNodeBacklogStat struct {
	Stage adminSummaryStageInfo
	Count int
	Days  []int
}

type adminSummaryProbeFieldStat struct {
	Key    string
	Name   string
	Total  int
	Filled int
}

func NewAdminSummaryService() AdminSummaryService {
	return AdminSummaryService{}
}

func (AdminSummaryService) Summary(ctx context.Context, queries ...AdminSummaryQuery) (map[string]any, error) {
	query := AdminSummaryQuery{}
	if len(queries) > 0 {
		query = queries[0]
	}
	query.Mode = adminSummaryMode(query.Mode)
	rangeValue := adminSummaryDateRange(query.DateFrom, query.DateTo)
	workflows := adminSummaryWorkflowOptions(ctx)
	if query.WorkflowID == 0 && (query.Mode == "business" || query.Mode == "all") && len(workflows) > 0 {
		query.WorkflowID = inputUint64(workflows[0]["id"])
	}

	result := map[string]any{
		"filters": map[string]any{
			"mode":          query.Mode,
			"workflow_id":   query.WorkflowID,
			"department_id": query.DepartmentID,
			"staff_id":      query.StaffID,
			"date_from":     rangeValue.Start.Format("2006-01-02"),
			"date_to":       rangeValue.End.Add(-time.Nanosecond).Format("2006-01-02"),
		},
		"filter_options": map[string]any{
			"workflows":   workflows,
			"departments": adminSummaryDepartmentOptions(ctx),
			"staff":       adminSummaryStaffOptions(ctx),
		},
		"generated_at": time.Now(),
	}

	if query.Mode == "all" || query.Mode == "business" {
		adminSummaryMerge(result, adminSummaryBusiness(ctx, query, rangeValue))
	}
	if query.Mode == "all" || query.Mode == "finance" {
		result["finance_summary"] = adminSummaryFinance(ctx, query, rangeValue)
	}
	if query.Mode == "all" || query.Mode == "performance" {
		staffRows := adminSummaryPerformance(ctx, query, rangeValue)
		result["staff_ranking"] = staffRows
		result["staff_output"] = staffRows
	}
	return result, nil
}

func adminSummaryMode(value string) string {
	switch strings.TrimSpace(value) {
	case "business", "finance", "performance":
		return strings.TrimSpace(value)
	default:
		return "all"
	}
}

func adminSummaryDateRange(dateFrom string, dateTo string) adminSummaryRange {
	start := adminSummaryTrendStart(adminSummaryTrendDays)
	end := workBeginningOfDay(time.Now()).AddDate(0, 0, 1)
	if parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(dateFrom), time.Local); err == nil {
		start = parsed
	}
	if parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(dateTo), time.Local); err == nil {
		end = parsed.AddDate(0, 0, 1)
	}
	if !start.Before(end) {
		start = end.AddDate(0, 0, -adminSummaryTrendDays)
	}
	if start.Before(end.AddDate(0, 0, -adminSummaryMaxRangeDays)) {
		start = end.AddDate(0, 0, -adminSummaryMaxRangeDays)
	}
	return adminSummaryRange{Start: start, End: end}
}

func adminSummaryInRange(value time.Time, rangeValue adminSummaryRange) bool {
	return !value.IsZero() && !value.Before(rangeValue.Start) && value.Before(rangeValue.End)
}

func adminSummaryMerge(target map[string]any, source map[string]any) {
	for key, value := range source {
		target[key] = value
	}
}

func adminSummaryBusiness(ctx context.Context, query AdminSummaryQuery, rangeValue adminSummaryRange) map[string]any {
	instanceFilters := map[string]any{}
	if query.WorkflowID > 0 {
		instanceFilters["workflow_id"] = query.WorkflowID
	}
	instances := crmmodel.NewWorkflowInstanceModel().Select(ctx, instanceFilters)
	activeInstances := make([]*crmmodel.WorkflowInstance, 0, len(instances))
	for _, instance := range instances {
		if instance != nil && instance.Status == crmmodel.ProgressStatusActive {
			activeInstances = append(activeInstances, instance)
		}
	}

	todoFilters := map[string]any{}
	if query.WorkflowID > 0 {
		todoFilters["workflow_id"] = query.WorkflowID
	}
	todos := crmmodel.NewWorkTodoModel().Select(ctx, todoFilters)
	tasks := adminSummaryTaskInfos(ctx)
	taskByID := adminSummaryTaskByID(tasks)
	pendingTodos := adminSummaryPendingManualTodos(todos, taskByID)
	eventFilters := map[string]any{}
	if query.WorkflowID > 0 {
		eventFilters["workflow_id"] = query.WorkflowID
	}
	events := crmmodel.NewStatEventModel().Select(ctx, eventFilters)
	stages := adminSummaryStageInfosForWorkflow(ctx, query.WorkflowID)
	customers := crmmodel.NewCustomerModel().Select(ctx, map[string]any{})
	assets := crmmodel.NewCustomerAssetModel().Select(ctx, map[string]any{})
	trendRows := adminSummaryBusinessTrend(customers, assets, todos, events, taskByID, query.WorkflowID, rangeValue)
	funnelRows := adminSummaryStageFunnel(events, stages, query.WorkflowID, rangeValue)

	return map[string]any{
		"metrics":          adminSummaryBusinessMetrics(instances, activeInstances, pendingTodos, rangeValue),
		"growth_trend":     trendRows,
		"execution_trend":  trendRows,
		"funnel":           funnelRows,
		"pipeline_funnel":  funnelRows,
		"node_backlog":     adminSummaryNodeBacklog(ctx, activeInstances, stages, pendingTodos),
		"task_breakdown":   adminSummaryTaskBreakdown(pendingTodos, taskByID),
		"probe_summary":    adminSummaryProbeSummary(ctx, assets, query.WorkflowID),
		"field_statistics": adminSummaryFieldStatistics(ctx, query, rangeValue),
	}
}

func adminSummaryMetric(key string, name string, value int, description string) map[string]any {
	return map[string]any{
		"key":         key,
		"name":        name,
		"value":       value,
		"description": description,
	}
}

func adminSummaryPercentMetric(key string, name string, value int, description string) map[string]any {
	row := adminSummaryMetric(key, name, value, description)
	row["unit"] = "%"
	return row
}

func adminSummaryUnitMetric(key string, name string, value int, unit string, description string) map[string]any {
	row := adminSummaryMetric(key, name, value, description)
	row["unit"] = unit
	return row
}

func adminSummaryBusinessMetrics(instances []*crmmodel.WorkflowInstance, activeInstances []*crmmodel.WorkflowInstance, pendingTodos []*crmmodel.WorkTodo, rangeValue adminSummaryRange) []map[string]any {
	completed := 0
	terminated := 0
	for _, instance := range instances {
		if instance == nil {
			continue
		}
		if instance.CompletedAt != nil && adminSummaryInRange(*instance.CompletedAt, rangeValue) {
			completed++
		}
		if instance.TerminatedAt != nil && adminSummaryInRange(*instance.TerminatedAt, rangeValue) {
			terminated++
		}
	}
	overdue := 0
	for _, todo := range pendingTodos {
		if todo != nil && todo.DueAt != nil && todo.DueAt.Before(time.Now()) {
			overdue++
		}
	}
	avgDays := 0
	if len(activeInstances) > 0 {
		totalDays := 0
		for _, instance := range activeInstances {
			totalDays += workStageDwellDays(instance.StartedAt)
		}
		avgDays = (totalDays + len(activeInstances)/2) / len(activeInstances)
	}
	closed := completed + terminated
	return []map[string]any{
		adminSummaryMetric("active_instances", "进行中流程", len(activeInstances), "当前仍在流转的流程实例"),
		adminSummaryMetric("pending_todos", "待办任务", len(pendingTodos), "当前尚未完成的人工任务"),
		adminSummaryMetric("overdue_todos", "超期待办", overdue, "已超过办理期限的人工任务"),
		adminSummaryMetric("completed_period", "期间完成", completed, "筛选日期内完成的流程"),
		adminSummaryPercentMetric("completion_rate", "完成率", adminSummaryPercent(completed, closed), "期间完成数占已结束流程的比例"),
		adminSummaryUnitMetric("avg_stage_days", "平均停留", avgDays, "天", "进行中流程当前阶段平均停留天数"),
	}
}

func adminSummaryWorkflowOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewWorkflowModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, map[string]any{"id": row.ID, "name": row.Name})
		}
	}
	return result
}

func adminSummaryDepartmentOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewDepartmentModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, map[string]any{"id": row.ID, "name": row.Name})
		}
	}
	return result
}

func adminSummaryStaffOptions(ctx context.Context) []map[string]any {
	rows := crmmodel.NewStaffModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, map[string]any{"id": row.ID, "name": row.Name, "department_id": row.DepartmentID})
		}
	}
	return result
}

func adminSummaryTaskByID(tasks []adminSummaryTaskInfo) map[uint64]adminSummaryTaskInfo {
	result := make(map[uint64]adminSummaryTaskInfo, len(tasks))
	for _, task := range tasks {
		result[task.ID] = task
	}
	return result
}

func adminSummaryPendingManualTodos(todos []*crmmodel.WorkTodo, tasks map[uint64]adminSummaryTaskInfo) []*crmmodel.WorkTodo {
	result := make([]*crmmodel.WorkTodo, 0, len(todos))
	for _, todo := range todos {
		if todo == nil || todo.Status != crmmodel.WorkTodoStatusPending || !adminSummaryIsManualTask(tasks, todo.TaskID) {
			continue
		}
		result = append(result, todo)
	}
	return result
}

func adminSummaryIsManualTask(tasks map[uint64]adminSummaryTaskInfo, taskID uint64) bool {
	task, exists := tasks[taskID]
	return exists && task.Type != crmmodel.TaskTypeRule
}

func adminSummaryAmountMetric(key string, name string, value float64, description string) map[string]any {
	return map[string]any{
		"key":         key,
		"name":        name,
		"value":       value,
		"unit":        "元",
		"description": description,
	}
}

func adminSummaryFinance(ctx context.Context, query AdminSummaryQuery, rangeValue adminSummaryRange) map[string]any {
	filters := map[string]any{}
	if query.DepartmentID > 0 {
		filters["department_id"] = query.DepartmentID
	}
	if query.StaffID > 0 {
		filters["staff_id"] = query.StaffID
	}
	rows := crmmodel.NewFinanceLedgerModel().Select(ctx, filters)
	ledgers := make([]*crmmodel.FinanceLedger, 0, len(rows))
	for _, ledger := range rows {
		if ledger == nil || !adminSummaryInRange(ledger.CreatedAt, rangeValue) {
			continue
		}
		if query.DepartmentID > 0 && ledger.DepartmentID != query.DepartmentID || query.StaffID > 0 && ledger.StaffID != query.StaffID {
			continue
		}
		ledgers = append(ledgers, ledger)
	}
	income, expense := adminSummaryFinanceTotals(ledgers)
	return map[string]any{
		"metrics": []map[string]any{
			adminSummaryAmountMetric("finance_income", "期间收入", income, "筛选日期内录入的收入流水"),
			adminSummaryAmountMetric("finance_expense", "期间支出", expense, "筛选日期内录入的支出流水"),
			adminSummaryAmountMetric("finance_net", "期间净额", income-expense, "筛选日期内收入减支出"),
			adminSummaryMetric("finance_ledger_count", "流水数量", len(ledgers), "筛选日期内录入的流水记录"),
		},
		"trend":          adminSummaryFinanceTrendRange(ledgers, rangeValue),
		"type_breakdown": adminSummaryFinanceTypeRows(ledgers),
	}
}

func adminSummaryFinanceTrendRange(ledgers []*crmmodel.FinanceLedger, rangeValue adminSummaryRange) []map[string]any {
	days := int(rangeValue.End.Sub(rangeValue.Start).Hours() / 24)
	if days <= 0 {
		return []map[string]any{}
	}
	rows := make([]map[string]any, 0, days)
	indexes := map[string]int{}
	for index := 0; index < days; index++ {
		day := rangeValue.Start.AddDate(0, 0, index)
		key := day.Format("2006-01-02")
		indexes[key] = index
		rows = append(rows, map[string]any{
			"date": key, "label": day.Format("01-02"), "income_amount": 0,
			"expense_amount": 0, "net_amount": 0, "ledger_count": 0,
		})
	}
	for _, ledger := range ledgers {
		if ledger == nil {
			continue
		}
		index, exists := indexes[workBeginningOfDay(ledger.CreatedAt).Format("2006-01-02")]
		if !exists {
			continue
		}
		row := rows[index]
		if ledger.Direction == crmmodel.FinanceDirectionExpense {
			row["expense_amount"] = numericValue(row["expense_amount"]) + ledger.Amount
		} else {
			row["income_amount"] = numericValue(row["income_amount"]) + ledger.Amount
		}
		row["ledger_count"] = inputInt(row["ledger_count"]) + 1
	}
	for _, row := range rows {
		row["net_amount"] = numericValue(row["income_amount"]) - numericValue(row["expense_amount"])
	}
	return rows
}

func adminSummaryFinanceTotals(ledgers []*crmmodel.FinanceLedger) (float64, float64) {
	var income float64
	var expense float64
	for _, ledger := range ledgers {
		if ledger == nil {
			continue
		}
		switch ledger.Direction {
		case crmmodel.FinanceDirectionExpense:
			expense += ledger.Amount
		default:
			income += ledger.Amount
		}
	}
	return income, expense
}

type adminSummaryFinanceTypeStat struct {
	Key       string
	Name      string
	Direction string
	Count     int
	Amount    float64
}

func adminSummaryFinanceTypeRows(ledgers []*crmmodel.FinanceLedger) []map[string]any {
	stats := map[string]*adminSummaryFinanceTypeStat{}
	totalsByDirection := map[string]float64{}
	for _, ledger := range ledgers {
		if ledger == nil {
			continue
		}
		key := ledger.FinanceTypeCode
		if key == "" {
			key = strconv.FormatUint(ledger.FinanceTypeID, 10)
		}
		if key == "" || key == "0" {
			key = "_unknown"
		}
		stat := stats[key]
		if stat == nil {
			stat = &adminSummaryFinanceTypeStat{
				Key:       key,
				Name:      ledger.FinanceTypeName,
				Direction: ledger.Direction,
			}
			if stat.Name == "" {
				stat.Name = "未分类财务"
			}
			stats[key] = stat
		}
		stat.Count++
		stat.Amount += ledger.Amount
		totalsByDirection[ledger.Direction] += ledger.Amount
	}
	rows := make([]*adminSummaryFinanceTypeStat, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, stat)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Amount != rows[j].Amount {
			return rows[i].Amount > rows[j].Amount
		}
		return rows[i].Key < rows[j].Key
	})
	result := make([]map[string]any, 0, len(rows))
	for _, stat := range rows {
		percent := 0
		if total := totalsByDirection[stat.Direction]; total > 0 {
			percent = int(stat.Amount / total * 100)
		}
		result = append(result, map[string]any{
			"key":       stat.Key,
			"name":      stat.Name,
			"direction": stat.Direction,
			"count":     stat.Count,
			"amount":    stat.Amount,
			"percent":   percent,
		})
	}
	return result
}

func adminSummaryBusinessTrend(customers []*crmmodel.Customer, assets []*crmmodel.CustomerAsset, todos []*crmmodel.WorkTodo, events []*crmmodel.StatEvent, tasks map[uint64]adminSummaryTaskInfo, workflowID uint64, rangeValue adminSummaryRange) []map[string]any {
	days := int(rangeValue.End.Sub(rangeValue.Start).Hours() / 24)
	if days <= 0 {
		return []map[string]any{}
	}
	rows := make([]map[string]any, 0, days)
	indexes := map[string]int{}
	for index := 0; index < days; index++ {
		day := rangeValue.Start.AddDate(0, 0, index)
		key := day.Format("2006-01-02")
		indexes[key] = index
		rows = append(rows, map[string]any{
			"date": key, "label": day.Format("01-02"), "customer_count": 0,
			"asset_count": 0, "task_count": 0, "transition_count": 0,
		})
	}
	for _, customer := range customers {
		if customer != nil {
			adminSummaryIncrementTrend(rows, indexes, rangeValue.Start, rangeValue.End, customer.CreatedAt, "customer_count")
		}
	}
	for _, asset := range assets {
		if asset != nil {
			adminSummaryIncrementTrend(rows, indexes, rangeValue.Start, rangeValue.End, asset.CreatedAt, "asset_count")
		}
	}
	for _, todo := range todos {
		if todo == nil || todo.Status != crmmodel.WorkTodoStatusDone || todo.CompletedAt == nil ||
			workflowID > 0 && todo.WorkflowID != workflowID || !adminSummaryIsManualTask(tasks, todo.TaskID) {
			continue
		}
		adminSummaryIncrementTrend(rows, indexes, rangeValue.Start, rangeValue.End, *todo.CompletedAt, "task_count")
	}
	for _, event := range events {
		if event == nil || event.EventType != crmmodel.StatEventTypeTransition || workflowID > 0 && event.WorkflowID != workflowID {
			continue
		}
		adminSummaryIncrementTrend(rows, indexes, rangeValue.Start, rangeValue.End, event.EventAt, "transition_count")
	}
	return rows
}

func adminSummaryIncrementTrend(rows []map[string]any, indexes map[string]int, start time.Time, end time.Time, eventAt time.Time, key string) {
	if eventAt.Before(start) || !eventAt.Before(end) {
		return
	}
	index, exists := indexes[workBeginningOfDay(eventAt).Format("2006-01-02")]
	if !exists {
		return
	}
	rows[index][key] = inputInt(rows[index][key]) + 1
}

func adminSummaryTrendStart(days int) time.Time {
	return workBeginningOfDay(time.Now()).AddDate(0, 0, -days+1)
}

func adminSummaryStageFunnel(events []*crmmodel.StatEvent, stages []adminSummaryStageInfo, workflowID uint64, rangeValue adminSummaryRange) []map[string]any {
	if len(stages) == 0 {
		return []map[string]any{}
	}
	cohort := map[uint64]bool{}
	for _, event := range events {
		if event != nil && event.EventType == crmmodel.StatEventTypeTransition && event.ToStageID == stages[0].ID &&
			(workflowID == 0 || event.WorkflowID == workflowID) && adminSummaryInRange(event.EventAt, rangeValue) {
			cohort[event.WorkflowInstanceID] = true
		}
	}
	enteredByStage := map[uint64]map[uint64]bool{}
	for _, event := range events {
		if event == nil || event.EventType != crmmodel.StatEventTypeTransition || event.ToStageID == 0 ||
			workflowID > 0 && event.WorkflowID != workflowID || !cohort[event.WorkflowInstanceID] || !event.EventAt.Before(rangeValue.End) {
			continue
		}
		if enteredByStage[event.ToStageID] == nil {
			enteredByStage[event.ToStageID] = map[uint64]bool{}
		}
		enteredByStage[event.ToStageID][event.WorkflowInstanceID] = true
	}
	total := len(cohort)
	previous := total
	rows := make([]map[string]any, 0, len(stages))
	for _, stage := range stages {
		count := len(enteredByStage[stage.ID])
		row := adminSummaryBreakdownRow(adminSummaryStageKey(stage.ID), stage.Name, count, total)
		adminSummaryAttachDropFields(row, previous, count)
		rows = append(rows, row)
		previous = count
	}
	return rows
}

func adminSummaryAttachDropFields(row map[string]any, previousCount int, count int) {
	if row == nil {
		return
	}
	dropCount := previousCount - count
	if dropCount < 0 {
		dropCount = 0
	}
	dropPercent := 0
	if previousCount > 0 {
		dropPercent = int(float64(dropCount) / float64(previousCount) * 100)
	}
	row["previous_count"] = previousCount
	row["drop_count"] = dropCount
	row["drop_percent"] = dropPercent
}

func adminSummaryNodeBacklog(ctx context.Context, targets []*crmmodel.WorkflowInstance, stages []adminSummaryStageInfo, todos []*crmmodel.WorkTodo) []map[string]any {
	stageByID := adminSummaryStageByID(stages)
	stats := map[uint64]*adminSummaryNodeBacklogStat{}
	for _, target := range targets {
		if target == nil {
			continue
		}
		stageID := target.StageID
		stat := stats[stageID]
		if stat == nil {
			stage := stageByID[stageID]
			if stage.ID == 0 {
				stage = adminSummaryStageInfo{Name: "未进入阶段"}
			}
			stat = &adminSummaryNodeBacklogStat{Stage: stage}
			stats[stageID] = stat
		}
		stat.Count++
		stat.Days = append(stat.Days, workStageDwellDays(workStageEnteredAt(ctx, target)))
	}

	pendingTodosByStage := map[uint64]int{}
	for _, todo := range todos {
		if todo == nil || todo.Status != crmmodel.WorkTodoStatusPending {
			continue
		}
		pendingTodosByStage[todo.StageID]++
	}

	rows := make([]map[string]any, 0, len(stats))
	total := len(targets)
	for _, stage := range stages {
		if stat := stats[stage.ID]; stat != nil {
			rows = append(rows, adminSummaryNodeBacklogRow(stat, pendingTodosByStage[stage.ID], total))
			delete(stats, stage.ID)
		}
	}
	for stageID, stat := range stats {
		rows = append(rows, adminSummaryNodeBacklogRow(stat, pendingTodosByStage[stageID], total))
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left := inputInt(rows[i]["stale_7d"])*100000 + inputInt(rows[i]["max_days"])*1000 + inputInt(rows[i]["count"])
		right := inputInt(rows[j]["stale_7d"])*100000 + inputInt(rows[j]["max_days"])*1000 + inputInt(rows[j]["count"])
		if left != right {
			return left > right
		}
		return inputText(rows[i]["name"]) < inputText(rows[j]["name"])
	})
	return rows
}

func adminSummaryNodeBacklogRow(stat *adminSummaryNodeBacklogStat, pendingTodoCount int, total int) map[string]any {
	sumDays := 0
	maxDays := 0
	stale3d := 0
	stale7d := 0
	stale15d := 0
	for _, days := range stat.Days {
		sumDays += days
		if days > maxDays {
			maxDays = days
		}
		if days >= 3 {
			stale3d++
		}
		if days >= 7 {
			stale7d++
		}
		if days >= 15 {
			stale15d++
		}
	}
	avgDays := 0
	if stat.Count > 0 {
		avgDays = (sumDays + stat.Count/2) / stat.Count
	}
	row := adminSummaryBreakdownRow(adminSummaryStageKey(stat.Stage.ID), stat.Stage.Name, stat.Count, total)
	row["pending_todo_count"] = pendingTodoCount
	row["avg_days"] = avgDays
	row["max_days"] = maxDays
	row["stale_3d"] = stale3d
	row["stale_7d"] = stale7d
	row["stale_15d"] = stale15d
	return row
}

func adminSummaryTaskBreakdown(todos []*crmmodel.WorkTodo, tasks map[uint64]adminSummaryTaskInfo) []map[string]any {
	counts := map[string]int{}
	total := 0
	for _, todo := range todos {
		if todo == nil {
			continue
		}
		key := tasks[todo.TaskID].Type
		if key == "" {
			key = "_unknown"
		}
		counts[key]++
		total++
	}

	rows := make([]map[string]any, 0, len(counts))
	for key, count := range counts {
		rows = append(rows, adminSummaryBreakdownRow(key, WorkTaskTypeName(key), count, total))
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return inputInt(rows[i]["count"]) > inputInt(rows[j]["count"])
	})
	return rows
}

func adminSummaryBreakdownRow(key string, name string, count int, total int) map[string]any {
	percent := 0
	if total > 0 {
		percent = int(float64(count) / float64(total) * 100)
	}
	return map[string]any{
		"key":     key,
		"name":    name,
		"count":   count,
		"percent": percent,
	}
}

func adminSummaryPerformance(ctx context.Context, query AdminSummaryQuery, rangeValue adminSummaryRange) []map[string]any {
	staffRows := crmmodel.NewStaffModel().Select(ctx, map[string]any{})
	staffByID := map[uint64]*crmmodel.Staff{}
	for _, staff := range staffRows {
		if staff != nil {
			staffByID[staff.ID] = staff
		}
	}
	departmentNames := map[uint64]string{}
	for _, department := range crmmodel.NewDepartmentModel().Select(ctx, map[string]any{}) {
		if department != nil {
			departmentNames[department.ID] = department.Name
		}
	}
	tasks := adminSummaryTaskByID(adminSummaryTaskInfos(ctx))
	stats := map[uint64]*adminSummaryStaffStat{}
	todoFilters := map[string]any{}
	if query.WorkflowID > 0 {
		todoFilters["workflow_id"] = query.WorkflowID
	}
	if query.StaffID > 0 {
		todoFilters["assignee_staff_id"] = query.StaffID
	}
	for _, todo := range crmmodel.NewWorkTodoModel().Select(ctx, todoFilters) {
		if todo == nil || todo.AssigneeStaffID == 0 || !adminSummaryIsManualTask(tasks, todo.TaskID) ||
			query.WorkflowID > 0 && todo.WorkflowID != query.WorkflowID ||
			!adminSummaryStaffMatches(staffByID, todo.AssigneeStaffID, query) {
			continue
		}
		if todo.Status == crmmodel.WorkTodoStatusPending {
			adminSummaryStaffStatFor(stats, todo.AssigneeStaffID).PendingTaskCount++
			continue
		}
		if todo.Status != crmmodel.WorkTodoStatusDone || todo.CompletedAt == nil || !adminSummaryInRange(*todo.CompletedAt, rangeValue) {
			continue
		}
		stat := adminSummaryStaffStatFor(stats, todo.AssigneeStaffID)
		stat.CompletedTaskCount++
		if duration := todo.CompletedAt.Sub(todo.CreatedAt).Seconds(); duration >= 0 {
			stat.DurationSeconds += duration
			stat.DurationSampleCount++
		}
		if todo.DueAt != nil {
			stat.OnTimeEligibleCount++
			if !todo.CompletedAt.After(*todo.DueAt) {
				stat.OnTimeCount++
			}
		}
		adminSummaryTouchStaffStat(stat, *todo.CompletedAt)
	}
	eventFilters := map[string]any{"event_type": crmmodel.StatEventTypeTransition}
	if query.WorkflowID > 0 {
		eventFilters["workflow_id"] = query.WorkflowID
	}
	if query.StaffID > 0 {
		eventFilters["operator_staff_id"] = query.StaffID
	}
	for _, event := range crmmodel.NewStatEventModel().Select(ctx, eventFilters) {
		if event == nil || event.OperatorStaffID == 0 || !adminSummaryInRange(event.EventAt, rangeValue) ||
			query.WorkflowID > 0 && event.WorkflowID != query.WorkflowID ||
			!adminSummaryStaffMatches(staffByID, event.OperatorStaffID, query) {
			continue
		}
		stat := adminSummaryStaffStatFor(stats, event.OperatorStaffID)
		stat.TransitionCount++
		adminSummaryTouchStaffStat(stat, event.EventAt)
	}

	list := make([]*adminSummaryStaffStat, 0, len(stats))
	for staffID, stat := range stats {
		staff := staffByID[staffID]
		if staff != nil {
			stat.Name = staff.Name
			stat.DepartmentName = departmentNames[staff.DepartmentID]
		}
		if stat.Name == "" {
			stat.Name = "未命名人员"
		}
		list = append(list, stat)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].CompletedTaskCount != list[j].CompletedTaskCount {
			return list[i].CompletedTaskCount > list[j].CompletedTaskCount
		}
		if list[i].TransitionCount != list[j].TransitionCount {
			return list[i].TransitionCount > list[j].TransitionCount
		}
		return list[i].ID < list[j].ID
	})

	rows := make([]map[string]any, 0, len(list))
	for _, stat := range list {
		avgHours := 0.0
		if stat.DurationSampleCount > 0 {
			avgHours = stat.DurationSeconds / float64(stat.DurationSampleCount) / 3600
		}
		rows = append(rows, map[string]any{
			"id":                   stat.ID,
			"name":                 stat.Name,
			"department_name":      stat.DepartmentName,
			"completed_task_count": stat.CompletedTaskCount,
			"transition_count":     stat.TransitionCount,
			"pending_task_count":   stat.PendingTaskCount,
			"on_time_rate":         adminSummaryPercent(stat.OnTimeCount, stat.OnTimeEligibleCount),
			"on_time_sample_count": stat.OnTimeEligibleCount,
			"avg_duration_hours":   avgHours,
			"last_active_at":       stat.LastActiveAt,
		})
	}
	return rows
}

func adminSummaryStaffMatches(staffByID map[uint64]*crmmodel.Staff, staffID uint64, query AdminSummaryQuery) bool {
	if query.StaffID > 0 && staffID != query.StaffID {
		return false
	}
	staff := staffByID[staffID]
	if staff == nil {
		return false
	}
	return query.DepartmentID == 0 || staff.DepartmentID == query.DepartmentID
}

func adminSummaryTouchStaffStat(stat *adminSummaryStaffStat, activeAt time.Time) {
	if stat == nil || activeAt.IsZero() {
		return
	}
	if stat.LastActiveAt.IsZero() || activeAt.After(stat.LastActiveAt) {
		stat.LastActiveAt = activeAt
	}
}

func adminSummaryProbeSummary(ctx context.Context, assets []*crmmodel.CustomerAsset, workflowID uint64) map[string]any {
	templates := adminSummaryProbeTemplates(ctx, workflowID)
	if len(templates) == 0 {
		return map[string]any{
			"asset_count":          len(assets),
			"started_asset_count":  0,
			"complete_asset_count": 0,
			"field_total":          0,
			"field_filled":         0,
			"percent":              0,
			"dimensions":           []map[string]any{},
			"missing_dimensions":   []map[string]any{},
		}
	}

	templateIDs := map[uint64]bool{}
	fieldsByTemplate := map[uint64][]*crmmodel.DataField{}
	fieldsByID := map[uint64]*crmmodel.DataField{}
	for _, template := range templates {
		if template == nil {
			continue
		}
		templateIDs[template.ID] = true
		fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
			"data_template_id": template.ID,
			"status":           crmmodel.StatusEnabled,
		})
		for _, field := range fields {
			if field == nil || field.FieldType == "group" {
				continue
			}
			fieldsByTemplate[template.ID] = append(fieldsByTemplate[template.ID], field)
			fieldsByID[field.ID] = field
		}
	}
	parentNames := adminSummaryProbeParentNames(ctx, fieldsByID)
	probeFieldsByID := map[uint64]*crmmodel.DataField{}
	for templateID, fields := range fieldsByTemplate {
		probeFieldCodes := elevenDimensionProbeFields(fields, parentNames)
		probeFields := make([]*crmmodel.DataField, 0, elevenDimensionProbeCount)
		for _, field := range fields {
			if _, ok := probeFieldCodes[field.ID]; !ok {
				continue
			}
			probeFields = append(probeFields, field)
			probeFieldsByID[field.ID] = field
		}
		fieldsByTemplate[templateID] = probeFields
	}

	valuesByAsset := map[uint64]map[uint64]any{}
	for _, record := range crmmodel.NewDataRecordModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled}) {
		if record == nil || record.AssetID == 0 || !templateIDs[record.DataTemplateID] {
			continue
		}
		values := valuesByAsset[record.AssetID]
		if values == nil {
			values = map[uint64]any{}
			valuesByAsset[record.AssetID] = values
		}
		for rawFieldID, value := range mapFromAny(record.RecordJSON) {
			fieldID := inputUint64(rawFieldID)
			if probeFieldsByID[fieldID] != nil {
				values[fieldID] = value
			}
		}
	}

	dimensionStats := map[string]*adminSummaryProbeFieldStat{}
	fieldTotal := 0
	fieldFilled := 0
	completeAssets := 0
	startedAssets := 0
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		values := valuesByAsset[asset.ID]
		if len(values) > 0 {
			startedAssets++
		}
		assetTotal := 0
		assetFilled := 0
		for templateID, fields := range fieldsByTemplate {
			if !templateIDs[templateID] {
				continue
			}
			for _, field := range fields {
				if field == nil {
					continue
				}
				code, ok := elevenDimensionProbeFieldCode(field, parentNames)
				if !ok {
					continue
				}
				stat := adminSummaryProbeDimensionStat(dimensionStats, field, parentNames, code)
				stat.Total++
				fieldTotal++
				assetTotal++
				if !emptyWorkFieldValue(values[field.ID]) {
					stat.Filled++
					fieldFilled++
					assetFilled++
				}
			}
		}
		if assetTotal > 0 && assetFilled == assetTotal {
			completeAssets++
		}
	}

	return map[string]any{
		"asset_count":          len(assets),
		"started_asset_count":  startedAssets,
		"complete_asset_count": completeAssets,
		"field_total":          fieldTotal,
		"field_filled":         fieldFilled,
		"percent":              adminSummaryPercent(fieldFilled, fieldTotal),
		"dimensions":           adminSummaryProbeDimensionRows(dimensionStats, false),
		"missing_dimensions":   adminSummaryProbeDimensionRows(dimensionStats, true),
	}
}

func adminSummaryProbeTemplates(ctx context.Context, workflowID uint64) []*crmmodel.DataTemplate {
	stageIDs := map[uint64]bool{}
	for _, stage := range adminSummaryStageInfosForWorkflow(ctx, workflowID) {
		stageIDs[stage.ID] = true
	}
	formIDs := map[uint64]bool{}
	for _, task := range adminSummaryEnabledTaskInfos(ctx) {
		if stageIDs[task.StageID] && task.FormID > 0 {
			formIDs[task.FormID] = true
		}
	}
	templateIDs := map[uint64]bool{}
	for formID := range formIDs {
		for _, field := range crmmodel.NewFormFieldModel().Select(ctx, map[string]any{"form_id": formID, "status": crmmodel.StatusEnabled}) {
			if field != nil && field.DataTemplateID > 0 {
				templateIDs[field.DataTemplateID] = true
			}
		}
	}
	templates := make([]*crmmodel.DataTemplate, 0, len(templateIDs))
	for templateID := range templateIDs {
		template := crmmodel.NewDataTemplateModel().Find(ctx, map[string]any{"id": templateID, "status": crmmodel.StatusEnabled})
		if template != nil && adminSummaryTemplateHasProbeDimensions(ctx, template.ID) {
			templates = append(templates, template)
		}
	}
	return templates
}

func adminSummaryTemplateHasProbeDimensions(ctx context.Context, templateID uint64) bool {
	fields := crmmodel.NewDataFieldModel().Select(ctx, map[string]any{
		"data_template_id": templateID,
		"status":           crmmodel.StatusEnabled,
	})
	parentNames := workDataCompletenessParentNames(ctx, fields)
	return isElevenDimensionProbeTemplate(elevenDimensionProbeFields(fields, parentNames))
}

func adminSummaryProbeParentNames(ctx context.Context, fieldsByID map[uint64]*crmmodel.DataField) map[uint64]string {
	parentIDs := map[uint64]bool{}
	for _, field := range fieldsByID {
		if field != nil && field.ParentFieldID > 0 {
			parentIDs[field.ParentFieldID] = true
		}
	}
	names := map[uint64]string{}
	for parentID := range parentIDs {
		if parent := crmmodel.NewDataFieldModel().Find(ctx, map[string]any{"id": parentID}); parent != nil {
			names[parentID] = parent.Name
		}
	}
	return names
}

func adminSummaryProbeDimensionStat(stats map[string]*adminSummaryProbeFieldStat, field *crmmodel.DataField, parentNames map[uint64]string, code string) *adminSummaryProbeFieldStat {
	name := workDataCompletenessFieldLabel(field, parentNames)
	if parentName := parentNames[field.ParentFieldID]; parentName != "" {
		name = parentName
	}
	if name == "" && field != nil {
		name = field.Name
	}
	stat := stats[code]
	if stat == nil {
		stat = &adminSummaryProbeFieldStat{Key: code, Name: name}
		stats[code] = stat
	}
	return stat
}

func adminSummaryProbeDimensionRows(stats map[string]*adminSummaryProbeFieldStat, missingFirst bool) []map[string]any {
	dimensions := make([]*adminSummaryProbeFieldStat, 0, len(stats))
	for _, stat := range stats {
		dimensions = append(dimensions, stat)
	}
	sort.SliceStable(dimensions, func(i, j int) bool {
		if missingFirst {
			leftMissing := dimensions[i].Total - dimensions[i].Filled
			rightMissing := dimensions[j].Total - dimensions[j].Filled
			if leftMissing != rightMissing {
				return leftMissing > rightMissing
			}
		}
		return dimensions[i].Key < dimensions[j].Key
	})
	if missingFirst && len(dimensions) > 8 {
		dimensions = dimensions[:8]
	}
	rows := make([]map[string]any, 0, len(dimensions))
	for _, stat := range dimensions {
		rows = append(rows, map[string]any{
			"key":           stat.Key,
			"name":          stat.Name,
			"total":         stat.Total,
			"filled":        stat.Filled,
			"missing_count": stat.Total - stat.Filled,
			"percent":       adminSummaryPercent(stat.Filled, stat.Total),
		})
	}
	return rows
}

func adminSummaryStaffStatFor(stats map[uint64]*adminSummaryStaffStat, staffID uint64) *adminSummaryStaffStat {
	if stats[staffID] == nil {
		stats[staffID] = &adminSummaryStaffStat{ID: staffID}
	}
	return stats[staffID]
}

func adminSummaryStageInfos(ctx context.Context) []adminSummaryStageInfo {
	rows := crmmodel.NewStageModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	stages := make([]adminSummaryStageInfo, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		stages = append(stages, adminSummaryStageInfo{
			ID:         row.ID,
			WorkflowID: row.WorkflowID,
			Name:       row.Name,
			Sort:       row.Sort,
		})
	}
	sort.SliceStable(stages, func(i, j int) bool {
		if stages[i].Sort != stages[j].Sort {
			return stages[i].Sort < stages[j].Sort
		}
		return stages[i].ID < stages[j].ID
	})
	return stages
}

func adminSummaryStageInfosForWorkflow(ctx context.Context, workflowID uint64) []adminSummaryStageInfo {
	stages := adminSummaryStageInfos(ctx)
	if workflowID == 0 {
		return stages
	}
	result := make([]adminSummaryStageInfo, 0, len(stages))
	for _, stage := range stages {
		if stage.WorkflowID == workflowID {
			result = append(result, stage)
		}
	}
	return result
}

func adminSummaryTaskInfos(ctx context.Context) []adminSummaryTaskInfo {
	return adminSummaryTaskInfosWithFilter(ctx, map[string]any{})
}

func adminSummaryEnabledTaskInfos(ctx context.Context) []adminSummaryTaskInfo {
	return adminSummaryTaskInfosWithFilter(ctx, map[string]any{"status": crmmodel.StatusEnabled})
}

func adminSummaryTaskInfosWithFilter(ctx context.Context, filter map[string]any) []adminSummaryTaskInfo {
	rows := crmmodel.NewTaskModel().Select(ctx, filter)
	tasks := make([]adminSummaryTaskInfo, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		tasks = append(tasks, adminSummaryTaskInfo{
			ID:      row.ID,
			StageID: row.StageID,
			Type:    row.TaskType,
			FormID:  row.FormID,
		})
	}
	return tasks
}

func adminSummaryStageByID(stages []adminSummaryStageInfo) map[uint64]adminSummaryStageInfo {
	result := map[uint64]adminSummaryStageInfo{}
	for _, stage := range stages {
		result[stage.ID] = stage
	}
	return result
}

func adminSummaryStageKey(stageID uint64) string {
	if stageID == 0 {
		return "_empty"
	}
	return strconv.FormatUint(stageID, 10)
}

func adminSummaryPercent(count int, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(count) / float64(total) * 100)
}

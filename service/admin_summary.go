package service

import (
	"context"
	"sort"
	"strconv"
	"time"

	crmmodel "github.com/dever-package/crm/model"
)

const adminSummaryTrendDays = 14

type AdminSummaryService struct{}

type adminSummaryStageInfo struct {
	ID   uint64
	Name string
	Sort int
}

type adminSummaryTaskInfo struct {
	ID      uint64
	StageID uint64
	Type    string
	Name    string
	FormID  uint64
}

type adminSummaryStaffStat struct {
	ID              uint64
	Name            string
	TaskCount       int
	TransitionCount int
	OperationCount  int
	TodoDoneCount   int
	LastActiveAt    time.Time
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

func (AdminSummaryService) Summary(ctx context.Context) (map[string]any, error) {
	customers := crmmodel.NewCustomerModel().Select(ctx, map[string]any{})
	assets := crmmodel.NewCustomerAssetModel().Select(ctx, map[string]any{})
	stageTargets := crmmodel.NewCustomerStageModel().Select(ctx, map[string]any{})
	stages := adminSummaryStageInfos(ctx)
	tasks := adminSummaryTaskInfos(ctx)
	statEvents := crmmodel.NewStatEventModel().Select(ctx, map[string]any{})
	operations := crmmodel.NewOperationLogModel().Select(ctx, map[string]any{})
	todos := crmmodel.NewWorkTodoModel().Select(ctx, map[string]any{})
	financeLedgers := crmmodel.NewFinanceLedgerModel().Select(ctx, map[string]any{})
	pendingTodoCount := int(crmmodel.NewWorkTodoModel().Count(ctx, map[string]any{
		"status": crmmodel.WorkTodoStatusPending,
	}))

	start := adminSummaryTrendStart(adminSummaryTrendDays)
	trendRows := adminSummaryTrendRows(customers, assets, statEvents, operations, adminSummaryTrendDays)
	funnelRows := adminSummaryStageFunnel(stageTargets, stages)
	staffRows := adminSummaryStaffOutput(ctx, statEvents, operations, todos, start)
	return map[string]any{
		"metrics":         adminSummaryMetrics(customers, assets, stageTargets, statEvents, operations, pendingTodoCount, start),
		"growth_trend":    trendRows,
		"execution_trend": trendRows,
		"funnel":          funnelRows,
		"pipeline_funnel": funnelRows,
		"node_backlog":    adminSummaryNodeBacklog(ctx, stageTargets, stages, tasks, todos),
		"task_breakdown":  adminSummaryTaskBreakdown(stageTargets, stages, tasks),
		"finance_summary": adminSummaryFinanceSummary(financeLedgers, start),
		"staff_ranking":   staffRows,
		"staff_output":    staffRows,
		"probe_summary":   adminSummaryProbeSummary(ctx, assets),
		"generated_at":    time.Now(),
	}, nil
}

func adminSummaryMetrics(customers []*crmmodel.Customer, assets []*crmmodel.CustomerAsset, stageTargets []*crmmodel.CustomerStage, events []*crmmodel.StatEvent, operations []*crmmodel.OperationLog, pendingTodoCount int, start time.Time) []map[string]any {
	taskCount, transitionCount := adminSummaryEventCountsSince(events, start)
	return []map[string]any{
		adminSummaryMetric("customers", "客户总数", len(customers), "CRM 当前全部客户"),
		adminSummaryMetric("assets", "资产总数", len(assets), "客户名下已建立资产"),
		adminSummaryMetric("stage_targets", "阶段对象", len(stageTargets), "正在阶段流转的客户或资产"),
		adminSummaryMetric("missing_assets", "未录资产", adminSummaryMissingAssetCustomers(customers, assets), "已建客户但尚未录入资产"),
		adminSummaryMetric("pending_todos", "待办任务", pendingTodoCount, "当前未完成的任务待办"),
		adminSummaryMetric("tasks_14d", "近14天任务", taskCount, "近14天完成的任务事件"),
		adminSummaryMetric("transitions_14d", "近14天流转", transitionCount, "近14天阶段流转次数"),
		adminSummaryMetric("operations_14d", "近14天操作", adminSummaryOperationCountSince(operations, start), "近14天提交的操作记录"),
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

func adminSummaryAmountMetric(key string, name string, value float64, description string) map[string]any {
	return map[string]any{
		"key":         key,
		"name":        name,
		"value":       value,
		"description": description,
	}
}

func adminSummaryFinanceSummary(ledgers []*crmmodel.FinanceLedger, start time.Time) map[string]any {
	totalIncome, totalExpense := adminSummaryFinanceTotals(ledgers, time.Time{})
	recentIncome, recentExpense := adminSummaryFinanceTotals(ledgers, start)
	return map[string]any{
		"metrics": []map[string]any{
			adminSummaryAmountMetric("finance_income", "累计收入", totalIncome, "全部财务收入流水金额"),
			adminSummaryAmountMetric("finance_expense", "累计支出", totalExpense, "全部财务支出流水金额"),
			adminSummaryAmountMetric("finance_net", "净额", totalIncome-totalExpense, "累计收入减累计支出"),
			adminSummaryMetric("finance_ledger_count", "流水数量", len(ledgers), "当前已生成的财务流水记录"),
			adminSummaryAmountMetric("finance_income_14d", "近14天收入", recentIncome, "近14天财务收入流水金额"),
			adminSummaryAmountMetric("finance_expense_14d", "近14天支出", recentExpense, "近14天财务支出流水金额"),
		},
		"trend":          adminSummaryFinanceTrendRows(ledgers, adminSummaryTrendDays),
		"type_breakdown": adminSummaryFinanceTypeRows(ledgers),
	}
}

func adminSummaryFinanceTotals(ledgers []*crmmodel.FinanceLedger, start time.Time) (float64, float64) {
	var income float64
	var expense float64
	for _, ledger := range ledgers {
		if ledger == nil || (!start.IsZero() && ledger.CreatedAt.Before(start)) {
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

func adminSummaryFinanceTrendRows(ledgers []*crmmodel.FinanceLedger, days int) []map[string]any {
	if days <= 0 {
		days = adminSummaryTrendDays
	}
	start := adminSummaryTrendStart(days)
	end := workBeginningOfDay(time.Now()).AddDate(0, 0, 1)
	rows := make([]map[string]any, 0, days)
	indexes := map[string]int{}
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		indexes[key] = i
		rows = append(rows, map[string]any{
			"date":           key,
			"label":          day.Format("01-02"),
			"income_amount":  0,
			"expense_amount": 0,
			"net_amount":     0,
			"ledger_count":   0,
		})
	}
	for _, ledger := range ledgers {
		if ledger == nil || ledger.CreatedAt.Before(start) || ledger.CreatedAt.After(end) {
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

type adminSummaryFinanceTypeStat struct {
	Key       string
	Name      string
	Direction string
	Count     int
	Amount    float64
}

func adminSummaryFinanceTypeRows(ledgers []*crmmodel.FinanceLedger) []map[string]any {
	stats := map[string]*adminSummaryFinanceTypeStat{}
	var totalAmount float64
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
		totalAmount += ledger.Amount
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
		if totalAmount > 0 {
			percent = int(stat.Amount / totalAmount * 100)
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

func adminSummaryTrendRows(customers []*crmmodel.Customer, assets []*crmmodel.CustomerAsset, events []*crmmodel.StatEvent, operations []*crmmodel.OperationLog, days int) []map[string]any {
	if days <= 0 {
		days = adminSummaryTrendDays
	}
	start := adminSummaryTrendStart(days)
	end := workBeginningOfDay(time.Now()).AddDate(0, 0, 1)
	rows := make([]map[string]any, 0, days)
	indexes := map[string]int{}
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		indexes[key] = i
		rows = append(rows, map[string]any{
			"date":             key,
			"label":            day.Format("01-02"),
			"customer_count":   0,
			"asset_count":      0,
			"task_count":       0,
			"transition_count": 0,
			"operation_count":  0,
		})
	}

	for _, customer := range customers {
		if customer == nil {
			continue
		}
		adminSummaryIncrementTrend(rows, indexes, start, end, customer.CreatedAt, "customer_count")
	}
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		adminSummaryIncrementTrend(rows, indexes, start, end, asset.CreatedAt, "asset_count")
	}
	for _, event := range events {
		if event == nil || event.EventAt.Before(start) || event.EventAt.After(end) {
			continue
		}
		switch event.EventType {
		case crmmodel.StatEventTypeTask:
			if event.ResultValue == workResultProgress {
				continue
			}
			adminSummaryIncrementTrend(rows, indexes, start, end, event.EventAt, "task_count")
		case crmmodel.StatEventTypeTransition:
			adminSummaryIncrementTrend(rows, indexes, start, end, event.EventAt, "transition_count")
		}
	}
	for _, operation := range operations {
		if operation == nil {
			continue
		}
		adminSummaryIncrementTrend(rows, indexes, start, end, operation.CreatedAt, "operation_count")
	}
	return rows
}

func adminSummaryIncrementTrend(rows []map[string]any, indexes map[string]int, start time.Time, end time.Time, eventAt time.Time, key string) {
	if eventAt.Before(start) || eventAt.After(end) {
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

func adminSummaryStageFunnel(targets []*crmmodel.CustomerStage, stages []adminSummaryStageInfo) []map[string]any {
	counts := map[uint64]int{}
	for _, target := range targets {
		if target == nil {
			continue
		}
		counts[target.StageID]++
	}
	stageByID := adminSummaryStageByID(stages)
	rows := make([]map[string]any, 0, len(counts))
	total := len(targets)
	previous := total
	for _, stage := range stages {
		count := counts[stage.ID]
		if count == 0 {
			continue
		}
		row := adminSummaryBreakdownRow(adminSummaryStageKey(stage.ID), stage.Name, count, total)
		adminSummaryAttachDropFields(row, previous, count)
		rows = append(rows, row)
		previous = count
		delete(counts, stage.ID)
	}
	for stageID, count := range counts {
		name := "未进入阶段"
		if stage, ok := stageByID[stageID]; ok && stage.Name != "" {
			name = stage.Name
		}
		row := adminSummaryBreakdownRow(adminSummaryStageKey(stageID), name, count, total)
		adminSummaryAttachDropFields(row, previous, count)
		rows = append(rows, row)
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

func adminSummaryNodeBacklog(ctx context.Context, targets []*crmmodel.CustomerStage, stages []adminSummaryStageInfo, tasks []adminSummaryTaskInfo, todos []*crmmodel.WorkTodo) []map[string]any {
	stageByID := adminSummaryStageByID(stages)
	stats := map[uint64]*adminSummaryNodeBacklogStat{}
	targetStageByKey := map[string]uint64{}
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
		targetStageByKey[adminSummaryTargetKey(target.CustomerID, target.AssetID)] = stageID
	}

	pendingTodosByStage := map[uint64]int{}
	for _, todo := range todos {
		if todo == nil || todo.Status != crmmodel.WorkTodoStatusPending {
			continue
		}
		stageID := targetStageByKey[adminSummaryTargetKey(todo.CustomerID, todo.AssetID)]
		pendingTodosByStage[stageID]++
	}

	tasksByStage := map[uint64]int{}
	for _, task := range tasks {
		tasksByStage[task.StageID]++
	}

	rows := make([]map[string]any, 0, len(stats))
	total := len(targets)
	for _, stage := range stages {
		if stat := stats[stage.ID]; stat != nil {
			rows = append(rows, adminSummaryNodeBacklogRow(stat, tasksByStage[stage.ID], pendingTodosByStage[stage.ID], total))
			delete(stats, stage.ID)
		}
	}
	for stageID, stat := range stats {
		rows = append(rows, adminSummaryNodeBacklogRow(stat, 0, pendingTodosByStage[stageID], total))
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

func adminSummaryNodeBacklogRow(stat *adminSummaryNodeBacklogStat, taskCount int, pendingTodoCount int, total int) map[string]any {
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
	row["task_count"] = taskCount
	row["pending_todo_count"] = pendingTodoCount
	row["avg_days"] = avgDays
	row["max_days"] = maxDays
	row["stale_3d"] = stale3d
	row["stale_7d"] = stale7d
	row["stale_15d"] = stale15d
	return row
}

func adminSummaryTaskBreakdown(targets []*crmmodel.CustomerStage, stages []adminSummaryStageInfo, tasks []adminSummaryTaskInfo) []map[string]any {
	tasksByStage := map[uint64][]adminSummaryTaskInfo{}
	for _, task := range tasks {
		tasksByStage[task.StageID] = append(tasksByStage[task.StageID], task)
	}

	counts := map[string]int{}
	total := 0
	for _, target := range targets {
		if target == nil {
			continue
		}
		for _, task := range tasksByStage[target.StageID] {
			key := task.Type
			if key == "" {
				key = "_unknown"
			}
			counts[key]++
			total++
		}
	}

	rows := make([]map[string]any, 0, len(counts))
	for key, count := range counts {
		rows = append(rows, adminSummaryBreakdownRow(key, workTaskTypeName(key), count, total))
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

func adminSummaryStaffOutput(ctx context.Context, events []*crmmodel.StatEvent, operations []*crmmodel.OperationLog, todos []*crmmodel.WorkTodo, start time.Time) []map[string]any {
	statsByStaff := map[uint64]*adminSummaryStaffStat{}
	for _, event := range events {
		if event == nil || event.OperatorStaffID == 0 || event.EventAt.Before(start) {
			continue
		}
		stat := adminSummaryStaffStatFor(statsByStaff, event.OperatorStaffID)
		adminSummaryTouchStaffStat(stat, event.EventAt)
		switch event.EventType {
		case crmmodel.StatEventTypeTask:
			if event.ResultValue == workResultProgress {
				continue
			}
			stat.TaskCount++
		case crmmodel.StatEventTypeTransition:
			stat.TransitionCount++
		}
	}
	for _, operation := range operations {
		if operation == nil || operation.OperatorStaffID == 0 || operation.CreatedAt.Before(start) {
			continue
		}
		stat := adminSummaryStaffStatFor(statsByStaff, operation.OperatorStaffID)
		stat.OperationCount++
		adminSummaryTouchStaffStat(stat, operation.CreatedAt)
	}
	for _, todo := range todos {
		if todo == nil || todo.AssigneeStaffID == 0 || todo.Status != crmmodel.WorkTodoStatusDone || todo.CompletedAt == nil || todo.CompletedAt.Before(start) {
			continue
		}
		stat := adminSummaryStaffStatFor(statsByStaff, todo.AssigneeStaffID)
		stat.TodoDoneCount++
		adminSummaryTouchStaffStat(stat, *todo.CompletedAt)
	}
	names := adminSummaryStaffNames(ctx)
	stats := make([]*adminSummaryStaffStat, 0, len(statsByStaff))
	for id, stat := range statsByStaff {
		stat.Name = names[id]
		if stat.Name == "" {
			stat.Name = "未命名人员"
		}
		stats = append(stats, stat)
	}
	sort.SliceStable(stats, func(i, j int) bool {
		left := adminSummaryStaffTotal(stats[i])
		right := adminSummaryStaffTotal(stats[j])
		if left != right {
			return left > right
		}
		return stats[i].ID < stats[j].ID
	})
	if len(stats) > 8 {
		stats = stats[:8]
	}

	rows := make([]map[string]any, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, map[string]any{
			"id":               stat.ID,
			"name":             stat.Name,
			"task_count":       stat.TaskCount,
			"transition_count": stat.TransitionCount,
			"operation_count":  stat.OperationCount,
			"todo_done_count":  stat.TodoDoneCount,
			"last_active_at":   stat.LastActiveAt,
			"total":            adminSummaryStaffTotal(stat),
		})
	}
	return rows
}

func adminSummaryTouchStaffStat(stat *adminSummaryStaffStat, activeAt time.Time) {
	if stat == nil || activeAt.IsZero() {
		return
	}
	if stat.LastActiveAt.IsZero() || activeAt.After(stat.LastActiveAt) {
		stat.LastActiveAt = activeAt
	}
}

func adminSummaryStaffTotal(stat *adminSummaryStaffStat) int {
	if stat == nil {
		return 0
	}
	return stat.TaskCount + stat.TransitionCount + stat.OperationCount + stat.TodoDoneCount
}

func adminSummaryProbeSummary(ctx context.Context, assets []*crmmodel.CustomerAsset) map[string]any {
	templates := adminSummaryProbeTemplates(ctx)
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
			if fieldsByID[fieldID] != nil {
				values[fieldID] = value
			}
		}
	}

	dimensionStats := map[string]*adminSummaryProbeFieldStat{}
	fieldTotal := 0
	fieldFilled := 0
	completeAssets := 0
	parentNames := adminSummaryProbeParentNames(ctx, fieldsByID)
	for _, values := range valuesByAsset {
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
				stat := adminSummaryProbeDimensionStat(dimensionStats, field, parentNames)
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
		"started_asset_count":  len(valuesByAsset),
		"complete_asset_count": completeAssets,
		"field_total":          fieldTotal,
		"field_filled":         fieldFilled,
		"percent":              adminSummaryPercent(fieldFilled, fieldTotal),
		"dimensions":           adminSummaryProbeDimensionRows(dimensionStats, false),
		"missing_dimensions":   adminSummaryProbeDimensionRows(dimensionStats, true),
	}
}

func adminSummaryProbeTemplates(ctx context.Context) []*crmmodel.DataTemplate {
	rows := crmmodel.NewDataTemplateModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	templates := make([]*crmmodel.DataTemplate, 0, len(rows))
	for _, row := range rows {
		if row != nil && workDataCompletenessTemplateIsProbe(row.Name) {
			templates = append(templates, row)
		}
	}
	return templates
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

func adminSummaryProbeDimensionStat(stats map[string]*adminSummaryProbeFieldStat, field *crmmodel.DataField, parentNames map[uint64]string) *adminSummaryProbeFieldStat {
	name := workDataCompletenessFieldLabel(field, parentNames)
	if parentName := parentNames[field.ParentFieldID]; parentName != "" {
		name = parentName
	}
	key := name
	if key == "" && field != nil {
		key = field.FieldKey
		name = field.Name
	}
	if key == "" {
		key = "unknown"
		name = "未命名维度"
	}
	stat := stats[key]
	if stat == nil {
		stat = &adminSummaryProbeFieldStat{Key: key, Name: name}
		stats[key] = stat
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

func adminSummaryStaffNames(ctx context.Context) map[uint64]string {
	rows := crmmodel.NewStaffModel().Select(ctx, map[string]any{})
	names := map[uint64]string{}
	for _, row := range rows {
		if row == nil {
			continue
		}
		names[row.ID] = row.Name
	}
	return names
}

func adminSummaryStageInfos(ctx context.Context) []adminSummaryStageInfo {
	rows := crmmodel.NewStageModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	stages := make([]adminSummaryStageInfo, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		stages = append(stages, adminSummaryStageInfo{
			ID:   row.ID,
			Name: row.Name,
			Sort: row.Sort,
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

func adminSummaryTaskInfos(ctx context.Context) []adminSummaryTaskInfo {
	rows := crmmodel.NewTaskModel().Select(ctx, map[string]any{"status": crmmodel.StatusEnabled})
	tasks := make([]adminSummaryTaskInfo, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		tasks = append(tasks, adminSummaryTaskInfo{
			ID:      row.ID,
			StageID: row.StageID,
			Type:    row.TaskType,
			Name:    row.Name,
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

func adminSummaryTargetKey(customerID uint64, assetID uint64) string {
	return strconv.FormatUint(customerID, 10) + ":" + strconv.FormatUint(assetID, 10)
}

func adminSummaryPercent(count int, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(count) / float64(total) * 100)
}

func adminSummaryMissingAssetCustomers(customers []*crmmodel.Customer, assets []*crmmodel.CustomerAsset) int {
	hasAsset := map[uint64]bool{}
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		hasAsset[asset.CustomerID] = true
	}
	count := 0
	for _, customer := range customers {
		if customer != nil && !hasAsset[customer.ID] {
			count++
		}
	}
	return count
}

func adminSummaryEventCountsSince(events []*crmmodel.StatEvent, start time.Time) (int, int) {
	taskCount := 0
	transitionCount := 0
	for _, event := range events {
		if event == nil || event.EventAt.Before(start) {
			continue
		}
		switch event.EventType {
		case crmmodel.StatEventTypeTask:
			if event.ResultValue == workResultProgress {
				continue
			}
			taskCount++
		case crmmodel.StatEventTypeTransition:
			transitionCount++
		}
	}
	return taskCount, transitionCount
}

func adminSummaryOperationCountSince(operations []*crmmodel.OperationLog, start time.Time) int {
	count := 0
	for _, operation := range operations {
		if operation != nil && !operation.CreatedAt.Before(start) {
			count++
		}
	}
	return count
}

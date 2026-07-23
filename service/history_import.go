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

type historyImportTarget struct {
	LeadID             uint64            `json:"lead_id,omitempty"`
	CustomerID         uint64            `json:"customer_id,omitempty"`
	AssetIDs           map[string]uint64 `json:"asset_ids,omitempty"`
	WorkflowInstanceID uint64            `json:"workflow_instance_id,omitempty"`
	OperationIDs       []uint64          `json:"operation_ids,omitempty"`
	MeetingIDs         []uint64          `json:"meeting_ids,omitempty"`
	GroupIDs           []uint64          `json:"group_ids,omitempty"`
	AttachmentIDs      []uint64          `json:"attachment_ids,omitempty"`
}

type historyImportCasePlan struct {
	Input          HistoryImportCaseInput
	Target         historyImportTarget
	SourceStates   map[string]*crmmodel.HistoryImportRecord
	Unchanged      bool
	RequiresCreate bool
	Issues         []HistoryImportIssue
}

func RunHistoryImport(
	ctx context.Context,
	batch HistoryImportBatchInput,
	options HistoryImportOptions,
) (result HistoryImportResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("飞书历史导入失败：%v", recovered)
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateHistoryImportBatch(batch, options); err != nil {
		return HistoryImportResult{}, err
	}
	catalog, err := loadHistoryImportCatalog(ctx)
	if err != nil {
		return HistoryImportResult{}, err
	}
	if options.Apply {
		if err := orm.Transaction(ctx, func(txCtx context.Context) error {
			return ensureHistoryImportConfiguration(txCtx, &catalog, batch)
		}); err != nil {
			return HistoryImportResult{}, err
		}
	}

	result = HistoryImportResult{
		BatchID: strings.TrimSpace(batch.BatchID),
		Applied: options.Apply,
		Cases:   make([]HistoryImportCaseResult, 0, len(batch.Cases)),
	}
	cases := append([]HistoryImportCaseInput(nil), batch.Cases...)
	sort.SliceStable(cases, func(i, j int) bool { return cases[i].CaseID < cases[j].CaseID })
	for _, input := range cases {
		caseResult := safeRunHistoryImportCase(ctx, catalog, input, batch.BatchID, options)
		result.Cases = append(result.Cases, caseResult)
		mergeHistoryImportCounts(&result.Counts, caseResult.Counts)
		result.Counts.Cases++
		result.Issues = append(result.Issues, caseResult.Issues...)
	}
	for _, source := range batch.OrphanRecords {
		issue := HistoryImportIssue{
			Severity:  HistoryImportSeverityWarning,
			Code:      "missing_case_id",
			SourceKey: source.SourceKey,
			Message:   "来源记录缺少内部案件ID，未创建业务对象",
		}
		result.Issues = append(result.Issues, issue)
		result.Counts.Skipped++
		if options.Apply {
			upsertHistoryImportAudit(ctx, batch.BatchID, source, historyImportTarget{}, crmmodel.HistoryImportStatusSkipped, issue.Message)
		}
	}
	return result, nil
}

func safeRunHistoryImportCase(
	ctx context.Context,
	catalog historyImportCatalog,
	input HistoryImportCaseInput,
	batchID string,
	options HistoryImportOptions,
) (result HistoryImportCaseResult) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message := fmt.Sprintf("案件导入异常：%v", recovered)
			result = HistoryImportCaseResult{
				CaseID: input.CaseID, Action: HistoryImportActionFailed, Error: message,
				Counts: HistoryImportCounts{Failed: 1},
				Issues: []HistoryImportIssue{{
					Severity: HistoryImportSeverityError, Code: "case_panic",
					CaseID: input.CaseID, Message: message,
				}},
			}
		}
	}()
	return runHistoryImportCase(ctx, catalog, input, batchID, options)
}

func validateHistoryImportBatch(batch HistoryImportBatchInput, options HistoryImportOptions) error {
	if strings.TrimSpace(batch.BatchID) == "" {
		return fmt.Errorf("导入批次不能为空")
	}
	if len(batch.Cases) == 0 {
		return fmt.Errorf("快照中没有可导入案件")
	}
	if options.Apply && !options.BackupConfirmed {
		return fmt.Errorf("正式写入前必须确认数据库已备份")
	}
	caseIDs := make(map[string]bool, len(batch.Cases))
	for _, input := range batch.Cases {
		caseID := strings.TrimSpace(input.CaseID)
		if caseID == "" {
			return fmt.Errorf("导入案件ID不能为空")
		}
		if caseIDs[caseID] {
			return fmt.Errorf("导入批次存在重复案件ID：%s", caseID)
		}
		caseIDs[caseID] = true
	}
	return nil
}

func runHistoryImportCase(
	ctx context.Context,
	catalog historyImportCatalog,
	input HistoryImportCaseInput,
	batchID string,
	options HistoryImportOptions,
) HistoryImportCaseResult {
	plan := prepareHistoryImportCase(ctx, catalog, input, options)
	result := HistoryImportCaseResult{
		CaseID:   input.CaseID,
		AssetIDs: copyHistoryAssetIDs(plan.Target.AssetIDs),
		Issues:   append([]HistoryImportIssue(nil), plan.Issues...),
	}
	applyHistoryTargetToResult(&result, plan.Target)
	if plan.Unchanged {
		result.Action = HistoryImportActionUnchanged
		result.Counts.Unchanged = 1
		if options.Apply {
			for _, source := range input.Sources {
				upsertHistoryImportAudit(ctx, batchID, source, plan.Target, crmmodel.HistoryImportStatusUnchanged, "")
			}
		}
		return result
	}
	if historyImportHasErrors(plan.Issues) {
		result.Action = HistoryImportActionConflict
		result.Counts.Conflicts = 1
		if options.Apply {
			message := historyImportIssueMessage(plan.Issues)
			for _, source := range input.Sources {
				upsertHistoryImportAudit(ctx, batchID, source, plan.Target, crmmodel.HistoryImportStatusConflict, message)
			}
		}
		return result
	}
	if !options.Apply {
		if plan.RequiresCreate {
			result.Action = HistoryImportActionCreate
			result.Counts.Created = 1
		} else {
			result.Action = HistoryImportActionUpdate
			result.Counts.Updated = 1
		}
		result.Counts = estimateHistoryImportCaseCounts(input, result.Counts)
		return result
	}

	var persisted historyImportTarget
	var counts HistoryImportCounts
	var writeIssues []HistoryImportIssue
	needsRetry := false
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		var persistErr error
		persisted, counts, writeIssues, persistErr = persistHistoryImportCase(txCtx, catalog, plan, options)
		if persistErr != nil {
			return persistErr
		}
		auditStatus := crmmodel.HistoryImportStatusImported
		auditMessage := ""
		needsRetry = historyImportNeedsRetry(writeIssues)
		if needsRetry {
			auditStatus = crmmodel.HistoryImportStatusPartial
			auditMessage = historyImportRetryMessage(writeIssues)
		}
		for _, source := range input.Sources {
			upsertHistoryImportAudit(txCtx, batchID, source, persisted, auditStatus, auditMessage)
		}
		return nil
	})
	result.Issues = append(result.Issues, writeIssues...)
	if err != nil {
		result.Action = HistoryImportActionFailed
		result.Error = err.Error()
		result.Counts.Failed = 1
		for _, source := range input.Sources {
			upsertHistoryImportAudit(ctx, batchID, source, plan.Target, crmmodel.HistoryImportStatusFailed, err.Error())
		}
		return result
	}
	applyHistoryTargetToResult(&result, persisted)
	result.Counts = counts
	if needsRetry {
		result.Action = HistoryImportActionPartial
		result.Counts.Partial++
	} else if plan.RequiresCreate {
		result.Action = HistoryImportActionCreate
		result.Counts.Created++
	} else {
		result.Action = HistoryImportActionUpdate
		result.Counts.Updated++
	}
	return result
}

func historyImportNeedsRetry(issues []HistoryImportIssue) bool {
	for _, issue := range issues {
		if issue.Severity == HistoryImportSeverityError || strings.HasPrefix(issue.Code, "attachment_") {
			return true
		}
	}
	return false
}

func historyImportRetryMessage(issues []HistoryImportIssue) string {
	parts := make([]string, 0)
	for _, issue := range issues {
		if issue.Severity == HistoryImportSeverityError || strings.HasPrefix(issue.Code, "attachment_") {
			parts = append(parts, strings.TrimSpace(issue.Message))
		}
	}
	return strings.Join(parts, "；")
}

func prepareHistoryImportCase(
	ctx context.Context,
	catalog historyImportCatalog,
	input HistoryImportCaseInput,
	options HistoryImportOptions,
) historyImportCasePlan {
	plan := historyImportCasePlan{
		Input:        input,
		Target:       historyImportTarget{AssetIDs: map[string]uint64{}},
		SourceStates: map[string]*crmmodel.HistoryImportRecord{},
		Issues:       append([]HistoryImportIssue(nil), input.Issues...),
	}
	for _, row := range crmmodel.NewHistoryImportRecordModel().Select(ctx, map[string]any{
		"internal_case_id": input.CaseID,
	}, map[string]any{"order": "id desc"}) {
		if row == nil {
			continue
		}
		plan.SourceStates[row.SourceKey] = row
		mergeHistoryAuditTarget(&plan.Target, row)
	}
	applyHistoryOverride(&plan.Target, input.TargetOverride)
	unchangedSources := 0
	for _, source := range input.Sources {
		row := plan.SourceStates[source.SourceKey]
		if row != nil && row.SourceChecksum == source.Checksum &&
			(row.Status == crmmodel.HistoryImportStatusImported || row.Status == crmmodel.HistoryImportStatusUnchanged) {
			unchangedSources++
		}
	}
	plan.Unchanged = len(input.Sources) > 0 && unchangedSources == len(input.Sources)
	if plan.Unchanged {
		if historyImportTargetsExist(ctx, plan.Target, input) {
			return plan
		}
		plan.Unchanged = false
		plan.Issues = append(plan.Issues, historyCaseError(input.CaseID, "import_target_missing", "历史导入审计对应的CRM目标已不存在，需要人工确认后重建"))
	}
	resolveHistoryExistingTargets(ctx, catalog, &plan)
	plan.RequiresCreate = plan.Target.LeadID == 0 && input.Lead != nil ||
		plan.Target.CustomerID == 0 && input.Customer != nil ||
		len(plan.Target.AssetIDs) == 0 && len(input.Assets) > 0
	plan.Issues = append(plan.Issues, validateHistoryCaseMappings(ctx, catalog, plan, options)...)
	return plan
}

func historyImportTargetsExist(
	ctx context.Context,
	target historyImportTarget,
	input HistoryImportCaseInput,
) bool {
	if input.Lead != nil && (target.LeadID == 0 || crmmodel.NewLeadModel().Find(ctx, map[string]any{"id": target.LeadID}) == nil) {
		return false
	}
	if input.Customer != nil && (target.CustomerID == 0 || crmmodel.NewCustomerModel().Find(ctx, map[string]any{"id": target.CustomerID}) == nil) {
		return false
	}
	for _, asset := range input.Assets {
		assetID := target.AssetIDs[asset.Key]
		if assetID == 0 || crmmodel.NewCustomerAssetModel().Find(ctx, map[string]any{"id": assetID}) == nil {
			return false
		}
	}
	if len(input.Assets) > 0 && (target.WorkflowInstanceID == 0 || crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": target.WorkflowInstanceID}) == nil) {
		return false
	}
	return true
}

func mergeHistoryAuditTarget(target *historyImportTarget, row *crmmodel.HistoryImportRecord) {
	if target == nil || row == nil {
		return
	}
	target.LeadID = preferredUint64(target.LeadID, row.LeadID)
	target.CustomerID = preferredUint64(target.CustomerID, row.CustomerID)
	target.WorkflowInstanceID = preferredUint64(target.WorkflowInstanceID, row.WorkflowInstanceID)
	if target.AssetIDs == nil {
		target.AssetIDs = map[string]uint64{}
	}
	if row.AssetID > 0 && target.AssetIDs["primary"] == 0 {
		target.AssetIDs["primary"] = row.AssetID
	}
	var decoded historyImportTarget
	if json.Unmarshal([]byte(row.TargetJSON), &decoded) == nil {
		if decoded.LeadID > 0 {
			target.LeadID = decoded.LeadID
		}
		if decoded.CustomerID > 0 {
			target.CustomerID = decoded.CustomerID
		}
		if decoded.WorkflowInstanceID > 0 {
			target.WorkflowInstanceID = decoded.WorkflowInstanceID
		}
		for key, id := range decoded.AssetIDs {
			if id > 0 {
				target.AssetIDs[key] = id
			}
		}
	}
}

func applyHistoryOverride(target *historyImportTarget, override HistoryImportTargetOverride) {
	if target == nil {
		return
	}
	if override.LeadID > 0 {
		target.LeadID = override.LeadID
	}
	if override.CustomerID > 0 {
		target.CustomerID = override.CustomerID
	}
	if override.WorkflowInstanceID > 0 {
		target.WorkflowInstanceID = override.WorkflowInstanceID
	}
	if target.AssetIDs == nil {
		target.AssetIDs = map[string]uint64{}
	}
	for key, id := range override.AssetIDs {
		if id > 0 {
			target.AssetIDs[key] = id
		}
	}
}

func applyHistoryTargetToResult(result *HistoryImportCaseResult, target historyImportTarget) {
	if result == nil {
		return
	}
	result.LeadID = target.LeadID
	result.CustomerID = target.CustomerID
	result.AssetIDs = copyHistoryAssetIDs(target.AssetIDs)
	result.WorkflowInstanceID = target.WorkflowInstanceID
}

func copyHistoryAssetIDs(values map[string]uint64) map[string]uint64 {
	result := make(map[string]uint64, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func historyImportHasErrors(issues []HistoryImportIssue) bool {
	for _, issue := range issues {
		if issue.Severity == HistoryImportSeverityError {
			return true
		}
	}
	return false
}

func historyImportIssueMessage(issues []HistoryImportIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Severity == HistoryImportSeverityError && strings.TrimSpace(issue.Message) != "" {
			parts = append(parts, strings.TrimSpace(issue.Message))
		}
	}
	return strings.Join(parts, "；")
}

func estimateHistoryImportCaseCounts(input HistoryImportCaseInput, counts HistoryImportCounts) HistoryImportCounts {
	if input.Lead != nil {
		counts.Leads = 1
	}
	if input.Customer != nil {
		counts.Customers = 1
	}
	counts.Assets = len(input.Assets)
	if input.Customer != nil {
		counts.Records = len(input.CustomerRecords)
	}
	for _, asset := range input.Assets {
		counts.Records += len(asset.Records)
		counts.Attachments += len(asset.Attachments)
	}
	counts.Operations = len(input.Operations)
	counts.Meetings = len(input.Meetings)
	counts.Groups = len(input.Groups)
	for _, meeting := range input.Meetings {
		counts.Attachments += len(meeting.Attachments)
	}
	return counts
}

func mergeHistoryImportCounts(target *HistoryImportCounts, value HistoryImportCounts) {
	if target == nil {
		return
	}
	target.Created += value.Created
	target.Updated += value.Updated
	target.Unchanged += value.Unchanged
	target.Partial += value.Partial
	target.Skipped += value.Skipped
	target.Conflicts += value.Conflicts
	target.Failed += value.Failed
	target.Leads += value.Leads
	target.Customers += value.Customers
	target.Assets += value.Assets
	target.Records += value.Records
	target.Operations += value.Operations
	target.Meetings += value.Meetings
	target.Groups += value.Groups
	target.Attachments += value.Attachments
}

func upsertHistoryImportAudit(
	ctx context.Context,
	batchID string,
	source HistoryImportSourceRecordInput,
	target historyImportTarget,
	status string,
	errorMessage string,
) {
	now := time.Now()
	targetJSON := jsonText(target)
	rawJSON := jsonText(map[string]any{
		"record_id":        source.RecordID,
		"checksum":         source.Checksum,
		"created_at":       source.CreatedAt,
		"last_modified_at": source.LastModifiedAt,
		"fields":           source.Fields,
	})
	assetID := target.AssetIDs["primary"]
	if assetID == 0 {
		for _, id := range target.AssetIDs {
			assetID = id
			break
		}
	}
	data := map[string]any{
		"batch_id":                strings.TrimSpace(batchID),
		"source_key":              source.SourceKey,
		"source_table_key":        source.TableKey,
		"source_table_name":       source.TableName,
		"source_table_id":         source.TableID,
		"source_record_id":        source.RecordID,
		"internal_case_id":        source.CaseID,
		"source_checksum":         source.Checksum,
		"lead_id":                 target.LeadID,
		"customer_id":             target.CustomerID,
		"asset_id":                assetID,
		"workflow_instance_id":    target.WorkflowInstanceID,
		"target_json":             targetJSON,
		"raw_snapshot_json":       rawJSON,
		"status":                  status,
		"error_message":           strings.TrimSpace(errorMessage),
		"source_created_at":       source.CreatedAt,
		"source_last_modified_at": source.LastModifiedAt,
		"imported_at":             &now,
		"updated_at":              now,
	}
	model := crmmodel.NewHistoryImportRecordModel()
	if existing := model.Find(ctx, map[string]any{"source_key": source.SourceKey}); existing != nil {
		model.Update(ctx, map[string]any{"id": existing.ID}, data)
		return
	}
	data["created_at"] = now
	model.Insert(ctx, data)
}

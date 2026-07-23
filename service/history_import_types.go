package service

import "time"

const (
	HistoryImportSeverityWarning = "warning"
	HistoryImportSeverityError   = "error"
)

const (
	HistoryImportActionCreate    = "create"
	HistoryImportActionUpdate    = "update"
	HistoryImportActionUnchanged = "unchanged"
	HistoryImportActionPartial   = "partial"
	HistoryImportActionSkipped   = "skipped"
	HistoryImportActionConflict  = "conflict"
	HistoryImportActionFailed    = "failed"
)

type HistoryImportIssue struct {
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	CaseID    string `json:"case_id,omitempty"`
	SourceKey string `json:"source_key,omitempty"`
	Field     string `json:"field,omitempty"`
	Message   string `json:"message"`
}

type HistoryImportPersonInput struct {
	OpenID string `json:"open_id,omitempty"`
	Name   string `json:"name,omitempty"`
}

type HistoryImportSourceRecordInput struct {
	SourceKey      string         `json:"source_key"`
	TableKey       string         `json:"table_key"`
	TableName      string         `json:"table_name"`
	TableID        string         `json:"table_id"`
	RecordID       string         `json:"record_id"`
	CaseID         string         `json:"case_id,omitempty"`
	Checksum       string         `json:"checksum"`
	Fields         map[string]any `json:"fields"`
	CreatedAt      *time.Time     `json:"created_at,omitempty"`
	LastModifiedAt *time.Time     `json:"last_modified_at,omitempty"`
}

type HistoryImportLeadInput struct {
	Name        string                   `json:"name,omitempty"`
	Phone       string                   `json:"phone,omitempty"`
	Wechat      string                   `json:"wechat,omitempty"`
	SourceCode  string                   `json:"source_code,omitempty"`
	SourceName  string                   `json:"source_name,omitempty"`
	ChannelCode string                   `json:"channel_code,omitempty"`
	ChannelName string                   `json:"channel_name,omitempty"`
	ExternalID  string                   `json:"external_id,omitempty"`
	City        string                   `json:"city,omitempty"`
	InitialNeed string                   `json:"initial_need,omitempty"`
	Owner       HistoryImportPersonInput `json:"owner,omitempty"`
	CreatedAt   *time.Time               `json:"created_at,omitempty"`
}

type HistoryImportCustomerInput struct {
	Name        string     `json:"name,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	Wechat      string     `json:"wechat,omitempty"`
	IDCard      string     `json:"id_card,omitempty"`
	SourceCode  string     `json:"source_code,omitempty"`
	SourceName  string     `json:"source_name,omitempty"`
	ChannelCode string     `json:"channel_code,omitempty"`
	ChannelName string     `json:"channel_name,omitempty"`
	Remark      string     `json:"remark,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

type HistoryImportDataRecordInput struct {
	TemplateName string         `json:"template_name"`
	Fields       map[string]any `json:"fields"`
}

type HistoryImportAttachmentInput struct {
	SourceKey    string `json:"source_key"`
	TableKey     string `json:"table_key"`
	RecordID     string `json:"record_id"`
	FieldName    string `json:"field_name"`
	TemplateName string `json:"template_name,omitempty"`
	FieldKey     string `json:"field_key,omitempty"`
	FileToken    string `json:"file_token"`
	FileName     string `json:"file_name"`
	MIME         string `json:"mime,omitempty"`
	DownloadURL  string `json:"download_url,omitempty"`
	LocalPath    string `json:"local_path,omitempty"`
}

type HistoryImportAssetInput struct {
	Key         string                         `json:"key"`
	Name        string                         `json:"name,omitempty"`
	Address     string                         `json:"address,omitempty"`
	Remark      string                         `json:"remark,omitempty"`
	Records     []HistoryImportDataRecordInput `json:"records,omitempty"`
	Attachments []HistoryImportAttachmentInput `json:"attachments,omitempty"`
}

type HistoryImportOperationInput struct {
	SourceKey string                   `json:"source_key"`
	Title     string                   `json:"title"`
	Content   string                   `json:"content,omitempty"`
	Result    string                   `json:"result,omitempty"`
	Operator  HistoryImportPersonInput `json:"operator,omitempty"`
	CreatedAt *time.Time               `json:"created_at,omitempty"`
	Snapshot  map[string]any           `json:"snapshot,omitempty"`
}

type HistoryImportMeetingInput struct {
	SourceKey     string                         `json:"source_key"`
	Title         string                         `json:"title"`
	Remark        string                         `json:"remark,omitempty"`
	Owner         HistoryImportPersonInput       `json:"owner,omitempty"`
	Participants  []HistoryImportPersonInput     `json:"participants,omitempty"`
	ResourceName  string                         `json:"resource_name,omitempty"`
	Attempt       int                            `json:"attempt"`
	StartAt       time.Time                      `json:"start_at"`
	EndAt         time.Time                      `json:"end_at"`
	Status        string                         `json:"status"`
	ArrivalStatus string                         `json:"arrival_status"`
	ArrivalAt     *time.Time                     `json:"arrival_at,omitempty"`
	NoShowReason  string                         `json:"no_show_reason,omitempty"`
	Attachments   []HistoryImportAttachmentInput `json:"attachments,omitempty"`
}

type HistoryImportGroupStaffInput struct {
	Person HistoryImportPersonInput `json:"person"`
	Role   string                   `json:"role"`
}

type HistoryImportCommunicationGroupInput struct {
	SourceKey      string                         `json:"source_key"`
	Name           string                         `json:"name"`
	ExternalID     string                         `json:"external_id,omitempty"`
	Status         string                         `json:"status"`
	EstablishedAt  time.Time                      `json:"established_at"`
	DissolvedAt    *time.Time                     `json:"dissolved_at,omitempty"`
	DissolveReason string                         `json:"dissolve_reason,omitempty"`
	Summary        string                         `json:"summary,omitempty"`
	Remark         string                         `json:"remark,omitempty"`
	Staff          []HistoryImportGroupStaffInput `json:"staff,omitempty"`
}

type HistoryImportWorkflowInput struct {
	StageName string                   `json:"stage_name,omitempty"`
	Status    string                   `json:"status"`
	Owner     HistoryImportPersonInput `json:"owner,omitempty"`
	StartedAt *time.Time               `json:"started_at,omitempty"`
	EndedAt   *time.Time               `json:"ended_at,omitempty"`
	Reason    string                   `json:"reason,omitempty"`
}

type HistoryImportTargetOverride struct {
	LeadID             uint64            `json:"lead_id,omitempty"`
	CustomerID         uint64            `json:"customer_id,omitempty"`
	AssetIDs           map[string]uint64 `json:"asset_ids,omitempty"`
	WorkflowInstanceID uint64            `json:"workflow_instance_id,omitempty"`
}

type HistoryImportCaseInput struct {
	CaseID          string                                 `json:"case_id"`
	Lead            *HistoryImportLeadInput                `json:"lead,omitempty"`
	Customer        *HistoryImportCustomerInput            `json:"customer,omitempty"`
	Assets          []HistoryImportAssetInput              `json:"assets,omitempty"`
	CustomerRecords []HistoryImportDataRecordInput         `json:"customer_records,omitempty"`
	Operations      []HistoryImportOperationInput          `json:"operations,omitempty"`
	Meetings        []HistoryImportMeetingInput            `json:"meetings,omitempty"`
	Groups          []HistoryImportCommunicationGroupInput `json:"groups,omitempty"`
	Workflow        HistoryImportWorkflowInput             `json:"workflow"`
	Sources         []HistoryImportSourceRecordInput       `json:"sources"`
	Issues          []HistoryImportIssue                   `json:"issues,omitempty"`
	TargetOverride  HistoryImportTargetOverride            `json:"target_override,omitempty"`
}

type HistoryImportBatchInput struct {
	BatchID       string                           `json:"batch_id"`
	SnapshotDir   string                           `json:"snapshot_dir"`
	Cases         []HistoryImportCaseInput         `json:"cases"`
	OrphanRecords []HistoryImportSourceRecordInput `json:"orphan_records,omitempty"`
}

type HistoryImportOptions struct {
	Apply                  bool   `json:"apply"`
	BackupConfirmed        bool   `json:"backup_confirmed"`
	RestoreActiveWorkflows bool   `json:"restore_active_workflows"`
	UploadRuleID           uint64 `json:"upload_rule_id"`
}

type HistoryImportCounts struct {
	Cases       int `json:"cases"`
	Created     int `json:"created"`
	Updated     int `json:"updated"`
	Unchanged   int `json:"unchanged"`
	Partial     int `json:"partial"`
	Skipped     int `json:"skipped"`
	Conflicts   int `json:"conflicts"`
	Failed      int `json:"failed"`
	Leads       int `json:"leads"`
	Customers   int `json:"customers"`
	Assets      int `json:"assets"`
	Records     int `json:"records"`
	Operations  int `json:"operations"`
	Meetings    int `json:"meetings"`
	Groups      int `json:"groups"`
	Attachments int `json:"attachments"`
}

type HistoryImportCaseResult struct {
	CaseID             string               `json:"case_id"`
	Action             string               `json:"action"`
	LeadID             uint64               `json:"lead_id,omitempty"`
	CustomerID         uint64               `json:"customer_id,omitempty"`
	AssetIDs           map[string]uint64    `json:"asset_ids,omitempty"`
	WorkflowInstanceID uint64               `json:"workflow_instance_id,omitempty"`
	Counts             HistoryImportCounts  `json:"counts"`
	Issues             []HistoryImportIssue `json:"issues,omitempty"`
	Error              string               `json:"error,omitempty"`
}

type HistoryImportResult struct {
	BatchID string                    `json:"batch_id"`
	Applied bool                      `json:"applied"`
	Counts  HistoryImportCounts       `json:"counts"`
	Cases   []HistoryImportCaseResult `json:"cases"`
	Issues  []HistoryImportIssue      `json:"issues,omitempty"`
}

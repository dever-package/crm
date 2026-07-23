package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

const (
	HistoryImportStatusImported  = "imported"
	HistoryImportStatusPartial   = "partial"
	HistoryImportStatusUnchanged = "unchanged"
	HistoryImportStatusSkipped   = "skipped"
	HistoryImportStatusConflict  = "conflict"
	HistoryImportStatusFailed    = "failed"
)

var historyImportStatusOptions = []map[string]any{
	{"id": HistoryImportStatusImported, "value": "已导入"},
	{"id": HistoryImportStatusPartial, "value": "部分成功，待重试"},
	{"id": HistoryImportStatusUnchanged, "value": "无变化"},
	{"id": HistoryImportStatusSkipped, "value": "已跳过"},
	{"id": HistoryImportStatusConflict, "value": "存在冲突"},
	{"id": HistoryImportStatusFailed, "value": "导入失败"},
}

type HistoryImportRecord struct {
	ID                   uint64     `dorm:"primaryKey;autoIncrement;comment:历史导入审计ID"`
	BatchID              string     `dorm:"type:varchar(96);not null;default:'';comment:导入批次"`
	SourceKey            string     `dorm:"type:varchar(192);not null;comment:来源唯一键"`
	SourceTableKey       string     `dorm:"type:varchar(64);not null;comment:来源表标识"`
	SourceTableName      string     `dorm:"type:varchar(128);not null;comment:来源表名称"`
	SourceTableID        string     `dorm:"type:varchar(64);not null;comment:飞书表ID"`
	SourceRecordID       string     `dorm:"type:varchar(64);not null;comment:飞书记录ID"`
	InternalCaseID       string     `dorm:"type:varchar(96);not null;default:'';comment:历史案件ID"`
	SourceChecksum       string     `dorm:"type:varchar(64);not null;comment:来源校验值"`
	LeadID               uint64     `dorm:"type:bigint;not null;default:0;comment:目标线索"`
	CustomerID           uint64     `dorm:"type:bigint;not null;default:0;comment:目标客户"`
	AssetID              uint64     `dorm:"type:bigint;not null;default:0;comment:目标客户资产"`
	WorkflowInstanceID   uint64     `dorm:"type:bigint;not null;default:0;comment:目标流程实例"`
	TargetJSON           string     `dorm:"type:text;not null;default:'{}';comment:目标记录JSON"`
	RawSnapshotJSON      string     `dorm:"type:text;not null;default:'{}';comment:来源快照JSON"`
	Status               string     `dorm:"type:varchar(32);not null;default:'imported';comment:导入状态"`
	ErrorMessage         string     `dorm:"type:text;not null;default:'';comment:错误信息"`
	SourceCreatedAt      *time.Time `dorm:"null;comment:来源创建时间"`
	SourceLastModifiedAt *time.Time `dorm:"null;comment:来源更新时间"`
	ImportedAt           *time.Time `dorm:"null;comment:导入时间"`
	CreatedAt            time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt            time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type HistoryImportRecordIndex struct {
	SourceKey        struct{} `unique:"source_key"`
	BatchStatus      struct{} `index:"batch_id,status,id"`
	CaseStatus       struct{} `index:"internal_case_id,status,id"`
	SourceTable      struct{} `index:"source_table_key,source_record_id,id"`
	Lead             struct{} `index:"lead_id,id"`
	CustomerAsset    struct{} `index:"customer_id,asset_id,id"`
	WorkflowInstance struct{} `index:"workflow_instance_id,id"`
}

func NewHistoryImportRecordModel() *orm.Model[HistoryImportRecord] {
	return orm.LoadModel[HistoryImportRecord]("飞书历史导入审计", "crm_history_import_record", orm.ModelConfig{
		Index:    HistoryImportRecordIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": historyImportStatusOptions,
		},
		Relations: []orm.Relation{
			leadRelation,
			customerRelation,
			assetRelation,
			workflowInstanceRelation,
		},
	})
}

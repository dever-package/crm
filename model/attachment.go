package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Attachment struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:附件ID"`
	CustomerID     uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID        uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	TaskID         uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	OperationLogID uint64    `dorm:"type:bigint;not null;default:0;comment:操作记录"`
	DataRecordID   uint64    `dorm:"type:bigint;not null;default:0;comment:沉淀数据"`
	FieldID        uint64    `dorm:"type:bigint;not null;default:0;comment:字段ID"`
	FileName       string    `dorm:"type:varchar(255);not null;comment:文件名"`
	FileURL        string    `dorm:"type:text;not null;comment:文件地址"`
	FileType       string    `dorm:"type:varchar(32);not null;default:'other';comment:文件类型"`
	UploaderID     uint64    `dorm:"type:bigint;not null;default:0;comment:上传人"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type AttachmentIndex struct {
	CustomerTime struct{} `index:"customer_id,created_at,id"`
	AssetTime    struct{} `index:"asset_id,created_at,id"`
	TaskTime     struct{} `index:"task_id,created_at,id"`
	Operation    struct{} `index:"operation_log_id,created_at,id"`
	DataRecord   struct{} `index:"data_record_id,created_at,id"`
}

func NewAttachmentModel() *orm.Model[Attachment] {
	return orm.LoadModel[Attachment]("CRM附件", "crm_attachment", orm.ModelConfig{
		Index:    AttachmentIndex{},
		Order:    "id desc",
		Database: "default",
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			taskRelation,
			dataRecordRelation,
			uploaderRelation,
		},
	})
}

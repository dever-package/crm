package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Lead struct {
	ID                  uint64     `dorm:"primaryKey;autoIncrement;comment:线索ID"`
	Code                string     `dorm:"type:varchar(32);not null;default:'';comment:线索编号"`
	Name                string     `dorm:"type:varchar(64);not null;comment:姓名"`
	Phone               string     `dorm:"type:varchar(32);not null;default:'';comment:手机号"`
	Wechat              string     `dorm:"type:varchar(64);not null;default:'';comment:微信号"`
	SourceID            uint64     `dorm:"type:bigint;not null;default:1;comment:来源"`
	ChannelID           uint64     `dorm:"type:bigint;not null;default:1;comment:渠道"`
	ExternalID          string     `dorm:"type:varchar(128);not null;default:'';comment:外部线索ID"`
	City                string     `dorm:"type:varchar(64);not null;default:'';comment:城市"`
	InitialNeed         string     `dorm:"type:text;not null;default:'';comment:初始诉求"`
	Status              string     `dorm:"type:varchar(24);not null;default:'pending';comment:状态"`
	DuplicateLeadID     uint64     `dorm:"type:bigint;not null;default:0;comment:重复线索"`
	DuplicateCustomerID uint64     `dorm:"type:bigint;not null;default:0;comment:重复客户"`
	DuplicateReason     string     `dorm:"type:varchar(255);not null;default:'';comment:重复原因"`
	InvalidReasonID     uint64     `dorm:"type:bigint;not null;default:0;comment:无效原因"`
	InvalidNote         string     `dorm:"type:text;not null;default:'';comment:无效说明"`
	CustomerID          uint64     `dorm:"type:bigint;not null;default:0;comment:转化客户"`
	RecordJSON          string     `dorm:"type:text;not null;default:'{}';comment:线索完整数据"`
	OwnerDepartmentID   uint64     `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	OwnerStaffID        uint64     `dorm:"type:bigint;not null;default:0;comment:负责人"`
	CreatedByStaffID    uint64     `dorm:"type:bigint;not null;default:0;comment:创建人员"`
	ConvertedByStaffID  uint64     `dorm:"type:bigint;not null;default:0;comment:转化人员"`
	ConvertedAt         *time.Time `dorm:"null;comment:转化时间"`
	CreatedAt           time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt           time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
	OriginTaskID        uint64     `dorm:"type:bigint;not null;default:0;comment:来源任务"`
	OriginFormID        uint64     `dorm:"type:bigint;not null;default:0;comment:来源表单"`
	InputSnapshotJSON   string     `dorm:"type:text;not null;default:'{}';comment:录入快照"`
}

type LeadIndex struct {
	Code              struct{} `unique:"code"`
	Phone             struct{} `index:"phone,id"`
	Wechat            struct{} `index:"wechat,id"`
	SourceExternal    struct{} `index:"source_id,external_id,id"`
	OwnerPage         struct{} `index:"owner_department_id,id"`
	OwnerStatusPage   struct{} `index:"owner_department_id,status,id"`
	Customer          struct{} `index:"customer_id,status,id"`
	DuplicateLead     struct{} `index:"duplicate_lead_id,id"`
	DuplicateCustomer struct{} `index:"duplicate_customer_id,id"`
}

func NewLeadModel() *orm.Model[Lead] {
	return orm.LoadModel[Lead]("线索", "crm_lead", orm.ModelConfig{
		Index:    LeadIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": leadStatusOptions,
		},
	})
}

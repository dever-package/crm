package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type PublicResourceBooking struct {
	ID                 uint64     `dorm:"primaryKey;autoIncrement;comment:预定ID"`
	ResourceID         uint64     `dorm:"type:bigint;not null;comment:公共资源"`
	CustomerID         uint64     `dorm:"type:bigint;not null;comment:客户"`
	AssetID            uint64     `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	TaskID             uint64     `dorm:"type:bigint;not null;default:0;comment:任务"`
	OperationLogID     uint64     `dorm:"type:bigint;not null;default:0;comment:操作记录"`
	StageCode          string     `dorm:"type:varchar(32);not null;default:'';comment:客户阶段"`
	BookingStatus      string     `dorm:"type:varchar(32);not null;default:'reserved';comment:预定状态"`
	Title              string     `dorm:"type:varchar(128);not null;default:'';comment:用途"`
	Remark             string     `dorm:"type:text;not null;default:'';comment:备注"`
	StartAt            time.Time  `dorm:"not null;comment:开始时间"`
	EndAt              time.Time  `dorm:"not null;comment:结束时间"`
	BookerStaffID      uint64     `dorm:"type:bigint;not null;default:0;comment:预定人员"`
	BookerDepartmentID uint64     `dorm:"type:bigint;not null;default:0;comment:预定部门"`
	ApprovedByStaffID  uint64     `dorm:"type:bigint;not null;default:0;comment:确认人员"`
	ApprovedAt         *time.Time `dorm:"null;comment:确认时间"`
	CreatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type PublicResourceBookingIndex struct {
	ResourceTime struct{} `index:"resource_id,start_at,end_at,booking_status,id"`
	CustomerTime struct{} `index:"customer_id,start_at,id"`
	StaffTime    struct{} `index:"booker_staff_id,start_at,id"`
	StatusTime   struct{} `index:"booking_status,start_at,id"`
	TaskTime     struct{} `index:"task_id,created_at,id"`
}

func NewPublicResourceBookingModel() *orm.Model[PublicResourceBooking] {
	return orm.LoadModel[PublicResourceBooking]("资源预定", "crm_public_resource_booking", orm.ModelConfig{
		Index:    PublicResourceBookingIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"booking_status": resourceBookingStatusOptions,
		},
		Relations: []orm.Relation{
			resourceRelation,
			customerRelation,
			assetRelation,
			taskRelation,
			bookerStaffRelation,
			bookerDepartmentRelation,
			approvedByStaffRelation,
		},
	})
}

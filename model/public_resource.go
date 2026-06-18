package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type PublicResource struct {
	ID                 uint64    `dorm:"primaryKey;autoIncrement;comment:资源ID"`
	ResourceCateID     uint64    `dorm:"type:bigint;not null;default:1;comment:资源分类"`
	Name               string    `dorm:"type:varchar(128);not null;comment:资源名称"`
	Location           string    `dorm:"type:varchar(128);not null;default:'';comment:位置"`
	Capacity           int       `dorm:"type:int;not null;default:0;comment:容量"`
	NeedConfirm        bool      `dorm:"not null;default:false;comment:是否需要确认"`
	OwnerDepartmentID  uint64    `dorm:"type:bigint;not null;default:0;comment:负责部门"`
	OwnerStaffID       uint64    `dorm:"type:bigint;not null;default:0;comment:负责人员"`
	AvailableStartTime string    `dorm:"type:varchar(16);not null;default:'';comment:可预定开始时间"`
	AvailableEndTime   string    `dorm:"type:varchar(16);not null;default:'';comment:可预定结束时间"`
	Remark             string    `dorm:"type:text;not null;default:'';comment:备注"`
	Status             int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort               int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type PublicResourceIndex struct {
	CateStatusSort struct{} `index:"resource_cate_id,status,sort,id"`
	OwnerStatus    struct{} `index:"owner_department_id,owner_staff_id,status,id"`
	StatusSort     struct{} `index:"status,sort,id"`
}

func NewPublicResourceModel() *orm.Model[PublicResource] {
	return orm.LoadModel[PublicResource]("公共资源", "crm_public_resource", orm.ModelConfig{
		Index:    PublicResourceIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"status": statusOptions,
		},
		Relations: []orm.Relation{
			resourceCateRelation,
			ownerDepartmentRelation,
			ownerStaffRelation,
		},
	})
}

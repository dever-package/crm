package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type DispatchPool struct {
	ID           uint64    `dorm:"primaryKey;autoIncrement;comment:派单池ID"`
	DepartmentID uint64    `dorm:"type:bigint;not null;comment:部门"`
	Name         string    `dorm:"type:varchar(64);not null;comment:派单池名称"`
	PoolType     string    `dorm:"type:varchar(32);not null;default:'group';comment:派单池类型"`
	Status       int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort         int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt    time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type DispatchPoolIndex struct {
	DepartmentStatus struct{} `index:"department_id,status,sort,id"`
	TypeStatus       struct{} `index:"department_id,pool_type,status,id"`
}

func NewDispatchPoolModel() *orm.Model[DispatchPool] {
	return orm.LoadModel[DispatchPool]("派单池", "crm_dispatch_pool", orm.ModelConfig{
		Index:    DispatchPoolIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"pool_type": dispatchPoolTypeOptions,
			"status":    statusOptions,
		},
		Relations: []orm.Relation{departmentRelation},
	})
}

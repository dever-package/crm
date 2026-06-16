package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FinanceType struct {
	ID        uint64    `dorm:"primaryKey;autoIncrement;comment:财务类型ID"`
	Code      string    `dorm:"type:varchar(64);not null;comment:类型编码"`
	Name      string    `dorm:"type:varchar(128);not null;comment:类型名称"`
	Direction string    `dorm:"type:varchar(16);not null;default:'income';comment:收支方向"`
	Status    int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort      int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type FinanceTypeIndex struct {
	Code       struct{} `unique:"code"`
	StatusSort struct{} `index:"status,sort,id"`
	Direction  struct{} `index:"direction,status,id"`
}

func NewFinanceTypeModel() *orm.Model[FinanceType] {
	return orm.LoadModel[FinanceType]("财务类型", "crm_finance_type", orm.ModelConfig{
		Index:    FinanceTypeIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"direction": financeDirectionOptions,
			"status":    statusOptions,
		},
	})
}

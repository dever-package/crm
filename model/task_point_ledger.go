package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type TaskPointLedger struct {
	ID             uint64    `dorm:"primaryKey;autoIncrement;comment:任务积分流水ID"`
	CustomerID     uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID        uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	TaskID         uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	OperationLogID uint64    `dorm:"type:bigint;not null;default:0;comment:操作记录"`
	TodoID         uint64    `dorm:"type:bigint;not null;default:0;comment:待办"`
	Points         float64   `dorm:"type:double precision;not null;default:0;comment:积分"`
	StaffID        uint64    `dorm:"type:bigint;not null;default:0;comment:获得人员"`
	DepartmentID   uint64    `dorm:"type:bigint;not null;default:0;comment:所属部门"`
	ResultValue    string    `dorm:"type:varchar(64);not null;default:'';comment:任务结果"`
	Source         string    `dorm:"type:varchar(64);not null;default:'task_complete';comment:来源"`
	CreatedAt      time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type TaskPointLedgerIndex struct {
	Operation      struct{} `index:"operation_log_id,todo_id,source"`
	StaffTime      struct{} `index:"staff_id,created_at,id"`
	DepartmentTime struct{} `index:"department_id,created_at,id"`
	TaskTime       struct{} `index:"task_id,created_at,id"`
	CustomerTime   struct{} `index:"customer_id,created_at,id"`
}

func NewTaskPointLedgerModel() *orm.Model[TaskPointLedger] {
	return orm.LoadModel[TaskPointLedger]("任务积分流水", "crm_task_point_ledger", orm.ModelConfig{
		Index:    TaskPointLedgerIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"source": taskPointLedgerSourceOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			taskRelation,
			operationLogRelation,
			todoRelation,
			staffRelation,
			departmentRelation,
		},
	})
}

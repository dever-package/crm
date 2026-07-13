package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type FinanceLedger struct {
	ID                 uint64    `dorm:"primaryKey;autoIncrement;comment:财务流水ID"`
	CustomerID         uint64    `dorm:"type:bigint;not null;comment:客户"`
	AssetID            uint64    `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	BusinessObjectID   uint64    `dorm:"type:bigint;not null;default:0;comment:业务对象"`
	WorkflowInstanceID uint64    `dorm:"type:bigint;not null;default:0;comment:流程实例"`
	CustomerProductID  uint64    `dorm:"type:bigint;not null;default:0;comment:客户产品"`
	TaskID             uint64    `dorm:"type:bigint;not null;default:0;comment:任务"`
	OperationLogID     uint64    `dorm:"type:bigint;not null;default:0;comment:操作记录"`
	DataFieldID        uint64    `dorm:"type:bigint;not null;default:0;comment:数据字段"`
	FinanceTypeID      uint64    `dorm:"type:bigint;not null;default:0;comment:财务类型"`
	FinanceTypeCode    string    `dorm:"type:varchar(64);not null;default:'';comment:财务类型编码"`
	FinanceTypeName    string    `dorm:"type:varchar(128);not null;default:'';comment:财务类型名称"`
	Direction          string    `dorm:"type:varchar(16);not null;default:'income';comment:收支方向"`
	Amount             float64   `dorm:"type:double precision;not null;default:0;comment:金额"`
	RawValue           string    `dorm:"type:text;not null;default:'';comment:原始值"`
	StaffID            uint64    `dorm:"type:bigint;not null;default:0;comment:操作人员"`
	DepartmentID       uint64    `dorm:"type:bigint;not null;default:0;comment:操作部门"`
	Source             string    `dorm:"type:varchar(32);not null;default:'form';comment:来源"`
	ReverseOfID        uint64    `dorm:"type:bigint;not null;default:0;comment:冲正来源"`
	CreatedAt          time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
}

type FinanceLedgerIndex struct {
	OperationField     struct{} `unique:"workflow_instance_id,operation_log_id,data_field_id,source"`
	CustomerTime       struct{} `index:"customer_id,created_at,id"`
	AssetTime          struct{} `index:"asset_id,created_at,id"`
	BusinessObjectTime struct{} `index:"business_object_id,created_at,id"`
	WorkflowTime       struct{} `index:"workflow_instance_id,created_at,id"`
	ProductTime        struct{} `index:"customer_product_id,created_at,id"`
	FinanceTime        struct{} `index:"finance_type_id,created_at,id"`
	FieldTime          struct{} `index:"data_field_id,created_at,id"`
	TaskTime           struct{} `index:"task_id,created_at,id"`
	StaffTime          struct{} `index:"staff_id,created_at,id"`
	DepartmentTime     struct{} `index:"department_id,created_at,id"`
	CreatedTime        struct{} `index:"created_at,id"`
}

func NewFinanceLedgerModel() *orm.Model[FinanceLedger] {
	return orm.LoadModel[FinanceLedger]("财务流水", "crm_finance_ledger", orm.ModelConfig{
		Index:    FinanceLedgerIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"direction": financeDirectionOptions,
			"source":    financeLedgerSourceOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			businessObjectRelation,
			workflowInstanceRelation,
			customerProductRelation,
			taskRelation,
			operationLogRelation,
			dataFieldRelation,
			financeTypeRelation,
			staffRelation,
			departmentRelation,
		},
	})
}

package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type WorkTodo struct {
	ID                      uint64     `dorm:"primaryKey;autoIncrement;comment:协作待办ID"`
	CustomerID              uint64     `dorm:"type:bigint;not null;comment:客户"`
	AssetID                 uint64     `dorm:"type:bigint;not null;default:0;comment:客户资产"`
	SourceTaskID            uint64     `dorm:"type:bigint;not null;default:0;comment:来源任务"`
	ParentOperationLogID    uint64     `dorm:"type:bigint;not null;default:0;comment:派单操作记录"`
	SubTaskName             string     `dorm:"type:varchar(128);not null;default:'';comment:子任务名称"`
	FormID                  uint64     `dorm:"type:bigint;not null;default:0;comment:处理资料模板"`
	AssigneeDepartmentID    uint64     `dorm:"type:bigint;not null;default:0;comment:处理部门"`
	AssigneeStaffID         uint64     `dorm:"type:bigint;not null;default:0;comment:处理人员"`
	Required                bool       `dorm:"not null;default:true;comment:是否必做"`
	Sort                    int        `dorm:"type:int;not null;default:100;comment:排序"`
	Status                  string     `dorm:"type:varchar(32);not null;default:'pending';comment:状态"`
	AssignedAt              time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:派单时间"`
	CompletedAt             *time.Time `dorm:"null;comment:完成时间"`
	CompletedOperationLogID uint64     `dorm:"type:bigint;not null;default:0;comment:完成操作记录"`
	CreatedByStaffID        uint64     `dorm:"type:bigint;not null;default:0;comment:创建人员"`
	CreatedAt               time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt               time.Time  `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type WorkTodoIndex struct {
	AssigneeStatus struct{} `index:"assignee_department_id,assignee_staff_id,status,sort,id"`
	CustomerStatus struct{} `index:"customer_id,asset_id,status,id"`
	SourceStatus   struct{} `index:"source_task_id,status,id"`
	ParentStatus   struct{} `index:"parent_operation_log_id,status,id"`
	CompletedLog   struct{} `index:"completed_operation_log_id,id"`
}

func NewWorkTodoModel() *orm.Model[WorkTodo] {
	return orm.LoadModel[WorkTodo]("协作待办", "crm_work_todo", orm.ModelConfig{
		Index:    WorkTodoIndex{},
		Order:    "id desc",
		Database: "default",
		Options: map[string]any{
			"status": workTodoStatusOptions,
		},
		Relations: []orm.Relation{
			customerRelation,
			assetRelation,
			{
				Field:      "source_task_id",
				Option:     "crm.NewTaskModel",
				OptionKeys: []string{"name", "task_type"},
			},
			{
				Field:      "assignee_department_id",
				Option:     "crm.NewDepartmentModel",
				OptionKeys: []string{"name", "code"},
			},
			{
				Field:      "assignee_staff_id",
				Option:     "crm.NewStaffModel",
				OptionKeys: []string{"name", "phone"},
			},
			{
				Field:      "created_by_staff_id",
				Option:     "crm.NewStaffModel",
				OptionKeys: []string{"name", "phone"},
			},
		},
	})
}

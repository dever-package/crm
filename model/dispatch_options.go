package model

const (
	DispatchPoolTypeDirect = "direct"
	DispatchPoolTypeGroup  = "group"
)

const (
	DispatchTypeAuto   = "auto"
	DispatchTypeManual = "manual"
)

const (
	DispatchSourceStage   = "stage"
	DispatchSourceTask    = "task"
	DispatchSourcePending = "pending"
	DispatchSourceManual  = "manual"
	DispatchSourceLead    = "lead_handoff"
)

const (
	LeadDispatchHandoffPending    = "pending"
	LeadDispatchHandoffProcessing = "processing"
	LeadDispatchHandoffCompleted  = "completed"
)

const DefaultDispatchScheduleJSON = `{"1":[[0,1440]],"2":[[0,1440]],"3":[[0,1440]],"4":[[0,1440]],"5":[[0,1440]],"6":[[0,1440]],"7":[[0,1440]]}`

var dispatchPoolTypeOptions = []map[string]any{
	{"id": DispatchPoolTypeDirect, "value": "按员工分配"},
	{"id": DispatchPoolTypeGroup, "value": "工作组"},
}

var dispatchTypeOptions = []map[string]any{
	{"id": DispatchTypeAuto, "value": "自动派单"},
	{"id": DispatchTypeManual, "value": "人工派单"},
}

var leadDispatchHandoffStatusOptions = []map[string]any{
	{"id": LeadDispatchHandoffPending, "value": "待派单"},
	{"id": LeadDispatchHandoffProcessing, "value": "派单中"},
	{"id": LeadDispatchHandoffCompleted, "value": "已派单"},
}

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	uploadrepo "github.com/dever-package/front/service/upload/repository"
	"github.com/shemic/dever/orm"

	crmmodel "github.com/dever-package/crm/model"
)

const maximumArrivalVideoCount = 5

func managedScheduleEventIDs(ctx context.Context, staff *WorkStaffSession) []uint64 {
	if staff == nil || staff.ID == 0 {
		return []uint64{}
	}
	resourceIDs := make([]uint64, 0)
	if staff.DepartmentID > 0 {
		for _, resource := range crmmodel.NewPublicResourceModel().Select(ctx, map[string]any{
			"owner_department_id": staff.DepartmentID,
		}) {
			if resource != nil {
				resourceIDs = append(resourceIDs, resource.ID)
			}
		}
	}
	for _, resource := range crmmodel.NewPublicResourceModel().Select(ctx, map[string]any{
		"owner_staff_id": staff.ID,
	}) {
		if resource != nil {
			resourceIDs = append(resourceIDs, resource.ID)
		}
	}
	resourceIDs = uniqueUint64Values(resourceIDs)
	if len(resourceIDs) == 0 {
		return []uint64{}
	}
	eventIDs := make([]uint64, 0)
	for _, booking := range crmmodel.NewPublicResourceBookingModel().Select(ctx, map[string]any{
		"resource_id": resourceIDs,
	}) {
		if booking != nil {
			eventIDs = append(eventIDs, booking.ScheduleEventID)
		}
	}
	return uniqueUint64Values(eventIDs)
}

func canManageMeetingEvidence(ctx context.Context, staff *WorkStaffSession, event *crmmodel.ScheduleEvent) bool {
	if staff == nil || staff.ID == 0 || event == nil || event.ScheduleType != crmmodel.ScheduleTypeMeeting {
		return false
	}
	if event.ArrivalStatus != crmmodel.MeetingArrivalArrived && event.CustomerArrivedAt == nil {
		return false
	}
	for _, booking := range crmmodel.NewPublicResourceBookingModel().Select(ctx, map[string]any{
		"schedule_event_id": event.ID,
	}) {
		if booking == nil {
			continue
		}
		resource := crmmodel.NewPublicResourceModel().Find(ctx, map[string]any{"id": booking.ResourceID})
		if resource != nil && (resource.OwnerStaffID == staff.ID || resource.OwnerDepartmentID > 0 && resource.OwnerDepartmentID == staff.DepartmentID) {
			return true
		}
	}
	return false
}

func scheduleArrivalVideoFiles(ctx context.Context, eventID uint64) []map[string]any {
	rows := crmmodel.NewAttachmentModel().Select(ctx, map[string]any{
		"schedule_event_id": eventID,
		"usage":             crmmodel.AttachmentUsageArrivalVideo,
	}, map[string]any{"order": "id asc"})
	fileIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.UploadFileID > 0 {
			fileIDs = append(fileIDs, row.UploadFileID)
		}
	}
	return workUploadFilePayloads(ctx, fileIDs)
}

func (WorkService) SaveScheduleArrivalVideos(ctx context.Context, staff *WorkStaffSession, payload map[string]any) (map[string]any, error) {
	if staff == nil || staff.ID == 0 {
		return nil, fmt.Errorf("请先登录")
	}
	eventID := firstUint64(payload, "schedule_event_id", "scheduleEventId", "id")
	event := crmmodel.NewScheduleEventModel().Find(ctx, map[string]any{"id": eventID})
	if event == nil || event.ScheduleType != crmmodel.ScheduleTypeMeeting {
		return nil, fmt.Errorf("会议日程不存在")
	}
	if !canManageMeetingEvidence(ctx, staff, event) {
		return nil, fmt.Errorf("无权维护该会议的到访视频")
	}
	fileIDs := uniqueUint64Values(uint64ListFromAny(firstPresent(payload, "file_ids", "fileIds")))
	if len(fileIDs) > maximumArrivalVideoCount {
		return nil, fmt.Errorf("到访视频最多上传 %d 个", maximumArrivalVideoCount)
	}
	files := make([]uploadrepo.UploadFile, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		file, err := uploadrepo.FindUploadFile(ctx, fileID)
		if err != nil || file.ID == 0 {
			return nil, fmt.Errorf("上传文件不存在或已失效")
		}
		if !isArrivalVideo(file) {
			return nil, fmt.Errorf("到访附件只能上传视频文件")
		}
		files = append(files, file)
	}
	err := orm.Transaction(ctx, func(txCtx context.Context) error {
		operationID := recordScheduleArrivalVideoOperation(txCtx, staff, event, fileIDs)
		if operationID == 0 {
			return fmt.Errorf("到访视频操作记录创建失败")
		}
		model := crmmodel.NewAttachmentModel()
		model.Delete(txCtx, map[string]any{
			"schedule_event_id": event.ID,
			"usage":             crmmodel.AttachmentUsageArrivalVideo,
		})
		now := time.Now()
		for _, file := range files {
			fileType := strings.TrimSpace(file.Mime)
			if fileType == "" {
				fileType = "video"
			}
			if model.Insert(txCtx, map[string]any{
				"customer_id":       event.CustomerID,
				"asset_id":          event.AssetID,
				"task_id":           event.SourceTaskID,
				"operation_log_id":  operationID,
				"schedule_event_id": event.ID,
				"upload_file_id":    file.ID,
				"usage":             crmmodel.AttachmentUsageArrivalVideo,
				"file_name":         file.Name,
				"file_url":          file.Path,
				"file_type":         fileType,
				"uploader_id":       staff.ID,
				"created_at":        now,
			}) == 0 {
				return fmt.Errorf("到访视频保存失败")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"schedule_event_id": event.ID,
		"file_ids":          fileIDs,
		"files":             scheduleArrivalVideoFiles(ctx, event.ID),
		"saved":             true,
	}, nil
}

func isArrivalVideo(file uploadrepo.UploadFile) bool {
	if strings.EqualFold(strings.TrimSpace(file.Kind), "video") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(file.Mime)), "video/") {
		return true
	}
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(file.Ext), ".")) {
	case "mp4", "mov", "avi", "mkv", "webm", "m4v", "3gp":
		return true
	default:
		return false
	}
}

func recordScheduleArrivalVideoOperation(ctx context.Context, staff *WorkStaffSession, event *crmmodel.ScheduleEvent, fileIDs []uint64) uint64 {
	if event == nil {
		return 0
	}
	if instance := crmmodel.NewWorkflowInstanceModel().Find(ctx, map[string]any{"id": event.SourceWorkflowInstanceID}); instance != nil {
		return recordWorkManagementOperation(ctx, staff, instance, "arrival_video_saved", "到访视频已保存", fmt.Sprintf("共 %d 个视频", len(fileIDs)), map[string]any{
			"schedule_event_id": event.ID,
			"file_ids":          fileIDs,
		})
	}
	return uint64(crmmodel.NewOperationLogModel().Insert(ctx, map[string]any{
		"customer_id":            event.CustomerID,
		"asset_id":               event.AssetID,
		"workflow_instance_id":   event.SourceWorkflowInstanceID,
		"task_id":                event.SourceTaskID,
		"result_value":           "arrival_video_saved",
		"title":                  "到访视频已保存",
		"content":                fmt.Sprintf("共 %d 个视频", len(fileIDs)),
		"data_snapshot_json":     jsonText(map[string]any{"schedule_event_id": event.ID, "file_ids": fileIDs}),
		"operator_staff_id":      staff.ID,
		"operator_department_id": staff.DepartmentID,
		"created_at":             time.Now(),
	}))
}

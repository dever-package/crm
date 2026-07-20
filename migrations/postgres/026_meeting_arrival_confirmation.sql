BEGIN;

ALTER TABLE gjj_crm_task
    ADD COLUMN IF NOT EXISTS meeting_arrival_required BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE gjj_crm_schedule_event
    ADD COLUMN IF NOT EXISTS customer_arrived_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS customer_arrived_by_staff_id BIGINT NOT NULL DEFAULT 0;

UPDATE gjj_crm_task AS task
SET name = '预约及到访确认',
    meeting_arrival_required = TRUE,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_stage AS stage,
     gjj_crm_workflow AS workflow
WHERE task.stage_id = stage.id
  AND stage.workflow_id = workflow.id
  AND workflow.name = '签约流程'
  AND stage.name = '邀约到访'
  AND task.name IN ('预约会议室', '预约及到访确认')
  AND task.task_type = 'form'
  AND task.meeting_enabled = TRUE
  AND (task.name = '预约会议室' OR task.meeting_arrival_required = FALSE);

COMMIT;

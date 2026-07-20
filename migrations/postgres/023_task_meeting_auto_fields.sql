BEGIN;

ALTER TABLE gjj_crm_task
    ADD COLUMN IF NOT EXISTS meeting_enabled BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE gjj_crm_task
SET meeting_enabled = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE meeting_enabled = FALSE
  AND meeting_start_field_id > 0
  AND meeting_duration_field_id > 0
  AND meeting_resource_field_id > 0;

UPDATE gjj_crm_form_field AS form_field
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_task AS task
WHERE task.meeting_enabled = TRUE
  AND form_field.form_id = task.form_id
  AND form_field.status = 1
  AND form_field.data_field_id IN (
      task.meeting_start_field_id,
      task.meeting_duration_field_id,
      task.meeting_resource_field_id
  );

COMMIT;

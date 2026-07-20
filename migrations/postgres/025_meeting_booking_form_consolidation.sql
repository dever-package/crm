BEGIN;

-- The system meeting start control replaces this duplicate form input.
-- Keep the data field and historical record values for compatibility.
UPDATE gjj_crm_form_field AS form_field
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_task AS task,
     gjj_crm_data_field AS field
WHERE task.meeting_enabled = TRUE
  AND task.form_id = form_field.form_id
  AND field.id = form_field.data_field_id
  AND field.field_key = 'yydf.yujidaofangshijian'
  AND form_field.status = 1;

COMMIT;

BEGIN;

CREATE TABLE IF NOT EXISTS gjj_crm_task_stat_field (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL,
    data_field_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_task_stat_field_task_field
    ON gjj_crm_task_stat_field (task_id, data_field_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_stat_field_field_task
    ON gjj_crm_task_stat_field (data_field_id, task_id, id);

ALTER TABLE gjj_crm_stat_field_value
    ADD COLUMN IF NOT EXISTS lead_id BIGINT NOT NULL DEFAULT 0;

DROP INDEX IF EXISTS uidx_gjj_crm_stat_field_value_owner_data_field;
DROP INDEX IF EXISTS idx_gjj_crm_stat_field_value_owner_data_field;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_stat_field_value_owner_data_field
    ON gjj_crm_stat_field_value (
        lead_id,
        customer_id,
        asset_id,
        workflow_instance_id,
        data_field_id
    );
CREATE INDEX IF NOT EXISTS idx_gjj_crm_stat_field_value_lead_time
    ON gjj_crm_stat_field_value (lead_id, updated_at, id);

-- Migration 024 retired every legacy usage row with status=2. The mapping rows
-- are retained for audit, so migrate statistic bindings by type and require the
-- task, form field, and data field themselves to remain enabled.
INSERT INTO gjj_crm_task_stat_field (task_id, data_field_id, created_at)
SELECT DISTINCT
    task.id,
    binding.data_field_id,
    CURRENT_TIMESTAMP
FROM gjj_crm_data_usage AS usage
INNER JOIN gjj_crm_data_usage_field AS binding
    ON binding.usage_id = usage.id
   AND binding.data_field_id > 0
INNER JOIN gjj_crm_data_field AS data_field
    ON data_field.id = binding.data_field_id
   AND data_field.status = 1
   AND data_field.field_type NOT IN ('group', 'attachment')
INNER JOIN gjj_crm_form_field AS form_field
    ON form_field.data_field_id = binding.data_field_id
   AND form_field.status = 1
INNER JOIN gjj_crm_task AS task
    ON task.form_id = form_field.form_id
   AND task.task_type IN ('form', 'approval')
   AND task.status = 1
WHERE usage.usage_type = 'stat'
ON CONFLICT (task_id, data_field_id) DO NOTHING;

COMMIT;

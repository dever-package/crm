BEGIN;

ALTER TABLE gjj_crm_task
    ADD COLUMN IF NOT EXISTS customer_follow_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS gjj_crm_task_finance_type (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL,
    finance_type_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_task_finance_type_task_finance
    ON gjj_crm_task_finance_type (task_id, finance_type_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_finance_type_finance_task
    ON gjj_crm_task_finance_type (finance_type_id, task_id, id);

ALTER TABLE gjj_crm_finance_ledger
    ADD COLUMN IF NOT EXISTS finance_source_key VARCHAR(96) NOT NULL DEFAULT '';

UPDATE gjj_crm_finance_ledger
SET finance_source_key = CASE
    WHEN data_field_id > 0 THEN 'field:' || data_field_id::TEXT
    ELSE 'legacy:' || id::TEXT
END
WHERE finance_source_key = '';

DROP INDEX IF EXISTS uidx_gjj_crm_finance_ledger_operation_field;
DROP INDEX IF EXISTS idx_gjj_crm_finance_ledger_operation_field;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_finance_ledger_operation_source
    ON gjj_crm_finance_ledger (
        workflow_instance_id,
        operation_log_id,
        finance_source_key,
        source
    );
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_source_key_time
    ON gjj_crm_finance_ledger (finance_source_key, created_at, id);

-- Copy valid finance field bindings to their owning form tasks.
INSERT INTO gjj_crm_task_finance_type (task_id, finance_type_id, created_at)
SELECT DISTINCT
    task.id,
    binding.finance_type_id,
    CURRENT_TIMESTAMP
FROM gjj_crm_data_usage AS usage
INNER JOIN gjj_crm_data_usage_field AS binding
    ON binding.usage_id = usage.id
   AND binding.status = 1
   AND binding.finance_type_id > 0
INNER JOIN gjj_crm_finance_type AS finance_type
    ON finance_type.id = binding.finance_type_id
   AND finance_type.status = 1
INNER JOIN gjj_crm_form_field AS form_field
    ON form_field.data_field_id = binding.data_field_id
   AND form_field.status = 1
INNER JOIN gjj_crm_task AS task
    ON task.form_id = form_field.form_id
   AND task.task_type = 'form'
WHERE usage.usage_type = 'finance'
  AND usage.status = 1
ON CONFLICT (task_id, finance_type_id) DO NOTHING;

-- Rent receipt had lost its old data field; attach it to the rental registration task.
INSERT INTO gjj_crm_task_finance_type (task_id, finance_type_id, created_at)
SELECT DISTINCT
    task.id,
    finance_type.id,
    CURRENT_TIMESTAMP
FROM gjj_crm_task AS task
INNER JOIN gjj_crm_form_field AS form_field
    ON form_field.form_id = task.form_id
   AND form_field.status = 1
INNER JOIN gjj_crm_data_field AS field
    ON field.id = form_field.data_field_id
   AND field.field_key = 'monthly_rent_amount'
   AND field.status = 1
CROSS JOIN gjj_crm_finance_type AS finance_type
WHERE task.task_type = 'form'
  AND finance_type.code = 'rent_income'
  AND finance_type.status = 1
ON CONFLICT (task_id, finance_type_id) DO NOTHING;

-- Moving subsidy is a direct moving-cost entry, not a reusable data-template amount field.
INSERT INTO gjj_crm_task_finance_type (task_id, finance_type_id, created_at)
SELECT DISTINCT
    task.id,
    finance_type.id,
    CURRENT_TIMESTAMP
FROM gjj_crm_task AS task
INNER JOIN gjj_crm_form_field AS form_field
    ON form_field.form_id = task.form_id
   AND form_field.status = 1
INNER JOIN gjj_crm_data_field AS field
    ON field.id = form_field.data_field_id
   AND field.field_key = 'moving_subsidy_amount'
   AND field.status = 1
CROSS JOIN gjj_crm_finance_type AS finance_type
WHERE task.task_type = 'form'
  AND finance_type.code = 'moving_cost'
  AND finance_type.status = 1
ON CONFLICT (task_id, finance_type_id) DO NOTHING;

-- Disable data-template amount controls replaced by task finance controls.
UPDATE gjj_crm_form_field AS form_field
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_task AS task,
     gjj_crm_data_usage AS usage,
     gjj_crm_data_usage_field AS binding
WHERE task.form_id = form_field.form_id
  AND task.task_type = 'form'
  AND usage.id = binding.usage_id
  AND usage.usage_type = 'finance'
  AND usage.status = 1
  AND binding.status = 1
  AND binding.finance_type_id > 0
  AND binding.data_field_id = form_field.data_field_id
  AND form_field.status = 1;

UPDATE gjj_crm_form_field AS form_field
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_data_field AS field
WHERE field.id = form_field.data_field_id
  AND field.field_key = 'moving_subsidy_amount'
  AND form_field.status = 1;

-- Convert the existing follow-up field binding into a task switch.
UPDATE gjj_crm_task AS task
SET customer_follow_enabled = TRUE,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_form_field AS form_field
INNER JOIN gjj_crm_data_usage_field AS binding
    ON binding.data_field_id = form_field.data_field_id
   AND binding.status = 1
INNER JOIN gjj_crm_data_usage AS usage
    ON usage.id = binding.usage_id
   AND usage.usage_type = 'customer_follow_at'
   AND usage.status = 1
WHERE task.form_id = form_field.form_id
  AND task.task_type = 'form'
  AND form_field.status = 1;

UPDATE gjj_crm_form_field AS form_field
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_data_usage_field AS binding
INNER JOIN gjj_crm_data_usage AS usage
    ON usage.id = binding.usage_id
   AND usage.usage_type = 'customer_follow_at'
   AND usage.status = 1
WHERE form_field.data_field_id = binding.data_field_id
  AND binding.status = 1
  AND form_field.status = 1;

-- Existing schedules remain intact; only obsolete dynamic-field pointers are detached.
UPDATE gjj_crm_schedule_event
SET data_usage_field_id = 0,
    data_record_id = 0,
    data_field_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE schedule_type = 'customer_follow'
  AND (data_usage_field_id > 0 OR data_record_id > 0 OR data_field_id > 0);

-- Keep the old configuration rows for audit/rollback, but retire them from all reads.
UPDATE gjj_crm_data_usage_field
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
WHERE status <> 2;

UPDATE gjj_crm_data_usage
SET status = 2,
    updated_at = CURRENT_TIMESTAMP
WHERE status <> 2;

COMMIT;

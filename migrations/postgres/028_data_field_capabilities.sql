BEGIN;

ALTER TABLE gjj_crm_data_field
    ADD COLUMN IF NOT EXISTS finance_type_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS stat_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_field_finance_status
    ON gjj_crm_data_field (finance_type_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_field_stat_status
    ON gjj_crm_data_field (stat_enabled, status, sort, id);

-- Existing ledgers are the strongest evidence of the field-to-finance mapping.
-- When historical rows disagree, retain the most frequently used recent mapping.
WITH ledger_counts AS (
    SELECT
        data_field_id,
        finance_type_id,
        COUNT(*) AS ledger_count,
        MAX(id) AS last_ledger_id
    FROM gjj_crm_finance_ledger
    WHERE data_field_id > 0
      AND finance_type_id > 0
    GROUP BY data_field_id, finance_type_id
),
ranked_ledger_mapping AS (
    SELECT
        data_field_id,
        finance_type_id,
        ROW_NUMBER() OVER (
            PARTITION BY data_field_id
            ORDER BY ledger_count DESC, last_ledger_id DESC, finance_type_id ASC
        ) AS mapping_rank
    FROM ledger_counts
)
UPDATE gjj_crm_data_field AS field
SET finance_type_id = mapping.finance_type_id,
    updated_at = CURRENT_TIMESTAMP
FROM ranked_ledger_mapping AS mapping
WHERE mapping.data_field_id = field.id
  AND mapping.mapping_rank = 1
  AND field.finance_type_id = 0;

-- Fill fields that were configured historically but have not produced a ledger yet.
WITH legacy_finance_mapping AS (
    SELECT DISTINCT ON (binding.data_field_id)
        binding.data_field_id,
        binding.finance_type_id
    FROM gjj_crm_data_usage_field AS binding
    INNER JOIN gjj_crm_data_usage AS usage
        ON usage.id = binding.usage_id
    WHERE usage.usage_type = 'finance'
      AND binding.data_field_id > 0
      AND binding.finance_type_id > 0
    ORDER BY binding.data_field_id, binding.updated_at DESC, binding.id DESC
)
UPDATE gjj_crm_data_field AS field
SET finance_type_id = mapping.finance_type_id,
    updated_at = CURRENT_TIMESTAMP
FROM legacy_finance_mapping AS mapping
WHERE mapping.data_field_id = field.id
  AND field.finance_type_id = 0;

-- Preserve the explicit mappings introduced for fields that had lost old usage rows.
UPDATE gjj_crm_data_field AS field
SET finance_type_id = finance_type.id,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_finance_type AS finance_type
WHERE field.field_key = 'monthly_rent_amount'
  AND finance_type.code = 'rent_income'
  AND field.finance_type_id = 0;

UPDATE gjj_crm_data_field AS field
SET finance_type_id = finance_type.id,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_finance_type AS finance_type
WHERE field.field_key = 'moving_subsidy_amount'
  AND finance_type.code = 'moving_cost'
  AND field.finance_type_id = 0;

-- Migration 024 hid finance fields after replacing them with task-generated inputs.
-- Restore only mapped fields on forms that are still used by form or approval tasks.
UPDATE gjj_crm_form_field AS form_field
SET status = 1,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_data_field AS field
WHERE field.id = form_field.data_field_id
  AND field.finance_type_id > 0
  AND form_field.status = 2
  AND EXISTS (
      SELECT 1
      FROM gjj_crm_task AS task
      WHERE task.form_id = form_field.form_id
        AND task.task_type IN ('form', 'approval')
        AND task.status = 1
  );

-- A field participates in statistics when any preserved configuration or snapshot
-- proves that it was previously selected. Existing snapshots remain untouched.
UPDATE gjj_crm_data_field AS field
SET stat_enabled = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE field.stat_enabled = FALSE
  AND field.id IN (
    SELECT data_field_id
    FROM gjj_crm_stat_field_value
    WHERE data_field_id > 0

    UNION

    SELECT data_field_id
    FROM gjj_crm_task_stat_field
    WHERE data_field_id > 0

    UNION

    SELECT binding.data_field_id
    FROM gjj_crm_data_usage_field AS binding
    INNER JOIN gjj_crm_data_usage AS usage
        ON usage.id = binding.usage_id
    WHERE usage.usage_type = 'stat'
      AND binding.data_field_id > 0
);

COMMIT;

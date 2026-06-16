-- Finance type is the configurable category used by finance data fields.
CREATE TABLE IF NOT EXISTS gjj_crm_finance_type (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    direction VARCHAR(16) NOT NULL DEFAULT 'income',
    status SMALLINT NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_finance_type_code
    ON gjj_crm_finance_type (code);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_type_status_sort
    ON gjj_crm_finance_type (status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_type_direction
    ON gjj_crm_finance_type (direction, status, id);

-- stat_id links finance data fields to gjj_crm_finance_type.
ALTER TABLE gjj_crm_data_field
    ADD COLUMN IF NOT EXISTS stat_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_field_stat_ref
    ON gjj_crm_data_field (stat_type, stat_id, status, id);

-- Finance ledger is append-only in this CRM. Adjustments should insert reverse rows.
CREATE TABLE IF NOT EXISTS gjj_crm_finance_ledger (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL DEFAULT 0,
    asset_id BIGINT NOT NULL DEFAULT 0,
    task_id BIGINT NOT NULL DEFAULT 0,
    operation_log_id BIGINT NOT NULL DEFAULT 0,
    data_field_id BIGINT NOT NULL DEFAULT 0,
    finance_type_id BIGINT NOT NULL DEFAULT 0,
    finance_type_code VARCHAR(64) NOT NULL DEFAULT '',
    finance_type_name VARCHAR(128) NOT NULL DEFAULT '',
    direction VARCHAR(16) NOT NULL DEFAULT 'income',
    amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    raw_value TEXT NOT NULL DEFAULT '',
    staff_id BIGINT NOT NULL DEFAULT 0,
    department_id BIGINT NOT NULL DEFAULT 0,
    source VARCHAR(32) NOT NULL DEFAULT 'form',
    reverse_of_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE gjj_crm_finance_ledger
    ADD COLUMN IF NOT EXISTS customer_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS asset_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS task_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS operation_log_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS data_field_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS finance_type_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS finance_type_code VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS finance_type_name VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS direction VARCHAR(16) NOT NULL DEFAULT 'income',
    ADD COLUMN IF NOT EXISTS amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_value TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS staff_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS department_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'form',
    ADD COLUMN IF NOT EXISTS reverse_of_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'gjj_crm_finance_ledger'
          AND column_name = 'finance_type'
    ) THEN
        EXECUTE $sql$
            UPDATE gjj_crm_finance_ledger
            SET finance_type_code = COALESCE(NULLIF(finance_type_code, ''), finance_type)
            WHERE finance_type <> ''
        $sql$;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'gjj_crm_finance_ledger'
          AND column_name = 'finance_name'
    ) THEN
        EXECUTE $sql$
            UPDATE gjj_crm_finance_ledger
            SET finance_type_name = COALESCE(NULLIF(finance_type_name, ''), finance_name)
            WHERE finance_name <> ''
        $sql$;
    END IF;
END $$;

ALTER TABLE gjj_crm_finance_ledger
    DROP COLUMN IF EXISTS data_template_id,
    DROP COLUMN IF EXISTS finance_type,
    DROP COLUMN IF EXISTS finance_name,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS updated_at;

DROP INDEX IF EXISTS uidx_gjj_crm_finance_ledger_operation_field;
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_operation_field
    ON gjj_crm_finance_ledger (operation_log_id, data_field_id, source);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_customer_time
    ON gjj_crm_finance_ledger (customer_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_asset_time
    ON gjj_crm_finance_ledger (asset_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_finance_time
    ON gjj_crm_finance_ledger (finance_type_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_field_time
    ON gjj_crm_finance_ledger (data_field_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_task_time
    ON gjj_crm_finance_ledger (task_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_staff_time
    ON gjj_crm_finance_ledger (staff_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_created_time
    ON gjj_crm_finance_ledger (created_at, id);

-- Task point ledger is append-only. Points are generated when a task or
-- collaboration todo is completed, then summarized by staff/department later.
CREATE TABLE IF NOT EXISTS gjj_crm_task_point_ledger (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL DEFAULT 0,
    asset_id BIGINT NOT NULL DEFAULT 0,
    task_id BIGINT NOT NULL DEFAULT 0,
    operation_log_id BIGINT NOT NULL DEFAULT 0,
    todo_id BIGINT NOT NULL DEFAULT 0,
    points DOUBLE PRECISION NOT NULL DEFAULT 0,
    staff_id BIGINT NOT NULL DEFAULT 0,
    department_id BIGINT NOT NULL DEFAULT 0,
    result_value VARCHAR(64) NOT NULL DEFAULT '',
    source VARCHAR(64) NOT NULL DEFAULT 'task_complete',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE gjj_crm_task_point_ledger
    ADD COLUMN IF NOT EXISTS customer_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS asset_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS task_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS operation_log_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS todo_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS points DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS staff_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS department_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS result_value VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source VARCHAR(64) NOT NULL DEFAULT 'task_complete',
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE gjj_crm_task_point_ledger
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS updated_at;

DROP INDEX IF EXISTS uidx_gjj_crm_task_point_ledger_operation;
DROP INDEX IF EXISTS idx_gjj_crm_task_point_ledger_operation;
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_point_ledger_operation
    ON gjj_crm_task_point_ledger (operation_log_id, todo_id, source);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_point_ledger_staff_time
    ON gjj_crm_task_point_ledger (staff_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_point_ledger_department_time
    ON gjj_crm_task_point_ledger (department_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_point_ledger_task_time
    ON gjj_crm_task_point_ledger (task_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_point_ledger_customer_time
    ON gjj_crm_task_point_ledger (customer_id, created_at, id);

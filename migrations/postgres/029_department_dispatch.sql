BEGIN;

CREATE TABLE IF NOT EXISTS gjj_crm_dispatch_pool (
    id BIGSERIAL PRIMARY KEY,
    department_id BIGINT NOT NULL,
    name VARCHAR(64) NOT NULL,
    pool_type VARCHAR(32) NOT NULL DEFAULT 'group',
    status SMALLINT NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_pool_department_status
    ON gjj_crm_dispatch_pool (department_id, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_pool_type_status
    ON gjj_crm_dispatch_pool (department_id, pool_type, status, id);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_dispatch_pool_direct
    ON gjj_crm_dispatch_pool (department_id)
    WHERE pool_type = 'direct';

CREATE TABLE IF NOT EXISTS gjj_crm_department_dispatch_setting (
    id BIGSERIAL PRIMARY KEY,
    department_id BIGINT NOT NULL,
    active_pool_id BIGINT NOT NULL DEFAULT 0,
    last_member_id BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_department_dispatch_setting_department
    ON gjj_crm_department_dispatch_setting (department_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_department_dispatch_setting_pool_status
    ON gjj_crm_department_dispatch_setting (active_pool_id, status, id);

CREATE TABLE IF NOT EXISTS gjj_crm_dispatch_pool_member (
    id BIGSERIAL PRIMARY KEY,
    pool_id BIGINT NOT NULL,
    department_id BIGINT NOT NULL,
    staff_id BIGINT NOT NULL,
    weekly_schedule_json TEXT NOT NULL DEFAULT '{}',
    daily_limit INTEGER NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_dispatch_pool_member_pool_staff
    ON gjj_crm_dispatch_pool_member (pool_id, staff_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_pool_member_pool_status_sort
    ON gjj_crm_dispatch_pool_member (pool_id, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_pool_member_department_staff
    ON gjj_crm_dispatch_pool_member (department_id, staff_id, status, id);

CREATE TABLE IF NOT EXISTS gjj_crm_dispatch_record (
    id BIGSERIAL PRIMARY KEY,
    dispatch_type VARCHAR(32) NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT '',
    department_id BIGINT NOT NULL,
    staff_id BIGINT NOT NULL,
    previous_staff_id BIGINT NOT NULL DEFAULT 0,
    workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    work_todo_id BIGINT NOT NULL DEFAULT 0,
    lead_id BIGINT NOT NULL DEFAULT 0,
    operator_staff_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_record_department_time
    ON gjj_crm_dispatch_record (department_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_record_staff_auto_time
    ON gjj_crm_dispatch_record (staff_id, dispatch_type, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_record_instance_time
    ON gjj_crm_dispatch_record (workflow_instance_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_record_todo_time
    ON gjj_crm_dispatch_record (work_todo_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_dispatch_record_lead_time
    ON gjj_crm_dispatch_record (lead_id, created_at, id);

INSERT INTO gjj_crm_dispatch_pool (
    department_id, name, pool_type, status, sort, created_at, updated_at
)
SELECT
    department.id,
    '按员工分配',
    'direct',
    1,
    10,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM gjj_crm_department AS department
WHERE department.status = 1
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_dispatch_pool AS pool
      WHERE pool.department_id = department.id
        AND pool.pool_type = 'direct'
  );

INSERT INTO gjj_crm_department_dispatch_setting (
    department_id, active_pool_id, last_member_id, version, status, created_at, updated_at
)
SELECT
    department.id,
    pool.id,
    0,
    1,
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM gjj_crm_department AS department
INNER JOIN gjj_crm_dispatch_pool AS pool
    ON pool.department_id = department.id
   AND pool.pool_type = 'direct'
WHERE department.status = 1
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_department_dispatch_setting AS setting
      WHERE setting.department_id = department.id
  );

WITH ranked_staff AS (
    SELECT
        staff.id AS staff_id,
        staff.department_id,
        (ROW_NUMBER() OVER (
            PARTITION BY staff.department_id
            ORDER BY staff.id
        ) * 10)::INTEGER AS member_sort
    FROM gjj_crm_staff AS staff
    WHERE staff.status = 1
)
INSERT INTO gjj_crm_dispatch_pool_member (
    pool_id,
    department_id,
    staff_id,
    weekly_schedule_json,
    daily_limit,
    status,
    sort,
    created_at,
    updated_at
)
SELECT
    pool.id,
    ranked_staff.department_id,
    ranked_staff.staff_id,
    '{"1":[[0,1440]],"2":[[0,1440]],"3":[[0,1440]],"4":[[0,1440]],"5":[[0,1440]],"6":[[0,1440]],"7":[[0,1440]]}',
    0,
    1,
    ranked_staff.member_sort,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM ranked_staff
INNER JOIN gjj_crm_dispatch_pool AS pool
    ON pool.department_id = ranked_staff.department_id
   AND pool.pool_type = 'direct'
ON CONFLICT (pool_id, staff_id) DO NOTHING;

UPDATE gjj_crm_staff
SET staff_type = 'employee'
WHERE staff_type = 'leader'
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_department AS department
      WHERE department.leader_staff_id = gjj_crm_staff.id
        AND department.id = gjj_crm_staff.department_id
  );

UPDATE gjj_crm_staff AS staff
SET staff_type = 'leader'
FROM gjj_crm_department AS department
WHERE department.leader_staff_id = staff.id
  AND department.id = staff.department_id
  AND staff.status = 1;

COMMIT;

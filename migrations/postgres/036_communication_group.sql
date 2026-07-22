BEGIN;

CREATE TABLE IF NOT EXISTS gjj_crm_communication_group_type (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_communication_group_type_code
    ON gjj_crm_communication_group_type (code);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_communication_group_type_status_sort
    ON gjj_crm_communication_group_type (status, sort, id);

CREATE TABLE IF NOT EXISTS gjj_crm_communication_group (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL,
    asset_id BIGINT NOT NULL DEFAULT 0,
    workflow_instance_id BIGINT NOT NULL,
    group_type_id BIGINT NOT NULL,
    name VARCHAR(160) NOT NULL,
    external_group_id VARCHAR(160) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    established_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    dissolved_at TIMESTAMPTZ,
    dissolve_reason TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    remark TEXT NOT NULL DEFAULT '',
    source_key VARCHAR(192),
    created_by_staff_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_communication_group_source_key
    ON gjj_crm_communication_group (source_key);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_communication_group_active_instance
    ON gjj_crm_communication_group (workflow_instance_id)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_gjj_crm_communication_group_instance_status
    ON gjj_crm_communication_group (workflow_instance_id, status, established_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_communication_group_customer_status
    ON gjj_crm_communication_group (customer_id, status, established_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_communication_group_asset_status
    ON gjj_crm_communication_group (asset_id, status, established_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_communication_group_type_status
    ON gjj_crm_communication_group (group_type_id, status, established_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_communication_group_external
    ON gjj_crm_communication_group (group_type_id, external_group_id, id);

CREATE TABLE IF NOT EXISTS gjj_crm_communication_group_staff (
    id BIGSERIAL PRIMARY KEY,
    communication_group_id BIGINT NOT NULL,
    staff_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'participant',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_communication_group_staff_group_staff
    ON gjj_crm_communication_group_staff (communication_group_id, staff_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_communication_group_staff_staff_group
    ON gjj_crm_communication_group_staff (staff_id, communication_group_id, id);

INSERT INTO gjj_crm_communication_group_type (
    code,
    name,
    description,
    status,
    sort,
    created_at,
    updated_at
)
SELECT
    'enterprise_wechat',
    '企业微信',
    '企业微信客户沟通群。',
    1,
    10,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_communication_group_type
    WHERE code = 'enterprise_wechat'
);

COMMIT;

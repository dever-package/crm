ALTER TABLE gjj_crm_data_field
    ADD COLUMN IF NOT EXISTS parent_field_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS option_set_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

DROP INDEX IF EXISTS idx_gjj_crm_data_field_stat_group;
DROP INDEX IF EXISTS idx_gjj_crm_data_field_stat_key;
DROP INDEX IF EXISTS idx_gjj_crm_data_field_stat_ref;
DROP INDEX IF EXISTS idx_gjj_crm_data_field_template_key;

ALTER TABLE gjj_crm_data_field
    DROP COLUMN IF EXISTS stat_enabled,
    DROP COLUMN IF EXISTS stat_type,
    DROP COLUMN IF EXISTS stat_id,
    DROP COLUMN IF EXISTS stat_group;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_data_field_field_key
    ON gjj_crm_data_field (field_key)
    WHERE field_key <> '';
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_field_parent_status
    ON gjj_crm_data_field (parent_field_id, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_field_template_status
    ON gjj_crm_data_field (data_template_id, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_field_option_set
    ON gjj_crm_data_field (option_set_id, status, id);

CREATE TABLE IF NOT EXISTS gjj_crm_option_set (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sort INTEGER NOT NULL DEFAULT 100,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_option_set_status_sort
    ON gjj_crm_option_set (status, sort, id);

CREATE TABLE IF NOT EXISTS gjj_crm_option_set_item (
    id BIGSERIAL PRIMARY KEY,
    option_set_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    value VARCHAR(255) NOT NULL,
    sort INTEGER NOT NULL DEFAULT 100,
    status SMALLINT NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_option_set_item_set_value
    ON gjj_crm_option_set_item (option_set_id, value);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_option_set_item_set_sort
    ON gjj_crm_option_set_item (option_set_id, status, sort, id);

CREATE TABLE IF NOT EXISTS gjj_crm_data_usage (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    usage_type VARCHAR(32) NOT NULL DEFAULT 'stat',
    description TEXT NOT NULL DEFAULT '',
    sort INTEGER NOT NULL DEFAULT 100,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_usage_type_status
    ON gjj_crm_data_usage (usage_type, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_usage_status_sort
    ON gjj_crm_data_usage (status, sort, id);

CREATE TABLE IF NOT EXISTS gjj_crm_data_usage_field (
    id BIGSERIAL PRIMARY KEY,
    usage_id BIGINT NOT NULL,
    data_template_id BIGINT NOT NULL,
    data_field_id BIGINT NOT NULL,
    value_type VARCHAR(32) NOT NULL DEFAULT 'text',
    aggregate_type VARCHAR(32) NOT NULL DEFAULT '',
    finance_type_id BIGINT NOT NULL DEFAULT 0,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    config_json TEXT NOT NULL DEFAULT '{}',
    sort INTEGER NOT NULL DEFAULT 100,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_data_usage_field_usage_field
    ON gjj_crm_data_usage_field (usage_id, data_field_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_usage_field_usage_sort
    ON gjj_crm_data_usage_field (usage_id, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_usage_field_field_usage
    ON gjj_crm_data_usage_field (data_field_id, usage_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_usage_field_finance
    ON gjj_crm_data_usage_field (finance_type_id, status, id);

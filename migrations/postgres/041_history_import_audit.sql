BEGIN;

ALTER TABLE gjj_crm_operation_log
    ADD COLUMN IF NOT EXISTS source_key VARCHAR(224);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_operation_log_source_key
    ON gjj_crm_operation_log (source_key);

ALTER TABLE gjj_crm_attachment
    ADD COLUMN IF NOT EXISTS source_key VARCHAR(224);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_attachment_source_key
    ON gjj_crm_attachment (source_key);

CREATE TABLE IF NOT EXISTS gjj_crm_history_import_record (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(96) NOT NULL DEFAULT '',
    source_key VARCHAR(192) NOT NULL,
    source_table_key VARCHAR(64) NOT NULL,
    source_table_name VARCHAR(128) NOT NULL,
    source_table_id VARCHAR(64) NOT NULL,
    source_record_id VARCHAR(64) NOT NULL,
    internal_case_id VARCHAR(96) NOT NULL DEFAULT '',
    source_checksum VARCHAR(64) NOT NULL,
    lead_id BIGINT NOT NULL DEFAULT 0,
    customer_id BIGINT NOT NULL DEFAULT 0,
    asset_id BIGINT NOT NULL DEFAULT 0,
    workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    target_json TEXT NOT NULL DEFAULT '{}',
    raw_snapshot_json TEXT NOT NULL DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'imported',
    error_message TEXT NOT NULL DEFAULT '',
    source_created_at TIMESTAMPTZ,
    source_last_modified_at TIMESTAMPTZ,
    imported_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_history_import_record_source_key
    ON gjj_crm_history_import_record (source_key);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_history_import_record_batch_status
    ON gjj_crm_history_import_record (batch_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_history_import_record_case_status
    ON gjj_crm_history_import_record (internal_case_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_history_import_record_source_table
    ON gjj_crm_history_import_record (source_table_key, source_record_id, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_history_import_record_lead
    ON gjj_crm_history_import_record (lead_id, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_history_import_record_customer_asset
    ON gjj_crm_history_import_record (customer_id, asset_id, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_history_import_record_workflow
    ON gjj_crm_history_import_record (workflow_instance_id, id);

COMMIT;

-- Workbench Feishu login binds Feishu open_id to CRM staff. Phone is optional
-- because Feishu may not return mobile without extra permission.
ALTER TABLE gjj_crm_staff
    ADD COLUMN IF NOT EXISTS feishu_open_id VARCHAR(128) NOT NULL DEFAULT '';

DROP INDEX IF EXISTS uidx_gjj_crm_staff_phone;
CREATE INDEX IF NOT EXISTS idx_gjj_crm_staff_phone
    ON gjj_crm_staff (phone, id);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_staff_feishu_open_id
    ON gjj_crm_staff (feishu_open_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_staff_feishu_open_id_not_empty
    ON gjj_crm_staff (feishu_open_id)
    WHERE feishu_open_id <> '';

ALTER TABLE gjj_crm_basic_config
    ADD COLUMN IF NOT EXISTS feishu_app_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS feishu_app_secret VARCHAR(255) NOT NULL DEFAULT '';

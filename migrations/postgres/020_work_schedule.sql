-- Initialize work schedules and bind the configurable customer follow-up time.
BEGIN;

CREATE TABLE IF NOT EXISTS gjj_crm_schedule_event (
    id BIGSERIAL PRIMARY KEY,
    schedule_type VARCHAR(32) NOT NULL DEFAULT 'personal',
    customer_id BIGINT NOT NULL DEFAULT 0,
    pending_customer_key VARCHAR(64),
    owner_staff_id BIGINT NOT NULL,
    created_by_staff_id BIGINT NOT NULL,
    source_workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    data_usage_field_id BIGINT NOT NULL DEFAULT 0,
    data_record_id BIGINT NOT NULL DEFAULT 0,
    data_field_id BIGINT NOT NULL DEFAULT 0,
    operation_log_id BIGINT NOT NULL DEFAULT 0,
    title VARCHAR(128) NOT NULL,
    remark TEXT NOT NULL DEFAULT '',
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    reminder_minutes INTEGER NOT NULL DEFAULT 0,
    remind_at TIMESTAMPTZ NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'calendar',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    completed_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_schedule_event_pending_customer
    ON gjj_crm_schedule_event (pending_customer_key);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_event_owner_status
    ON gjj_crm_schedule_event (owner_staff_id, status, start_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_event_customer_status
    ON gjj_crm_schedule_event (customer_id, schedule_type, status, start_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_event_reminder_status
    ON gjj_crm_schedule_event (remind_at, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_event_source_workflow
    ON gjj_crm_schedule_event (source_workflow_instance_id, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_event_data_record_field
    ON gjj_crm_schedule_event (data_record_id, data_field_id, status, id);

CREATE TABLE IF NOT EXISTS gjj_crm_schedule_participant (
    id BIGSERIAL PRIMARY KEY,
    schedule_event_id BIGINT NOT NULL,
    staff_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'participant',
    workbench_read_at TIMESTAMPTZ,
    feishu_sent_at TIMESTAMPTZ,
    feishu_claimed_at TIMESTAMPTZ,
    feishu_attempts INTEGER NOT NULL DEFAULT 0,
    feishu_last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_schedule_participant_event_staff
    ON gjj_crm_schedule_participant (schedule_event_id, staff_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_participant_staff_event
    ON gjj_crm_schedule_participant (staff_id, schedule_event_id, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_participant_feishu_state
    ON gjj_crm_schedule_participant (feishu_sent_at, feishu_attempts, id);

ALTER TABLE gjj_crm_public_resource_booking
    ADD COLUMN IF NOT EXISTS schedule_event_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_gjj_crm_public_resource_booking_schedule_time
    ON gjj_crm_public_resource_booking (schedule_event_id, start_at, id);

DO $$
DECLARE
    follow_field_count INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO follow_field_count
    FROM gjj_crm_data_field AS field
    INNER JOIN gjj_crm_data_template AS template
        ON template.id = field.data_template_id
    WHERE field.field_key = 'next_follow_at'
      AND field.field_type = 'datetime'
      AND field.status = 1
      AND template.cate_id = 1
      AND template.status = 1;

    IF follow_field_count <> 1 THEN
        RAISE EXCEPTION '客户下次跟进时间需要唯一绑定一个启用的客户日期时间字段';
    END IF;
END $$;

INSERT INTO gjj_crm_data_usage (
    name,
    usage_type,
    description,
    sort,
    status,
    created_at,
    updated_at
)
SELECT
    '客户下次跟进时间',
    'customer_follow_at',
    '日程与客户资料双向同步使用的日期时间字段。',
    40,
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_data_usage
    WHERE usage_type = 'customer_follow_at'
      AND status = 1
);

WITH target_usage AS (
    SELECT id
    FROM gjj_crm_data_usage
    WHERE usage_type = 'customer_follow_at'
      AND status = 1
    ORDER BY sort, id
    LIMIT 1
), target_field AS (
    SELECT field.id, field.data_template_id, field.name
    FROM gjj_crm_data_field AS field
    INNER JOIN gjj_crm_data_template AS template
        ON template.id = field.data_template_id
    WHERE field.field_key = 'next_follow_at'
      AND field.field_type = 'datetime'
      AND field.status = 1
      AND template.cate_id = 1
      AND template.status = 1
    ORDER BY field.id
    LIMIT 1
)
INSERT INTO gjj_crm_data_usage_field (
    usage_id,
    data_template_id,
    data_field_id,
    value_type,
    aggregate_type,
    finance_type_id,
    display_name,
    config_json,
    sort,
    status,
    created_at,
    updated_at
)
SELECT
    target_usage.id,
    target_field.data_template_id,
    target_field.id,
    'time',
    '',
    0,
    target_field.name,
    '{}',
    10,
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM target_usage
CROSS JOIN target_field
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_data_usage_field AS binding
    INNER JOIN gjj_crm_data_usage AS usage
        ON usage.id = binding.usage_id
    WHERE usage.usage_type = 'customer_follow_at'
      AND usage.status = 1
      AND binding.status = 1
);

COMMIT;

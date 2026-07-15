-- Keep channel-specific Douyin metadata configurable and separate from the
-- core lead table. The task form remains unchanged; these fields are mainly
-- populated by the Douyin importer and displayed in lead details.
BEGIN;

INSERT INTO gjj_crm_data_template (cate_id, name, status, sort, created_at, updated_at)
SELECT 4, '抖音来客信息', 1, 20, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_data_template
    WHERE cate_id = 4 AND name = '抖音来客信息'
);

WITH target_template AS (
    SELECT id
    FROM gjj_crm_data_template
    WHERE cate_id = 4 AND name = '抖音来客信息'
    ORDER BY id
    LIMIT 1
), field_seed(name, field_key, field_type, sort) AS (
    VALUES
        ('抖音账户', 'douyin_account_name', 'text', 10),
        ('推广计划', 'douyin_promotion_name', 'text', 20),
        ('推广产品', 'douyin_product_name', 'text', 30),
        ('进线时间', 'douyin_entered_at', 'datetime', 40),
        ('分配时间', 'douyin_assigned_at', 'datetime', 50),
        ('抖音跟进人', 'douyin_owner_name', 'text', 60),
        ('抖音线索阶段', 'douyin_stage', 'text', 70),
        ('抖音跟进状态', 'douyin_follow_status', 'text', 80),
        ('线索标签', 'douyin_tags', 'textarea', 90),
        ('跟进记录', 'douyin_follow_record', 'textarea', 100),
        ('流量类型', 'douyin_traffic_type', 'text', 110),
        ('线索成本', 'douyin_lead_cost', 'money', 120)
)
INSERT INTO gjj_crm_data_field (
    data_template_id,
    parent_field_id,
    option_set_id,
    name,
    field_key,
    field_type,
    default_value,
    sort,
    status,
    created_at,
    updated_at
)
SELECT
    target_template.id,
    0,
    0,
    field_seed.name,
    field_seed.field_key,
    field_seed.field_type,
    '',
    field_seed.sort,
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM target_template
CROSS JOIN field_seed
ON CONFLICT (field_key) DO UPDATE SET
    data_template_id = EXCLUDED.data_template_id,
    name = EXCLUDED.name,
    field_type = EXCLUDED.field_type,
    sort = EXCLUDED.sort,
    status = 1,
    updated_at = CURRENT_TIMESTAMP;

CREATE UNIQUE INDEX IF NOT EXISTS gjj_crm_lead_source_external_unique
ON gjj_crm_lead (source_id, external_id)
WHERE external_id <> '';

COMMIT;

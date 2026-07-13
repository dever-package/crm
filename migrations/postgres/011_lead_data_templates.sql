-- Add configurable lead fields without mixing them into customer data.
INSERT INTO gjj_crm_data_template_cate (id, name, status, sort, created_at)
SELECT 4, '线索信息', 1, 5, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_data_template_cate WHERE id = 4);

UPDATE gjj_crm_data_template_cate
SET name = '线索信息', status = 1, sort = 5
WHERE id = 4;

INSERT INTO gjj_crm_data_template (cate_id, name, status, sort, created_at, updated_at)
SELECT 4, '线索补充信息', 1, 10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (
    SELECT 1 FROM gjj_crm_data_template
    WHERE cate_id = 4 AND name = '线索补充信息'
);

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
SELECT template.id, 0, 0, '隐私授权', 'privacy_authorized', 'select', '', 10, 1,
       CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM gjj_crm_data_template AS template
WHERE template.cate_id = 4
  AND template.name = '线索补充信息'
  AND NOT EXISTS (
      SELECT 1 FROM gjj_crm_data_field WHERE field_key = 'privacy_authorized'
  )
ORDER BY template.id
LIMIT 1;

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
SELECT template.id, 0, 0, '线索质量', 'source_quality_label', 'select', '', 20, 1,
       CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM gjj_crm_data_template AS template
WHERE template.cate_id = 4
  AND template.name = '线索补充信息'
  AND NOT EXISTS (
      SELECT 1 FROM gjj_crm_data_field WHERE field_key = 'source_quality_label'
  )
ORDER BY template.id
LIMIT 1;

UPDATE gjj_crm_data_field
SET data_template_id = (
        SELECT id FROM gjj_crm_data_template
        WHERE cate_id = 4 AND name = '线索补充信息'
        ORDER BY id LIMIT 1
    ),
    sort = CASE field_key
        WHEN 'privacy_authorized' THEN 10
        WHEN 'source_quality_label' THEN 20
        ELSE sort
    END,
    status = 1,
    updated_at = CURRENT_TIMESTAMP
WHERE field_key IN ('privacy_authorized', 'source_quality_label');

INSERT INTO gjj_crm_data_field_option (data_field_id, name, value, sort)
SELECT field.id, seed.name, seed.value, seed.sort
FROM (
    VALUES
        ('privacy_authorized', '已授权', 'authorized', 10),
        ('privacy_authorized', '待补授权', 'pending', 20),
        ('privacy_authorized', '拒绝授权', 'refused', 30),
        ('source_quality_label', '高意向', 'high', 10),
        ('source_quality_label', '需跟进', 'medium', 20),
        ('source_quality_label', '低意向', 'low', 30),
        ('source_quality_label', '无效', 'invalid', 40)
) AS seed(field_key, name, value, sort)
JOIN gjj_crm_data_field AS field ON field.field_key = seed.field_key
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_data_field_option AS option
    WHERE option.data_field_id = field.id
      AND option.value = seed.value
);

UPDATE gjj_crm_data_template
SET status = 2, updated_at = CURRENT_TIMESTAMP
WHERE cate_id = 1 AND name = '客户来源与基础建档';

UPDATE gjj_crm_form
SET status = 2, updated_at = CURRENT_TIMESTAMP
WHERE name = '客户来源与基础建档';

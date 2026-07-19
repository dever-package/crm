-- Add configurable customer tags grouped by customer level. This migration
-- does not infer relations from legacy free-text customer tags.
BEGIN;

ALTER TABLE gjj_crm_customer
    ALTER COLUMN tags TYPE TEXT,
    ALTER COLUMN level_id SET DEFAULT 0;

CREATE TABLE IF NOT EXISTS gjj_crm_customer_tag (
    id BIGSERIAL PRIMARY KEY,
    level_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_customer_tag_level_name
    ON gjj_crm_customer_tag (level_id, name);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_customer_tag_level_sort
    ON gjj_crm_customer_tag (level_id, status, sort, id);

CREATE TABLE IF NOT EXISTS gjj_crm_customer_tag_relation (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_customer_tag_relation_customer_tag
    ON gjj_crm_customer_tag_relation (customer_id, tag_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_customer_tag_relation_tag_customer
    ON gjj_crm_customer_tag_relation (tag_id, customer_id, id);

SELECT setval(
    pg_get_serial_sequence('gjj_crm_customer_level', 'id'),
    GREATEST(COALESCE(MAX(id), 1), 1),
    COUNT(*) > 0
)
FROM gjj_crm_customer_level;

WITH level_seed(code, name, sort) AS (
    VALUES
        ('high_intent', '高意向', 10),
        ('medium_intent', '中意向', 20),
        ('no_intent', '无意向', 30),
        ('unreachable', '联系不上', 40)
)
INSERT INTO gjj_crm_customer_level (
    code,
    name,
    status,
    sort,
    created_at
)
SELECT
    level_seed.code,
    level_seed.name,
    1,
    level_seed.sort,
    CURRENT_TIMESTAMP
FROM level_seed
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_customer_level AS configured
    WHERE configured.name = level_seed.name
)
AND NOT EXISTS (
    SELECT 1
    FROM gjj_crm_customer_level AS code_owner
    WHERE code_owner.code = level_seed.code
);

-- Keep historical customer references unchanged, but remove the legacy
-- placeholder from the active level configuration.
UPDATE gjj_crm_customer_level
SET status = 2
WHERE name = '普通';

DO $$
DECLARE
    configured_level_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT name)
    INTO configured_level_count
    FROM gjj_crm_customer_level
    WHERE name IN ('高意向', '中意向', '无意向', '联系不上')
      AND status = 1;

    IF configured_level_count <> 4 THEN
        RAISE EXCEPTION '客户等级配置不完整，需要启用高意向、中意向、无意向、联系不上四个等级';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form
        WHERE name = '接单建档'
          AND status = 1
    ) THEN
        RAISE EXCEPTION '缺少启用的接单建档资料模板';
    END IF;
END $$;

WITH level_target AS (
    SELECT DISTINCT ON (name) id, name
    FROM gjj_crm_customer_level
    WHERE name IN ('高意向', '中意向', '无意向', '联系不上')
      AND status = 1
    ORDER BY name, sort, id
), tag_seed(level_name, tag_name, sort) AS (
    VALUES
        ('高意向', '即将逾期', 10),
        ('高意向', '已经逾期', 20),
        ('高意向', '资不抵债', 30),
        ('中意向', '不认可方案', 10),
        ('中意向', '产权存在争议', 20),
        ('中意向', '单纯咨询/了解观望', 30),
        ('中意向', '房产已查封', 40),
        ('中意向', '房租过低', 50),
        ('中意向', '自己无法做主', 60),
        ('中意向', '资可抵债', 70),
        ('无意向', '无保房需求', 10),
        ('无意向', '潜在沟通', 20),
        ('无意向', '非保房业务', 30),
        ('联系不上', '首次未接通', 10),
        ('联系不上', '二次未接通', 20),
        ('联系不上', '三次未接通', 30),
        ('联系不上', '四次未接通', 40),
        ('联系不上', '五次未接通（无效）', 50),
        ('联系不上', '关机/停机/空号', 60),
        ('联系不上', '联系方式不统一', 70)
)
INSERT INTO gjj_crm_customer_tag (
    level_id,
    name,
    status,
    sort,
    created_at,
    updated_at
)
SELECT
    level_target.id,
    tag_seed.tag_name,
    1,
    tag_seed.sort,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM tag_seed
INNER JOIN level_target ON level_target.name = tag_seed.level_name
ON CONFLICT (level_id, name) DO UPDATE SET
    status = 1,
    sort = EXCLUDED.sort,
    updated_at = CURRENT_TIMESTAMP;

WITH entry_form AS (
    SELECT id
    FROM gjj_crm_form
    WHERE name = '接单建档'
      AND status = 1
    ORDER BY id
    LIMIT 1
)
INSERT INTO gjj_crm_form_field (
    form_id,
    data_template_cate_id,
    data_template_id,
    field_source,
    field_path,
    main_field,
    data_field_id,
    name,
    required,
    readonly,
    sort,
    status,
    created_at,
    updated_at
)
SELECT
    entry_form.id,
    1,
    0,
    'main:1:tags',
    '["cate:1","main_table:1","main:1:tags"]',
    'tags',
    0,
    '标签',
    TRUE,
    FALSE,
    70,
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM entry_form
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_form_field AS field
    WHERE field.form_id = entry_form.id
      AND field.main_field = 'tags'
);

UPDATE gjj_crm_form_field AS field
SET readonly = TRUE,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_form AS form
WHERE field.form_id = form.id
  AND form.name = '接单建档'
  AND form.status = 1
  AND field.main_field IN ('source_id', 'channel_id');

UPDATE gjj_crm_form_field AS field
SET required = TRUE,
    readonly = FALSE,
    status = 1,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_form AS form
WHERE field.form_id = form.id
  AND form.name = '接单建档'
  AND form.status = 1
  AND field.main_field = 'tags';

DELETE FROM gjj_crm_form_field AS field
USING gjj_crm_form AS form
WHERE field.form_id = form.id
  AND form.name = '接单建档'
  AND form.status = 1
  AND field.main_field = 'level_id';

COMMIT;

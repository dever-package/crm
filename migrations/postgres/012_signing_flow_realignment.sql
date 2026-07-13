-- Realign the default signing workflow around the current workflow-instance runtime.
-- Migration 011 must run first so lead-only fields stay outside customer templates.
BEGIN;

SELECT setval(
    pg_get_serial_sequence('gjj_crm_data_template_cate', 'id'),
    COALESCE((SELECT MAX(id) FROM gjj_crm_data_template_cate), 1),
    EXISTS (SELECT 1 FROM gjj_crm_data_template_cate)
);

-- Some existing environments only applied part of the workflow-instance cutover.
-- Add the missing ownership columns before rebuilding configuration.
ALTER TABLE IF EXISTS gjj_crm_data_record
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_stat_field_value
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_finance_ledger
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_operation_log
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_stat_event
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_task_todo
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_record_instance_template
    ON gjj_crm_data_record (workflow_instance_id, data_template_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_record_product_template
    ON gjj_crm_data_record (customer_product_id, data_template_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_stat_field_value_instance_time
    ON gjj_crm_stat_field_value (workflow_instance_id, updated_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_stat_field_value_product_time
    ON gjj_crm_stat_field_value (customer_product_id, updated_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_instance_time
    ON gjj_crm_finance_ledger (workflow_instance_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_product_time
    ON gjj_crm_finance_ledger (customer_product_id, created_at, id);

-- Workflow definitions are intentionally reset. Customer, asset and lead master data
-- remain intact; old workflow instances and pending assignments are not compatible.
DELETE FROM gjj_crm_task_todo;
DELETE FROM gjj_crm_stat_event
WHERE workflow_id > 0 OR workflow_instance_id > 0 OR customer_product_id > 0;
DELETE FROM gjj_crm_operation_log
WHERE workflow_id > 0 OR workflow_instance_id > 0 OR customer_product_id > 0;
DELETE FROM gjj_crm_data_record
WHERE workflow_instance_id > 0 OR customer_product_id > 0;
DELETE FROM gjj_crm_stat_field_value
WHERE workflow_instance_id > 0 OR customer_product_id > 0;
DELETE FROM gjj_crm_finance_ledger
WHERE workflow_instance_id > 0 OR customer_product_id > 0;
DELETE FROM gjj_crm_workflow_instance;
DELETE FROM gjj_crm_customer_product;

DROP INDEX IF EXISTS uidx_gjj_crm_task_todo_stage_task;
DROP INDEX IF EXISTS idx_gjj_crm_task_todo_stage_task;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_task_todo_instance_task
    ON gjj_crm_task_todo (workflow_instance_id, stage_id, task_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_todo_instance_status
    ON gjj_crm_task_todo (workflow_instance_id, status, id);

-- Keep a source snapshot so the entry form can be rebuilt idempotently from the
-- existing customer, NPL and asset field definitions.
CREATE TEMP TABLE crm_signing_form_field_source ON COMMIT DROP AS
SELECT field.*
FROM gjj_crm_form_field AS field
JOIN gjj_crm_form AS form ON form.id = field.form_id
WHERE form.name IN (
    '客户来源与基础建档',
    'NPL接单首呼与建群',
    '接单建档',
    '客户资料与资产建档'
);

-- Data templates describe storage only. Lead collection is separate; signing
-- workflow forms reuse customer and asset storage fields.
UPDATE gjj_crm_data_template
SET status = 2, updated_at = CURRENT_TIMESTAMP
WHERE cate_id = 1 AND name = '客户来源与基础建档';

UPDATE gjj_crm_data_template
SET name = '接单与建群', cate_id = 1, status = 1, sort = 10,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('NPL接单首呼与建群', '接单建档', '接单与建群');

UPDATE gjj_crm_data_template
SET name = '资产基础信息', cate_id = 2, status = 1, sort = 10,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('客户资料与资产建档', '资产基础信息');

UPDATE gjj_crm_data_template
SET name = '十一维资料', cate_id = 2, status = 1, sort = 20,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('P01-P12十一维探针', '十一维资料');

UPDATE gjj_crm_data_template
SET name = '诊断结果', cate_id = 2, status = 1, sort = 30,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('PM案件判断与产品确认', '诊断结果');

UPDATE gjj_crm_data_template
SET name = '专业协作意见', cate_id = 3, status = 1, sort = 10,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('专业门禁协作', '签约协作', '专业协作意见');

UPDATE gjj_crm_data_template
SET name = '合同信息', cate_id = 3, status = 1, sort = 20,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('合同与费用', '合同签署', '合同信息');

UPDATE gjj_crm_data_template
SET name = '服务交付', cate_id = 3, status = 1, sort = 30,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('服务交付与验收', '服务交付');

-- T-node values are diagnosis results, not raw eleven-dimension input fields.
UPDATE gjj_crm_data_field
SET data_template_id = (
        SELECT id FROM gjj_crm_data_template
        WHERE name = '诊断结果' ORDER BY id LIMIT 1
    ),
    status = 1,
    sort = CASE field_key
        WHEN 'candidate_t_node' THEN 10
        WHEN 'candidate_t_confidence_level' THEN 20
        ELSE sort
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE field_key IN ('candidate_t_node', 'candidate_t_confidence_level');

-- Signing direction and duplicated product fields are superseded by the product task.
UPDATE gjj_crm_data_field
SET status = 2, updated_at = CURRENT_TIMESTAMP
WHERE field_key IN (
    'signing_business_type_candidate',
    'signing_business_type_confidence',
    'npl_candidate_s_product_codes',
    'npl_primary_s_product_code',
    'npl_s_product_reason_summary',
    'npl_submit_pm_confirmation_status',
    'pm_confirmed_signing_business_type',
    'final_signing_business_type',
    'pm_confirmed_s_product_code',
    's_product_confirmation_status'
);

-- Business-data forms must write against a workflow instance.
UPDATE gjj_crm_form_field
SET data_template_cate_id = 3, updated_at = CURRENT_TIMESTAMP
WHERE data_template_id IN (
    SELECT id FROM gjj_crm_data_template WHERE cate_id = 3
);

DO $$
DECLARE
    entry_form_id BIGINT;
BEGIN
    SELECT id INTO entry_form_id
    FROM gjj_crm_form
    WHERE name IN ('接单建档', 'NPL接单首呼与建群')
    ORDER BY CASE WHEN name = '接单建档' THEN 0 ELSE 1 END, id
    LIMIT 1;

    IF entry_form_id IS NULL THEN
        RAISE EXCEPTION '缺少接单建档资料模板';
    END IF;

    UPDATE gjj_crm_form
    SET name = '接单建档',
        description = 'NPL接单后补充客户基础信息、首呼建群结果和资产基础资料。',
        status = 1,
        sort = 10,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = entry_form_id;

    DELETE FROM gjj_crm_form_field WHERE form_id = entry_form_id;

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
        entry_form_id,
        prepared.data_template_cate_id,
        prepared.data_template_id,
        prepared.field_source,
        prepared.field_path,
        prepared.main_field,
        prepared.data_field_id,
        prepared.name,
        prepared.required,
        prepared.readonly,
        prepared.target_sort,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM (
        SELECT
            source.*,
            CASE source.field_source
                WHEN 'main:1:name' THEN 10
                WHEN 'main:1:phone' THEN 20
                WHEN 'main:1:wechat' THEN 30
                WHEN 'main:1:id_card' THEN 40
                WHEN 'main:1:source_id' THEN 50
                WHEN 'main:1:channel_id' THEN 60
                WHEN 'main:1:level_id' THEN 70
                WHEN 'main:1:tags' THEN 80
                WHEN 'main:1:remark' THEN 90
                WHEN 'main:2:asset_name' THEN 210
                WHEN 'main:2:asset_status_id' THEN 220
                WHEN 'main:2:remark' THEN 230
                ELSE CASE
                    WHEN source.data_template_id = (
                        SELECT id FROM gjj_crm_data_template
                        WHERE name = '接单与建群' ORDER BY id LIMIT 1
                    ) THEN 100 + COALESCE(data_field.sort, 100)
                    ELSE 300 + COALESCE(data_field.sort, 100)
                END
            END AS target_sort
        FROM (
            SELECT DISTINCT ON (field_source) field.*
            FROM crm_signing_form_field_source AS field
            WHERE field.field_source LIKE 'main:1:%'
               OR field.field_source LIKE 'main:2:%'
               OR field.data_template_id IN (
                    SELECT id FROM gjj_crm_data_template
                    WHERE name IN ('接单与建群', '资产基础信息')
               )
            ORDER BY field_source, id
        ) AS source
        LEFT JOIN gjj_crm_data_field AS data_field ON data_field.id = source.data_field_id
    ) AS prepared
    ORDER BY prepared.target_sort, prepared.id;
END $$;

UPDATE gjj_crm_form
SET status = 2, updated_at = CURRENT_TIMESTAMP
WHERE name IN ('客户来源与基础建档', '客户资料与资产建档');

DO $$
DECLARE
    eleven_form_id BIGINT;
    eleven_template_id BIGINT;
BEGIN
    SELECT id INTO eleven_form_id
    FROM gjj_crm_form
    WHERE name IN ('十一维资料收集', 'P01-P12十一维探针')
    ORDER BY CASE WHEN name = '十一维资料收集' THEN 0 ELSE 1 END, id
    LIMIT 1;

    SELECT id INTO eleven_template_id
    FROM gjj_crm_data_template
    WHERE name = '十一维资料'
    ORDER BY id
    LIMIT 1;

    IF eleven_form_id IS NULL OR eleven_template_id IS NULL THEN
        RAISE EXCEPTION '缺少十一维资料模板';
    END IF;

    UPDATE gjj_crm_form
    SET name = '十一维资料收集',
        description = 'PM按P01-P12分组收集资料；每组探针选项必填，证据和备注可分次补充。',
        status = 1,
        sort = 20,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = eleven_form_id;

    DELETE FROM gjj_crm_form_field WHERE form_id = eleven_form_id;

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
        eleven_form_id,
        2,
        child.data_template_id,
        'data:' || child.id,
        '["cate:2","template:' || child.data_template_id || '","data:' || child.id || '"]',
        '',
        child.id,
        child.name,
        child.name = '探针选项',
        FALSE,
        parent.sort * 100 + child.sort,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS child
    JOIN gjj_crm_data_field AS parent ON parent.id = child.parent_field_id
    WHERE child.data_template_id = eleven_template_id
      AND child.parent_field_id > 0
      AND child.field_type <> 'group'
      AND child.status = 1
      AND parent.status = 1
    ORDER BY parent.sort, parent.id, child.sort, child.id;
END $$;

UPDATE gjj_crm_form
SET status = 2, updated_at = CURRENT_TIMESTAMP
WHERE name IN (
    'PM案件判断与产品确认',
    '专业门禁协作',
    '律师正式T与合同法律边界',
    'ALA资产运营与回款支撑',
    '财务费用与收款审核',
    '合同路径与文本门禁'
);

UPDATE gjj_crm_form
SET name = '合同签署登记',
    description = '登记合同组合、正式签署状态、合同文件及费用信息。',
    status = 1,
    sort = 30,
    updated_at = CURRENT_TIMESTAMP
WHERE name IN ('合同与费用', '合同签署登记');

-- Preserve the current T-node decision logic while making its result persistent.
UPDATE gjj_crm_rule_script
SET script = replace(
        script,
        'return { value: value, reason: reason };',
        'return { value: value, reason: reason, fields: { candidate_t_node: value, candidate_t_confidence_level: value === "T0" ? "pending" : "high" } };'
    ),
    status = 1,
    updated_at = CURRENT_TIMESTAMP
WHERE name = '十一维T节点自动判断'
  AND script NOT LIKE '%candidate_t_node%';

UPDATE gjj_crm_rule_script
SET status = 2, updated_at = CURRENT_TIMESTAMP
WHERE name = 'P01-P12签约方向自动判断';

-- Product category is display/configuration metadata. Only asset-operation products
-- start the operation workflow after the default signing workflow completes.
UPDATE gjj_crm_product
SET category_id = (SELECT id FROM gjj_crm_product_category WHERE name = '司法服务' ORDER BY id LIMIT 1),
    service_workflow_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE code = ANY (ARRAY['S01', 'S03', 'S08', 'S10', 'S11', 'S12', 'S13', 'S19']);

UPDATE gjj_crm_product
SET category_id = (SELECT id FROM gjj_crm_product_category WHERE name = '债务结构' ORDER BY id LIMIT 1),
    service_workflow_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE code = ANY (ARRAY['S04', 'S05', 'S06', 'S07', 'S09', 'S17', 'S18']);

UPDATE gjj_crm_product
SET category_id = (SELECT id FROM gjj_crm_product_category WHERE name = '资产运营' ORDER BY id LIMIT 1),
    service_workflow_id = COALESCE((
        SELECT id FROM gjj_crm_workflow
        WHERE name IN ('运营流程', '租赁运营流程') AND status = 1
        ORDER BY CASE WHEN name = '租赁运营流程' THEN 0 ELSE 1 END, id
        LIMIT 1
    ), 0),
    updated_at = CURRENT_TIMESTAMP
WHERE code = ANY (ARRAY['S14', 'S15', 'S16', 'S22-07', 'S22-13']);

UPDATE gjj_crm_product
SET category_id = (SELECT id FROM gjj_crm_product_category WHERE name = '风险处置' ORDER BY id LIMIT 1),
    service_workflow_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE code = ANY (ARRAY['S20', 'S21', 'S23', 'S24', 'S25', 'S26']);

UPDATE gjj_crm_product
SET category_id = (SELECT id FROM gjj_crm_product_category WHERE name = '咨询服务' ORDER BY id LIMIT 1),
    service_workflow_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE category_id = 0;

DO $$
DECLARE
    signing_workflow_id BIGINT;
    npl_department_id BIGINT;
    pm_department_id BIGINT;
    law_department_id BIGINT;
    ala_department_id BIGINT;
    fin_department_id BIGINT;
    contract_department_id BIGINT;
    entry_form_id BIGINT;
    eleven_form_id BIGINT;
    contract_form_id BIGINT;
    t_rule_id BIGINT;
    current_stage_id BIGINT;
BEGIN
    SELECT id INTO signing_workflow_id
    FROM gjj_crm_workflow
    WHERE default_entry = TRUE OR name IN ('签约流程', '签署流程')
    ORDER BY CASE WHEN default_entry THEN 0 ELSE 1 END, id
    LIMIT 1;

    IF signing_workflow_id IS NULL THEN
        INSERT INTO gjj_crm_workflow (
            name, default_entry, sort, status, created_at, updated_at
        ) VALUES (
            '签约流程', TRUE, 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO signing_workflow_id;
    END IF;

    SELECT id INTO npl_department_id FROM gjj_crm_department WHERE code = 'NPL' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO pm_department_id FROM gjj_crm_department WHERE code = 'PM' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO law_department_id FROM gjj_crm_department WHERE code = 'LAW' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO ala_department_id FROM gjj_crm_department WHERE code = 'ALA' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO fin_department_id FROM gjj_crm_department WHERE code = 'FIN' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO contract_department_id FROM gjj_crm_department WHERE code = 'CONTRACT' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO entry_form_id FROM gjj_crm_form WHERE name = '接单建档' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO eleven_form_id FROM gjj_crm_form WHERE name = '十一维资料收集' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO contract_form_id FROM gjj_crm_form WHERE name = '合同签署登记' AND status = 1 ORDER BY id LIMIT 1;
    SELECT id INTO t_rule_id FROM gjj_crm_rule_script WHERE name = '十一维T节点自动判断' AND status = 1 ORDER BY id LIMIT 1;

    IF npl_department_id IS NULL OR pm_department_id IS NULL
       OR law_department_id IS NULL OR ala_department_id IS NULL
       OR fin_department_id IS NULL OR contract_department_id IS NULL THEN
        RAISE EXCEPTION '签约流程所需部门未完整启用';
    END IF;
    IF entry_form_id IS NULL OR eleven_form_id IS NULL
       OR contract_form_id IS NULL OR t_rule_id IS NULL THEN
        RAISE EXCEPTION '签约流程所需资料模板或规则未完整启用';
    END IF;

    DELETE FROM gjj_crm_task
    WHERE stage_id IN (
        SELECT id FROM gjj_crm_stage
        WHERE workflow_id = signing_workflow_id OR workflow_id = 0
    );
    DELETE FROM gjj_crm_stage
    WHERE workflow_id = signing_workflow_id OR workflow_id = 0;

    UPDATE gjj_crm_workflow
    SET default_entry = FALSE, updated_at = CURRENT_TIMESTAMP
    WHERE id <> signing_workflow_id;
    UPDATE gjj_crm_workflow
    SET name = '签约流程', default_entry = TRUE, sort = 10, status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = signing_workflow_id;

    INSERT INTO gjj_crm_stage (
        workflow_id, name, owner_department_id, assignment_mode,
        sort, status, created_at, updated_at
    ) VALUES (
        signing_workflow_id, '接单建档', npl_department_id, 'auto',
        10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    ) RETURNING id INTO current_stage_id;
    INSERT INTO gjj_crm_task (
        stage_id, name, task_type, required, assignee_mode,
        assignee_department_id, form_id, script_id, due_days,
        sort, status, created_at, updated_at
    ) VALUES (
        current_stage_id, '完成接单建档', 'form', TRUE, 'stage',
        0, entry_form_id, 0, 0,
        10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    );

    INSERT INTO gjj_crm_stage (
        workflow_id, name, owner_department_id, assignment_mode,
        sort, status, created_at, updated_at
    ) VALUES (
        signing_workflow_id, '资料收集', pm_department_id, 'auto',
        20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    ) RETURNING id INTO current_stage_id;
    INSERT INTO gjj_crm_task (
        stage_id, name, task_type, required, assignee_mode,
        assignee_department_id, form_id, script_id, due_days,
        sort, status, created_at, updated_at
    ) VALUES (
        current_stage_id, '收集十一维资料', 'form', TRUE, 'stage',
        0, eleven_form_id, 0, 0,
        10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    );

    INSERT INTO gjj_crm_stage (
        workflow_id, name, owner_department_id, assignment_mode,
        sort, status, created_at, updated_at
    ) VALUES (
        signing_workflow_id, '诊断核验', pm_department_id, 'auto',
        30, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    ) RETURNING id INTO current_stage_id;
    INSERT INTO gjj_crm_task (
        stage_id, name, task_type, required, assignee_mode,
        assignee_department_id, form_id, script_id, due_days,
        sort, status, created_at, updated_at
    ) VALUES
        (
            current_stage_id, '自动判断T节点', 'rule', TRUE, 'stage',
            0, 0, t_rule_id, 0,
            10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ),
        (
            current_stage_id, '确认诊断结果', 'approval', TRUE, 'stage',
            0, 0, 0, 0,
            20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        );

    INSERT INTO gjj_crm_stage (
        workflow_id, name, owner_department_id, assignment_mode,
        sort, status, created_at, updated_at
    ) VALUES (
        signing_workflow_id, '产品确认', pm_department_id, 'auto',
        40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    ) RETURNING id INTO current_stage_id;
    INSERT INTO gjj_crm_task (
        stage_id, name, task_type, required, assignee_mode,
        assignee_department_id, form_id, script_id, due_days,
        sort, status, created_at, updated_at
    ) VALUES (
        current_stage_id, '确认适用产品', 'product', TRUE, 'stage',
        0, 0, 0, 0,
        10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    );

    INSERT INTO gjj_crm_stage (
        workflow_id, name, owner_department_id, assignment_mode,
        sort, status, created_at, updated_at
    ) VALUES (
        signing_workflow_id, '签约协作', pm_department_id, 'auto',
        50, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    ) RETURNING id INTO current_stage_id;
    INSERT INTO gjj_crm_task (
        stage_id, name, task_type, required, assignee_mode,
        assignee_department_id, form_id, script_id, due_days,
        sort, status, created_at, updated_at
    ) VALUES
        (
            current_stage_id, '律师合同审核', 'approval', TRUE, 'auto',
            law_department_id, 0, 0, 0,
            10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ),
        (
            current_stage_id, 'ALA运营条件确认', 'approval', TRUE, 'auto',
            ala_department_id, 0, 0, 0,
            20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ),
        (
            current_stage_id, '财务费用确认', 'approval', TRUE, 'auto',
            fin_department_id, 0, 0, 0,
            30, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        );

    INSERT INTO gjj_crm_stage (
        workflow_id, name, owner_department_id, assignment_mode,
        sort, status, created_at, updated_at
    ) VALUES (
        signing_workflow_id, '合同签署', contract_department_id, 'auto',
        60, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    ) RETURNING id INTO current_stage_id;
    INSERT INTO gjj_crm_task (
        stage_id, name, task_type, required, assignee_mode,
        assignee_department_id, form_id, script_id, due_days,
        sort, status, created_at, updated_at
    ) VALUES (
        current_stage_id, '合同签署登记', 'form', TRUE, 'stage',
        0, contract_form_id, 0, 0,
        10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    );
END $$;

-- Remove tables and columns from the superseded pre-instance workflow model.
DROP INDEX IF EXISTS idx_gjj_crm_product_category;
DROP INDEX IF EXISTS idx_gjj_crm_product_signing_type;
ALTER TABLE IF EXISTS gjj_crm_product
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS default_signing_business_type,
    DROP COLUMN IF EXISTS signing_direction,
    DROP COLUMN IF EXISTS default_signing_direction,
    DROP COLUMN IF EXISTS need_pm_review,
    DROP COLUMN IF EXISTS need_lawyer_review,
    DROP COLUMN IF EXISTS need_ala_review,
    DROP COLUMN IF EXISTS need_finance_review,
    DROP COLUMN IF EXISTS need_contract_review;

DROP INDEX IF EXISTS idx_gjj_crm_workflow_next_workflow;
ALTER TABLE IF EXISTS gjj_crm_workflow DROP COLUMN IF EXISTS next_workflow_id;

DROP INDEX IF EXISTS idx_gjj_crm_data_template_cate_target;
DROP INDEX IF EXISTS idx_gjj_crm_data_template_cate_business_object_type;
ALTER TABLE IF EXISTS gjj_crm_data_template_cate
    DROP COLUMN IF EXISTS target_table,
    DROP COLUMN IF EXISTS business_object_type_id;

DROP INDEX IF EXISTS idx_gjj_crm_data_record_business_object_template;
ALTER TABLE IF EXISTS gjj_crm_data_record DROP COLUMN IF EXISTS business_object_id;
DROP INDEX IF EXISTS idx_gjj_crm_stat_field_value_business_object_time;
ALTER TABLE IF EXISTS gjj_crm_stat_field_value DROP COLUMN IF EXISTS business_object_id;
DROP INDEX IF EXISTS idx_gjj_crm_finance_ledger_business_object_time;
ALTER TABLE IF EXISTS gjj_crm_finance_ledger DROP COLUMN IF EXISTS business_object_id;

DROP TABLE IF EXISTS gjj_crm_asset_progress;
DROP TABLE IF EXISTS gjj_crm_work_todo;
DROP TABLE IF EXISTS gjj_crm_business_object;
DROP TABLE IF EXISTS gjj_crm_business_object_type;

COMMIT;

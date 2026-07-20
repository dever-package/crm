-- Add configurable collaboration routes, conditional contract fields and task meetings.
BEGIN;

ALTER TABLE gjj_crm_task
    ADD COLUMN IF NOT EXISTS activation_mode VARCHAR(32) NOT NULL DEFAULT 'stage',
    ADD COLUMN IF NOT EXISTS condition_script_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reject_target_task_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS complete_target_task_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS meeting_start_field_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS meeting_duration_field_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS meeting_resource_field_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS include_in_meeting BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE gjj_crm_task
SET activation_mode = 'stage'
WHERE activation_mode IS NULL OR BTRIM(activation_mode) = '';

CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_condition_script
    ON gjj_crm_task (condition_script_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_reject_target
    ON gjj_crm_task (reject_target_task_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_complete_target
    ON gjj_crm_task (complete_target_task_id, status, id);

ALTER TABLE gjj_crm_form_field
    ADD COLUMN IF NOT EXISTS visible_when_field_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS visible_when_operator VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS visible_when_value TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_gjj_crm_form_field_visible_when
    ON gjj_crm_form_field (visible_when_field_id, status, id);

ALTER TABLE gjj_crm_schedule_event
    ADD COLUMN IF NOT EXISTS asset_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS source_task_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS meeting_source_key VARCHAR(96);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_schedule_event_meeting_source
    ON gjj_crm_schedule_event (meeting_source_key);

ALTER TABLE gjj_crm_schedule_participant
    ADD COLUMN IF NOT EXISTS checked_in_at TIMESTAMPTZ;

DO $$
DECLARE
    contract_template_id BIGINT;
    contract_template_cate_id BIGINT;
    contract_form_id BIGINT;
    share_option_set_id BIGINT;
BEGIN
    SELECT id, cate_id
    INTO contract_template_id, contract_template_cate_id
    FROM gjj_crm_data_template
    WHERE name = '合同信息' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO contract_form_id
    FROM gjj_crm_form
    WHERE name = '合同签署登记' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO share_option_set_id
    FROM gjj_crm_option_set
    WHERE name = '房东分成百分比' AND status = 1
    ORDER BY id
    LIMIT 1;

    IF contract_template_id IS NULL OR contract_form_id IS NULL OR share_option_set_id IS NULL THEN
        RAISE EXCEPTION '合同动态字段所需的合同模板、表单或房东分成选项集未完整启用';
    END IF;

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
        contract_template_id,
        0,
        CASE WHEN seed.field_key = 'landlord_share_rate' THEN share_option_set_id ELSE 0 END,
        seed.name,
        seed.field_key,
        seed.field_type,
        '',
        seed.sort,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('service_period', '服务周期', 'text', 100),
            ('payment_method', '支付方式', 'text', 110),
            ('expected_handover_at', '预计收房时间', 'datetime', 120),
            ('custody_operation_start_at', '托管运营开始时间', 'date', 130),
            ('custody_operation_end_at', '托管运营结束时间', 'date', 140),
            ('landlord_share_rate', '房东分成百分比', 'radio', 150),
            ('moving_subsidy_amount', '搬家费补助', 'money', 160)
    ) AS seed(field_key, name, field_type, sort)
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_data_field AS existing
        WHERE existing.field_key = seed.field_key
    );

    UPDATE gjj_crm_data_field AS field
    SET data_template_id = contract_template_id,
        parent_field_id = 0,
        option_set_id = CASE WHEN field.field_key = 'landlord_share_rate' THEN share_option_set_id ELSE 0 END,
        name = seed.name,
        field_type = seed.field_type,
        sort = seed.sort,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('service_period', '服务周期', 'text', 100),
            ('payment_method', '支付方式', 'text', 110),
            ('expected_handover_at', '预计收房时间', 'datetime', 120),
            ('custody_operation_start_at', '托管运营开始时间', 'date', 130),
            ('custody_operation_end_at', '托管运营结束时间', 'date', 140),
            ('landlord_share_rate', '房东分成百分比', 'radio', 150),
            ('moving_subsidy_amount', '搬家费补助', 'money', 160)
    ) AS seed(field_key, name, field_type, sort)
    WHERE field.field_key = seed.field_key;

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
        contract_form_id,
        contract_template_cate_id,
        field.data_template_id,
        'data:' || field.id,
        json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || field.data_template_id,
            'data:' || field.id
        )::TEXT,
        '',
        field.id,
        field.name,
        FALSE,
        FALSE,
        field.sort,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE field.field_key = ANY (ARRAY[
        'service_period',
        'payment_method',
        'expected_handover_at',
        'custody_operation_start_at',
        'custody_operation_end_at',
        'landlord_share_rate',
        'moving_subsidy_amount'
    ])
      AND NOT EXISTS (
          SELECT 1
          FROM gjj_crm_form_field AS existing
          WHERE existing.form_id = contract_form_id
            AND existing.data_field_id = field.id
      );

    UPDATE gjj_crm_form_field AS form_field
    SET name = data_field.name,
        required = data_field.field_key IN ('contract_combination_id', 'formal_signing_status', 'service_fee_amount'),
        readonly = FALSE,
        visible_when_field_id = CASE
            WHEN data_field.field_key IN (
                'service_period',
                'service_fee_amount',
                'payment_method',
                'deposit_amount',
                'expected_handover_at',
                'custody_operation_start_at',
                'custody_operation_end_at',
                'landlord_share_rate',
                'htxx.yufuzujin',
                'moving_subsidy_amount'
            ) THEN (
                SELECT id
                FROM gjj_crm_data_field
                WHERE field_key = 'contract_combination_id'
                ORDER BY id
                LIMIT 1
            )
            ELSE 0
        END,
        visible_when_operator = CASE
            WHEN data_field.field_key IN (
                'service_period', 'service_fee_amount', 'payment_method',
                'deposit_amount', 'expected_handover_at',
                'custody_operation_start_at', 'custody_operation_end_at',
                'landlord_share_rate', 'htxx.yufuzujin', 'moving_subsidy_amount'
            ) THEN 'equals'
            ELSE ''
        END,
        visible_when_value = CASE
            WHEN data_field.field_key IN ('service_period', 'service_fee_amount', 'payment_method')
                THEN 'SS-SERVICE-PACKAGE-20260704'
            WHEN data_field.field_key IN (
                'deposit_amount', 'expected_handover_at',
                'custody_operation_start_at', 'custody_operation_end_at',
                'landlord_share_rate', 'htxx.yufuzujin', 'moving_subsidy_amount'
            ) THEN 'NS-MAIN-PACKAGE-20260704'
            ELSE ''
        END,
        sort = CASE data_field.field_key
            WHEN 'contract_combination_id' THEN 10
            WHEN 'contract_status' THEN 20
            WHEN 'formal_signing_status' THEN 30
            WHEN 'contract_file' THEN 40
            WHEN 'service_period' THEN 50
            WHEN 'service_fee_amount' THEN 60
            WHEN 'payment_method' THEN 70
            WHEN 'deposit_amount' THEN 80
            WHEN 'expected_handover_at' THEN 90
            WHEN 'custody_operation_start_at' THEN 100
            WHEN 'custody_operation_end_at' THEN 110
            WHEN 'landlord_share_rate' THEN 120
            WHEN 'htxx.yufuzujin' THEN 130
            WHEN 'moving_subsidy_amount' THEN 140
            ELSE form_field.sort
        END,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS data_field
    WHERE form_field.form_id = contract_form_id
      AND form_field.data_field_id = data_field.id
      AND data_field.field_key = ANY (ARRAY[
          'contract_combination_id', 'contract_status', 'formal_signing_status', 'contract_file',
          'service_period', 'service_fee_amount', 'payment_method',
          'deposit_amount', 'expected_handover_at',
          'custody_operation_start_at', 'custody_operation_end_at',
          'landlord_share_rate', 'htxx.yufuzujin', 'moving_subsidy_amount'
      ]);

    UPDATE gjj_crm_form_field AS form_field
    SET status = 2,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS data_field
    WHERE form_field.form_id = contract_form_id
      AND form_field.data_field_id = data_field.id
      AND data_field.field_key IN ('htxx.banjiafeibuzhu', 'third_party_cost');

    UPDATE gjj_crm_form
    SET description = '根据合同组合展示对应签署、费用和运营交付字段。',
        updated_at = CURRENT_TIMESTAMP
    WHERE id = contract_form_id;
END $$;

DO $$
DECLARE
    sealed_status_id BIGINT;
    unsealed_status_ids TEXT;
    service_fee_code TEXT;
    ala_script TEXT;
    sealed_delivery_script TEXT;
BEGIN
    SELECT id INTO sealed_status_id
    FROM gjj_crm_asset_status
    WHERE name = '已查封' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT json_agg(id ORDER BY id)::TEXT INTO unsealed_status_ids
    FROM gjj_crm_asset_status
    WHERE name LIKE '未查封%' AND status = 1;

    SELECT code INTO service_fee_code
    FROM gjj_crm_finance_type
    WHERE name = '服务费收入' AND status = 1
    ORDER BY id
    LIMIT 1;

    IF sealed_status_id IS NULL OR unsealed_status_ids IS NULL OR service_fee_code IS NULL THEN
        RAISE EXCEPTION '流程条件所需的资产状态或服务费财务类型未完整启用';
    END IF;

    ala_script := format($script$
function evaluate(input) {
  var current = input && input.current ? input.current : {};
  var asset = current.asset || {};
  var allowedStatusIDs = %s;
  var statusID = Number(asset.asset_status_id || 0);
  var passed = allowedStatusIDs.indexOf(statusID) >= 0;
  return {
    passed: passed,
    reason: passed ? "未查封资产需要ALA运营条件确认" : "当前资产无需ALA运营条件确认"
  };
}
$script$, unsealed_status_ids);

    sealed_delivery_script := format($script$
function evaluate(input) {
  var current = input && input.current ? input.current : {};
  var asset = current.asset || {};
  var finance = asset.finance || {};
  var serviceFee = finance[%s] || {};
  var sealed = Number(asset.asset_status_id || 0) === %s;
  var received = Number(serviceFee.amount || 0) > 0;
  return {
    passed: sealed && received,
    reason: sealed ? (received ? "已收到服务费" : "尚未收到服务费") : "非查封资产不走查封交付接单"
  };
}
$script$, to_json(service_fee_code)::TEXT, sealed_status_id);

    INSERT INTO gjj_crm_rule_script (
        cate_id, name, description, script, status, sort, created_at, updated_at
    )
    SELECT 0, '非查封资产协作条件', '用于ALA确认和非查封交付路由。', ala_script, 1, 200,
           CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1 FROM gjj_crm_rule_script WHERE name = '非查封资产协作条件'
    );

    UPDATE gjj_crm_rule_script
    SET description = '用于ALA确认和非查封交付路由。',
        script = ala_script,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE name = '非查封资产协作条件';

    INSERT INTO gjj_crm_rule_script (
        cate_id, name, description, script, status, sort, created_at, updated_at
    )
    SELECT 0, '查封资产交付接单条件', '查封资产收到首笔服务费后激活交付接单。', sealed_delivery_script, 1, 210,
           CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1 FROM gjj_crm_rule_script WHERE name = '查封资产交付接单条件'
    );

    UPDATE gjj_crm_rule_script
    SET description = '查封资产收到首笔服务费后激活交付接单。',
        script = sealed_delivery_script,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE name = '查封资产交付接单条件';
END $$;

DO $$
DECLARE
    signing_workflow_id BIGINT;
    operation_workflow_id BIGINT;
    collaboration_stage_id BIGINT;
    contract_stage_id BIGINT;
    visit_stage_id BIGINT;
    receive_stage_id BIGINT;
    service_department_id BIGINT;
    ala_condition_script_id BIGINT;
    sealed_delivery_script_id BIGINT;
    ala_task_id BIGINT;
    pm_correction_task_id BIGINT;
    contract_task_id BIGINT;
    sealed_delivery_task_id BIGINT;
    receive_task_id BIGINT;
    nonsealed_delivery_task_id BIGINT;
    meeting_task_id BIGINT;
    meeting_start_data_field_id BIGINT;
    meeting_duration_data_field_id BIGINT;
    meeting_resource_data_field_id BIGINT;
BEGIN
    SELECT id INTO signing_workflow_id
    FROM gjj_crm_workflow
    WHERE subject_type = 'customer_asset' AND default_entry = TRUE AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO operation_workflow_id
    FROM gjj_crm_workflow
    WHERE subject_type = 'customer_asset' AND name = '运营流程' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO service_department_id
    FROM gjj_crm_department
    WHERE code = 'SERVICE' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO ala_condition_script_id
    FROM gjj_crm_rule_script
    WHERE name = '非查封资产协作条件' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO sealed_delivery_script_id
    FROM gjj_crm_rule_script
    WHERE name = '查封资产交付接单条件' AND status = 1
    ORDER BY id
    LIMIT 1;

    IF signing_workflow_id IS NULL OR operation_workflow_id IS NULL
       OR service_department_id IS NULL
       OR ala_condition_script_id IS NULL OR sealed_delivery_script_id IS NULL THEN
        RAISE EXCEPTION '协作流程所需的流程、部门或条件规则未完整启用';
    END IF;

    SELECT id INTO collaboration_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = signing_workflow_id AND name = '签约协作' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO contract_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = signing_workflow_id AND name = '合同签署' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO visit_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = signing_workflow_id AND name = '邀约到访' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO receive_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = operation_workflow_id AND name = '收房' AND status = 1
    ORDER BY id
    LIMIT 1;

    IF collaboration_stage_id IS NULL OR contract_stage_id IS NULL
       OR visit_stage_id IS NULL OR receive_stage_id IS NULL THEN
        RAISE EXCEPTION '协作流程所需的阶段未完整启用';
    END IF;

    SELECT id INTO ala_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id AND name = 'ALA运营条件确认' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO contract_task_id
    FROM gjj_crm_task
    WHERE stage_id = contract_stage_id AND name = '合同签署登记' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO receive_task_id
    FROM gjj_crm_task
    WHERE stage_id = receive_stage_id AND name = '收房确认' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO meeting_task_id
    FROM gjj_crm_task
    WHERE stage_id = visit_stage_id AND name = '预约会议室' AND status = 1
    ORDER BY sort, id
    LIMIT 1;

    IF ala_task_id IS NULL OR contract_task_id IS NULL OR receive_task_id IS NULL OR meeting_task_id IS NULL THEN
        RAISE EXCEPTION '协作流程所需的源任务未完整启用';
    END IF;

    SELECT id INTO pm_correction_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id AND name = 'PM补充签约资料'
    ORDER BY id
    LIMIT 1;

    IF pm_correction_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required, assignee_mode,
            assignee_department_id, form_id, script_id, activation_mode,
            condition_script_id, reject_target_task_id, complete_target_task_id,
            due_days, sort, status, created_at, updated_at
        ) VALUES (
            collaboration_stage_id, 'PM补充签约资料', 'todo', TRUE, 'stage',
            0, 0, 0, 'route',
            ala_condition_script_id, 0, ala_task_id,
            0, 90, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO pm_correction_task_id;
    END IF;

    UPDATE gjj_crm_task
    SET name = 'PM补充签约资料',
        task_type = 'todo',
        required = TRUE,
        assignee_mode = 'stage',
        assignee_department_id = 0,
        activation_mode = 'route',
        condition_script_id = ala_condition_script_id,
        reject_target_task_id = 0,
        complete_target_task_id = ala_task_id,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = pm_correction_task_id;

    UPDATE gjj_crm_task
    SET activation_mode = 'stage',
        condition_script_id = ala_condition_script_id,
        reject_target_task_id = pm_correction_task_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = ala_task_id;

    SELECT id INTO sealed_delivery_task_id
    FROM gjj_crm_task
    WHERE stage_id = contract_stage_id AND name = '交付部接单'
    ORDER BY id
    LIMIT 1;

    IF sealed_delivery_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required, assignee_mode,
            assignee_department_id, form_id, script_id, activation_mode,
            condition_script_id, due_days, sort, status, created_at, updated_at
        ) VALUES (
            contract_stage_id, '交付部接单', 'todo', TRUE, 'auto',
            service_department_id, 0, 0, 'route',
            sealed_delivery_script_id, 0, 90, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO sealed_delivery_task_id;
    END IF;

    UPDATE gjj_crm_task
    SET task_type = 'todo',
        required = TRUE,
        assignee_mode = 'auto',
        assignee_department_id = service_department_id,
        activation_mode = 'route',
        condition_script_id = sealed_delivery_script_id,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = sealed_delivery_task_id;

    UPDATE gjj_crm_task
    SET complete_target_task_id = sealed_delivery_task_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = contract_task_id;

    SELECT id INTO nonsealed_delivery_task_id
    FROM gjj_crm_task
    WHERE stage_id = receive_stage_id AND name = '交付部接单'
    ORDER BY id
    LIMIT 1;

    IF nonsealed_delivery_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required, assignee_mode,
            assignee_department_id, form_id, script_id, activation_mode,
            condition_script_id, due_days, sort, status, created_at, updated_at
        ) VALUES (
            receive_stage_id, '交付部接单', 'todo', TRUE, 'auto',
            service_department_id, 0, 0, 'route',
            ala_condition_script_id, 0, 90, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO nonsealed_delivery_task_id;
    END IF;

    UPDATE gjj_crm_task
    SET task_type = 'todo',
        required = TRUE,
        assignee_mode = 'auto',
        assignee_department_id = service_department_id,
        activation_mode = 'route',
        condition_script_id = ala_condition_script_id,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = nonsealed_delivery_task_id;

    UPDATE gjj_crm_task
    SET complete_target_task_id = nonsealed_delivery_task_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = receive_task_id;

    SELECT id INTO meeting_start_data_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'yydf.yaoyueshijian' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO meeting_duration_data_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'yydf.yuyuehuiyishishizhang' AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO meeting_resource_data_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'yydf.yuyuehuiyishi' AND status = 1
    ORDER BY id
    LIMIT 1;

    IF meeting_start_data_field_id IS NULL OR meeting_duration_data_field_id IS NULL OR meeting_resource_data_field_id IS NULL THEN
        RAISE EXCEPTION '会议预约字段未完整启用';
    END IF;

    UPDATE gjj_crm_data_field
    SET name = '预约会议室时长（分钟）',
        default_value = CASE WHEN BTRIM(default_value) = '' THEN '60' ELSE default_value END,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = meeting_duration_data_field_id;

    UPDATE gjj_crm_form_field
    SET name = '预约会议室时长（分钟）',
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = (SELECT form_id FROM gjj_crm_task WHERE id = meeting_task_id)
      AND data_field_id = meeting_duration_data_field_id;

    UPDATE gjj_crm_task
    SET meeting_start_field_id = meeting_start_data_field_id,
        meeting_duration_field_id = meeting_duration_data_field_id,
        meeting_resource_field_id = meeting_resource_data_field_id,
        include_in_meeting = TRUE,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = meeting_task_id;

    UPDATE gjj_crm_task AS task
    SET include_in_meeting = TRUE,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_stage AS stage
    WHERE task.stage_id = stage.id
      AND stage.workflow_id = signing_workflow_id
      AND task.status = 1
      AND (
          task.name IN ('确认适用产品', 'PM谈话笔录', 'ALA运营条件确认')
          OR task.id = meeting_task_id
      );
END $$;

COMMIT;

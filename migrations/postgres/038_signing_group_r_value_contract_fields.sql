-- Route eligible cases to data-center group creation and align signing fields.
BEGIN;

ALTER TABLE gjj_crm_task
    ADD COLUMN IF NOT EXISTS communication_group_enabled BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
DECLARE
    signing_workflow_id BIGINT;
    intake_stage_id BIGINT;
    intake_task_id BIGINT;
    intake_task_target_id BIGINT;
    intake_form_id BIGINT;
    group_form_id BIGINT;
    data_center_department_id BIGINT;
    group_template_id BIGINT;
    group_template_cate_id BIGINT;
    group_status_field_id BIGINT;
    group_decision_field_id BIGINT;
    group_decision_option_set_id BIGINT;
    group_condition_script_id BIGINT;
    group_task_id BIGINT;
    group_condition_script TEXT;
BEGIN
    SELECT id INTO signing_workflow_id
    FROM gjj_crm_workflow
    WHERE subject_type = 'customer_asset'
      AND default_entry = TRUE
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO intake_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = signing_workflow_id
      AND name = '接单建档'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id, form_id, complete_target_task_id
    INTO intake_task_id, intake_form_id, intake_task_target_id
    FROM gjj_crm_task
    WHERE stage_id = intake_stage_id
      AND name = '完成接单建档'
      AND task_type = 'form'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO data_center_department_id
    FROM gjj_crm_department
    WHERE status = 1
      AND (code = 'CONTRACT' OR name = '数据中台')
    ORDER BY CASE WHEN name = '数据中台' THEN 0 ELSE 1 END, id
    LIMIT 1;

    SELECT field.id, field.data_template_id, template.cate_id
    INTO group_status_field_id, group_template_id, group_template_cate_id
    FROM gjj_crm_data_field AS field
    INNER JOIN gjj_crm_data_template AS template
        ON template.id = field.data_template_id
    WHERE field.field_key = 'service_group_status'
      AND field.status = 1
      AND template.status = 1
    ORDER BY field.id
    LIMIT 1;

    IF signing_workflow_id IS NULL
       OR intake_stage_id IS NULL
       OR intake_task_id IS NULL
       OR intake_form_id IS NULL
       OR data_center_department_id IS NULL
       OR group_status_field_id IS NULL
       OR group_template_id IS NULL THEN
        RAISE EXCEPTION '建企微群路由所需的签约流程、接单任务、数据中台或建群字段未完整启用';
    END IF;

    SELECT id INTO group_decision_option_set_id
    FROM gjj_crm_option_set
    WHERE name = '建群结论'
    ORDER BY id
    LIMIT 1;

    IF group_decision_option_set_id IS NULL THEN
        INSERT INTO gjj_crm_option_set (
            name, sort, status, created_at, updated_at
        ) VALUES (
            '建群结论', 65, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO group_decision_option_set_id;
    ELSE
        UPDATE gjj_crm_option_set
        SET sort = 65,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = group_decision_option_set_id;
    END IF;

    INSERT INTO gjj_crm_option_set_item (
        option_set_id, name, value, sort, status
    )
    SELECT group_decision_option_set_id, seed.name, seed.value, seed.sort, 1
    FROM (
        VALUES
            ('可建群', 'eligible', 10),
            ('暂不建群', 'not_eligible', 20)
    ) AS seed(name, value, sort)
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_option_set_item AS existing
        WHERE existing.option_set_id = group_decision_option_set_id
          AND existing.value = seed.value
    );

    UPDATE gjj_crm_option_set_item AS item
    SET name = seed.name,
        sort = seed.sort,
        status = 1
    FROM (
        VALUES
            ('可建群', 'eligible', 10),
            ('暂不建群', 'not_eligible', 20)
    ) AS seed(name, value, sort)
    WHERE item.option_set_id = group_decision_option_set_id
      AND item.value = seed.value;

    SELECT id INTO group_decision_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'group_creation_decision'
    ORDER BY id
    LIMIT 1;

    IF group_decision_field_id IS NULL THEN
        INSERT INTO gjj_crm_data_field (
            data_template_id, parent_field_id, option_set_id,
            name, field_key, field_type, default_value,
            finance_type_id, stat_enabled, sort, status,
            created_at, updated_at
        ) VALUES (
            group_template_id, 0, group_decision_option_set_id,
            '建群结论', 'group_creation_decision', 'select', '',
            0, FALSE, 85, 1,
            CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO group_decision_field_id;
    ELSE
        UPDATE gjj_crm_data_field
        SET data_template_id = group_template_id,
            parent_field_id = 0,
            option_set_id = group_decision_option_set_id,
            name = '建群结论',
            field_type = 'select',
            default_value = '',
            finance_type_id = 0,
            sort = 85,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = group_decision_field_id;
    END IF;

    INSERT INTO gjj_crm_form_field (
        form_id, data_template_cate_id, data_template_id,
        field_source, field_path, main_field, data_field_id,
        name, required, readonly,
        visible_when_field_id, visible_when_operator, visible_when_value,
        sort, status, created_at, updated_at
    )
    SELECT
        intake_form_id, group_template_cate_id, group_template_id,
        'data:' || group_decision_field_id,
        json_build_array(
            'cate:' || group_template_cate_id,
            'template:' || group_template_id,
            'data:' || group_decision_field_id
        )::TEXT,
        '', group_decision_field_id,
        '建群结论', TRUE, FALSE,
        0, '', '',
        9, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS existing
        WHERE existing.form_id = intake_form_id
          AND existing.data_field_id = group_decision_field_id
    );

    UPDATE gjj_crm_form_field
    SET data_template_cate_id = group_template_cate_id,
        data_template_id = group_template_id,
        field_source = 'data:' || group_decision_field_id,
        field_path = json_build_array(
            'cate:' || group_template_cate_id,
            'template:' || group_template_id,
            'data:' || group_decision_field_id
        )::TEXT,
        main_field = '',
        name = '建群结论',
        required = TRUE,
        readonly = FALSE,
        visible_when_field_id = 0,
        visible_when_operator = '',
        visible_when_value = '',
        sort = 9,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = intake_form_id
      AND data_field_id = group_decision_field_id;

    UPDATE gjj_crm_form_field
    SET status = 2,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = intake_form_id
      AND data_field_id = group_status_field_id;

    UPDATE gjj_crm_form
    SET description = 'NPL完成客户建档、首呼和建群结论；可建群时自动转数据中台建群。',
        updated_at = CURRENT_TIMESTAMP
    WHERE id = intake_form_id;

    SELECT id INTO group_form_id
    FROM gjj_crm_form
    WHERE name = '建企微群'
    ORDER BY id
    LIMIT 1;

    IF group_form_id IS NULL THEN
        INSERT INTO gjj_crm_form (
            name, description, sort, status, created_at, updated_at
        ) VALUES (
            '建企微群',
            '数据中台维护企业微信等客户沟通群信息。',
            12,
            1,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO group_form_id;
    ELSE
        UPDATE gjj_crm_form
        SET description = '数据中台维护企业微信等客户沟通群信息。',
            sort = 12,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = group_form_id;
    END IF;

    INSERT INTO gjj_crm_form_field (
        form_id, data_template_cate_id, data_template_id,
        field_source, field_path, main_field, data_field_id,
        name, required, readonly,
        visible_when_field_id, visible_when_operator, visible_when_value,
        sort, status, created_at, updated_at
    )
    SELECT
        group_form_id, group_template_cate_id, group_template_id,
        'data:' || group_decision_field_id,
        json_build_array(
            'cate:' || group_template_cate_id,
            'template:' || group_template_id,
            'data:' || group_decision_field_id
        )::TEXT,
        '', group_decision_field_id,
        '建群结论', FALSE, TRUE,
        0, '', '',
        10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS existing
        WHERE existing.form_id = group_form_id
          AND existing.data_field_id = group_decision_field_id
    );

    UPDATE gjj_crm_form_field
    SET data_template_cate_id = group_template_cate_id,
        data_template_id = group_template_id,
        field_source = 'data:' || group_decision_field_id,
        field_path = json_build_array(
            'cate:' || group_template_cate_id,
            'template:' || group_template_id,
            'data:' || group_decision_field_id
        )::TEXT,
        main_field = '',
        name = '建群结论',
        required = FALSE,
        readonly = TRUE,
        visible_when_field_id = 0,
        visible_when_operator = '',
        visible_when_value = '',
        sort = 10,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = group_form_id
      AND data_field_id = group_decision_field_id;

    group_condition_script := $script$
function evaluate(input) {
  var customer = input && input.customer ? input.customer : {};
  var fields = customer.fields || {};
  var passed = String(fields.group_creation_decision || "") === "eligible";
  return {
    passed: passed,
    reason: passed ? "NPL确认可建群" : "当前无需建群"
  };
}
$script$;

    SELECT id INTO group_condition_script_id
    FROM gjj_crm_rule_script
    WHERE name = '可建群任务条件'
    ORDER BY id
    LIMIT 1;

    IF group_condition_script_id IS NULL THEN
        INSERT INTO gjj_crm_rule_script (
            cate_id, name, description, script,
            status, sort, created_at, updated_at
        ) VALUES (
            0,
            '可建群任务条件',
            'NPL选择可建群后激活数据中台建企微群任务。',
            group_condition_script,
            1,
            220,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO group_condition_script_id;
    ELSE
        UPDATE gjj_crm_rule_script
        SET description = 'NPL选择可建群后激活数据中台建企微群任务。',
            script = group_condition_script,
            status = 1,
            sort = 220,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = group_condition_script_id;
    END IF;

    SELECT id INTO group_task_id
    FROM gjj_crm_task
    WHERE stage_id = intake_stage_id
      AND name = '建企微群'
    ORDER BY id
    LIMIT 1;

    IF group_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required,
            assignee_mode, assignee_department_id,
            form_id, script_id,
            activation_mode, condition_script_id,
            reject_action, reject_target_task_id, complete_target_task_id,
            opinion_requirement, reject_submit_form,
            meeting_enabled, meeting_arrival_required,
            customer_follow_enabled, communication_group_enabled,
            include_in_meeting, due_days,
            sort, status, created_at, updated_at
        ) VALUES (
            intake_stage_id, '建企微群', 'form', TRUE,
            'auto', data_center_department_id,
            group_form_id, 0,
            'route', group_condition_script_id,
            'stay', 0, 0,
            'optional', FALSE,
            FALSE, FALSE,
            FALSE, TRUE,
            FALSE, 0,
            20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO group_task_id;
    ELSE
        UPDATE gjj_crm_task
        SET task_type = 'form',
            required = TRUE,
            assignee_mode = 'auto',
            assignee_department_id = data_center_department_id,
            form_id = group_form_id,
            script_id = 0,
            activation_mode = 'route',
            condition_script_id = group_condition_script_id,
            reject_action = 'stay',
            reject_target_task_id = 0,
            complete_target_task_id = 0,
            opinion_requirement = 'optional',
            reject_submit_form = FALSE,
            meeting_enabled = FALSE,
            meeting_arrival_required = FALSE,
            customer_follow_enabled = FALSE,
            communication_group_enabled = TRUE,
            include_in_meeting = FALSE,
            due_days = 0,
            sort = 20,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = group_task_id;
    END IF;

    IF intake_task_target_id > 0 AND intake_task_target_id <> group_task_id THEN
        RAISE EXCEPTION '完成接单建档已配置其他完成后任务，不能覆盖现有流转';
    END IF;

    UPDATE gjj_crm_task
    SET complete_target_task_id = group_task_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = intake_task_id;
END $$;

DO $$
DECLARE
    contract_template_id BIGINT;
    contract_template_cate_id BIGINT;
    ala_form_id BIGINT;
    ala_r_value_field_id BIGINT;
BEGIN
    SELECT id, cate_id
    INTO contract_template_id, contract_template_cate_id
    FROM gjj_crm_data_template
    WHERE name = '合同信息'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO ala_form_id
    FROM gjj_crm_form
    WHERE name = 'ALA运营条件确认'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    IF contract_template_id IS NULL OR ala_form_id IS NULL THEN
        RAISE EXCEPTION 'ALA评估R值所需的合同模板或ALA运营条件确认表单未启用';
    END IF;

    SELECT id INTO ala_r_value_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'ala_assessed_r_value'
    ORDER BY id
    LIMIT 1;

    IF ala_r_value_field_id IS NULL THEN
        INSERT INTO gjj_crm_data_field (
            data_template_id, parent_field_id, option_set_id,
            name, field_key, field_type, default_value,
            finance_type_id, stat_enabled, sort, status,
            created_at, updated_at
        ) VALUES (
            contract_template_id, 0, 0,
            'ALA评估R值', 'ala_assessed_r_value', 'number', '',
            0, TRUE, 190, 1,
            CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO ala_r_value_field_id;
    ELSE
        UPDATE gjj_crm_data_field
        SET data_template_id = contract_template_id,
            parent_field_id = 0,
            option_set_id = 0,
            name = 'ALA评估R值',
            field_type = 'number',
            default_value = '',
            finance_type_id = 0,
            stat_enabled = TRUE,
            sort = 190,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = ala_r_value_field_id;
    END IF;

    INSERT INTO gjj_crm_form_field (
        form_id, data_template_cate_id, data_template_id,
        field_source, field_path, main_field, data_field_id,
        name, required, readonly,
        visible_when_field_id, visible_when_operator, visible_when_value,
        sort, status, created_at, updated_at
    )
    SELECT
        ala_form_id, contract_template_cate_id, contract_template_id,
        'data:' || ala_r_value_field_id,
        json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || ala_r_value_field_id
        )::TEXT,
        '', ala_r_value_field_id,
        'ALA评估R值', TRUE, FALSE,
        0, '', '',
        20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS existing
        WHERE existing.form_id = ala_form_id
          AND existing.data_field_id = ala_r_value_field_id
    );

    UPDATE gjj_crm_form_field
    SET data_template_cate_id = contract_template_cate_id,
        data_template_id = contract_template_id,
        field_source = 'data:' || ala_r_value_field_id,
        field_path = json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || ala_r_value_field_id
        )::TEXT,
        main_field = '',
        name = 'ALA评估R值',
        required = TRUE,
        readonly = FALSE,
        visible_when_field_id = 0,
        visible_when_operator = '',
        visible_when_value = '',
        sort = 20,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = ala_form_id
      AND data_field_id = ala_r_value_field_id;

    UPDATE gjj_crm_form
    SET description = 'ALA填写评估租金和评估R值后提交运营条件审核。',
        updated_at = CURRENT_TIMESTAMP
    WHERE id = ala_form_id;
END $$;

DO $$
DECLARE
    contract_template_id BIGINT;
    contract_template_cate_id BIGINT;
    contract_form_id BIGINT;
    payment_option_set_id BIGINT;
    share_option_set_id BIGINT;
    contract_combination_field_id BIGINT;
    payment_method_field_id BIGINT;
BEGIN
    SELECT id, cate_id
    INTO contract_template_id, contract_template_cate_id
    FROM gjj_crm_data_template
    WHERE name = '合同信息'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO contract_form_id
    FROM gjj_crm_form
    WHERE name = '合同签署登记'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO contract_combination_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'contract_combination_id'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO share_option_set_id
    FROM gjj_crm_option_set
    WHERE name = '房东分成百分比'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    IF contract_template_id IS NULL
       OR contract_form_id IS NULL
       OR contract_combination_field_id IS NULL
       OR share_option_set_id IS NULL THEN
        RAISE EXCEPTION '合同字段调整所需的合同模板、表单、合同组合或分成选项集未完整启用';
    END IF;

    SELECT id INTO payment_option_set_id
    FROM gjj_crm_option_set
    WHERE name = '合同支付方式'
    ORDER BY id
    LIMIT 1;

    IF payment_option_set_id IS NULL THEN
        INSERT INTO gjj_crm_option_set (
            name, sort, status, created_at, updated_at
        ) VALUES (
            '合同支付方式', 66, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO payment_option_set_id;
    ELSE
        UPDATE gjj_crm_option_set
        SET sort = 66,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = payment_option_set_id;
    END IF;

    INSERT INTO gjj_crm_option_set_item (
        option_set_id, name, value, sort, status
    )
    SELECT payment_option_set_id, seed.name, seed.value, seed.sort, 1
    FROM (
        VALUES
            ('一次性支付', 'one_time', 10),
            ('分两期支付', 'two_installments', 20),
            ('三方代付', 'third_party', 30)
    ) AS seed(name, value, sort)
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_option_set_item AS existing
        WHERE existing.option_set_id = payment_option_set_id
          AND existing.value = seed.value
    );

    UPDATE gjj_crm_option_set_item AS item
    SET name = seed.name,
        sort = seed.sort,
        status = 1
    FROM (
        VALUES
            ('一次性支付', 'one_time', 10),
            ('分两期支付', 'two_installments', 20),
            ('三方代付', 'third_party', 30)
    ) AS seed(name, value, sort)
    WHERE item.option_set_id = payment_option_set_id
      AND item.value = seed.value;

    INSERT INTO gjj_crm_data_field (
        data_template_id, parent_field_id, option_set_id,
        name, field_key, field_type, default_value,
        finance_type_id, stat_enabled, sort, status,
        created_at, updated_at
    )
    SELECT
        contract_template_id,
        0,
        CASE WHEN seed.use_share_options THEN share_option_set_id ELSE 0 END,
        seed.name,
        seed.field_key,
        seed.field_type,
        '',
        0,
        FALSE,
        seed.sort,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('payment_first_date', '首款日期', 'date', FALSE, 200),
            ('payment_first_amount', '首款金额', 'money', FALSE, 210),
            ('payment_final_date', '尾款日期', 'date', FALSE, 220),
            ('payment_final_amount', '尾款金额', 'money', FALSE, 230),
            ('sole_housing_subsidy_rate', '唯一住房补贴比例', 'number', FALSE, 240),
            ('custody_phase_one_start_at', '托管运营期第一阶段开始时间', 'date', FALSE, 250),
            ('custody_phase_one_end_at', '托管运营期第一阶段结束时间', 'date', FALSE, 260),
            ('custody_phase_one_landlord_share_rate', '托管运营期第一阶段房东分成百分比', 'select', TRUE, 270),
            ('custody_phase_two_start_at', '托管运营期第二阶段开始时间', 'date', FALSE, 280),
            ('custody_phase_two_end_at', '托管运营期第二阶段结束时间', 'date', FALSE, 290),
            ('custody_phase_two_landlord_share_rate', '托管运营期第二阶段房东分成百分比', 'select', TRUE, 300)
    ) AS seed(field_key, name, field_type, use_share_options, sort)
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_data_field AS existing
        WHERE existing.field_key = seed.field_key
    );

    UPDATE gjj_crm_data_field AS field
    SET data_template_id = contract_template_id,
        parent_field_id = 0,
        option_set_id = CASE WHEN seed.use_share_options THEN share_option_set_id ELSE 0 END,
        name = seed.name,
        field_type = seed.field_type,
        default_value = '',
        finance_type_id = 0,
        sort = seed.sort,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('payment_first_date', '首款日期', 'date', FALSE, 200),
            ('payment_first_amount', '首款金额', 'money', FALSE, 210),
            ('payment_final_date', '尾款日期', 'date', FALSE, 220),
            ('payment_final_amount', '尾款金额', 'money', FALSE, 230),
            ('sole_housing_subsidy_rate', '唯一住房补贴比例', 'number', FALSE, 240),
            ('custody_phase_one_start_at', '托管运营期第一阶段开始时间', 'date', FALSE, 250),
            ('custody_phase_one_end_at', '托管运营期第一阶段结束时间', 'date', FALSE, 260),
            ('custody_phase_one_landlord_share_rate', '托管运营期第一阶段房东分成百分比', 'select', TRUE, 270),
            ('custody_phase_two_start_at', '托管运营期第二阶段开始时间', 'date', FALSE, 280),
            ('custody_phase_two_end_at', '托管运营期第二阶段结束时间', 'date', FALSE, 290),
            ('custody_phase_two_landlord_share_rate', '托管运营期第二阶段房东分成百分比', 'select', TRUE, 300)
    ) AS seed(field_key, name, field_type, use_share_options, sort)
    WHERE field.field_key = seed.field_key;

    SELECT id INTO payment_method_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'payment_method'
    ORDER BY id
    LIMIT 1;

    IF payment_method_field_id IS NULL THEN
        RAISE EXCEPTION '合同支付方式字段不存在';
    END IF;

    UPDATE gjj_crm_data_field
    SET data_template_id = contract_template_id,
        parent_field_id = 0,
        option_set_id = payment_option_set_id,
        name = '支付方式',
        field_type = 'select',
        default_value = '',
        sort = 110,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = payment_method_field_id;

    INSERT INTO gjj_crm_form_field (
        form_id, data_template_cate_id, data_template_id,
        field_source, field_path, main_field, data_field_id,
        name, required, readonly,
        visible_when_field_id, visible_when_operator, visible_when_value,
        sort, status, created_at, updated_at
    )
    SELECT
        contract_form_id,
        contract_template_cate_id,
        contract_template_id,
        'data:' || field.id,
        json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || field.id
        )::TEXT,
        '',
        field.id,
        field.name,
        TRUE,
        FALSE,
        CASE
            WHEN field.field_key LIKE 'payment_%'
              OR field.field_key = 'sole_housing_subsidy_rate'
                THEN payment_method_field_id
            ELSE contract_combination_field_id
        END,
        'equals',
        CASE
            WHEN field.field_key LIKE 'payment_%'
              OR field.field_key = 'sole_housing_subsidy_rate'
                THEN 'two_installments'
            ELSE 'NS-MAIN-PACKAGE-20260704'
        END,
        CASE field.field_key
            WHEN 'payment_first_date' THEN 80
            WHEN 'payment_first_amount' THEN 90
            WHEN 'payment_final_date' THEN 100
            WHEN 'payment_final_amount' THEN 110
            WHEN 'sole_housing_subsidy_rate' THEN 120
            WHEN 'custody_phase_one_start_at' THEN 170
            WHEN 'custody_phase_one_end_at' THEN 180
            WHEN 'custody_phase_one_landlord_share_rate' THEN 190
            WHEN 'custody_phase_two_start_at' THEN 200
            WHEN 'custody_phase_two_end_at' THEN 210
            WHEN 'custody_phase_two_landlord_share_rate' THEN 220
        END,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE field.field_key = ANY (ARRAY[
        'payment_first_date',
        'payment_first_amount',
        'payment_final_date',
        'payment_final_amount',
        'sole_housing_subsidy_rate',
        'custody_phase_one_start_at',
        'custody_phase_one_end_at',
        'custody_phase_one_landlord_share_rate',
        'custody_phase_two_start_at',
        'custody_phase_two_end_at',
        'custody_phase_two_landlord_share_rate'
    ])
      AND NOT EXISTS (
          SELECT 1
          FROM gjj_crm_form_field AS existing
          WHERE existing.form_id = contract_form_id
            AND existing.data_field_id = field.id
      );

    UPDATE gjj_crm_form_field AS form_field
    SET data_template_cate_id = contract_template_cate_id,
        data_template_id = contract_template_id,
        field_source = 'data:' || field.id,
        field_path = json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || field.id
        )::TEXT,
        main_field = '',
        name = field.name,
        required = TRUE,
        readonly = FALSE,
        visible_when_field_id = CASE
            WHEN field.field_key LIKE 'payment_%'
              OR field.field_key = 'sole_housing_subsidy_rate'
                THEN payment_method_field_id
            ELSE contract_combination_field_id
        END,
        visible_when_operator = 'equals',
        visible_when_value = CASE
            WHEN field.field_key LIKE 'payment_%'
              OR field.field_key = 'sole_housing_subsidy_rate'
                THEN 'two_installments'
            ELSE 'NS-MAIN-PACKAGE-20260704'
        END,
        sort = CASE field.field_key
            WHEN 'payment_first_date' THEN 80
            WHEN 'payment_first_amount' THEN 90
            WHEN 'payment_final_date' THEN 100
            WHEN 'payment_final_amount' THEN 110
            WHEN 'sole_housing_subsidy_rate' THEN 120
            WHEN 'custody_phase_one_start_at' THEN 170
            WHEN 'custody_phase_one_end_at' THEN 180
            WHEN 'custody_phase_one_landlord_share_rate' THEN 190
            WHEN 'custody_phase_two_start_at' THEN 200
            WHEN 'custody_phase_two_end_at' THEN 210
            WHEN 'custody_phase_two_landlord_share_rate' THEN 220
        END,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE form_field.form_id = contract_form_id
      AND form_field.data_field_id = field.id
      AND field.field_key = ANY (ARRAY[
          'payment_first_date',
          'payment_first_amount',
          'payment_final_date',
          'payment_final_amount',
          'sole_housing_subsidy_rate',
          'custody_phase_one_start_at',
          'custody_phase_one_end_at',
          'custody_phase_one_landlord_share_rate',
          'custody_phase_two_start_at',
          'custody_phase_two_end_at',
          'custody_phase_two_landlord_share_rate'
      ]);

    UPDATE gjj_crm_form_field
    SET required = TRUE,
        readonly = FALSE,
        visible_when_field_id = contract_combination_field_id,
        visible_when_operator = 'equals',
        visible_when_value = 'SS-SERVICE-PACKAGE-20260704',
        sort = 70,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = contract_form_id
      AND data_field_id = payment_method_field_id;

    -- Disable only current form references. Data fields, historical records and
    -- finance ledgers remain available; the newer moving_subsidy_amount field
    -- replaces the legacy htxx.banjiafeibuzhu form reference.
    UPDATE gjj_crm_form_field AS form_field
    SET status = 2,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE form_field.form_id = contract_form_id
      AND form_field.data_field_id = field.id
      AND (
          field.field_key IN (
              'third_party_cost',
              'lawyer_fee',
              'legal_fee',
              'htxx.banjiafeibuzhu',
              'custody_operation_start_at',
              'custody_operation_end_at',
              'landlord_share_rate'
          )
          OR field.name = '律师费'
      );

    UPDATE gjj_crm_form_field AS form_field
    SET sort = CASE field.field_key
            WHEN 'deposit_amount' THEN 130
            WHEN 'expected_handover_at' THEN 140
            WHEN 'htxx.yufuzujin' THEN 150
            WHEN 'moving_subsidy_amount' THEN 160
            ELSE form_field.sort
        END,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE form_field.form_id = contract_form_id
      AND form_field.data_field_id = field.id
      AND field.field_key IN (
          'deposit_amount',
          'expected_handover_at',
          'htxx.yufuzujin',
          'moving_subsidy_amount'
      );

    UPDATE gjj_crm_form
    SET description = '按合同组合展示查封服务支付信息或非查封托管分阶段信息。',
        updated_at = CURRENT_TIMESTAMP
    WHERE id = contract_form_id;
END $$;

COMMIT;

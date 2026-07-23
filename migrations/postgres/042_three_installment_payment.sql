BEGIN;

DO $$
DECLARE
    contract_template_id BIGINT;
    contract_template_cate_id BIGINT;
    contract_form_id BIGINT;
    payment_option_set_id BIGINT;
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

    SELECT id INTO payment_option_set_id
    FROM gjj_crm_option_set
    WHERE name = '合同支付方式'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO payment_method_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'payment_method'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    IF contract_template_id IS NULL
       OR contract_form_id IS NULL
       OR payment_option_set_id IS NULL
       OR payment_method_field_id IS NULL THEN
        RAISE EXCEPTION '三期支付配置所需的合同模板、表单、支付方式选项集或字段未完整启用';
    END IF;

    INSERT INTO gjj_crm_option_set_item (
        option_set_id, name, value, sort, status
    )
    SELECT payment_option_set_id, '分三期支付', 'three_installments', 30, 1
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_option_set_item AS existing
        WHERE existing.option_set_id = payment_option_set_id
          AND existing.value = 'three_installments'
    );

    UPDATE gjj_crm_option_set_item
    SET name = '分三期支付',
        sort = 30,
        status = 1
    WHERE option_set_id = payment_option_set_id
      AND value = 'three_installments';

    UPDATE gjj_crm_option_set_item
    SET sort = 40
    WHERE option_set_id = payment_option_set_id
      AND value = 'third_party';

    INSERT INTO gjj_crm_data_field (
        data_template_id, parent_field_id, option_set_id,
        name, field_key, field_type, default_value,
        finance_type_id, stat_enabled, sort, status,
        created_at, updated_at
    )
    SELECT
        contract_template_id, 0, 0,
        seed.name, seed.field_key, seed.field_type, '',
        0, FALSE, seed.sort, 1,
        CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('payment_middle_date', '中款日期', 'date', 220),
            ('payment_middle_amount', '中款金额', 'money', 230)
    ) AS seed(field_key, name, field_type, sort)
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_data_field AS existing
        WHERE existing.field_key = seed.field_key
    );

    UPDATE gjj_crm_data_field AS field
    SET data_template_id = contract_template_id,
        parent_field_id = 0,
        option_set_id = 0,
        name = seed.name,
        field_type = seed.field_type,
        default_value = '',
        finance_type_id = 0,
        stat_enabled = FALSE,
        sort = seed.sort,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('payment_first_date', '首款日期', 'date', 200),
            ('payment_first_amount', '首款金额', 'money', 210),
            ('payment_middle_date', '中款日期', 'date', 220),
            ('payment_middle_amount', '中款金额', 'money', 230),
            ('payment_final_date', '尾款日期', 'date', 240),
            ('payment_final_amount', '尾款金额', 'money', 250)
    ) AS seed(field_key, name, field_type, sort)
    WHERE field.field_key = seed.field_key;

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
        payment_method_field_id,
        'equals',
        'three_installments',
        CASE field.field_key
            WHEN 'payment_middle_date' THEN 100
            WHEN 'payment_middle_amount' THEN 110
        END,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE field.field_key IN ('payment_middle_date', 'payment_middle_amount')
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
        visible_when_field_id = payment_method_field_id,
        visible_when_operator = CASE
            WHEN field.field_key IN ('payment_middle_date', 'payment_middle_amount') THEN 'equals'
            ELSE 'in'
        END,
        visible_when_value = CASE
            WHEN field.field_key IN ('payment_middle_date', 'payment_middle_amount') THEN 'three_installments'
            ELSE 'two_installments,three_installments'
        END,
        sort = CASE field.field_key
            WHEN 'payment_first_date' THEN 80
            WHEN 'payment_first_amount' THEN 90
            WHEN 'payment_middle_date' THEN 100
            WHEN 'payment_middle_amount' THEN 110
            WHEN 'payment_final_date' THEN 120
            WHEN 'payment_final_amount' THEN 130
        END,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE form_field.form_id = contract_form_id
      AND form_field.data_field_id = field.id
      AND field.field_key IN (
          'payment_first_date',
          'payment_first_amount',
          'payment_middle_date',
          'payment_middle_amount',
          'payment_final_date',
          'payment_final_amount'
      );

    UPDATE gjj_crm_form_field AS form_field
    SET sort = 140,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS field
    WHERE form_field.form_id = contract_form_id
      AND form_field.data_field_id = field.id
      AND field.field_key = 'sole_housing_subsidy_rate';
END $$;

COMMIT;

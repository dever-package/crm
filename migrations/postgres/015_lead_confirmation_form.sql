-- Add a configurable confirmation form to the default lead workflow.
-- Lead creation remains a separate entry action; this form gates conversion.
BEGIN;

DO $$
DECLARE
    lead_form_id BIGINT;
    lead_workflow_id BIGINT;
    lead_stage_id BIGINT;
    lead_task_id BIGINT;
BEGIN
    SELECT id INTO lead_form_id
    FROM gjj_crm_form
    WHERE name = '线索确认'
    ORDER BY id
    LIMIT 1;

    IF lead_form_id IS NULL THEN
        INSERT INTO gjj_crm_form (
            name, description, sort, status, created_at, updated_at
        ) VALUES (
            '线索确认',
            'MKT补充并确认线索信息；完成后方可转为客户。',
            5,
            1,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO lead_form_id;
    ELSE
        UPDATE gjj_crm_form
        SET description = 'MKT补充并确认线索信息；完成后方可转为客户。',
            sort = 5,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = lead_form_id;
    END IF;

    DELETE FROM gjj_crm_form_field
    WHERE form_id = lead_form_id;

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
        lead_form_id,
        4,
        0,
        'main:4:' || seed.field_key,
        '["cate:4","main_table:4","main:4:' || seed.field_key || '"]',
        seed.field_key,
        0,
        seed.field_name,
        seed.required,
        FALSE,
        seed.sort,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('name', '姓名', TRUE, 10),
            ('phone', '手机号', FALSE, 20),
            ('wechat', '微信号', FALSE, 30),
            ('source_id', '来源', FALSE, 40),
            ('channel_id', '渠道', FALSE, 50),
            ('external_id', '外部线索ID', FALSE, 60),
            ('city', '城市', FALSE, 70),
            ('initial_need', '初始诉求', FALSE, 80)
    ) AS seed(field_key, field_name, required, sort);

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
        lead_form_id,
        template.cate_id,
        template.id,
        'data:' || field.id,
        '["cate:4","template:' || template.id || '","data:' || field.id || '"]',
        '',
        field.id,
        field.name,
        FALSE,
        FALSE,
        1000 + template.sort * 100 + field.sort,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM gjj_crm_data_template AS template
    INNER JOIN gjj_crm_data_field AS field
        ON field.data_template_id = template.id
       AND field.status = 1
       AND field.field_type <> 'group'
    WHERE template.cate_id = 4
      AND template.status = 1
    ORDER BY template.sort, template.id, field.sort, field.id;

    SELECT id INTO lead_workflow_id
    FROM gjj_crm_workflow
    WHERE subject_type = 'lead'
      AND status = 1
    ORDER BY default_entry DESC, sort, id
    LIMIT 1;

    IF lead_workflow_id IS NULL THEN
        RAISE EXCEPTION '缺少启用的线索流程';
    END IF;

    SELECT id INTO lead_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = lead_workflow_id
      AND status = 1
    ORDER BY sort, id
    LIMIT 1;

    IF lead_stage_id IS NULL THEN
        RAISE EXCEPTION '默认线索流程缺少启用阶段';
    END IF;

    SELECT id INTO lead_task_id
    FROM gjj_crm_task
    WHERE stage_id = lead_stage_id
      AND name IN ('确认线索', '确认线索信息')
    ORDER BY CASE WHEN name = '确认线索' THEN 0 ELSE 1 END, sort, id
    LIMIT 1;

    IF lead_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id,
            name,
            task_type,
            required,
            assignee_mode,
            assignee_department_id,
            form_id,
            script_id,
            due_days,
            sort,
            status,
            created_at,
            updated_at
        ) VALUES (
            lead_stage_id,
            '确认线索',
            'form',
            TRUE,
            'stage',
            0,
            lead_form_id,
            0,
            0,
            10,
            1,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        );
    ELSE
        UPDATE gjj_crm_task
        SET name = '确认线索',
            task_type = 'form',
            required = TRUE,
            assignee_mode = 'stage',
            assignee_department_id = 0,
            form_id = lead_form_id,
            script_id = 0,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = lead_task_id;
    END IF;
END $$;

COMMIT;

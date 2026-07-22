-- Complete approval rejection loops and add the task-specific business fields.
BEGIN;

DO $$
DECLARE
    signing_workflow_id BIGINT;
    diagnosis_stage_id BIGINT;
    collaboration_stage_id BIGINT;
    npl_department_id BIGINT;
    contract_template_id BIGINT;
    contract_template_cate_id BIGINT;
    eleven_form_id BIGINT;
    diagnosis_task_id BIGINT;
    npl_correction_task_id BIGINT;
    pm_interview_task_id BIGINT;
    ala_task_id BIGINT;
    pm_correction_task_id BIGINT;
    pm_interview_form_id BIGINT;
    ala_form_id BIGINT;
    pm_interview_attachment_field_id BIGINT;
    ala_assessed_rent_field_id BIGINT;
BEGIN
    SELECT id INTO signing_workflow_id
    FROM gjj_crm_workflow
    WHERE subject_type = 'customer_asset'
      AND default_entry = TRUE
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO diagnosis_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = signing_workflow_id
      AND name = '诊断核验'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO collaboration_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = signing_workflow_id
      AND name = '签约协作'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO npl_department_id
    FROM gjj_crm_department
    WHERE code = 'NPL'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id, cate_id
    INTO contract_template_id, contract_template_cate_id
    FROM gjj_crm_data_template
    WHERE name = '合同信息'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO eleven_form_id
    FROM gjj_crm_form
    WHERE name = '十一维资料收集'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO diagnosis_task_id
    FROM gjj_crm_task
    WHERE stage_id = diagnosis_stage_id
      AND name = '确认诊断结果'
      AND task_type = 'approval'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO pm_interview_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'PM谈话笔录'
      AND task_type = 'approval'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO ala_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'ALA运营条件确认'
      AND task_type = 'approval'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO pm_correction_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'PM补充签约资料'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    IF signing_workflow_id IS NULL
       OR diagnosis_stage_id IS NULL
       OR collaboration_stage_id IS NULL
       OR npl_department_id IS NULL
       OR contract_template_id IS NULL
       OR eleven_form_id IS NULL
       OR diagnosis_task_id IS NULL
       OR pm_interview_task_id IS NULL
       OR ala_task_id IS NULL
       OR pm_correction_task_id IS NULL THEN
        RAISE EXCEPTION '审批回流所需的流程、部门、模板、表单或任务未完整启用';
    END IF;

    SELECT id INTO pm_interview_form_id
    FROM gjj_crm_form
    WHERE name = 'PM谈话笔录'
    ORDER BY id
    LIMIT 1;

    IF pm_interview_form_id IS NULL THEN
        INSERT INTO gjj_crm_form (
            name, description, sort, status, created_at, updated_at
        ) VALUES (
            'PM谈话笔录',
            'PM上传本次谈话笔录附件后提交审核结果。',
            45,
            1,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO pm_interview_form_id;
    ELSE
        UPDATE gjj_crm_form
        SET description = 'PM上传本次谈话笔录附件后提交审核结果。',
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = pm_interview_form_id;
    END IF;

    SELECT id INTO ala_form_id
    FROM gjj_crm_form
    WHERE name = 'ALA运营条件确认'
    ORDER BY id
    LIMIT 1;

    IF ala_form_id IS NULL THEN
        INSERT INTO gjj_crm_form (
            name, description, sort, status, created_at, updated_at
        ) VALUES (
            'ALA运营条件确认',
            'ALA填写评估租金后提交运营条件审核。',
            46,
            1,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO ala_form_id;
    ELSE
        UPDATE gjj_crm_form
        SET description = 'ALA填写评估租金后提交运营条件审核。',
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = ala_form_id;
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
        0,
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
            ('pm_interview_attachment', '谈话笔录附件', 'attachment', 170),
            ('ala_assessed_rent', '评估租金', 'money', 180)
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
        sort = seed.sort,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM (
        VALUES
            ('pm_interview_attachment', '谈话笔录附件', 'attachment', 170),
            ('ala_assessed_rent', '评估租金', 'money', 180)
    ) AS seed(field_key, name, field_type, sort)
    WHERE field.field_key = seed.field_key;

    SELECT id INTO pm_interview_attachment_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'pm_interview_attachment'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO ala_assessed_rent_field_id
    FROM gjj_crm_data_field
    WHERE field_key = 'ala_assessed_rent'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    IF pm_interview_attachment_field_id IS NULL OR ala_assessed_rent_field_id IS NULL THEN
        RAISE EXCEPTION '谈话笔录附件或评估租金字段创建失败';
    END IF;

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
        pm_interview_form_id,
        contract_template_cate_id,
        contract_template_id,
        'data:' || pm_interview_attachment_field_id,
        json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || pm_interview_attachment_field_id
        )::TEXT,
        '',
        pm_interview_attachment_field_id,
        '上传附件',
        TRUE,
        FALSE,
        10,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS existing
        WHERE existing.form_id = pm_interview_form_id
          AND existing.data_field_id = pm_interview_attachment_field_id
    );

    UPDATE gjj_crm_form_field
    SET data_template_cate_id = contract_template_cate_id,
        data_template_id = contract_template_id,
        field_source = 'data:' || pm_interview_attachment_field_id,
        field_path = json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || pm_interview_attachment_field_id
        )::TEXT,
        main_field = '',
        name = '上传附件',
        required = TRUE,
        readonly = FALSE,
        sort = 10,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = pm_interview_form_id
      AND data_field_id = pm_interview_attachment_field_id;

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
        ala_form_id,
        contract_template_cate_id,
        contract_template_id,
        'data:' || ala_assessed_rent_field_id,
        json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || ala_assessed_rent_field_id
        )::TEXT,
        '',
        ala_assessed_rent_field_id,
        '评估租金（元）',
        TRUE,
        FALSE,
        10,
        1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS existing
        WHERE existing.form_id = ala_form_id
          AND existing.data_field_id = ala_assessed_rent_field_id
    );

    UPDATE gjj_crm_form_field
    SET data_template_cate_id = contract_template_cate_id,
        data_template_id = contract_template_id,
        field_source = 'data:' || ala_assessed_rent_field_id,
        field_path = json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || ala_assessed_rent_field_id
        )::TEXT,
        main_field = '',
        name = '评估租金（元）',
        required = TRUE,
        readonly = FALSE,
        sort = 10,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = ala_form_id
      AND data_field_id = ala_assessed_rent_field_id;

    UPDATE gjj_crm_task
    SET form_id = pm_interview_form_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = pm_interview_task_id;

    UPDATE gjj_crm_task
    SET form_id = ala_form_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = ala_task_id;

    SELECT id INTO npl_correction_task_id
    FROM gjj_crm_task
    WHERE stage_id = diagnosis_stage_id
      AND name = 'NPL补充诊断资料'
    ORDER BY id
    LIMIT 1;

    IF npl_correction_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id,
            name,
            task_type,
            required,
            assignee_mode,
            assignee_department_id,
            form_id,
            script_id,
            activation_mode,
            condition_script_id,
            reject_target_task_id,
            complete_target_task_id,
            due_days,
            sort,
            status,
            created_at,
            updated_at
        ) VALUES (
            diagnosis_stage_id,
            'NPL补充诊断资料',
            'form',
            TRUE,
            'previous',
            npl_department_id,
            eleven_form_id,
            0,
            'route',
            0,
            0,
            diagnosis_task_id,
            0,
            30,
            1,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO npl_correction_task_id;
    ELSE
        UPDATE gjj_crm_task
        SET task_type = 'form',
            required = TRUE,
            assignee_mode = 'previous',
            assignee_department_id = npl_department_id,
            form_id = eleven_form_id,
            script_id = 0,
            activation_mode = 'route',
            condition_script_id = 0,
            reject_target_task_id = 0,
            complete_target_task_id = diagnosis_task_id,
            due_days = 0,
            sort = 30,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = npl_correction_task_id;
    END IF;

    UPDATE gjj_crm_task
    SET activation_mode = 'stage',
        reject_target_task_id = npl_correction_task_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = diagnosis_task_id;

    UPDATE gjj_crm_task
    SET activation_mode = 'route',
        condition_script_id = 0,
        complete_target_task_id = ala_task_id,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = pm_correction_task_id;

    UPDATE gjj_crm_task
    SET reject_target_task_id = pm_correction_task_id,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = ala_task_id;

    -- Repair active diagnosis approvals whose last action was a rejection before
    -- the correction route existed. Prefer the previous NPL assignee.
    INSERT INTO gjj_crm_task_todo (
        lead_id,
        customer_id,
        asset_id,
        workflow_instance_id,
        customer_product_id,
        workflow_id,
        stage_id,
        task_id,
        assignee_department_id,
        assignee_staff_id,
        required,
        status,
        due_at,
        result,
        completed_at,
        created_at,
        updated_at
    )
    SELECT
        source_todo.lead_id,
        source_todo.customer_id,
        source_todo.asset_id,
        source_todo.workflow_instance_id,
        source_todo.customer_product_id,
        source_todo.workflow_id,
        source_todo.stage_id,
        npl_correction_task_id,
        npl_department_id,
        COALESCE(previous_npl.assignee_staff_id, fallback_npl.id, 0),
        TRUE,
        'pending',
        NULL,
        '',
        NULL,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM gjj_crm_task_todo AS source_todo
    JOIN gjj_crm_workflow_instance AS instance
      ON instance.id = source_todo.workflow_instance_id
     AND instance.stage_id = source_todo.stage_id
     AND instance.status = 'active'
    JOIN LATERAL (
        SELECT operation.result_value, operation.content
        FROM gjj_crm_operation_log AS operation
        WHERE operation.workflow_instance_id = source_todo.workflow_instance_id
          AND operation.task_id = source_todo.task_id
        ORDER BY operation.id DESC
        LIMIT 1
    ) AS latest_operation ON latest_operation.result_value = 'rejected'
    LEFT JOIN LATERAL (
        SELECT previous_todo.assignee_staff_id
        FROM gjj_crm_task_todo AS previous_todo
        JOIN gjj_crm_staff AS previous_staff
          ON previous_staff.id = previous_todo.assignee_staff_id
         AND previous_staff.department_id = npl_department_id
         AND previous_staff.status = 1
        WHERE previous_todo.workflow_instance_id = source_todo.workflow_instance_id
          AND previous_todo.assignee_department_id = npl_department_id
          AND previous_todo.assignee_staff_id > 0
        ORDER BY previous_todo.updated_at DESC, previous_todo.id DESC
        LIMIT 1
    ) AS previous_npl ON TRUE
    LEFT JOIN LATERAL (
        SELECT staff.id
        FROM gjj_crm_staff AS staff
        WHERE staff.department_id = npl_department_id
          AND staff.status = 1
        ORDER BY staff.id
        LIMIT 1
    ) AS fallback_npl ON TRUE
    WHERE source_todo.task_id = diagnosis_task_id
      AND source_todo.status = 'pending'
    ON CONFLICT (workflow_instance_id, stage_id, task_id) DO UPDATE SET
        assignee_department_id = EXCLUDED.assignee_department_id,
        assignee_staff_id = EXCLUDED.assignee_staff_id,
        required = TRUE,
        status = 'pending',
        due_at = NULL,
        result = '',
        completed_at = NULL,
        updated_at = CURRENT_TIMESTAMP;

    WITH rejected_pending AS (
        SELECT source_todo.id, latest_operation.content
        FROM gjj_crm_task_todo AS source_todo
        JOIN LATERAL (
            SELECT operation.result_value, operation.content
            FROM gjj_crm_operation_log AS operation
            WHERE operation.workflow_instance_id = source_todo.workflow_instance_id
              AND operation.task_id = source_todo.task_id
            ORDER BY operation.id DESC
            LIMIT 1
        ) AS latest_operation ON latest_operation.result_value = 'rejected'
        WHERE source_todo.task_id = diagnosis_task_id
          AND source_todo.status = 'pending'
          AND EXISTS (
              SELECT 1
              FROM gjj_crm_workflow_instance AS instance
              WHERE instance.id = source_todo.workflow_instance_id
                AND instance.stage_id = source_todo.stage_id
                AND instance.status = 'active'
          )
          AND EXISTS (
              SELECT 1
              FROM gjj_crm_task_todo AS correction_todo
              WHERE correction_todo.workflow_instance_id = source_todo.workflow_instance_id
                AND correction_todo.stage_id = source_todo.stage_id
                AND correction_todo.task_id = npl_correction_task_id
                AND correction_todo.status = 'pending'
          )
    )
    UPDATE gjj_crm_task_todo AS source_todo
    SET status = 'done',
        result = '审核驳回：' || rejected_pending.content,
        completed_at = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    FROM rejected_pending
    WHERE source_todo.id = rejected_pending.id;

    -- Apply the same repair to any ALA rejection that predates its PM route.
    INSERT INTO gjj_crm_task_todo (
        lead_id,
        customer_id,
        asset_id,
        workflow_instance_id,
        customer_product_id,
        workflow_id,
        stage_id,
        task_id,
        assignee_department_id,
        assignee_staff_id,
        required,
        status,
        due_at,
        result,
        completed_at,
        created_at,
        updated_at
    )
    SELECT
        source_todo.lead_id,
        source_todo.customer_id,
        source_todo.asset_id,
        source_todo.workflow_instance_id,
        source_todo.customer_product_id,
        source_todo.workflow_id,
        source_todo.stage_id,
        pm_correction_task_id,
        instance.owner_department_id,
        instance.owner_staff_id,
        TRUE,
        'pending',
        NULL,
        '',
        NULL,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    FROM gjj_crm_task_todo AS source_todo
    JOIN gjj_crm_workflow_instance AS instance
      ON instance.id = source_todo.workflow_instance_id
     AND instance.stage_id = source_todo.stage_id
     AND instance.status = 'active'
    JOIN LATERAL (
        SELECT operation.result_value
        FROM gjj_crm_operation_log AS operation
        WHERE operation.workflow_instance_id = source_todo.workflow_instance_id
          AND operation.task_id = source_todo.task_id
        ORDER BY operation.id DESC
        LIMIT 1
    ) AS latest_operation ON latest_operation.result_value = 'rejected'
    WHERE source_todo.task_id = ala_task_id
      AND source_todo.status = 'pending'
    ON CONFLICT (workflow_instance_id, stage_id, task_id) DO UPDATE SET
        assignee_department_id = EXCLUDED.assignee_department_id,
        assignee_staff_id = EXCLUDED.assignee_staff_id,
        required = TRUE,
        status = 'pending',
        due_at = NULL,
        result = '',
        completed_at = NULL,
        updated_at = CURRENT_TIMESTAMP;

    WITH rejected_pending AS (
        SELECT source_todo.id, latest_operation.content
        FROM gjj_crm_task_todo AS source_todo
        JOIN LATERAL (
            SELECT operation.result_value, operation.content
            FROM gjj_crm_operation_log AS operation
            WHERE operation.workflow_instance_id = source_todo.workflow_instance_id
              AND operation.task_id = source_todo.task_id
            ORDER BY operation.id DESC
            LIMIT 1
        ) AS latest_operation ON latest_operation.result_value = 'rejected'
        WHERE source_todo.task_id = ala_task_id
          AND source_todo.status = 'pending'
          AND EXISTS (
              SELECT 1
              FROM gjj_crm_workflow_instance AS instance
              WHERE instance.id = source_todo.workflow_instance_id
                AND instance.stage_id = source_todo.stage_id
                AND instance.status = 'active'
          )
          AND EXISTS (
              SELECT 1
              FROM gjj_crm_task_todo AS correction_todo
              WHERE correction_todo.workflow_instance_id = source_todo.workflow_instance_id
                AND correction_todo.stage_id = source_todo.stage_id
                AND correction_todo.task_id = pm_correction_task_id
                AND correction_todo.status = 'pending'
          )
    )
    UPDATE gjj_crm_task_todo AS source_todo
    SET status = 'done',
        result = '审核驳回：' || rejected_pending.content,
        completed_at = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    FROM rejected_pending
    WHERE source_todo.id = rejected_pending.id;
END $$;

COMMIT;

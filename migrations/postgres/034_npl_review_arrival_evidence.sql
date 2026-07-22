BEGIN;

ALTER TABLE gjj_crm_task
    ADD COLUMN IF NOT EXISTS reject_action VARCHAR(32) NOT NULL DEFAULT 'stay',
    ADD COLUMN IF NOT EXISTS opinion_requirement VARCHAR(32) NOT NULL DEFAULT 'reject_required',
    ADD COLUMN IF NOT EXISTS reject_submit_form BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE gjj_crm_task
SET reject_action = 'route',
    updated_at = CURRENT_TIMESTAMP
WHERE reject_target_task_id > 0
  AND reject_action = 'stay';

ALTER TABLE gjj_crm_schedule_event
    ADD COLUMN IF NOT EXISTS meeting_attempt INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS arrival_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS arrival_confirmed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS arrival_confirmed_by_staff_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS no_show_reason TEXT NOT NULL DEFAULT '';

UPDATE gjj_crm_schedule_event
SET meeting_attempt = 1
WHERE meeting_attempt < 1;

UPDATE gjj_crm_schedule_event
SET arrival_status = 'arrived',
    arrival_confirmed_at = COALESCE(arrival_confirmed_at, customer_arrived_at),
    arrival_confirmed_by_staff_id = CASE
        WHEN arrival_confirmed_by_staff_id > 0 THEN arrival_confirmed_by_staff_id
        ELSE customer_arrived_by_staff_id
    END
WHERE customer_arrived_at IS NOT NULL
  AND arrival_status = 'pending';

UPDATE gjj_crm_schedule_event
SET status = 'completed',
    completed_at = COALESCE(completed_at, customer_arrived_at),
    updated_at = CURRENT_TIMESTAMP
WHERE customer_arrived_at IS NOT NULL
  AND status = 'pending';

UPDATE gjj_crm_public_resource_booking AS booking
SET booking_status = 'done',
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_schedule_event AS event
WHERE event.id = booking.schedule_event_id
  AND event.customer_arrived_at IS NOT NULL
  AND booking.booking_status NOT IN ('canceled', 'rejected', 'done');

CREATE INDEX IF NOT EXISTS idx_gjj_crm_schedule_event_source_task_attempt
    ON gjj_crm_schedule_event (source_workflow_instance_id, source_task_id, meeting_attempt, id);

ALTER TABLE gjj_crm_attachment
    ADD COLUMN IF NOT EXISTS schedule_event_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS upload_file_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS usage VARCHAR(32) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_gjj_crm_attachment_schedule_usage
    ON gjj_crm_attachment (schedule_event_id, usage, created_at, id);

DO $$
DECLARE
    signing_workflow_id BIGINT;
    diagnosis_stage_id BIGINT;
    collaboration_stage_id BIGINT;
    npl_department_id BIGINT;
    administrative_department_id BIGINT;
    diagnosis_task_id BIGINT;
    pm_interview_task_id BIGINT;
    ala_task_id BIGINT;
    diagnosis_review_task_id BIGINT;
    pm_review_task_id BIGINT;
    ala_review_task_id BIGINT;
    legacy_diagnosis_correction_task_id BIGINT;
    legacy_pm_correction_task_id BIGINT;
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

    SELECT id INTO administrative_department_id
    FROM gjj_crm_department
    WHERE name = '行政人事'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO diagnosis_task_id
    FROM gjj_crm_task
    WHERE stage_id = diagnosis_stage_id
      AND name = '确认诊断结果'
      AND task_type = 'approval'
    ORDER BY id
    LIMIT 1;

    SELECT id INTO pm_interview_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'PM谈话笔录'
      AND task_type = 'approval'
    ORDER BY id
    LIMIT 1;

    SELECT id INTO ala_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'ALA运营条件确认'
      AND task_type = 'approval'
    ORDER BY id
    LIMIT 1;

    IF signing_workflow_id IS NULL
       OR diagnosis_stage_id IS NULL
       OR collaboration_stage_id IS NULL
       OR npl_department_id IS NULL
       OR diagnosis_task_id IS NULL
       OR pm_interview_task_id IS NULL
       OR ala_task_id IS NULL THEN
        RAISE EXCEPTION 'NPL复核配置所需的流程、阶段、部门或任务不存在';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM gjj_crm_staff
        WHERE id = (
            SELECT leader_staff_id
            FROM gjj_crm_department
            WHERE id = npl_department_id
        )
          AND department_id = npl_department_id
          AND status = 1
    ) THEN
        RAISE EXCEPTION 'NPL部门未配置有效负责人';
    END IF;

    SELECT id INTO diagnosis_review_task_id
    FROM gjj_crm_task
    WHERE stage_id = diagnosis_stage_id
      AND name = 'NPL复核诊断结果'
    ORDER BY id
    LIMIT 1;

    IF diagnosis_review_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required,
            assignee_mode, assignee_department_id, form_id, script_id,
            activation_mode, condition_script_id,
            reject_action, reject_target_task_id, complete_target_task_id,
            opinion_requirement, due_days, sort, status, created_at, updated_at
        ) VALUES (
            diagnosis_stage_id, 'NPL复核诊断结果', 'approval', TRUE,
            'department_leader', npl_department_id, 0, 0,
            'route', 0,
            'terminate', 0, 0,
            'reject_required', 0, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO diagnosis_review_task_id;
    ELSE
        UPDATE gjj_crm_task
        SET task_type = 'approval',
            required = TRUE,
            assignee_mode = 'department_leader',
            assignee_department_id = npl_department_id,
            form_id = 0,
            script_id = 0,
            activation_mode = 'route',
            condition_script_id = 0,
            reject_action = 'terminate',
            reject_target_task_id = 0,
            complete_target_task_id = 0,
            opinion_requirement = 'reject_required',
            due_days = 0,
            sort = 40,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = diagnosis_review_task_id;
    END IF;

    SELECT id INTO pm_review_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'NPL复核PM会审'
    ORDER BY id
    LIMIT 1;

    IF pm_review_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required,
            assignee_mode, assignee_department_id, form_id, script_id,
            activation_mode, condition_script_id,
            reject_action, reject_target_task_id, complete_target_task_id,
            opinion_requirement, due_days, sort, status, created_at, updated_at
        ) VALUES (
            collaboration_stage_id, 'NPL复核PM会审', 'approval', TRUE,
            'department_leader', npl_department_id, 0, 0,
            'route', 0,
            'terminate', 0, ala_task_id,
            'reject_required', 0, 15, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO pm_review_task_id;
    ELSE
        UPDATE gjj_crm_task
        SET task_type = 'approval',
            required = TRUE,
            assignee_mode = 'department_leader',
            assignee_department_id = npl_department_id,
            form_id = 0,
            script_id = 0,
            activation_mode = 'route',
            condition_script_id = 0,
            reject_action = 'terminate',
            reject_target_task_id = 0,
            complete_target_task_id = ala_task_id,
            opinion_requirement = 'reject_required',
            due_days = 0,
            sort = 15,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = pm_review_task_id;
    END IF;

    SELECT id INTO ala_review_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'NPL复核ALA条件'
    ORDER BY id
    LIMIT 1;

    IF ala_review_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required,
            assignee_mode, assignee_department_id, form_id, script_id,
            activation_mode, condition_script_id,
            reject_action, reject_target_task_id, complete_target_task_id,
            opinion_requirement, due_days, sort, status, created_at, updated_at
        ) VALUES (
            collaboration_stage_id, 'NPL复核ALA条件', 'approval', TRUE,
            'department_leader', npl_department_id, 0, 0,
            'route', 0,
            'terminate', 0, 0,
            'reject_required', 0, 25, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO ala_review_task_id;
    ELSE
        UPDATE gjj_crm_task
        SET task_type = 'approval',
            required = TRUE,
            assignee_mode = 'department_leader',
            assignee_department_id = npl_department_id,
            form_id = 0,
            script_id = 0,
            activation_mode = 'route',
            condition_script_id = 0,
            reject_action = 'terminate',
            reject_target_task_id = 0,
            complete_target_task_id = 0,
            opinion_requirement = 'reject_required',
            due_days = 0,
            sort = 25,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = ala_review_task_id;
    END IF;

    UPDATE gjj_crm_task
    SET reject_action = 'route',
        reject_target_task_id = diagnosis_review_task_id,
        complete_target_task_id = 0,
        opinion_requirement = 'reject_required',
        reject_submit_form = FALSE,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = diagnosis_task_id;

    UPDATE gjj_crm_task
    SET reject_action = 'route',
        reject_target_task_id = pm_review_task_id,
        complete_target_task_id = ala_task_id,
        opinion_requirement = 'optional',
        reject_submit_form = TRUE,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = pm_interview_task_id;

    UPDATE gjj_crm_task
    SET activation_mode = 'route',
        reject_action = 'route',
        reject_target_task_id = ala_review_task_id,
        complete_target_task_id = 0,
        opinion_requirement = 'optional',
        reject_submit_form = TRUE,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = ala_task_id;

    UPDATE gjj_crm_task
    SET status = 2,
        updated_at = CURRENT_TIMESTAMP
    WHERE (stage_id = diagnosis_stage_id AND name = 'NPL补充诊断资料')
       OR (stage_id = collaboration_stage_id AND name = 'PM补充签约资料');

    SELECT id INTO legacy_diagnosis_correction_task_id
    FROM gjj_crm_task
    WHERE stage_id = diagnosis_stage_id
      AND name = 'NPL补充诊断资料'
    ORDER BY id
    LIMIT 1;

    SELECT id INTO legacy_pm_correction_task_id
    FROM gjj_crm_task
    WHERE stage_id = collaboration_stage_id
      AND name = 'PM补充签约资料'
    ORDER BY id
    LIMIT 1;

    IF legacy_diagnosis_correction_task_id IS NOT NULL THEN
        INSERT INTO gjj_crm_task_todo (
            lead_id, customer_id, asset_id, workflow_instance_id,
            customer_product_id, workflow_id, stage_id, task_id,
            assignee_department_id, assignee_staff_id, required, status,
            due_at, result, completed_at, created_at, updated_at
        )
        SELECT
            legacy.lead_id, legacy.customer_id, legacy.asset_id, legacy.workflow_instance_id,
            legacy.customer_product_id, legacy.workflow_id, legacy.stage_id, diagnosis_review_task_id,
            npl_department_id, department.leader_staff_id, TRUE, 'pending',
            NULL, '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        FROM gjj_crm_task_todo AS legacy
        JOIN gjj_crm_workflow_instance AS instance
          ON instance.id = legacy.workflow_instance_id
         AND instance.stage_id = legacy.stage_id
         AND instance.status = 'active'
        JOIN gjj_crm_department AS department
          ON department.id = npl_department_id
        WHERE legacy.task_id = legacy_diagnosis_correction_task_id
          AND legacy.status = 'pending'
        ON CONFLICT (workflow_instance_id, stage_id, task_id) DO UPDATE SET
            assignee_department_id = EXCLUDED.assignee_department_id,
            assignee_staff_id = EXCLUDED.assignee_staff_id,
            required = TRUE,
            status = 'pending',
            due_at = NULL,
            result = '',
            completed_at = NULL,
            updated_at = CURRENT_TIMESTAMP;

        UPDATE gjj_crm_task_todo AS legacy
        SET status = 'canceled',
            result = '已切换为NPL负责人复核',
            updated_at = CURRENT_TIMESTAMP
        WHERE legacy.task_id = legacy_diagnosis_correction_task_id
          AND legacy.status = 'pending'
          AND EXISTS (
              SELECT 1
              FROM gjj_crm_task_todo AS review
              WHERE review.workflow_instance_id = legacy.workflow_instance_id
                AND review.stage_id = legacy.stage_id
                AND review.task_id = diagnosis_review_task_id
                AND review.status = 'pending'
          );
    END IF;

    IF legacy_pm_correction_task_id IS NOT NULL THEN
        INSERT INTO gjj_crm_task_todo (
            lead_id, customer_id, asset_id, workflow_instance_id,
            customer_product_id, workflow_id, stage_id, task_id,
            assignee_department_id, assignee_staff_id, required, status,
            due_at, result, completed_at, created_at, updated_at
        )
        SELECT
            legacy.lead_id, legacy.customer_id, legacy.asset_id, legacy.workflow_instance_id,
            legacy.customer_product_id, legacy.workflow_id, legacy.stage_id, ala_review_task_id,
            npl_department_id, department.leader_staff_id, TRUE, 'pending',
            NULL, '', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        FROM gjj_crm_task_todo AS legacy
        JOIN gjj_crm_workflow_instance AS instance
          ON instance.id = legacy.workflow_instance_id
         AND instance.stage_id = legacy.stage_id
         AND instance.status = 'active'
        JOIN gjj_crm_department AS department
          ON department.id = npl_department_id
        WHERE legacy.task_id = legacy_pm_correction_task_id
          AND legacy.status = 'pending'
        ON CONFLICT (workflow_instance_id, stage_id, task_id) DO UPDATE SET
            assignee_department_id = EXCLUDED.assignee_department_id,
            assignee_staff_id = EXCLUDED.assignee_staff_id,
            required = TRUE,
            status = 'pending',
            due_at = NULL,
            result = '',
            completed_at = NULL,
            updated_at = CURRENT_TIMESTAMP;

        UPDATE gjj_crm_task_todo AS legacy
        SET status = 'canceled',
            result = '已切换为NPL负责人复核',
            updated_at = CURRENT_TIMESTAMP
        WHERE legacy.task_id = legacy_pm_correction_task_id
          AND legacy.status = 'pending'
          AND EXISTS (
              SELECT 1
              FROM gjj_crm_task_todo AS review
              WHERE review.workflow_instance_id = legacy.workflow_instance_id
                AND review.stage_id = legacy.stage_id
                AND review.task_id = ala_review_task_id
                AND review.status = 'pending'
          );
    END IF;

    UPDATE gjj_crm_task_todo AS ala
    SET status = 'canceled',
        result = '等待PM会审通过后重新创建',
        updated_at = CURRENT_TIMESTAMP
    WHERE ala.task_id = ala_task_id
      AND ala.status = 'pending'
      AND EXISTS (
          SELECT 1
          FROM gjj_crm_workflow_instance AS instance
          WHERE instance.id = ala.workflow_instance_id
            AND instance.stage_id = collaboration_stage_id
            AND instance.status = 'active'
      )
      AND (
          EXISTS (
              SELECT 1
              FROM gjj_crm_task_todo AS pm
              WHERE pm.workflow_instance_id = ala.workflow_instance_id
                AND pm.stage_id = ala.stage_id
                AND pm.task_id = pm_interview_task_id
                AND pm.status = 'pending'
          )
          OR EXISTS (
              SELECT 1
              FROM gjj_crm_task_todo AS review
              WHERE review.workflow_instance_id = ala.workflow_instance_id
                AND review.stage_id = ala.stage_id
                AND review.task_id = pm_review_task_id
                AND review.status = 'pending'
          )
      );

    UPDATE gjj_crm_form_field
    SET name = '上传附件-录音',
        required = TRUE,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = (SELECT form_id FROM gjj_crm_task WHERE id = pm_interview_task_id)
      AND data_field_id = (
          SELECT id
          FROM gjj_crm_data_field
          WHERE field_key = 'pm_interview_attachment'
          ORDER BY id
          LIMIT 1
      );

    UPDATE gjj_crm_data_field
    SET field_type = 'audio',
        updated_at = CURRENT_TIMESTAMP
    WHERE field_key = 'pm_interview_attachment';

    UPDATE gjj_crm_form_field
    SET name = '评估租金（元）',
        required = TRUE,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = (SELECT form_id FROM gjj_crm_task WHERE id = ala_task_id)
      AND data_field_id = (
          SELECT id
          FROM gjj_crm_data_field
          WHERE field_key = 'ala_assessed_rent'
          ORDER BY id
          LIMIT 1
      );

    IF administrative_department_id IS NOT NULL THEN
        UPDATE gjj_crm_public_resource
        SET owner_department_id = administrative_department_id,
            owner_staff_id = 0,
            updated_at = CURRENT_TIMESTAMP
        WHERE resource_cate_id IN (
            SELECT id
            FROM gjj_crm_public_resource_cate
            WHERE name = '会议室'
              AND status = 1
        );
    END IF;
END $$;

COMMIT;

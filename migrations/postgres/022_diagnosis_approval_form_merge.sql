-- Merge the formal T form into the diagnosis approval without rewriting history.
BEGIN;

DO $$
DECLARE
    diagnosis_approval_task_id BIGINT;
    formal_t_task_id BIGINT;
    formal_t_form_id BIGINT;
    formal_t_task_sort INTEGER;
BEGIN
    SELECT approval_task.id,
           formal_task.id,
           formal_task.form_id,
           formal_task.sort
    INTO diagnosis_approval_task_id,
         formal_t_task_id,
         formal_t_form_id,
         formal_t_task_sort
    FROM gjj_crm_stage AS stage
    JOIN gjj_crm_workflow AS workflow
      ON workflow.id = stage.workflow_id
     AND workflow.status = 1
    JOIN gjj_crm_task AS approval_task
      ON approval_task.stage_id = stage.id
     AND approval_task.name = '确认诊断结果'
     AND approval_task.task_type = 'approval'
     AND approval_task.status = 1
    JOIN gjj_crm_task AS formal_task
      ON formal_task.stage_id = stage.id
     AND formal_task.name = '确认正式T'
     AND formal_task.task_type = 'form'
     AND formal_task.form_id > 0
     AND formal_task.status IN (1, 2)
    WHERE stage.name = '诊断核验'
      AND stage.status = 1
    ORDER BY workflow.default_entry DESC,
             workflow.sort,
             workflow.id,
             stage.sort,
             stage.id,
             approval_task.id,
             formal_task.id
    LIMIT 1;

    IF diagnosis_approval_task_id IS NULL
       OR formal_t_task_id IS NULL
       OR formal_t_form_id IS NULL THEN
        RAISE EXCEPTION '诊断确认任务或正式T表单配置不完整';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS form_field
        JOIN gjj_crm_data_field AS data_field
          ON data_field.id = form_field.data_field_id
         AND data_field.field_key = 'formal_t'
         AND data_field.status = 1
        WHERE form_field.form_id = formal_t_form_id
          AND form_field.status = 1
    ) THEN
        RAISE EXCEPTION '正式T确认表单缺少已启用的 formal_t 字段';
    END IF;

    UPDATE gjj_crm_task
    SET form_id = formal_t_form_id,
        sort = LEAST(sort, formal_t_task_sort),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = diagnosis_approval_task_id;

    UPDATE gjj_crm_task
    SET status = 2,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = formal_t_task_id;

    UPDATE gjj_crm_task_todo AS formal_t_todo
    SET status = 'canceled',
        result = '任务已合并至确认诊断结果',
        updated_at = CURRENT_TIMESTAMP
    WHERE formal_t_todo.task_id = formal_t_task_id
      AND formal_t_todo.status = 'pending'
      AND EXISTS (
          SELECT 1
          FROM gjj_crm_task_todo AS approval_todo
          WHERE approval_todo.workflow_instance_id = formal_t_todo.workflow_instance_id
            AND approval_todo.stage_id = formal_t_todo.stage_id
            AND approval_todo.task_id = diagnosis_approval_task_id
            AND approval_todo.status = 'pending'
      );
END $$;

COMMIT;

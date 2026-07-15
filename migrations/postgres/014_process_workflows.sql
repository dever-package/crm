-- Treat leads and customer assets as two workflow subjects without introducing
-- another workflow engine. Existing customer, asset, contract and rental data
-- stay in place.
BEGIN;

ALTER TABLE IF EXISTS gjj_crm_workflow
    ADD COLUMN IF NOT EXISTS subject_type VARCHAR(32) NOT NULL DEFAULT 'customer_asset';

UPDATE gjj_crm_workflow
SET subject_type = 'customer_asset'
WHERE subject_type IS NULL OR subject_type = '';

ALTER TABLE IF EXISTS gjj_crm_workflow_instance
    ADD COLUMN IF NOT EXISTS lead_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_workflow_instance
    ALTER COLUMN customer_id SET DEFAULT 0,
    ALTER COLUMN asset_id SET DEFAULT 0;

ALTER TABLE IF EXISTS gjj_crm_task_todo
    ADD COLUMN IF NOT EXISTS lead_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_task_todo
    ALTER COLUMN customer_id SET DEFAULT 0,
    ALTER COLUMN asset_id SET DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_subject_status
    ON gjj_crm_workflow (subject_type, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_instance_lead_flow
    ON gjj_crm_workflow_instance (lead_id, workflow_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_todo_lead_status
    ON gjj_crm_task_todo (lead_id, status, id);

DO $$
DECLARE
    lead_workflow_id BIGINT;
    lead_stage_id BIGINT;
    lead_task_id BIGINT;
    lead_department_id BIGINT;
BEGIN
    SELECT department.id INTO lead_department_id
    FROM gjj_crm_department AS department
    WHERE department.status = 1
      AND EXISTS (
          SELECT 1
          FROM gjj_crm_staff AS staff
          WHERE staff.department_id = department.id AND staff.status = 1
      )
    ORDER BY CASE WHEN UPPER(department.code) = 'MKT' THEN 0 ELSE 1 END,
             department.sort,
             department.id
    LIMIT 1;

    IF lead_department_id IS NULL THEN
        RAISE EXCEPTION '线索流程至少需要一个拥有启用人员的部门';
    END IF;

    SELECT id INTO lead_workflow_id
    FROM gjj_crm_workflow
    WHERE subject_type = 'lead'
    ORDER BY default_entry DESC, status ASC, sort ASC, id ASC
    LIMIT 1;

    IF lead_workflow_id IS NULL THEN
        INSERT INTO gjj_crm_workflow (
            name, subject_type, default_entry, sort, status, created_at, updated_at
        ) VALUES (
            '线索流程', 'lead', TRUE, 5, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO lead_workflow_id;
    ELSE
        UPDATE gjj_crm_workflow
        SET default_entry = TRUE, status = 1, updated_at = CURRENT_TIMESTAMP
        WHERE id = lead_workflow_id;
    END IF;

    UPDATE gjj_crm_workflow
    SET default_entry = FALSE, updated_at = CURRENT_TIMESTAMP
    WHERE subject_type = 'lead' AND id <> lead_workflow_id AND default_entry = TRUE;

    SELECT id INTO lead_stage_id
    FROM gjj_crm_stage
    WHERE workflow_id = lead_workflow_id
    ORDER BY sort, id
    LIMIT 1;

    IF lead_stage_id IS NULL THEN
        INSERT INTO gjj_crm_stage (
            workflow_id, name, owner_department_id, assignment_mode,
            sort, status, created_at, updated_at
        ) VALUES (
            lead_workflow_id, '线索确认', lead_department_id, 'auto',
            10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO lead_stage_id;
    END IF;

    SELECT id INTO lead_task_id
    FROM gjj_crm_task
    WHERE stage_id = lead_stage_id
    ORDER BY sort, id
    LIMIT 1;

    IF lead_task_id IS NULL THEN
        INSERT INTO gjj_crm_task (
            stage_id, name, task_type, required, assignee_mode,
            assignee_department_id, form_id, script_id, due_days,
            sort, status, created_at, updated_at
        ) VALUES (
            lead_stage_id, '确认线索', 'todo', TRUE, 'stage',
            0, 0, 0, 0,
            10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO lead_task_id;
    END IF;

    INSERT INTO gjj_crm_workflow_instance (
        lead_id, customer_id, asset_id, customer_product_id,
        workflow_id, stage_id, owner_department_id, owner_staff_id,
        status, started_at, completed_at, terminated_at,
        terminated_reason, updated_at
    )
    SELECT
        lead.id, 0, 0, 0,
        lead_workflow_id, lead_stage_id,
        lead_department_id,
        COALESCE(owner.id, fallback_owner.id),
        CASE
            WHEN lead.status = 'converted' THEN 'completed'
            WHEN lead.status IN ('invalid', 'duplicate') THEN 'terminated'
            ELSE 'active'
        END,
        lead.created_at,
        CASE WHEN lead.status = 'converted'
            THEN COALESCE(lead.converted_at, lead.updated_at) END,
        CASE WHEN lead.status IN ('invalid', 'duplicate')
            THEN lead.updated_at END,
        CASE
            WHEN lead.status = 'invalid' THEN COALESCE(NULLIF(lead.invalid_note, ''), '无效线索')
            WHEN lead.status = 'duplicate' THEN COALESCE(NULLIF(lead.duplicate_reason, ''), '重复线索')
            ELSE ''
        END,
        lead.updated_at
    FROM gjj_crm_lead AS lead
    LEFT JOIN gjj_crm_staff AS owner
        ON owner.id = lead.owner_staff_id
       AND owner.department_id = lead_department_id
       AND owner.status = 1
    LEFT JOIN LATERAL (
        SELECT staff.id
        FROM gjj_crm_staff AS staff
        WHERE staff.department_id = lead_department_id AND staff.status = 1
        ORDER BY staff.id
        LIMIT 1
    ) AS fallback_owner ON TRUE
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_workflow_instance AS instance
        WHERE instance.lead_id = lead.id AND instance.workflow_id = lead_workflow_id
    );

    INSERT INTO gjj_crm_task_todo (
        lead_id, customer_id, asset_id, workflow_instance_id,
        customer_product_id, workflow_id, stage_id, task_id,
        assignee_department_id, assignee_staff_id, required, status,
        due_at, result, completed_at, created_at, updated_at
    )
    SELECT
        instance.lead_id, 0, 0, instance.id,
        0, instance.workflow_id, instance.stage_id, lead_task_id,
        instance.owner_department_id, instance.owner_staff_id, TRUE,
        CASE
            WHEN instance.status = 'completed' THEN 'done'
            WHEN instance.status = 'terminated' THEN 'canceled'
            ELSE 'pending'
        END,
        NULL,
        CASE
            WHEN instance.status = 'completed' THEN '线索已转化'
            WHEN instance.status = 'terminated' THEN instance.terminated_reason
            ELSE ''
        END,
        CASE WHEN instance.status = 'completed' THEN instance.completed_at END,
        instance.started_at,
        instance.updated_at
    FROM gjj_crm_workflow_instance AS instance
    WHERE instance.workflow_id = lead_workflow_id
      AND instance.lead_id > 0
      AND NOT EXISTS (
          SELECT 1
          FROM gjj_crm_task_todo AS todo
          WHERE todo.workflow_instance_id = instance.id
            AND todo.stage_id = instance.stage_id
            AND todo.task_id = lead_task_id
      );
END $$;

-- Entry and product-triggered workflows have no previous-stage owner who can
-- choose the first assignee. Their first enabled stage must use automatic assignment.
WITH ranked_entry_stage AS (
    SELECT stage.id,
           ROW_NUMBER() OVER (
               PARTITION BY stage.workflow_id
               ORDER BY stage.sort, stage.id
           ) AS row_number
    FROM gjj_crm_stage AS stage
    INNER JOIN gjj_crm_workflow AS workflow ON workflow.id = stage.workflow_id
    WHERE stage.status = 1
      AND (
          workflow.default_entry = TRUE
          OR EXISTS (
              SELECT 1
              FROM gjj_crm_product AS product
              WHERE product.service_workflow_id = workflow.id
          )
      )
)
UPDATE gjj_crm_stage AS stage
SET assignment_mode = 'auto', updated_at = CURRENT_TIMESTAMP
FROM ranked_entry_stage
WHERE stage.id = ranked_entry_stage.id AND ranked_entry_stage.row_number = 1;

-- Data templates now have only three visible storage scopes. Legacy business
-- templates are asset extensions and keep their IDs and recorded values.
UPDATE gjj_crm_data_template
SET cate_id = 2, updated_at = CURRENT_TIMESTAMP
WHERE cate_id = 3;

UPDATE gjj_crm_form_field
SET data_template_cate_id = 2, updated_at = CURRENT_TIMESTAMP
WHERE data_template_cate_id = 3;

UPDATE gjj_crm_data_template_cate
SET name = '业务数据（旧）', status = 2, sort = 30
WHERE id = 3;

-- Keep exactly one default entry per workflow subject.
WITH ranked_default AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY subject_type
               ORDER BY status ASC, sort ASC, id ASC
           ) AS row_number
    FROM gjj_crm_workflow
    WHERE default_entry = TRUE
)
UPDATE gjj_crm_workflow AS workflow
SET default_entry = FALSE, updated_at = CURRENT_TIMESTAMP
FROM ranked_default
WHERE workflow.id = ranked_default.id AND ranked_default.row_number > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_workflow_subject_default
    ON gjj_crm_workflow (subject_type)
    WHERE default_entry = TRUE;

COMMIT;

-- Simplified workflow: workflow -> ordered stage -> unified task.
ALTER TABLE gjj_crm_workflow
    ADD COLUMN IF NOT EXISTS default_entry BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS next_workflow_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_default_entry
    ON gjj_crm_workflow (default_entry, status, id);
DROP INDEX IF EXISTS idx_gjj_crm_workflow_next;
CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_next_workflow
    ON gjj_crm_workflow (next_workflow_id, id);

ALTER TABLE gjj_crm_stage
    ADD COLUMN IF NOT EXISTS assignment_mode VARCHAR(32) NOT NULL DEFAULT 'auto';

UPDATE gjj_crm_stage
SET assignment_mode = 'auto'
WHERE assignment_mode NOT IN ('auto', 'manual') OR assignment_mode = '';

DROP INDEX IF EXISTS idx_gjj_crm_task_assignee_status;
ALTER TABLE gjj_crm_task
    DROP COLUMN IF EXISTS assignee_staff_id;
CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_assignee_status
    ON gjj_crm_task (assignee_department_id, status, id);

UPDATE gjj_crm_task
SET assignee_mode = CASE assignee_mode
    WHEN 'department' THEN 'auto'
    WHEN 'staff' THEN 'manual'
    WHEN 'stage' THEN 'stage'
    WHEN 'auto' THEN 'auto'
    WHEN 'manual' THEN 'manual'
    ELSE 'stage'
END;

UPDATE gjj_crm_task
SET task_type = CASE
    WHEN task_type IN ('todo', 'form', 'approval', 'rule') THEN task_type
    WHEN task_type = 'decision' THEN 'approval'
    WHEN task_type = 'create' AND form_id > 0 THEN 'form'
    ELSE 'todo'
END;

ALTER TABLE gjj_crm_staff
    ADD COLUMN IF NOT EXISTS can_dispatch BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS last_assigned_at TIMESTAMPTZ NULL;

UPDATE gjj_crm_staff
SET can_dispatch = TRUE
WHERE id = 1;

ALTER TABLE gjj_crm_asset_progress
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS terminated_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS terminated_reason TEXT NOT NULL DEFAULT '';

UPDATE gjj_crm_asset_progress
SET completed_at = COALESCE(completed_at, updated_at)
WHERE status = 'completed';

WITH owner_candidates AS (
    SELECT progress.id AS progress_id,
           (
               SELECT staff.id
               FROM gjj_crm_staff staff
               WHERE staff.department_id = progress.owner_department_id
                 AND staff.status = 1
               ORDER BY staff.id
               LIMIT 1
           ) AS staff_id
    FROM gjj_crm_asset_progress progress
    WHERE progress.status = 'active'
      AND progress.owner_staff_id = 0
)
UPDATE gjj_crm_asset_progress progress
SET owner_staff_id = owner_candidates.staff_id,
    updated_at = CURRENT_TIMESTAMP
FROM owner_candidates
WHERE progress.id = owner_candidates.progress_id
  AND owner_candidates.staff_id IS NOT NULL;

DO $$
DECLARE
    signing_workflow_id BIGINT;
    operation_workflow_id BIGINT;
    default_department_id BIGINT;
    npl_department_id BIGINT;
    pm_department_id BIGINT;
    lawyer_department_id BIGINT;
    ala_department_id BIGINT;
    finance_department_id BIGINT;
    contract_department_id BIGINT;
    service_department_id BIGINT;
    intake_stage_id BIGINT;
    collection_stage_id BIGINT;
    diagnosis_stage_id BIGINT;
    product_stage_id BIGINT;
    contract_stage_id BIGINT;
    signing_confirm_stage_id BIGINT;
    operation_intake_stage_id BIGINT;
    leasing_stage_id BIGINT;
    lease_signing_stage_id BIGINT;
    delivery_stage_id BIGINT;
    active_lease_stage_id BIGINT;
    checkout_stage_id BIGINT;
BEGIN
    SELECT id INTO default_department_id
    FROM gjj_crm_department
    WHERE status = 1
    ORDER BY CASE WHEN code = 'default' THEN 0 ELSE 1 END, sort, id
    LIMIT 1;

    SELECT COALESCE((SELECT id FROM gjj_crm_department WHERE code = 'NPL' AND status = 1 ORDER BY id LIMIT 1), default_department_id)
    INTO npl_department_id;
    SELECT COALESCE((SELECT id FROM gjj_crm_department WHERE code = 'PM' AND status = 1 ORDER BY id LIMIT 1), default_department_id)
    INTO pm_department_id;
    SELECT COALESCE((SELECT id FROM gjj_crm_department WHERE code = 'LAW' AND status = 1 ORDER BY id LIMIT 1), default_department_id)
    INTO lawyer_department_id;
    SELECT COALESCE((SELECT id FROM gjj_crm_department WHERE code = 'ALA' AND status = 1 ORDER BY id LIMIT 1), default_department_id)
    INTO ala_department_id;
    SELECT COALESCE((SELECT id FROM gjj_crm_department WHERE code = 'FIN' AND status = 1 ORDER BY id LIMIT 1), default_department_id)
    INTO finance_department_id;
    SELECT COALESCE((SELECT id FROM gjj_crm_department WHERE code = 'CONTRACT' AND status = 1 ORDER BY id LIMIT 1), default_department_id)
    INTO contract_department_id;
    SELECT COALESCE((SELECT id FROM gjj_crm_department WHERE code = 'SERVICE' AND status = 1 ORDER BY id LIMIT 1), default_department_id)
    INTO service_department_id;

    INSERT INTO gjj_crm_workflow (name, default_entry, next_workflow_id, sort, status)
    SELECT '签署流程', FALSE, 0, 10, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_workflow WHERE name = '签署流程');

    INSERT INTO gjj_crm_workflow (name, default_entry, next_workflow_id, sort, status)
    SELECT '运营流程', FALSE, 0, 20, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_workflow WHERE name = '运营流程');

    SELECT id INTO signing_workflow_id
    FROM gjj_crm_workflow WHERE name = '签署流程' ORDER BY id LIMIT 1;
    SELECT id INTO operation_workflow_id
    FROM gjj_crm_workflow WHERE name = '运营流程' ORDER BY id LIMIT 1;

    IF NOT EXISTS (SELECT 1 FROM gjj_crm_workflow WHERE default_entry = TRUE) THEN
        UPDATE gjj_crm_workflow SET default_entry = TRUE WHERE id = signing_workflow_id;
    END IF;
    UPDATE gjj_crm_workflow
    SET next_workflow_id = operation_workflow_id
    WHERE id = signing_workflow_id
      AND next_workflow_id = 0;

    UPDATE gjj_crm_stage
    SET name = '签约确认', sort = 60, owner_department_id = contract_department_id
    WHERE workflow_id = signing_workflow_id
      AND name = '签署确认'
      AND NOT EXISTS (
          SELECT 1 FROM gjj_crm_stage
          WHERE workflow_id = signing_workflow_id AND name = '签约确认'
      );

    UPDATE gjj_crm_stage
    SET sort = 20,
        owner_department_id = npl_department_id
    WHERE workflow_id = signing_workflow_id
      AND name = '资料收集'
      AND sort = 10;

    UPDATE gjj_crm_stage
    SET name = '待运营', sort = 10, owner_department_id = ala_department_id
    WHERE workflow_id = operation_workflow_id
      AND name = '运营启动'
      AND NOT EXISTS (
          SELECT 1 FROM gjj_crm_stage
          WHERE workflow_id = operation_workflow_id AND name = '待运营'
      );

    INSERT INTO gjj_crm_stage (workflow_id, name, owner_department_id, assignment_mode, sort, status)
    SELECT signing_workflow_id, stage.name, stage.department_id, 'auto', stage.sort, 1
    FROM (VALUES
        ('接单建档', npl_department_id, 10),
        ('资料收集', npl_department_id, 20),
        ('诊断核验', npl_department_id, 30),
        ('产品确认', pm_department_id, 40),
        ('合同签署', contract_department_id, 50),
        ('签约确认', contract_department_id, 60)
    ) AS stage(name, department_id, sort)
    WHERE NOT EXISTS (
        SELECT 1 FROM gjj_crm_stage existing
        WHERE existing.workflow_id = signing_workflow_id
          AND existing.name = stage.name
    );

    INSERT INTO gjj_crm_stage (workflow_id, name, owner_department_id, assignment_mode, sort, status)
    SELECT operation_workflow_id, stage.name, stage.department_id, 'auto', stage.sort, 1
    FROM (VALUES
        ('待运营', ala_department_id, 10),
        ('招租', ala_department_id, 20),
        ('签租', ala_department_id, 30),
        ('交付', service_department_id, 40),
        ('在租', ala_department_id, 50),
        ('退租', service_department_id, 60)
    ) AS stage(name, department_id, sort)
    WHERE NOT EXISTS (
        SELECT 1 FROM gjj_crm_stage existing
        WHERE existing.workflow_id = operation_workflow_id
          AND existing.name = stage.name
    );

    UPDATE gjj_crm_stage
    SET assignment_mode = 'auto', status = 1
    WHERE workflow_id IN (signing_workflow_id, operation_workflow_id)
      AND assignment_mode NOT IN ('auto', 'manual');

    SELECT id INTO intake_stage_id FROM gjj_crm_stage WHERE workflow_id = signing_workflow_id AND name = '接单建档' ORDER BY id LIMIT 1;
    SELECT id INTO collection_stage_id FROM gjj_crm_stage WHERE workflow_id = signing_workflow_id AND name = '资料收集' ORDER BY id LIMIT 1;
    SELECT id INTO diagnosis_stage_id FROM gjj_crm_stage WHERE workflow_id = signing_workflow_id AND name = '诊断核验' ORDER BY id LIMIT 1;
    SELECT id INTO product_stage_id FROM gjj_crm_stage WHERE workflow_id = signing_workflow_id AND name = '产品确认' ORDER BY id LIMIT 1;
    SELECT id INTO contract_stage_id FROM gjj_crm_stage WHERE workflow_id = signing_workflow_id AND name = '合同签署' ORDER BY id LIMIT 1;
    SELECT id INTO signing_confirm_stage_id FROM gjj_crm_stage WHERE workflow_id = signing_workflow_id AND name = '签约确认' ORDER BY id LIMIT 1;
    SELECT id INTO operation_intake_stage_id FROM gjj_crm_stage WHERE workflow_id = operation_workflow_id AND name = '待运营' ORDER BY id LIMIT 1;
    SELECT id INTO leasing_stage_id FROM gjj_crm_stage WHERE workflow_id = operation_workflow_id AND name = '招租' ORDER BY id LIMIT 1;
    SELECT id INTO lease_signing_stage_id FROM gjj_crm_stage WHERE workflow_id = operation_workflow_id AND name = '签租' ORDER BY id LIMIT 1;
    SELECT id INTO delivery_stage_id FROM gjj_crm_stage WHERE workflow_id = operation_workflow_id AND name = '交付' ORDER BY id LIMIT 1;
    SELECT id INTO active_lease_stage_id FROM gjj_crm_stage WHERE workflow_id = operation_workflow_id AND name = '在租' ORDER BY id LIMIT 1;
    SELECT id INTO checkout_stage_id FROM gjj_crm_stage WHERE workflow_id = operation_workflow_id AND name = '退租' ORDER BY id LIMIT 1;

    UPDATE gjj_crm_task
    SET stage_id = diagnosis_stage_id,
        name = '自动核验资料',
        task_type = 'rule',
        assignee_mode = 'stage',
        assignee_department_id = 0,
        sort = 10
    WHERE stage_id = collection_stage_id
      AND script_id > 0
      AND name IN ('自动判断T节点', '自动判断 T 节点', '十一维T节点自动判断');

    UPDATE gjj_crm_task
    SET name = '收集十一维资料', task_type = 'form', assignee_mode = 'stage', assignee_department_id = 0, sort = 10
    WHERE stage_id = collection_stage_id
      AND form_id > 0;

    UPDATE gjj_crm_task
    SET name = '签约确认', task_type = 'approval', assignee_mode = 'stage', assignee_department_id = 0, sort = 10
    WHERE stage_id = signing_confirm_stage_id
      AND name IN ('签署审核', '签署确认');

    UPDATE gjj_crm_task
    SET name = '确认运营接单', task_type = 'todo', assignee_mode = 'stage', assignee_department_id = 0, sort = 10
    WHERE stage_id = operation_intake_stage_id
      AND name = '启动运营';

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT intake_stage_id, '接单建档与建群', CASE WHEN form_ref.id IS NULL THEN 'todo' ELSE 'form' END,
           TRUE, 'stage', 0, COALESCE(form_ref.id, 0), 0, 0, 10, 1
    FROM (SELECT MIN(id) AS id FROM gjj_crm_form WHERE name = 'NPL接单首呼与建群') form_ref
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = intake_stage_id AND name = '接单建档与建群');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT collection_stage_id, '收集十一维资料', CASE WHEN form_ref.id IS NULL THEN 'todo' ELSE 'form' END,
           TRUE, 'stage', 0, COALESCE(form_ref.id, 0), 0, 0, 10, 1
    FROM (SELECT MIN(id) AS id FROM gjj_crm_form WHERE name = 'P01-P12十一维探针') form_ref
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = collection_stage_id AND name = '收集十一维资料');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT diagnosis_stage_id, '自动核验资料', 'rule', TRUE, 'stage', 0, 0, rule_ref.id, 0, 10, 1
    FROM (SELECT MIN(id) AS id FROM gjj_crm_rule_script WHERE name = '十一维T节点自动判断') rule_ref
    WHERE rule_ref.id IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = diagnosis_stage_id AND name = '自动核验资料');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT diagnosis_stage_id, '确认诊断结果', 'approval', TRUE, 'stage', 0, 0, 0, 0, 20, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = diagnosis_stage_id AND name = '确认诊断结果');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT product_stage_id, '确认适用产品', CASE WHEN form_ref.id IS NULL THEN 'todo' ELSE 'form' END,
           TRUE, 'stage', 0, COALESCE(form_ref.id, 0), 0, 0, 10, 1
    FROM (SELECT MIN(id) AS id FROM gjj_crm_form WHERE name = 'PM案件判断与产品确认') form_ref
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = product_stage_id AND name = '确认适用产品');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT contract_stage_id, '合同签署登记', CASE WHEN form_ref.id IS NULL THEN 'todo' ELSE 'form' END,
           TRUE, 'stage', 0, COALESCE(form_ref.id, 0), 0, 0, 10, 1
    FROM (SELECT MIN(id) AS id FROM gjj_crm_form WHERE name = '合同与费用') form_ref
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = contract_stage_id AND name = '合同签署登记');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT contract_stage_id, task.name, 'approval', TRUE, 'auto', task.department_id, 0, 0, 0, task.sort, 1
    FROM (VALUES
        ('律师合同审核', lawyer_department_id, 20),
        ('ALA运营条件确认', ala_department_id, 30),
        ('财务费用确认', finance_department_id, 40)
    ) AS task(name, department_id, sort)
    WHERE NOT EXISTS (
        SELECT 1 FROM gjj_crm_task existing
        WHERE existing.stage_id = contract_stage_id
          AND existing.name = task.name
    );

    UPDATE gjj_crm_task existing
    SET assignee_mode = 'auto',
        assignee_department_id = task.department_id,
        task_type = 'approval',
        required = TRUE,
        status = 1
    FROM (VALUES
        ('律师合同审核', lawyer_department_id),
        ('ALA运营条件确认', ala_department_id),
        ('财务费用确认', finance_department_id)
    ) AS task(name, department_id)
    WHERE existing.stage_id = contract_stage_id
      AND existing.name = task.name;

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT signing_confirm_stage_id, '签约确认', 'approval', TRUE, 'stage', 0, 0, 0, 0, 10, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = signing_confirm_stage_id AND name = '签约确认');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT operation_intake_stage_id, '确认运营接单', 'todo', TRUE, 'stage', 0, 0, 0, 0, 10, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = operation_intake_stage_id AND name = '确认运营接单');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT leasing_stage_id, '推进招租', 'todo', TRUE, 'stage', 0, 0, 0, 0, 10, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = leasing_stage_id AND name = '推进招租');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT lease_signing_stage_id, '登记租赁与收款', CASE WHEN form_ref.id IS NULL THEN 'todo' ELSE 'form' END,
           TRUE, 'stage', 0, COALESCE(form_ref.id, 0), 0, 0, 10, 1
    FROM (SELECT MIN(id) AS id FROM gjj_crm_form WHERE name = '租赁记录与租户收款') form_ref
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = lease_signing_stage_id AND name = '登记租赁与收款');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT delivery_stage_id, '确认交付', 'todo', TRUE, 'stage', 0, 0, 0, 0, 10, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = delivery_stage_id AND name = '确认交付');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT active_lease_stage_id, '在租运营', CASE WHEN form_ref.id IS NULL THEN 'todo' ELSE 'form' END,
           TRUE, 'stage', 0, COALESCE(form_ref.id, 0), 0, 0, 10, 1
    FROM (SELECT MIN(id) AS id FROM gjj_crm_form WHERE name = '出租运营成本登记') form_ref
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = active_lease_stage_id AND name = '在租运营');

    INSERT INTO gjj_crm_task (stage_id, name, task_type, required, assignee_mode, assignee_department_id, form_id, script_id, due_days, sort, status)
    SELECT checkout_stage_id, '完成退租', 'todo', TRUE, 'stage', 0, 0, 0, 0, 10, 1
    WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_task WHERE stage_id = checkout_stage_id AND name = '完成退租');
END $$;

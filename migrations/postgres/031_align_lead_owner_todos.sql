BEGIN;

-- 线索负责人改派后，同负责部门的未完成待办应由当前线索负责人继续办理。
UPDATE gjj_crm_task_todo AS todo
SET assignee_department_id = instance.owner_department_id,
    assignee_staff_id = instance.owner_staff_id,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_workflow_instance AS instance
WHERE instance.id = todo.workflow_instance_id
  AND instance.status = 'active'
  AND instance.lead_id > 0
  AND instance.owner_staff_id > 0
  AND todo.status = 'pending'
  AND todo.stage_id = instance.stage_id
  AND todo.assignee_department_id = instance.owner_department_id
  AND todo.assignee_staff_id <> instance.owner_staff_id;

COMMIT;

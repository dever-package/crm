-- Remove obsolete signing task forms after the simplified workflow cutover.
-- Active forms and any form still referenced by a task are preserved.
BEGIN;

CREATE TEMP TABLE crm_obsolete_task_form ON COMMIT DROP AS
SELECT form.id
FROM gjj_crm_form AS form
WHERE form.status = 2
  AND form.name IN (
      '客户来源与基础建档',
      '客户资料与资产建档',
      'PM案件判断与产品确认',
      '专业门禁协作',
      '律师正式T与合同法律边界',
      'ALA资产运营与回款支撑',
      '财务费用与收款审核',
      '合同路径与文本门禁'
  )
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_task AS task
      WHERE task.form_id = form.id
  );

DELETE FROM gjj_crm_form_field AS field
USING crm_obsolete_task_form AS obsolete
WHERE field.form_id = obsolete.id;

DELETE FROM gjj_crm_form AS form
USING crm_obsolete_task_form AS obsolete
WHERE form.id = obsolete.id;

COMMIT;

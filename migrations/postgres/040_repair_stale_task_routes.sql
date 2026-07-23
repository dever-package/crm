BEGIN;

-- A routed task can only target an enabled task in the same stage. Historical
-- configuration allowed a target to be disabled without clearing references,
-- which made the source task impossible to complete at runtime.
UPDATE gjj_crm_task AS source
SET complete_target_task_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE source.complete_target_task_id > 0
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_task AS target
      WHERE target.id = source.complete_target_task_id
        AND target.stage_id = source.stage_id
        AND target.status = 1
  );

UPDATE gjj_crm_task AS source
SET reject_action = 'stay',
    reject_target_task_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE source.reject_target_task_id > 0
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_task AS target
      WHERE target.id = source.reject_target_task_id
        AND target.stage_id = source.stage_id
        AND target.status = 1
  );

COMMIT;

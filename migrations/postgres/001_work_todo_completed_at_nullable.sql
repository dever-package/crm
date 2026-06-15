-- Pending collaboration todos have no completion time until they are completed.
ALTER TABLE gjj_crm_work_todo
    ALTER COLUMN completed_at DROP NOT NULL;

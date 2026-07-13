-- Remove legacy configuration and runtime tables after workflow-instance cutover.
DROP INDEX IF EXISTS idx_gjj_crm_product_category;
DROP INDEX IF EXISTS idx_gjj_crm_product_signing_type;
ALTER TABLE IF EXISTS gjj_crm_product
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS default_signing_business_type,
    DROP COLUMN IF EXISTS signing_direction,
    DROP COLUMN IF EXISTS default_signing_direction,
    DROP COLUMN IF EXISTS need_pm_review,
    DROP COLUMN IF EXISTS need_lawyer_review,
    DROP COLUMN IF EXISTS need_ala_review,
    DROP COLUMN IF EXISTS need_finance_review,
    DROP COLUMN IF EXISTS need_contract_review;

DROP INDEX IF EXISTS idx_gjj_crm_workflow_next_workflow;
ALTER TABLE IF EXISTS gjj_crm_workflow
    DROP COLUMN IF EXISTS next_workflow_id;

DROP INDEX IF EXISTS idx_gjj_crm_data_template_cate_target;
DROP INDEX IF EXISTS idx_gjj_crm_data_template_cate_business_object_type;
ALTER TABLE IF EXISTS gjj_crm_data_template_cate
    DROP COLUMN IF EXISTS target_table,
    DROP COLUMN IF EXISTS business_object_type_id;

DROP INDEX IF EXISTS idx_gjj_crm_data_record_business_object_template;
ALTER TABLE IF EXISTS gjj_crm_data_record
    DROP COLUMN IF EXISTS business_object_id;

DROP INDEX IF EXISTS idx_gjj_crm_stat_field_value_business_object_time;
ALTER TABLE IF EXISTS gjj_crm_stat_field_value
    DROP COLUMN IF EXISTS business_object_id;

DROP INDEX IF EXISTS idx_gjj_crm_finance_ledger_business_object_time;
ALTER TABLE IF EXISTS gjj_crm_finance_ledger
    DROP COLUMN IF EXISTS business_object_id;

DROP TABLE IF EXISTS gjj_crm_asset_progress;
DROP TABLE IF EXISTS gjj_crm_business_object;
DROP TABLE IF EXISTS gjj_crm_business_object_type;

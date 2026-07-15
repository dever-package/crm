-- Remove configuration and schema left behind by the legacy CRM workflow.
-- The migration keeps active lead, customer and customer-asset data scopes.
BEGIN;

-- Legacy business templates now belong to the customer-asset scope. Preserve
-- any tenant-created rows that were not covered by the earlier migration.
UPDATE gjj_crm_data_template
SET cate_id = 2, updated_at = CURRENT_TIMESTAMP
WHERE cate_id = 3;

UPDATE gjj_crm_form_field
SET data_template_cate_id = 2, updated_at = CURRENT_TIMESTAMP
WHERE data_template_cate_id = 3;

DELETE FROM gjj_crm_data_template_cate
WHERE id = 3;

-- Reuse common option sets when a field's private options are exactly equal.
-- A private option is retained when no exact common set exists.
CREATE TEMP TABLE crm_matching_option_set ON COMMIT DROP AS
WITH field_signature AS (
    SELECT
        option.data_field_id,
        COUNT(*) AS option_count,
        STRING_AGG(
            option.value || CHR(31) || option.name,
            CHR(30) ORDER BY option.sort, option.id
        ) AS option_signature
    FROM gjj_crm_data_field_option AS option
    GROUP BY option.data_field_id
),
option_set_signature AS (
    SELECT
        item.option_set_id,
        COUNT(*) AS option_count,
        STRING_AGG(
            item.value || CHR(31) || item.name,
            CHR(30) ORDER BY item.sort, item.id
        ) AS option_signature
    FROM gjj_crm_option_set_item AS item
    WHERE item.status = 1
    GROUP BY item.option_set_id
)
SELECT
    field_signature.data_field_id,
    MIN(option_set_signature.option_set_id) AS option_set_id
FROM field_signature
INNER JOIN option_set_signature
    ON option_set_signature.option_count = field_signature.option_count
   AND option_set_signature.option_signature = field_signature.option_signature
INNER JOIN gjj_crm_option_set AS option_set
    ON option_set.id = option_set_signature.option_set_id
   AND option_set.status = 1
GROUP BY field_signature.data_field_id;

UPDATE gjj_crm_data_field AS field
SET option_set_id = matching.option_set_id,
    updated_at = CURRENT_TIMESTAMP
FROM crm_matching_option_set AS matching
WHERE field.id = matching.data_field_id
  AND field.option_set_id = 0;

DELETE FROM gjj_crm_data_field_option AS option
USING gjj_crm_data_field AS field
WHERE option.data_field_id = field.id
  AND field.option_set_id > 0;

-- Remove enabled forms that no current task uses.
CREATE TEMP TABLE crm_obsolete_form ON COMMIT DROP AS
SELECT form.id
FROM gjj_crm_form AS form
WHERE form.name IN (
    '服务交付与验收',
    'PM服务编排与SLA',
    'PM助理客户前置同步',
    'ALA资产运营交付',
    '律师司法服务推进',
    '财务对账与回款确认'
)
AND NOT EXISTS (
    SELECT 1
    FROM gjj_crm_task AS task
    WHERE task.form_id = form.id
);

DELETE FROM gjj_crm_form_field AS field
USING crm_obsolete_form AS obsolete
WHERE field.form_id = obsolete.id;

DELETE FROM gjj_crm_form AS form
USING crm_obsolete_form AS obsolete
WHERE form.id = obsolete.id;

-- Report mappings have no runtime consumer.
CREATE TEMP TABLE crm_obsolete_data_usage ON COMMIT DROP AS
SELECT id
FROM gjj_crm_data_usage
WHERE usage_type = 'report';

DELETE FROM gjj_crm_data_usage_field AS field
USING crm_obsolete_data_usage AS obsolete
WHERE field.usage_id = obsolete.id;

DELETE FROM gjj_crm_data_usage AS usage
USING crm_obsolete_data_usage AS obsolete
WHERE usage.id = obsolete.id;

-- Remove templates and fields left by the previous signing/service design.
CREATE TEMP TABLE crm_obsolete_data_template ON COMMIT DROP AS
SELECT template.id
FROM gjj_crm_data_template AS template
WHERE template.name IN (
    '客户来源与基础建档',
    '专业协作意见',
    '服务交付'
)
AND NOT EXISTS (
    SELECT 1
    FROM gjj_crm_form_field AS form_field
    WHERE form_field.data_template_id = template.id
);

CREATE TEMP TABLE crm_obsolete_data_field ON COMMIT DROP AS
SELECT field.id
FROM gjj_crm_data_field AS field
WHERE field.data_template_id IN (
    SELECT id FROM crm_obsolete_data_template
)
OR (
    field.field_key IN (
        'signing_business_type_candidate',
        'signing_business_type_confidence',
        'npl_candidate_s_product_codes',
        'npl_primary_s_product_code',
        'pm_confirmed_signing_business_type',
        'final_signing_business_type',
        'pm_confirmed_s_product_code',
        's_product_confirmation_status'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS form_field
        WHERE form_field.data_field_id = field.id
    )
);

DELETE FROM gjj_crm_finance_ledger
WHERE data_field_id IN (SELECT id FROM crm_obsolete_data_field);

DELETE FROM gjj_crm_stat_field_value
WHERE data_field_id IN (SELECT id FROM crm_obsolete_data_field)
   OR data_template_id IN (SELECT id FROM crm_obsolete_data_template);

DELETE FROM gjj_crm_data_record
WHERE data_template_id IN (SELECT id FROM crm_obsolete_data_template);

DELETE FROM gjj_crm_data_usage_field
WHERE data_field_id IN (SELECT id FROM crm_obsolete_data_field)
   OR data_template_id IN (SELECT id FROM crm_obsolete_data_template);

DELETE FROM gjj_crm_form_field
WHERE data_field_id IN (SELECT id FROM crm_obsolete_data_field)
   OR data_template_id IN (SELECT id FROM crm_obsolete_data_template);

DELETE FROM gjj_crm_data_field_option
WHERE data_field_id IN (SELECT id FROM crm_obsolete_data_field);

DELETE FROM gjj_crm_data_field
WHERE id IN (SELECT id FROM crm_obsolete_data_field);

DELETE FROM gjj_crm_data_template
WHERE id IN (SELECT id FROM crm_obsolete_data_template);

-- Remove disabled rules and finance types with no active consumer.
DELETE FROM gjj_crm_rule_script AS script
WHERE script.name = 'P01-P12签约方向自动判断'
  AND script.status = 2
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_task AS task
      WHERE task.script_id = script.id
  );

DELETE FROM gjj_crm_finance_type AS finance_type
WHERE finance_type.code IN ('operation_income', 'refund_expense')
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_data_usage_field AS usage_field
      WHERE usage_field.finance_type_id = finance_type.id
  )
  AND NOT EXISTS (
      SELECT 1
      FROM gjj_crm_finance_ledger AS ledger
      WHERE ledger.finance_type_id = finance_type.id
  );

-- Delete common sets that remain unreferenced after field consolidation.
DELETE FROM gjj_crm_option_set_item AS item
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_data_field AS field
    WHERE field.option_set_id = item.option_set_id
);

DELETE FROM gjj_crm_option_set AS option_set
WHERE NOT EXISTS (
    SELECT 1
    FROM gjj_crm_data_field AS field
    WHERE field.option_set_id = option_set.id
);

-- These tables belong to workflow/domain models that no longer exist in the
-- current package. Migration 016 is the last migration that reads the product
-- eligibility table, so it is safe to remove here.
DROP TABLE IF EXISTS
    gjj_crm_asset_cate,
    gjj_crm_asset_type,
    gjj_crm_business_role,
    gjj_crm_configuration_installation,
    gjj_crm_customer_resource,
    gjj_crm_customer_stage,
    gjj_crm_customer_state,
    gjj_crm_department_function,
    gjj_crm_flow_edge,
    gjj_crm_flow_node,
    gjj_crm_flow_release,
    gjj_crm_flow_stage,
    gjj_crm_flow_template,
    gjj_crm_function,
    gjj_crm_function_cate,
    gjj_crm_function_result,
    gjj_crm_opportunity_type,
    gjj_crm_product_eligibility_rule,
    gjj_crm_product_match_run,
    gjj_crm_resource_task,
    gjj_crm_stage_transition,
    gjj_crm_stage_transition_log,
    gjj_crm_status,
    gjj_crm_status_transition,
    gjj_crm_task_field,
    gjj_crm_task_point_ledger,
    gjj_crm_task_record,
    gjj_crm_task_result,
    gjj_crm_task_template,
    gjj_crm_task_template_cate,
    gjj_crm_task_transition,
    gjj_crm_timeline
CASCADE;

COMMIT;

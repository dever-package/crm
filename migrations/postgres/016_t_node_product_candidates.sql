-- Recommend the configured judicial product after the eleven-dimension rule
-- determines a T node. Product confirmation remains a manual PM decision, and
-- operation products can still be added independently.
BEGIN;

UPDATE gjj_crm_rule_script
SET script = replace(
        replace(
            script,
            E'  function result(value, reason) {',
            E'  var productCodesByT = {\n'
                || E'    T0: ["S03"],\n'
                || E'    T1: ["S04"],\n'
                || E'    T2: ["S08"],\n'
                || E'    T3: ["S10"],\n'
                || E'    T4: ["S11"],\n'
                || E'    T5: ["S12"],\n'
                || E'    T6: ["S13"],\n'
                || E'    T7: ["S20"],\n'
                || E'    T8: ["S23"],\n'
                || E'    T9: ["S24"],\n'
                || E'    T10: ["S26"]\n'
                || E'  };\n\n'
                || E'  function result(value, reason) {'
        ),
        'return { value: value, reason: reason, fields: { candidate_t_node: value, candidate_t_confidence_level: value === "T0" ? "pending" : "high" } };',
        'return { value: value, reason: reason, fields: { candidate_t_node: value, candidate_t_confidence_level: value === "T0" ? "pending" : "high" }, product_codes: productCodesByT[value] || [] };'
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE name = '十一维T节点自动判断'
  AND status = 1
  AND script NOT LIKE '%product_codes%';

-- Backfill active entry workflows whose T-node rule finished before this
-- migration. Existing product decisions are left untouched.
INSERT INTO gjj_crm_customer_product (
    customer_id,
    asset_id,
    product_id,
    source_workflow_instance_id,
    status,
    created_at,
    updated_at
)
SELECT
    instance.customer_id,
    instance.asset_id,
    product.id,
    instance.id,
    'candidate',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM gjj_crm_workflow_instance AS instance
INNER JOIN gjj_crm_workflow AS workflow
    ON workflow.id = instance.workflow_id
   AND workflow.default_entry = TRUE
   AND workflow.subject_type = 'customer_asset'
INNER JOIN gjj_crm_data_field AS t_field
    ON t_field.field_key = 'candidate_t_node'
   AND t_field.status = 1
INNER JOIN LATERAL (
    SELECT record.record_json::jsonb ->> t_field.id::text AS t_node
    FROM gjj_crm_data_record AS record
    WHERE record.customer_id = instance.customer_id
      AND record.asset_id = instance.asset_id
      AND record.status = 1
      AND record.record_json::jsonb ? t_field.id::text
    ORDER BY record.updated_at DESC, record.id DESC
    LIMIT 1
) AS decision ON decision.t_node <> ''
INNER JOIN gjj_crm_product_eligibility_rule AS eligibility
    ON eligibility.data_field_id = t_field.id
   AND eligibility.operator = 'equals'
   AND eligibility.expected_value = decision.t_node
   AND eligibility.effect = 'eligible'
   AND eligibility.status = 1
INNER JOIN gjj_crm_product AS product
    ON product.id = eligibility.product_id
   AND product.status = 1
WHERE instance.status = 'active'
  AND instance.customer_id > 0
  AND instance.asset_id > 0
  AND instance.customer_product_id = 0
ON CONFLICT (source_workflow_instance_id, product_id) DO NOTHING;

COMMIT;

BEGIN;

-- Migration 028 restored finance-bound form fields. This legacy field kept
-- its old percentage option set, so finance ledger creation parsed every
-- selected value (for example, "10%") as zero. Keep the finance binding but
-- restore the field's current business meaning as a monetary input.
UPDATE gjj_crm_data_field AS field
SET name = '第三方成本',
    field_type = 'money',
    option_set_id = 0,
    updated_at = CURRENT_TIMESTAMP
FROM gjj_crm_finance_type AS finance_type
WHERE field.field_key = 'third_party_cost'
  AND field.finance_type_id = finance_type.id
  AND finance_type.code = 'third_party_cost'
  AND (
      field.name <> '第三方成本'
      OR field.field_type <> 'money'
      OR field.option_set_id <> 0
  );

COMMIT;

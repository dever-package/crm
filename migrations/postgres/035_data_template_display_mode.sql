-- Configure whether a data template is visible in lead, customer and asset details.
BEGIN;

ALTER TABLE gjj_crm_data_template
    ADD COLUMN IF NOT EXISTS display_mode VARCHAR(16) NOT NULL DEFAULT 'always';

UPDATE gjj_crm_data_template
SET display_mode = 'always'
WHERE display_mode IS NULL
   OR BTRIM(display_mode) = ''
   OR display_mode NOT IN ('always', 'filled', 'hidden');

COMMIT;

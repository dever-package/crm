-- Customers may provide only WeChat as contact information, so empty phone values
-- cannot be globally unique.
DROP INDEX IF EXISTS uidx_gjj_crm_customer_phone;
CREATE INDEX IF NOT EXISTS idx_gjj_crm_customer_phone
    ON gjj_crm_customer (phone, id);

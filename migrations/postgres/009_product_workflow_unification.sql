-- Add product categories, customer products and workflow instances before runtime cutover.
CREATE TABLE IF NOT EXISTS gjj_crm_product_category (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_product_category_status_sort
    ON gjj_crm_product_category (status, sort, id);

INSERT INTO gjj_crm_product_category (name, status, sort)
SELECT seed.name, 1, seed.sort
FROM (VALUES
    ('司法推进类', 10),
    ('资产运营类', 20),
    ('债务结构类', 30),
    ('阶段性服务类', 40),
    ('风险处置类', 50),
    ('咨询/预审类', 60)
) AS seed(name, sort)
WHERE NOT EXISTS (
    SELECT 1 FROM gjj_crm_product_category current WHERE current.name = seed.name
);

ALTER TABLE IF EXISTS gjj_crm_product
    ADD COLUMN IF NOT EXISTS category_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS service_workflow_id BIGINT NOT NULL DEFAULT 0;

UPDATE gjj_crm_product product
SET category_id = category.id
FROM gjj_crm_product_category category
WHERE product.category_id = 0
  AND category.name = CASE product.category
      WHEN 'judicial' THEN '司法推进类'
      WHEN 'asset_operation' THEN '资产运营类'
      WHEN 'debt_structure' THEN '债务结构类'
      WHEN 'stage_service' THEN '阶段性服务类'
      WHEN 'risk_disposal' THEN '风险处置类'
      ELSE '咨询/预审类'
  END;

UPDATE gjj_crm_product product
SET category_id = category.id
FROM (
    SELECT id FROM gjj_crm_product_category WHERE status = 1 ORDER BY sort, id LIMIT 1
) category
WHERE product.category_id = 0;

UPDATE gjj_crm_product product
SET service_workflow_id = workflow.id
FROM (
    SELECT id FROM gjj_crm_workflow
    WHERE status = 1 AND default_entry = FALSE AND name IN ('运营流程', '租赁运营流程')
    ORDER BY CASE WHEN name = '租赁运营流程' THEN 0 ELSE 1 END, sort, id
    LIMIT 1
) workflow
WHERE product.service_workflow_id = 0
  AND product.category = 'asset_operation';

CREATE INDEX IF NOT EXISTS idx_gjj_crm_product_category_status
    ON gjj_crm_product (category_id, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_product_service_workflow
    ON gjj_crm_product (service_workflow_id, status, id);

INSERT INTO gjj_crm_data_template_cate (
    id, name, target_table, business_object_type_id, status, sort, created_at
)
SELECT 3, '业务数据', 'business_object', 0, 1, 30, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM gjj_crm_data_template_cate WHERE id = 3);

UPDATE gjj_crm_data_template_cate
SET name = '业务数据', status = 1, sort = 30
WHERE id = 3;

CREATE TABLE IF NOT EXISTS gjj_crm_customer_product (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL,
    asset_id BIGINT NOT NULL DEFAULT 0,
    product_id BIGINT NOT NULL,
    source_workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_customer_product_source_product
    ON gjj_crm_customer_product (source_workflow_instance_id, product_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_customer_product_customer_status
    ON gjj_crm_customer_product (customer_id, asset_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_customer_product_product_status
    ON gjj_crm_customer_product (product_id, status, id);

CREATE TABLE IF NOT EXISTS gjj_crm_workflow_instance (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL,
    asset_id BIGINT NOT NULL DEFAULT 0,
    customer_product_id BIGINT NOT NULL DEFAULT 0,
    workflow_id BIGINT NOT NULL,
    stage_id BIGINT NOT NULL,
    owner_department_id BIGINT NOT NULL DEFAULT 0,
    owner_staff_id BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ NULL,
    terminated_at TIMESTAMPTZ NULL,
    terminated_reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_instance_customer_asset
    ON gjj_crm_workflow_instance (customer_id, asset_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_instance_product_flow
    ON gjj_crm_workflow_instance (customer_product_id, workflow_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_instance_workflow_stage
    ON gjj_crm_workflow_instance (workflow_id, stage_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_workflow_instance_owner_status
    ON gjj_crm_workflow_instance (owner_department_id, owner_staff_id, status, id);

INSERT INTO gjj_crm_workflow_instance (
    id, customer_id, asset_id, customer_product_id, workflow_id, stage_id,
    owner_department_id, owner_staff_id, status, started_at, completed_at,
    terminated_at, terminated_reason, updated_at
)
SELECT progress.id, progress.customer_id, progress.asset_id, 0,
       progress.workflow_id, progress.stage_id, progress.owner_department_id,
       progress.owner_staff_id, progress.status, progress.started_at,
       progress.completed_at, progress.terminated_at,
       COALESCE(progress.terminated_reason, ''), progress.updated_at
FROM gjj_crm_asset_progress progress
WHERE NOT EXISTS (
    SELECT 1 FROM gjj_crm_workflow_instance instance WHERE instance.id = progress.id
);

SELECT setval(
    pg_get_serial_sequence('gjj_crm_workflow_instance', 'id'),
    COALESCE((SELECT MAX(id) FROM gjj_crm_workflow_instance), 1),
    EXISTS (SELECT 1 FROM gjj_crm_workflow_instance)
);

ALTER TABLE IF EXISTS gjj_crm_task_todo
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_operation_log
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_stat_event
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_data_record
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_stat_field_value
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS gjj_crm_finance_ledger
    ADD COLUMN IF NOT EXISTS workflow_instance_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS customer_product_id BIGINT NOT NULL DEFAULT 0;

UPDATE gjj_crm_task_todo todo
SET workflow_instance_id = instance.id,
    customer_product_id = instance.customer_product_id
FROM gjj_crm_workflow_instance instance
WHERE todo.workflow_instance_id = 0
  AND instance.customer_id = todo.customer_id
  AND instance.asset_id = todo.asset_id
  AND instance.workflow_id = todo.workflow_id;

UPDATE gjj_crm_operation_log operation
SET workflow_instance_id = instance.id,
    customer_product_id = instance.customer_product_id
FROM gjj_crm_workflow_instance instance
WHERE operation.workflow_instance_id = 0
  AND instance.customer_id = operation.customer_id
  AND instance.asset_id = operation.asset_id
  AND instance.workflow_id = operation.workflow_id;

UPDATE gjj_crm_stat_event event
SET workflow_instance_id = instance.id,
    customer_product_id = instance.customer_product_id
FROM gjj_crm_workflow_instance instance
WHERE event.workflow_instance_id = 0
  AND instance.customer_id = event.customer_id
  AND instance.asset_id = event.asset_id
  AND instance.workflow_id = event.workflow_id;

DROP INDEX IF EXISTS uidx_gjj_crm_task_todo_stage_task;
DROP INDEX IF EXISTS idx_gjj_crm_task_todo_stage_task;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_task_todo_instance_task
    ON gjj_crm_task_todo (workflow_instance_id, stage_id, task_id);

CREATE INDEX IF NOT EXISTS idx_gjj_crm_task_todo_instance_status
    ON gjj_crm_task_todo (workflow_instance_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_operation_log_instance_time
    ON gjj_crm_operation_log (workflow_instance_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_operation_log_product_time
    ON gjj_crm_operation_log (customer_product_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_stat_event_instance_time
    ON gjj_crm_stat_event (workflow_instance_id, event_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_stat_event_product_time
    ON gjj_crm_stat_event (customer_product_id, event_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_record_instance_template
    ON gjj_crm_data_record (workflow_instance_id, data_template_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_data_record_product_template
    ON gjj_crm_data_record (customer_product_id, data_template_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_stat_field_value_instance_time
    ON gjj_crm_stat_field_value (workflow_instance_id, updated_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_stat_field_value_product_time
    ON gjj_crm_stat_field_value (customer_product_id, updated_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_instance_time
    ON gjj_crm_finance_ledger (workflow_instance_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_finance_ledger_product_time
    ON gjj_crm_finance_ledger (customer_product_id, created_at, id);

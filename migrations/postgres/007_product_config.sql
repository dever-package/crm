CREATE TABLE IF NOT EXISTS gjj_crm_product (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    category VARCHAR(32) NOT NULL DEFAULT 'consulting',
    default_signing_business_type VARCHAR(64) NOT NULL DEFAULT 'manual_review',
    description TEXT NOT NULL DEFAULT '',
    need_pm_review BOOLEAN NOT NULL DEFAULT TRUE,
    need_lawyer_review BOOLEAN NOT NULL DEFAULT FALSE,
    need_ala_review BOOLEAN NOT NULL DEFAULT FALSE,
    need_finance_review BOOLEAN NOT NULL DEFAULT FALSE,
    need_contract_review BOOLEAN NOT NULL DEFAULT TRUE,
    status SMALLINT NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE gjj_crm_product
    ADD COLUMN IF NOT EXISTS code VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS name VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS category VARCHAR(32) NOT NULL DEFAULT 'consulting',
    ADD COLUMN IF NOT EXISTS default_signing_business_type VARCHAR(64) NOT NULL DEFAULT 'manual_review',
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS need_pm_review BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS need_lawyer_review BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS need_ala_review BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS need_finance_review BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS need_contract_review BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS sort INTEGER NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_product_code
    ON gjj_crm_product (code);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_product_status_sort
    ON gjj_crm_product (status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_product_category
    ON gjj_crm_product (category, status, sort, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_product_signing_type
    ON gjj_crm_product (default_signing_business_type, status, id);

INSERT INTO gjj_crm_product (
    code, name, category, default_signing_business_type, description,
    need_pm_review, need_lawyer_review, need_ala_review, need_finance_review, need_contract_review,
    status, sort
) VALUES
    ('S01', '司法节点风险规避服务', 'judicial', 'sealed_asset_service_signing', '查封服务签约/司法节点服务。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 10),
    ('S03', '材料完整性与证据准备服务', 'judicial', 'sealed_asset_service_signing', '查封服务签约/前置材料服务。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 30),
    ('S04', '贷款逾期金额核验服务', 'debt_structure', 'manual_review', '根据P02/P05判断，可进入非查封前置或查封服务。', TRUE, FALSE, FALSE, TRUE, TRUE, 1, 40),
    ('S05', '债务底账与主触发债务识别服务', 'debt_structure', 'manual_review', '根据司法阶段和债务结构判断。', TRUE, FALSE, FALSE, TRUE, TRUE, 1, 50),
    ('S06', '资金缺口与维持成本测算服务', 'debt_structure', 'manual_review', '根据是否已执行/查封判断。', TRUE, FALSE, FALSE, TRUE, TRUE, 1, 60),
    ('S07', '第三方资金可行性评估服务', 'debt_structure', 'manual_review', '需PM、财务、ALA共同判断。', TRUE, FALSE, TRUE, TRUE, TRUE, 1, 70),
    ('S08', '催收材料整理与沟通准备服务', 'judicial', 'manual_review', '多为司法前置，需结合P02判断。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 80),
    ('S09', '银行协商方案准备服务', 'debt_structure', 'manual_review', '多为司法前置，需结合P02判断。', TRUE, FALSE, FALSE, TRUE, TRUE, 1, 90),
    ('S10', '诉前调解准备与协同服务', 'judicial', 'sealed_asset_service_signing', '查封服务签约/司法前置。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 100),
    ('S11', '起诉材料核验与诉讼节点准备服务', 'judicial', 'sealed_asset_service_signing', '查封服务签约/诉讼节点。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 110),
    ('S12', '开庭准备与出庭协同服务', 'judicial', 'sealed_asset_service_signing', '查封服务签约/庭审节点。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 120),
    ('S13', '判决调解履行期管理与执行前预警服务', 'judicial', 'sealed_asset_service_signing', '查封服务签约/执行前。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 130),
    ('S14', '房屋交付条件预审服务', 'asset_operation', 'non_sealed_asset_signing', '非查封资产签约前置。', TRUE, FALSE, TRUE, FALSE, TRUE, 1, 140),
    ('S15', '租赁真实性与风险核验服务', 'asset_operation', 'non_sealed_asset_signing', '非查封资产签约前置/风险核验。', TRUE, TRUE, TRUE, FALSE, TRUE, 1, 150),
    ('S16', '资产运营可行性评估服务', 'asset_operation', 'non_sealed_asset_signing', '非查封资产签约核心前置。', TRUE, FALSE, TRUE, TRUE, TRUE, 1, 160),
    ('S17', '企业代押与担保关系专项预审服务', 'debt_structure', 'manual_review', '需律师、财务复核。', TRUE, TRUE, FALSE, TRUE, TRUE, 1, 170),
    ('S18', '多债权与抵押顺位专项评估服务', 'debt_structure', 'manual_review', '需律师、ALA复核，可能转查封服务。', TRUE, TRUE, TRUE, TRUE, TRUE, 1, 180),
    ('S19', '权属共有人与授权专项核验服务', 'judicial', 'manual_review', '需律师复核，影响两类签约门禁。', TRUE, TRUE, FALSE, FALSE, TRUE, 1, 190),
    ('S20', '执行查封后重评与风险说明服务', 'risk_disposal', 'sealed_asset_service_signing', '查封服务签约。', TRUE, TRUE, TRUE, TRUE, TRUE, 1, 200),
    ('S21', '深处置阶段风险说明与收尾服务', 'risk_disposal', 'sealed_asset_service_signing', '查封服务签约。', TRUE, TRUE, TRUE, TRUE, TRUE, 1, 210),
    ('S22-07', '一换七资产运营权益产品', 'asset_operation', 'non_sealed_asset_signing', '非查封资产运营路径，需ALA强审。', TRUE, FALSE, TRUE, TRUE, TRUE, 1, 220),
    ('S22-13', '二换十三资产运营权益产品', 'asset_operation', 'non_sealed_asset_signing', '非查封资产运营路径，需ALA强审。', TRUE, FALSE, TRUE, TRUE, TRUE, 1, 230),
    ('S23', '执行查封事务协助服务', 'risk_disposal', 'sealed_asset_service_signing', '查封服务签约。', TRUE, TRUE, TRUE, TRUE, TRUE, 1, 240),
    ('S24', '司法拍卖与合法参拍事务协助服务', 'risk_disposal', 'sealed_asset_service_signing', '查封服务签约。', TRUE, TRUE, TRUE, TRUE, TRUE, 1, 250),
    ('S25', '住房保障及变价款租金预留申请协助服务', 'risk_disposal', 'sealed_asset_service_signing', '查封服务签约。', TRUE, TRUE, TRUE, TRUE, TRUE, 1, 260),
    ('S26', '成交腾退与余债收尾服务', 'risk_disposal', 'sealed_asset_service_signing', '查封服务签约。', TRUE, TRUE, TRUE, TRUE, TRUE, 1, 270)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    category = EXCLUDED.category,
    default_signing_business_type = EXCLUDED.default_signing_business_type,
    description = EXCLUDED.description,
    need_pm_review = EXCLUDED.need_pm_review,
    need_lawyer_review = EXCLUDED.need_lawyer_review,
    need_ala_review = EXCLUDED.need_ala_review,
    need_finance_review = EXCLUDED.need_finance_review,
    need_contract_review = EXCLUDED.need_contract_review,
    sort = EXCLUDED.sort,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO gjj_crm_option_set (name, sort, status)
SELECT 'S产品', 130, 1
WHERE NOT EXISTS (
    SELECT 1 FROM gjj_crm_option_set WHERE name = 'S产品'
);

DELETE FROM gjj_crm_option_set_item
WHERE option_set_id = (SELECT id FROM gjj_crm_option_set WHERE name = 'S产品' ORDER BY id LIMIT 1);

INSERT INTO gjj_crm_option_set_item (option_set_id, name, value, sort, status)
SELECT option_set.id, product.name, product.code, product.sort, product.status
FROM gjj_crm_product product
CROSS JOIN LATERAL (
    SELECT id FROM gjj_crm_option_set WHERE name = 'S产品' ORDER BY id LIMIT 1
) option_set
WHERE product.status = 1
ORDER BY product.sort, product.id
ON CONFLICT (option_set_id, value) DO UPDATE SET
    name = EXCLUDED.name,
    sort = EXCLUDED.sort,
    status = EXCLUDED.status;

DO $$
DECLARE
    product_template_id BIGINT;
    product_form_id BIGINT;
    probe_stage_id BIGINT;
    product_task_id BIGINT;
    product_auth_id BIGINT;
BEGIN
    INSERT INTO gjj_crm_data_template (cate_id, name, status, sort)
    SELECT 2, 'NPL产品候选与提交PM', 1, 45
    WHERE NOT EXISTS (
        SELECT 1 FROM gjj_crm_data_template WHERE name = 'NPL产品候选与提交PM'
    );

    SELECT id INTO product_template_id
    FROM gjj_crm_data_template
    WHERE name = 'NPL产品候选与提交PM'
    ORDER BY id
    LIMIT 1;

    UPDATE gjj_crm_data_field
    SET data_template_id = product_template_id,
        parent_field_id = 0,
        sort = CASE field_key
            WHEN 'npl_candidate_s_product_codes' THEN 10
            WHEN 'npl_primary_s_product_code' THEN 20
            WHEN 'npl_s_product_reason_summary' THEN 30
            WHEN 'npl_submit_pm_confirmation_status' THEN 40
            ELSE sort
        END,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE field_key IN (
        'npl_candidate_s_product_codes',
        'npl_primary_s_product_code',
        'npl_s_product_reason_summary',
        'npl_submit_pm_confirmation_status'
    );

    INSERT INTO gjj_crm_form (name, description, sort, status)
    SELECT 'NPL产品候选与提交PM', '十一维探针自动判断后，由NPL选择候选产品并提交PM确认。', 45, 1
    WHERE NOT EXISTS (
        SELECT 1 FROM gjj_crm_form WHERE name = 'NPL产品候选与提交PM'
    );

    SELECT id INTO product_form_id
    FROM gjj_crm_form
    WHERE name = 'NPL产品候选与提交PM'
    ORDER BY id
    LIMIT 1;

    UPDATE gjj_crm_form_field AS ff
    SET form_id = product_form_id,
        data_template_id = product_template_id,
        sort = CASE df.field_key
            WHEN 'npl_candidate_s_product_codes' THEN 10
            WHEN 'npl_primary_s_product_code' THEN 20
            WHEN 'npl_s_product_reason_summary' THEN 30
            WHEN 'npl_submit_pm_confirmation_status' THEN 40
            ELSE ff.sort
        END,
        required = CASE df.field_key
            WHEN 'npl_primary_s_product_code' THEN TRUE
            WHEN 'npl_submit_pm_confirmation_status' THEN TRUE
            ELSE FALSE
        END,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS df
    WHERE ff.data_field_id = df.id
      AND df.field_key IN (
        'npl_candidate_s_product_codes',
        'npl_primary_s_product_code',
        'npl_s_product_reason_summary',
        'npl_submit_pm_confirmation_status'
      );

    SELECT id INTO probe_stage_id
    FROM gjj_crm_stage
    WHERE code = 'S04'
    ORDER BY id
    LIMIT 1;

    INSERT INTO gjj_crm_task (
        stage_id, name, task_type, form_id, trigger_type, trigger_task_id,
        script_id, config_json, sort, status
    )
    SELECT
        probe_stage_id,
        'NPL产品候选与提交PM',
        'form',
        product_form_id,
        'manual',
        0,
        0,
        jsonb_build_object(
            'completion_mode', 'manual',
            'next_stage_code', 'S05',
            'task_points', 1,
            'visible_when', jsonb_build_object(
                'data_field_id', (SELECT id FROM gjj_crm_data_field WHERE field_key = 'signing_business_type_candidate' ORDER BY id LIMIT 1),
                'operator', 'notEmpty'
            )
        )::text,
        65,
        1
    WHERE probe_stage_id IS NOT NULL
      AND product_form_id IS NOT NULL
      AND NOT EXISTS (
        SELECT 1 FROM gjj_crm_task WHERE name = 'NPL产品候选与提交PM'
      );

    SELECT id INTO product_task_id
    FROM gjj_crm_task
    WHERE name = 'NPL产品候选与提交PM'
    ORDER BY id
    LIMIT 1;

    IF product_task_id IS NOT NULL THEN
        UPDATE gjj_crm_task
        SET stage_id = probe_stage_id,
            task_type = 'form',
            form_id = product_form_id,
            trigger_type = 'manual',
            config_json = jsonb_build_object(
                'completion_mode', 'manual',
                'next_stage_code', 'S05',
                'task_points', 1,
                'visible_when', jsonb_build_object(
                    'data_field_id', (SELECT id FROM gjj_crm_data_field WHERE field_key = 'signing_business_type_candidate' ORDER BY id LIMIT 1),
                    'operator', 'notEmpty'
                )
            )::text,
            sort = 65,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = product_task_id;
    END IF;

    INSERT INTO gjj_auth (key, name, icon, path, parent_id, type, sort, source_type, source_name, managed)
    SELECT 'crm/product/list', '产品配置', 'package-check', 'crm/product/list', parent.id, 1, 7, 'page', 'crm', 1
    FROM gjj_auth parent
    WHERE parent.key = 'crm-business-config'
      AND NOT EXISTS (SELECT 1 FROM gjj_auth WHERE key = 'crm/product/list');

    SELECT id INTO product_auth_id
    FROM gjj_auth
    WHERE key = 'crm/product/list'
    ORDER BY id
    LIMIT 1;

    IF product_auth_id IS NOT NULL THEN
        UPDATE gjj_auth
        SET name = '产品配置',
            icon = 'package-check',
            path = 'crm/product/list',
            parent_id = (SELECT id FROM gjj_auth WHERE key = 'crm-business-config' ORDER BY id LIMIT 1),
            type = 1,
            sort = 7
        WHERE id = product_auth_id;

        INSERT INTO gjj_auth (key, name, icon, path, parent_id, type, sort, source_type, source_name, managed)
        SELECT 'crm/product/create', '产品新增', '', 'crm/product/update', product_auth_id, 2, 0, 'page', 'crm', 1
        WHERE NOT EXISTS (SELECT 1 FROM gjj_auth WHERE key = 'crm/product/create');

        INSERT INTO gjj_auth (key, name, icon, path, parent_id, type, sort, source_type, source_name, managed)
        SELECT 'crm/product/update', '产品编辑', '', 'crm/product/update', product_auth_id, 2, 0, 'page', 'crm', 1
        WHERE NOT EXISTS (SELECT 1 FROM gjj_auth WHERE key = 'crm/product/update');

        UPDATE gjj_auth
        SET parent_id = product_auth_id,
            type = 2,
            path = 'crm/product/update'
        WHERE key IN ('crm/product/create', 'crm/product/update');

        INSERT INTO gjj_role_auth (role_id, auth_id)
        SELECT role.id, auth.id
        FROM gjj_role role
        CROSS JOIN gjj_auth auth
        WHERE auth.key IN ('crm/product/list', 'crm/product/create', 'crm/product/update')
          AND NOT EXISTS (
            SELECT 1
            FROM gjj_role_auth existing
            WHERE existing.role_id = role.id
              AND existing.auth_id = auth.id
          );
    END IF;
END $$;

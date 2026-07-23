-- Configure script-driven ALA rent assessment without hard-coding task names at runtime.
BEGIN;

ALTER TABLE gjj_crm_form
    ADD COLUMN IF NOT EXISTS calculation_script_id BIGINT NOT NULL DEFAULT 0;

DO $$
DECLARE
    contract_template_id BIGINT;
    contract_template_cate_id BIGINT;
    ala_form_id BIGINT;
    calculation_cate_id BIGINT;
    selected_calculation_script_id BIGINT;
    assessment_group_id BIGINT;
    selected_option_set_id BIGINT;
    seed RECORD;
    calculation_script TEXT;
BEGIN
    SELECT id, cate_id
    INTO contract_template_id, contract_template_cate_id
    FROM gjj_crm_data_template
    WHERE name = '合同信息'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    SELECT id INTO ala_form_id
    FROM gjj_crm_form
    WHERE name = 'ALA运营条件确认'
      AND status = 1
    ORDER BY id
    LIMIT 1;

    IF contract_template_id IS NULL OR ala_form_id IS NULL THEN
        RAISE EXCEPTION 'ALA租金评估所需的合同信息模板或任务表单未启用';
    END IF;

    SELECT id INTO calculation_cate_id
    FROM gjj_crm_rule_script_cate
    WHERE name = '业务计算'
    ORDER BY id
    LIMIT 1;

    IF calculation_cate_id IS NULL THEN
        INSERT INTO gjj_crm_rule_script_cate (
            name, sort, status, created_at
        ) VALUES (
            '业务计算', 30, 1, CURRENT_TIMESTAMP
        ) RETURNING id INTO calculation_cate_id;
    ELSE
        UPDATE gjj_crm_rule_script_cate
        SET sort = 30,
            status = 1
        WHERE id = calculation_cate_id;
    END IF;

    calculation_script := $script$
function evaluate(input) {
  var form = input && input.form ? input.form : {};

  function numberValue(value) {
    var result = Number(value);
    return isFinite(result) ? result : 0;
  }

  function booleanValue(value) {
    return value === true || value === 1 || value === "1" || value === "true";
  }

  function median(values) {
    var sorted = values.slice().sort(function (left, right) { return left - right; });
    var middle = Math.floor(sorted.length / 2);
    return sorted.length % 2 === 0
      ? (sorted[middle - 1] + sorted[middle]) / 2
      : sorted[middle];
  }

  var rates = {
    level_0: 0,
    level_1: 0.01,
    level_2: 0.02,
    level_3: 0.03,
    level_4: 0.04,
    level_5: 0.05
  };
  var definitions = {
    lease_sale: {
      label: "房屋连租带售",
      field: "ala_rent_lease_sale_level",
      weight: 1,
      rates: { none: 0, triggered: 0.05 },
      descriptions: {
        none: "不属于房屋连租带售项目，或不存在后续出售、看房、处置配合等额外不确定性。",
        triggered: "后续可能需要租户配合看房、售卖、银行或诉讼材料及资产处置事项，直接按5%扣减。"
      }
    },
    furniture: {
      label: "家具家电配置及可用性",
      field: "ala_rent_furniture_level",
      weight: 1.2,
      rates: rates,
      descriptions: {
        level_0: "配置完整且核心家具家电正常可用，达到同类房源正常出租水平。",
        level_1: "基本齐全，仅有轻微老旧、外观磨损或非核心小件缺失。",
        level_2: "存在非核心缺失、可用性不明或多件物品明显老旧。",
        level_3: "存在核心项缺失、损坏或多个非核心配置缺失。",
        level_4: "多个核心项缺失、损坏、成色差或需要维修补配。",
        level_5: "家具家电严重不齐或多数不可用，未处理前难以正常出租。"
      }
    },
    listing: {
      label: "同小区同户型挂租数量",
      field: "ala_rent_listing_level",
      weight: 0.8,
      rates: rates,
      descriptions: {
        level_0: "0-1套有效挂租，竞争压力低。",
        level_1: "2-4套有效挂租，存在轻微竞争。",
        level_2: "约5套有效挂租，竞争开始明显。",
        level_3: "6-9套有效挂租，供应较多。",
        level_4: "10-14套有效挂租，竞争压力较大。",
        level_5: "15套及以上有效挂租，竞争压力很大，需要主管关注。"
      }
    },
    renovation: {
      label: "装修新旧程度",
      field: "ala_rent_renovation_level",
      weight: 1.4,
      rates: rates,
      descriptions: {
        level_0: "装修较新、维护良好，不低于同类房源平均水平。",
        level_1: "有轻微使用痕迹或局部小瑕疵，不影响多数租客判断。",
        level_2: "局部老旧或污损明显，需要清洁或小修。",
        level_3: "整体偏旧，多处老化，展示效果低于同类平均水平。",
        level_4: "明显老旧或存在破损、渗水、霉斑、异味等问题。",
        level_5: "装修破旧或严重污损，未处理前难以按正常价格出租。"
      }
    }
  };

  var validQuotes = [];
  for (var index = 1; index <= 5; index += 1) {
    var amount = numberValue(form["ala_rent_quote_" + index + "_amount"]);
    if (amount > 0 && booleanValue(form["ala_rent_quote_" + index + "_valid"])) {
      validQuotes.push(amount);
    }
  }

  var quoteMedian = validQuotes.length >= 3 ? median(validQuotes) : 0;
  var automaticBase = quoteMedian > 0 ? Math.round(quoteMedian * 0.85) : 0;
  var manualBase = numberValue(form.ala_rent_manual_discounted_base);
  var effectiveBase = manualBase > 0 ? manualBase : automaticBase;
  var calculationMode = manualBase > 0 ? "manual" : "quote";
  var missing = [];
  var items = [];
  var combinedRate = 0;

  if (effectiveBase <= 0) {
    missing.push("请填写手工八五折基准租金，或至少录入3条有效询价");
  }

  Object.keys(definitions).forEach(function (key) {
    var definition = definitions[key];
    var selection = String(form[definition.field] || "");
    var rate = Object.prototype.hasOwnProperty.call(definition.rates, selection)
      ? definition.rates[selection]
      : null;
    if (rate === null) {
      missing.push("请选择" + definition.label);
      items.push({
        key: key,
        label: definition.label,
        rate: null,
        weight: definition.weight,
        weighted_rate: null,
        description: ""
      });
      return;
    }
    var weightedRate = rate * definition.weight;
    combinedRate += weightedRate;
    items.push({
      key: key,
      label: definition.label,
      rate: rate,
      weight: definition.weight,
      weighted_rate: weightedRate,
      description: definition.descriptions[selection] || ""
    });
  });

  var partialFields = {
    ala_rent_quote_median: quoteMedian > 0 ? quoteMedian : "",
    ala_rent_auto_discounted_base: automaticBase > 0 ? automaticBase : "",
    ala_rent_effective_base: effectiveBase > 0 ? effectiveBase : "",
    ala_rent_combined_decay_rate: missing.length === 0 ? combinedRate : "",
    ala_rent_reduction_amount: "",
    ala_rent_review_required: false,
    ala_rent_review_message: "",
    ala_assessed_r_value: "",
    ala_assessed_rent: ""
  };

  if (missing.length > 0) {
    return {
      passed: false,
      reason: missing.join("；"),
      fields: partialFields,
      items: items,
      valid_quote_count: validQuotes.length,
      calculation_mode: calculationMode,
      review_required: false,
      review_reasons: []
    };
  }

  var finalRent = Math.round(effectiveBase * (1 - combinedRate) / 10) * 10;
  var reductionAmount = effectiveBase - finalRent;
  var reviewReasons = [];
  if (combinedRate > 0.15) {
    reviewReasons.push("综合衰减率超过15%");
  }
  if (String(form.ala_rent_lease_sale_level || "") === "triggered") {
    reviewReasons.push("触发房屋连租带售5%特殊扣减");
  }
  if (String(form.ala_rent_listing_level || "") === "level_5") {
    reviewReasons.push("同小区同户型有效挂租达到15套及以上");
  }
  if (manualBase > 0 && validQuotes.length < 3) {
    reviewReasons.push("采用手工八五折基准且有效询价不足3条");
  }
  var reviewMessage = reviewReasons.length > 0
    ? "需主管复核：" + reviewReasons.join("；")
    : "可作为初步销售计提金额，仍需按流程复核归档";

  return {
    passed: true,
    value: String(finalRent),
    reason: reviewMessage,
    fields: {
      ala_rent_quote_median: quoteMedian > 0 ? quoteMedian : "",
      ala_rent_auto_discounted_base: automaticBase > 0 ? automaticBase : "",
      ala_rent_effective_base: effectiveBase,
      ala_rent_combined_decay_rate: combinedRate,
      ala_rent_reduction_amount: reductionAmount,
      ala_rent_review_required: reviewReasons.length > 0,
      ala_rent_review_message: reviewMessage,
      ala_assessed_r_value: finalRent,
      ala_assessed_rent: finalRent
    },
    items: items,
    valid_quote_count: validQuotes.length,
    calculation_mode: calculationMode,
    review_required: reviewReasons.length > 0,
    review_reasons: reviewReasons
  };
}
$script$;

    SELECT id INTO selected_calculation_script_id
    FROM gjj_crm_rule_script
    WHERE name = 'ALA销售计提租金测算'
    ORDER BY id
    LIMIT 1;

    IF selected_calculation_script_id IS NULL THEN
        INSERT INTO gjj_crm_rule_script (
            cate_id, name, description, script,
            status, sort, created_at, updated_at
        ) VALUES (
            calculation_cate_id,
            'ALA销售计提租金测算',
            '根据有效询价、八五折基准和四项加权衰减计算ALA评估R值。',
            calculation_script,
            1, 100, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO selected_calculation_script_id;
    ELSE
        UPDATE gjj_crm_rule_script
        SET cate_id = calculation_cate_id,
            description = '根据有效询价、八五折基准和四项加权衰减计算ALA评估R值。',
            script = calculation_script,
            status = 1,
            sort = 100,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = selected_calculation_script_id;
    END IF;

    FOR seed IN
        SELECT * FROM (VALUES
            ('ALA房屋连租带售评估', 100),
            ('ALA家具家电评估', 110),
            ('ALA同户型挂租评估', 120),
            ('ALA装修程度评估', 130)
        ) AS option_seed(name, sort)
    LOOP
        SELECT id INTO selected_option_set_id
        FROM gjj_crm_option_set
        WHERE name = seed.name
        ORDER BY id
        LIMIT 1;

        IF selected_option_set_id IS NULL THEN
            INSERT INTO gjj_crm_option_set (
                name, sort, status, created_at, updated_at
            ) VALUES (
                seed.name, seed.sort, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
            );
        ELSE
            UPDATE gjj_crm_option_set
            SET sort = seed.sort,
                status = 1,
                updated_at = CURRENT_TIMESTAMP
            WHERE id = selected_option_set_id;
        END IF;
    END LOOP;

    WITH option_seed(option_set_name, name, value, sort) AS (
        VALUES
            ('ALA房屋连租带售评估', '0%｜不属于房屋连租带售', 'none', 10),
            ('ALA房屋连租带售评估', '5%｜触发房屋连租带售', 'triggered', 20),
            ('ALA家具家电评估', '0%｜配置完整且正常可用', 'level_0', 10),
            ('ALA家具家电评估', '1%｜基本齐全轻微老旧', 'level_1', 20),
            ('ALA家具家电评估', '2%｜非核心缺失/可用性不明', 'level_2', 30),
            ('ALA家具家电评估', '3%｜核心项缺失或损坏', 'level_3', 40),
            ('ALA家具家电评估', '4%｜多个核心项缺失/需维修', 'level_4', 50),
            ('ALA家具家电评估', '5%｜严重不齐或多数不可用', 'level_5', 60),
            ('ALA同户型挂租评估', '0%｜0-1套有效挂租', 'level_0', 10),
            ('ALA同户型挂租评估', '1%｜2-4套有效挂租', 'level_1', 20),
            ('ALA同户型挂租评估', '2%｜约5套有效挂租', 'level_2', 30),
            ('ALA同户型挂租评估', '3%｜6-9套有效挂租', 'level_3', 40),
            ('ALA同户型挂租评估', '4%｜10-14套有效挂租', 'level_4', 50),
            ('ALA同户型挂租评估', '5%｜15套及以上有效挂租', 'level_5', 60),
            ('ALA装修程度评估', '0%｜装修较新/优于均值', 'level_0', 10),
            ('ALA装修程度评估', '1%｜轻微使用痕迹', 'level_1', 20),
            ('ALA装修程度评估', '2%｜局部老旧/需小修', 'level_2', 30),
            ('ALA装修程度评估', '3%｜整体偏旧', 'level_3', 40),
            ('ALA装修程度评估', '4%｜明显老旧/需维修清洁', 'level_4', 50),
            ('ALA装修程度评估', '5%｜破旧/严重影响出租', 'level_5', 60)
    )
    INSERT INTO gjj_crm_option_set_item (
        option_set_id, name, value, sort, status
    )
    SELECT option_set.id, option_seed.name, option_seed.value, option_seed.sort, 1
    FROM option_seed
    INNER JOIN gjj_crm_option_set AS option_set
        ON option_set.name = option_seed.option_set_name
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_option_set_item AS existing
        WHERE existing.option_set_id = option_set.id
          AND existing.value = option_seed.value
    );

    WITH option_seed(option_set_name, name, value, sort) AS (
        VALUES
            ('ALA房屋连租带售评估', '0%｜不属于房屋连租带售', 'none', 10),
            ('ALA房屋连租带售评估', '5%｜触发房屋连租带售', 'triggered', 20),
            ('ALA家具家电评估', '0%｜配置完整且正常可用', 'level_0', 10),
            ('ALA家具家电评估', '1%｜基本齐全轻微老旧', 'level_1', 20),
            ('ALA家具家电评估', '2%｜非核心缺失/可用性不明', 'level_2', 30),
            ('ALA家具家电评估', '3%｜核心项缺失或损坏', 'level_3', 40),
            ('ALA家具家电评估', '4%｜多个核心项缺失/需维修', 'level_4', 50),
            ('ALA家具家电评估', '5%｜严重不齐或多数不可用', 'level_5', 60),
            ('ALA同户型挂租评估', '0%｜0-1套有效挂租', 'level_0', 10),
            ('ALA同户型挂租评估', '1%｜2-4套有效挂租', 'level_1', 20),
            ('ALA同户型挂租评估', '2%｜约5套有效挂租', 'level_2', 30),
            ('ALA同户型挂租评估', '3%｜6-9套有效挂租', 'level_3', 40),
            ('ALA同户型挂租评估', '4%｜10-14套有效挂租', 'level_4', 50),
            ('ALA同户型挂租评估', '5%｜15套及以上有效挂租', 'level_5', 60),
            ('ALA装修程度评估', '0%｜装修较新/优于均值', 'level_0', 10),
            ('ALA装修程度评估', '1%｜轻微使用痕迹', 'level_1', 20),
            ('ALA装修程度评估', '2%｜局部老旧/需小修', 'level_2', 30),
            ('ALA装修程度评估', '3%｜整体偏旧', 'level_3', 40),
            ('ALA装修程度评估', '4%｜明显老旧/需维修清洁', 'level_4', 50),
            ('ALA装修程度评估', '5%｜破旧/严重影响出租', 'level_5', 60)
    )
    UPDATE gjj_crm_option_set_item AS item
    SET name = option_seed.name,
        sort = option_seed.sort,
        status = 1
    FROM option_seed
    INNER JOIN gjj_crm_option_set AS option_set
        ON option_set.name = option_seed.option_set_name
    WHERE item.option_set_id = option_set.id
      AND item.value = option_seed.value;

    SELECT id INTO assessment_group_id
    FROM gjj_crm_data_field
    WHERE field_key = 'ala_rent_assessment'
    ORDER BY id
    LIMIT 1;

    IF assessment_group_id IS NULL THEN
        INSERT INTO gjj_crm_data_field (
            data_template_id, parent_field_id, option_set_id,
            name, field_key, field_type, default_value,
            finance_type_id, stat_enabled, sort, status,
            created_at, updated_at
        ) VALUES (
            contract_template_id, 0, 0,
            'ALA租金评估', 'ala_rent_assessment', 'group', '',
            0, FALSE, 185, 1,
            CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        ) RETURNING id INTO assessment_group_id;
    ELSE
        UPDATE gjj_crm_data_field
        SET data_template_id = contract_template_id,
            parent_field_id = 0,
            option_set_id = 0,
            name = 'ALA租金评估',
            field_type = 'group',
            default_value = '',
            finance_type_id = 0,
            stat_enabled = FALSE,
            sort = 185,
            status = 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = assessment_group_id;
    END IF;

    FOR seed IN
        SELECT * FROM (VALUES
            ('ala_rent_manual_discounted_base', '手工八五折基准租金', 'money', '', '', 10, FALSE),
            ('ala_rent_quote_1_platform', '询价1中介/平台', 'text', '', '', 20, FALSE),
            ('ala_rent_quote_1_reference', '询价1参考房源', 'text', '', '', 21, FALSE),
            ('ala_rent_quote_1_amount', '询价1报价金额', 'money', '', '', 22, FALSE),
            ('ala_rent_quote_1_valid', '询价1是否有效', 'boolean', 'true', '', 23, FALSE),
            ('ala_rent_quote_1_remark', '询价1备注', 'text', '', '', 24, FALSE),
            ('ala_rent_quote_2_platform', '询价2中介/平台', 'text', '', '', 30, FALSE),
            ('ala_rent_quote_2_reference', '询价2参考房源', 'text', '', '', 31, FALSE),
            ('ala_rent_quote_2_amount', '询价2报价金额', 'money', '', '', 32, FALSE),
            ('ala_rent_quote_2_valid', '询价2是否有效', 'boolean', 'true', '', 33, FALSE),
            ('ala_rent_quote_2_remark', '询价2备注', 'text', '', '', 34, FALSE),
            ('ala_rent_quote_3_platform', '询价3中介/平台', 'text', '', '', 40, FALSE),
            ('ala_rent_quote_3_reference', '询价3参考房源', 'text', '', '', 41, FALSE),
            ('ala_rent_quote_3_amount', '询价3报价金额', 'money', '', '', 42, FALSE),
            ('ala_rent_quote_3_valid', '询价3是否有效', 'boolean', 'true', '', 43, FALSE),
            ('ala_rent_quote_3_remark', '询价3备注', 'text', '', '', 44, FALSE),
            ('ala_rent_quote_4_platform', '询价4中介/平台', 'text', '', '', 50, FALSE),
            ('ala_rent_quote_4_reference', '询价4参考房源', 'text', '', '', 51, FALSE),
            ('ala_rent_quote_4_amount', '询价4报价金额', 'money', '', '', 52, FALSE),
            ('ala_rent_quote_4_valid', '询价4是否有效', 'boolean', 'true', '', 53, FALSE),
            ('ala_rent_quote_4_remark', '询价4备注', 'text', '', '', 54, FALSE),
            ('ala_rent_quote_5_platform', '询价5中介/平台', 'text', '', '', 60, FALSE),
            ('ala_rent_quote_5_reference', '询价5参考房源', 'text', '', '', 61, FALSE),
            ('ala_rent_quote_5_amount', '询价5报价金额', 'money', '', '', 62, FALSE),
            ('ala_rent_quote_5_valid', '询价5是否有效', 'boolean', 'true', '', 63, FALSE),
            ('ala_rent_quote_5_remark', '询价5备注', 'text', '', '', 64, FALSE),
            ('ala_rent_lease_sale_level', '房屋连租带售', 'select', '', 'ALA房屋连租带售评估', 100, FALSE),
            ('ala_rent_furniture_level', '家具家电配置及可用性', 'select', '', 'ALA家具家电评估', 110, FALSE),
            ('ala_rent_listing_level', '同小区同户型挂租数量', 'select', '', 'ALA同户型挂租评估', 120, FALSE),
            ('ala_rent_renovation_level', '装修新旧程度', 'select', '', 'ALA装修程度评估', 130, FALSE),
            ('ala_rent_quote_median', '基础询价租金', 'money', '', '', 200, FALSE),
            ('ala_rent_auto_discounted_base', '自动八五折基准租金', 'money', '', '', 210, FALSE),
            ('ala_rent_effective_base', '实际八五折基准租金', 'money', '', '', 220, FALSE),
            ('ala_rent_combined_decay_rate', '综合衰减率', 'number', '', '', 230, TRUE),
            ('ala_rent_reduction_amount', '预计核减金额', 'money', '', '', 240, TRUE),
            ('ala_rent_review_required', '需要主管复核', 'boolean', 'false', '', 250, FALSE),
            ('ala_rent_review_message', '复核提示', 'textarea', '', '', 260, FALSE),
            ('ala_assessed_r_value', 'ALA评估R值', 'money', '', '', 270, TRUE),
            ('ala_assessed_rent', '评估租金（元）', 'money', '', '', 280, TRUE)
        ) AS field_seed(field_key, name, field_type, default_value, option_set_name, sort, stat_enabled)
    LOOP
        SELECT id INTO selected_option_set_id
        FROM gjj_crm_option_set
        WHERE name = seed.option_set_name
        ORDER BY id
        LIMIT 1;

        IF seed.option_set_name = '' THEN
            selected_option_set_id := 0;
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM gjj_crm_data_field WHERE field_key = seed.field_key
        ) THEN
            INSERT INTO gjj_crm_data_field (
                data_template_id, parent_field_id, option_set_id,
                name, field_key, field_type, default_value,
                finance_type_id, stat_enabled, sort, status,
                created_at, updated_at
            ) VALUES (
                contract_template_id, assessment_group_id, selected_option_set_id,
                seed.name, seed.field_key, seed.field_type, seed.default_value,
                0, seed.stat_enabled, seed.sort, 1,
                CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
            );
        ELSE
            UPDATE gjj_crm_data_field
            SET data_template_id = contract_template_id,
                parent_field_id = assessment_group_id,
                option_set_id = selected_option_set_id,
                name = seed.name,
                field_type = seed.field_type,
                default_value = seed.default_value,
                finance_type_id = 0,
                stat_enabled = seed.stat_enabled,
                sort = seed.sort,
                status = 1,
                updated_at = CURRENT_TIMESTAMP
            WHERE field_key = seed.field_key;
        END IF;
    END LOOP;

    UPDATE gjj_crm_form_field AS form_field
    SET status = 2,
        updated_at = CURRENT_TIMESTAMP
    FROM gjj_crm_data_field AS data_field
    WHERE form_field.form_id = ala_form_id
      AND form_field.data_field_id = data_field.id
      AND data_field.parent_field_id = assessment_group_id;

    INSERT INTO gjj_crm_form_field (
        form_id, data_template_cate_id, data_template_id,
        field_source, field_path, main_field, data_field_id,
        name, required, readonly,
        visible_when_field_id, visible_when_operator, visible_when_value,
        sort, status, created_at, updated_at
    )
    SELECT
        ala_form_id, contract_template_cate_id, contract_template_id,
        'data:' || assessment_group_id,
        json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || assessment_group_id
        )::TEXT,
        '', assessment_group_id,
        'ALA租金评估', FALSE, FALSE,
        0, '', '',
        10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1
        FROM gjj_crm_form_field AS existing
        WHERE existing.form_id = ala_form_id
          AND existing.data_field_id = assessment_group_id
    );

    UPDATE gjj_crm_form_field
    SET data_template_cate_id = contract_template_cate_id,
        data_template_id = contract_template_id,
        field_source = 'data:' || assessment_group_id,
        field_path = json_build_array(
            'cate:' || contract_template_cate_id,
            'template:' || contract_template_id,
            'data:' || assessment_group_id
        )::TEXT,
        main_field = '',
        name = 'ALA租金评估',
        required = FALSE,
        readonly = FALSE,
        visible_when_field_id = 0,
        visible_when_operator = '',
        visible_when_value = '',
        sort = 10,
        status = 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE form_id = ala_form_id
      AND data_field_id = assessment_group_id;

    UPDATE gjj_crm_form
    SET calculation_script_id = selected_calculation_script_id,
        description = 'ALA按有效询价、八五折基准和四项衰减测算R值，最终金额同步为评估租金。',
        updated_at = CURRENT_TIMESTAMP
    WHERE id = ala_form_id;
END $$;

COMMIT;

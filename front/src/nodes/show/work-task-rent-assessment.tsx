import { AlertTriangle, Calculator, CheckCircle2, Loader2 } from "lucide-react";

import {
  displayText,
  setWorkStoreValue,
  textValue,
  workIsRecord,
  workStoreValue,
  workTaskCalculationPath,
  workTaskFormDataPath,
  type WorkFormCalculationState,
  type WorkStoreLike,
  type WorkTaskFormField,
} from "./work-core";
import {
  emptyWorkTaskRecord,
  WorkTaskFieldControl,
  useWorkTaskStoreValue,
} from "./work-task-form-fields";

export const alaRentAssessmentGroupKey = "ala_rent_assessment";

const quoteRows = [1, 2, 3, 4, 5];
const decayRows = [
  {
    key: "lease_sale",
    fieldKey: "ala_rent_lease_sale_level",
    label: "房屋连租带售",
  },
  {
    key: "furniture",
    fieldKey: "ala_rent_furniture_level",
    label: "家具家电配置及可用性",
  },
  {
    key: "listing",
    fieldKey: "ala_rent_listing_level",
    label: "同小区同户型挂租数量",
  },
  {
    key: "renovation",
    fieldKey: "ala_rent_renovation_level",
    label: "装修新旧程度",
  },
];

export function WorkTaskRentAssessment({
  fields,
  store,
}: {
  fields: WorkTaskFormField[];
  store?: WorkStoreLike;
}) {
  const formValues = useWorkTaskStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    emptyWorkTaskRecord,
  );
  const calculation = useWorkTaskStoreValue<WorkFormCalculationState>(
    store,
    workTaskCalculationPath,
    { status: "idle" },
  );
  const fieldsByKey = new Map(
    fields.map((field) => [textValue(field.meta?.["dataFieldKey"]), field]),
  );
  const result = workIsRecord(calculation.result) ? calculation.result : {};
  const itemResults = assessmentItems(result["items"]);
  const reviewRequired = Boolean(result["review_required"]);
  const reviewMessage =
    textValue(calculation.reason) ||
    assessmentValue(fieldsByKey, formValues, "ala_rent_review_message");
  const validQuoteCount = Number(result["valid_quote_count"] || 0);
  const calculationMode = textValue(result["calculation_mode"]);

  return (
    <div className="min-w-0 space-y-7">
      <section className="space-y-4">
        <div className="flex flex-wrap items-end justify-between gap-3 border-b border-border/70 pb-3">
          <div>
            <h3 className="text-base font-semibold text-foreground">询价基准</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              至少录入 3 条有效报价自动取中位数；手工八五折金额优先。
            </p>
          </div>
          <div className="text-sm text-muted-foreground">
            有效询价 <span className="font-semibold text-foreground">{validQuoteCount}</span> / 3
          </div>
        </div>

        <div className="grid gap-2 md:max-w-md">
          <AssessmentLabel label="手工八五折基准租金" optional />
          <AssessmentControl
            field={fieldsByKey.get("ala_rent_manual_discounted_base")}
            store={store}
          />
          <p className="text-xs text-muted-foreground">
            已有八五折金额时可直接填写；留空则按有效询价自动计算。
          </p>
        </div>

        <div className="overflow-x-auto rounded-md border border-border/80">
          <table className="w-full min-w-[920px] table-fixed text-sm">
            <colgroup>
              <col className="w-12" />
              <col className="w-40" />
              <col className="w-64" />
              <col className="w-40" />
              <col className="w-24" />
              <col />
            </colgroup>
            <thead className="bg-muted/45 text-left text-xs font-medium text-muted-foreground">
              <tr>
                <th className="px-3 py-3 text-center">序号</th>
                <th className="px-3 py-3">中介/平台</th>
                <th className="px-3 py-3">参考房源/同户型说明</th>
                <th className="px-3 py-3">报价金额</th>
                <th className="px-3 py-3 text-center">有效</th>
                <th className="px-3 py-3">备注/截图说明</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/70">
              {quoteRows.map((row) => (
                <QuoteRow
                  key={row}
                  row={row}
                  fieldsByKey={fieldsByKey}
                  formValues={formValues}
                  store={store}
                />
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="space-y-4">
        <div className="border-b border-border/70 pb-3">
          <h3 className="text-base font-semibold text-foreground">衰减评估</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            选择等级后由脚本返回扣减率、权重和判断说明。
          </p>
        </div>
        <div className="overflow-x-auto rounded-md border border-border/80">
          <table className="w-full min-w-[980px] table-fixed text-sm">
            <colgroup>
              <col className="w-52" />
              <col className="w-72" />
              <col className="w-24" />
              <col className="w-20" />
              <col className="w-28" />
              <col />
            </colgroup>
            <thead className="bg-muted/45 text-left text-xs font-medium text-muted-foreground">
              <tr>
                <th className="px-3 py-3">衰减项目</th>
                <th className="px-3 py-3">等级选择</th>
                <th className="px-3 py-3 text-right">扣减率</th>
                <th className="px-3 py-3 text-right">权重</th>
                <th className="px-3 py-3 text-right">加权扣减率</th>
                <th className="px-3 py-3">判断说明</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/70">
              {decayRows.map((row) => {
                const item = itemResults.get(row.key);
                return (
                  <tr key={row.key} className="align-top">
                    <td className="px-3 py-4 font-medium text-foreground">
                      {row.label}
                    </td>
                    <td className="px-3 py-3">
                      <AssessmentControl
                        field={fieldsByKey.get(row.fieldKey)}
                        store={store}
                      />
                    </td>
                    <td className="px-3 py-4 text-right tabular-nums">
                      {formatPercent(item?.rate)}
                    </td>
                    <td className="px-3 py-4 text-right tabular-nums">
                      {formatWeight(item?.weight)}
                    </td>
                    <td className="px-3 py-4 text-right font-medium tabular-nums">
                      {formatPercent(item?.weighted_rate)}
                    </td>
                    <td className="px-3 py-4 text-sm leading-6 text-muted-foreground">
                      {displayText(item?.description, "选择后显示判断说明")}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/70 pb-3">
          <div>
            <h3 className="text-base font-semibold text-foreground">测算结果</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {calculationMode === "manual"
                ? "当前采用手工八五折基准租金"
                : "当前采用有效询价中位数的八五折金额"}
            </p>
          </div>
          <CalculationStatus calculation={calculation} />
        </div>

        <div className="grid overflow-hidden rounded-md border border-border/80 sm:grid-cols-2 xl:grid-cols-5">
          <ResultValue
            label="询价中位数"
            value={formatMoney(
              assessmentValue(fieldsByKey, formValues, "ala_rent_quote_median"),
            )}
          />
          <ResultValue
            label="实际八五折基准"
            value={formatMoney(
              assessmentValue(fieldsByKey, formValues, "ala_rent_effective_base"),
            )}
          />
          <ResultValue
            label="综合衰减率"
            value={formatPercent(
              assessmentValue(
                fieldsByKey,
                formValues,
                "ala_rent_combined_decay_rate",
              ),
            )}
          />
          <ResultValue
            label="预计核减金额"
            value={formatMoney(
              assessmentValue(
                fieldsByKey,
                formValues,
                "ala_rent_reduction_amount",
              ),
            )}
          />
          <ResultValue
            label="ALA评估R值"
            value={formatMoney(
              assessmentValue(fieldsByKey, formValues, "ala_assessed_r_value"),
            )}
            emphasis
          />
        </div>

        {calculation.status === "success" &&
        (reviewRequired || reviewMessage) ? (
          <div
            className={`flex items-start gap-3 rounded-md border px-4 py-3 ${
              reviewRequired
                ? "border-amber-300 bg-amber-50 text-amber-950"
                : "border-emerald-200 bg-emerald-50 text-emerald-950"
            }`}
          >
            {reviewRequired ? (
              <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
            ) : (
              <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0" />
            )}
            <div>
              <div className="text-sm font-semibold">
                {reviewRequired ? "需要主管复核" : "测算条件完整"}
              </div>
              <p className="mt-1 whitespace-pre-wrap text-sm leading-6 opacity-80">
                {reviewMessage}
              </p>
            </div>
          </div>
        ) : null}
      </section>
    </div>
  );
}

function QuoteRow({
  row,
  fieldsByKey,
  formValues,
  store,
}: {
  row: number;
  fieldsByKey: Map<string, WorkTaskFormField>;
  formValues: Record<string, unknown>;
  store?: WorkStoreLike;
}) {
  const prefix = `ala_rent_quote_${row}`;
  const validField = fieldsByKey.get(`${prefix}_valid`);
  const checked = validField
    ? assessmentBoolean(formValues[validField.formKey])
    : false;
  return (
    <tr className="align-middle">
      <td className="px-3 py-3 text-center font-medium text-muted-foreground">
        {row}
      </td>
      {["platform", "reference", "amount"].map((suffix) => (
        <td key={suffix} className="px-3 py-2.5">
          <AssessmentControl
            field={fieldsByKey.get(`${prefix}_${suffix}`)}
            store={store}
          />
        </td>
      ))}
      <td className="px-3 py-2.5 text-center">
        {validField ? (
          <label
            className="inline-flex cursor-pointer items-center justify-center gap-2 text-sm"
            data-work-form-key={validField.formKey}
          >
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-border accent-primary"
              checked={checked}
              onChange={(event) =>
                setAssessmentFieldValue(
                  store,
                  validField,
                  event.currentTarget.checked,
                )
              }
            />
            <span className="text-muted-foreground">计入</span>
          </label>
        ) : (
          "-"
        )}
      </td>
      <td className="px-3 py-2.5">
        <AssessmentControl
          field={fieldsByKey.get(`${prefix}_remark`)}
          store={store}
        />
      </td>
    </tr>
  );
}

function AssessmentControl({
  field,
  store,
}: {
  field?: WorkTaskFormField;
  store?: WorkStoreLike;
}) {
  if (!field) {
    return <span className="text-sm text-muted-foreground">未配置字段</span>;
  }
  return (
    <div data-work-form-key={field.formKey} tabIndex={-1}>
      <WorkTaskFieldControl field={field} store={store} />
    </div>
  );
}

function AssessmentLabel({
  label,
  optional = false,
}: {
  label: string;
  optional?: boolean;
}) {
  return (
    <div className="flex items-center gap-2 text-sm font-medium text-foreground">
      <span>{label}</span>
      {optional ? (
        <span className="text-xs font-normal text-muted-foreground">可选</span>
      ) : null}
    </div>
  );
}

function ResultValue({
  label,
  value,
  emphasis = false,
}: {
  label: string;
  value: string;
  emphasis?: boolean;
}) {
  return (
    <div className="border-b border-border/70 px-4 py-4 last:border-b-0 sm:border-r xl:border-b-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className={`mt-1.5 tabular-nums ${
          emphasis
            ? "text-xl font-semibold text-foreground"
            : "text-base font-medium text-foreground"
        }`}
      >
        {value}
      </div>
    </div>
  );
}

function CalculationStatus({
  calculation,
}: {
  calculation: WorkFormCalculationState;
}) {
  if (calculation.status === "calculating") {
    return (
      <span className="inline-flex items-center gap-1.5 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        计算中
      </span>
    );
  }
  if (calculation.status === "error") {
    return (
      <span className="inline-flex items-center gap-1.5 text-sm text-destructive">
        <AlertTriangle className="h-4 w-4" />
        {displayText(calculation.error, "计算失败")}
      </span>
    );
  }
  if (calculation.status === "incomplete") {
    return (
      <span className="inline-flex items-center gap-1.5 text-sm text-amber-700">
        <AlertTriangle className="h-4 w-4" />
        {displayText(calculation.reason, "请补充测算信息")}
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1.5 text-sm text-muted-foreground">
      <Calculator className="h-4 w-4" />
      自动测算
    </span>
  );
}

function setAssessmentFieldValue(
  store: WorkStoreLike | undefined,
  field: WorkTaskFormField,
  value: unknown,
) {
  const current = workStoreValue<Record<string, unknown>>(
    store,
    workTaskFormDataPath,
    emptyWorkTaskRecord,
  );
  setWorkStoreValue(store, workTaskFormDataPath, {
    ...current,
    [field.formKey]: value,
  });
}

function assessmentValue(
  fieldsByKey: Map<string, WorkTaskFormField>,
  values: Record<string, unknown>,
  fieldKey: string,
): unknown {
  const field = fieldsByKey.get(fieldKey);
  return field ? values[field.formKey] : "";
}

function assessmentBoolean(value: unknown): boolean {
  return value === true || value === 1 || value === "1" || value === "true";
}

function assessmentItems(value: unknown): Map<string, Record<string, unknown>> {
  const result = new Map<string, Record<string, unknown>>();
  if (!Array.isArray(value)) return result;
  for (const item of value) {
    if (!workIsRecord(item)) continue;
    const key = textValue(item["key"]);
    if (key) result.set(key, item);
  }
  return result;
}

function formatMoney(value: unknown): string {
  const amount = Number(value);
  if (!Number.isFinite(amount) || amount <= 0) return "-";
  return `¥${new Intl.NumberFormat("zh-CN", {
    maximumFractionDigits: 0,
  }).format(amount)}`;
}

function formatPercent(value: unknown): string {
  if (value === "" || value === null || value === undefined) return "-";
  const rate = Number(value);
  if (!Number.isFinite(rate)) return "-";
  return `${(rate * 100).toFixed(rate === 0 ? 0 : 1)}%`;
}

function formatWeight(value: unknown): string {
  if (value === "" || value === null || value === undefined) return "-";
  const weight = Number(value);
  if (!Number.isFinite(weight)) return "-";
  return weight.toFixed(1);
}

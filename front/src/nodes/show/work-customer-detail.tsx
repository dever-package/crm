import { useEffect, useMemo, useState } from "react";
import {
  Download,
  FileText,
  Loader2,
  Paperclip,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { downloadUploadFile, type UploadFileItem } from "@/lib/upload";
import { normalizeUploadItems } from "@/lib/resource";

import {
  displayText,
  type WorkAsset,
  type WorkCustomer,
  type WorkDetailField,
  type WorkDetailSection,
  type WorkOperation,
} from "./work-core";
import { WorkTaskUploadPreviewDialog } from "./work-upload";

export type WorkCustomerFlowEntryView = {
  id: string;
  title: string;
  description: string;
  badge: string;
  badgeClassName: string;
  dotClassName: string;
  stageName: string;
  operatorName: string;
  time: string;
  operation: WorkOperation;
};

export type WorkCustomerOperationScope = "all" | "mine";
export type WorkDetailTab = "info" | "records";

const operationScopeOptions: Array<{
  value: WorkCustomerOperationScope;
  label: string;
}> = [
  { value: "all", label: "全部记录" },
  { value: "mine", label: "我的记录" },
];

export function WorkCustomerDetailStyles() {
  return (
    <style>{`
      .crm-customer-detail-data-grid {
        display: grid;
        grid-template-columns: 184px minmax(0, 1fr);
        gap: 20px;
      }

      .crm-customer-detail-section-nav {
        align-self: start;
        grid-auto-rows: 56px;
      }

      .crm-customer-detail-section-nav button {
        height: 56px;
        min-height: 56px;
        max-height: 56px;
      }

      .crm-customer-detail-fields {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        column-gap: 24px;
      }

      .crm-customer-flow-dot {
        left: -27px;
      }

      @media (max-width: 767px) {
        .crm-customer-detail-data-grid {
          grid-template-columns: minmax(0, 1fr);
          gap: 14px;
        }

        .crm-customer-detail-section-nav {
          display: flex;
          overflow-x: auto;
          padding-bottom: 4px;
        }

        .crm-customer-detail-section-nav button {
          min-width: 148px;
        }

        .crm-customer-detail-fields {
          grid-template-columns: minmax(0, 1fr);
        }

      }
    `}</style>
  );
}

export function WorkDetailTabs({
  activeTab,
  onChange,
}: {
  activeTab: WorkDetailTab;
  onChange: (tab: WorkDetailTab) => void;
}) {
  const tabs: Array<{ key: WorkDetailTab; label: string }> = [
    { key: "records", label: "记录" },
    { key: "info", label: "资料" },
  ];

  return (
    <div className="border-b border-border/70">
      <div className="flex gap-1">
        {tabs.map((tab) => (
          <Button
            type="button"
            key={tab.key}
            variant="ghost"
            aria-pressed={activeTab === tab.key}
            className={`h-auto rounded-none border-b-2 px-1.5 py-3 text-sm font-medium ${
              activeTab === tab.key
                ? "border-primary text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => onChange(tab.key)}
          >
            {tab.label}
          </Button>
        ))}
      </div>
    </div>
  );
}

export function workDetailValueEmpty(value: unknown): boolean {
  if (value === null || value === undefined || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  return false;
}

export function WorkCustomerDetailData({
  customer,
  asset,
  sections,
}: {
  customer: WorkCustomer;
  asset?: WorkAsset;
  sections: WorkDetailSection[];
}) {
  const allSections = useMemo(
    () => workCustomerDetailSections(customer, asset, sections),
    [asset, customer, sections],
  );

  return <WorkDetailSectionsData sections={allSections} />;
}

export function WorkDetailSectionsData({
  sections,
}: {
  sections: WorkDetailSection[];
}) {
  const allSections = sections;
  const [activeSectionID, setActiveSectionID] = useState(
    allSections[0]?.id || "",
  );
  const [previewFile, setPreviewFile] = useState<UploadFileItem | null>(null);

  useEffect(() => {
    if (!allSections.some((section) => section.id === activeSectionID)) {
      setActiveSectionID(allSections[0]?.id || "");
    }
  }, [activeSectionID, allSections]);

  const activeSection =
    allSections.find((section) => section.id === activeSectionID) ||
    allSections[0];

  if (!activeSection) {
    return <div className="py-10 text-center text-sm text-muted-foreground">暂无资料</div>;
  }

  return (
    <div className="crm-customer-detail-data-grid">
      <nav className="crm-customer-detail-section-nav grid content-start gap-1">
        {allSections.map((section) => {
          const active = section.id === activeSection.id;
          return (
            <Button
              type="button"
              key={section.id}
              variant="ghost"
              aria-pressed={active}
              className={`h-auto w-full flex-col items-stretch gap-0 rounded-md px-3 py-2.5 text-left ${
                active
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              }`}
              onClick={() => setActiveSectionID(section.id)}
            >
              <span className="block truncate text-sm font-medium">
                {section.productName
                  ? `${section.productName} / ${section.name}`
                  : section.name}
              </span>
              <span className="mt-1 block text-xs opacity-75">
                {section.filled} / {section.total} · {section.percent}%
              </span>
            </Button>
          );
        })}
      </nav>

      <section className="min-w-0">
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border/70 pb-3">
          <div>
            <h3 className="text-sm font-semibold text-foreground">
              {activeSection.name}
            </h3>
            {activeSection.productName ? (
              <p className="mt-1 text-xs text-muted-foreground">
                所属产品：{activeSection.productName}
              </p>
            ) : null}
          </div>
          <span className="text-xs text-muted-foreground">
            已填写 {activeSection.filled} / {activeSection.total}
          </span>
        </div>

        <WorkCustomerDetailFieldGroups
          fields={activeSection.fields}
          onPreviewFile={setPreviewFile}
        />
      </section>

      <WorkTaskUploadPreviewDialog
        file={previewFile}
        onOpenChange={(open) => {
          if (!open) setPreviewFile(null);
        }}
      />
    </div>
  );
}

function workCustomerDetailSections(
  customer: WorkCustomer,
  asset: WorkAsset | undefined,
  sections: WorkDetailSection[],
): WorkDetailSection[] {
  const sourceLead = customer.source_lead;
  const sourceSections = sections.filter(
    (section) => section.targetType === "lead",
  );
  const customerSections = sections.filter(
    (section) => section.targetType !== "lead",
  );
  const leadBaseSection: WorkDetailSection | null = sourceLead
    ? {
      id: "base:lead",
      name: "来源线索",
      targetType: "lead",
      filled: 0,
      total: 0,
      percent: 0,
      fields: [
        workBaseDetailField("线索编号", sourceLead.code),
        workBaseDetailField("姓名", sourceLead.name),
        workBaseDetailField("手机号", sourceLead.phone),
        workBaseDetailField("微信", sourceLead.wechat),
        workBaseDetailField("来源", sourceLead.source_name),
        workBaseDetailField("渠道", sourceLead.channel_name),
        workBaseDetailField("外部线索ID", sourceLead.external_id),
        workBaseDetailField("城市", sourceLead.city),
        workBaseDetailField("初始诉求", sourceLead.initial_need),
      ],
    }
    : null;
  const baseSections: WorkDetailSection[] = [
    {
      id: "base:customer",
      name: "客户基础信息",
      targetType: "customer",
      filled: 0,
      total: 0,
      percent: 0,
      fields: [
        workBaseDetailField("姓名", customer.name || customer.customer_name),
        workBaseDetailField("手机号", customer.phone || customer.mobile),
        workBaseDetailField("微信", customer.wechat),
        workBaseDetailField("来源", customer.source_name || customer.source),
        workBaseDetailField("渠道", customer.channel_name || customer.channel),
        workBaseDetailField("等级", customer.level_name || customer.customer_level),
      ],
    },
  ];
  if (asset) {
    baseSections.push({
      id: "base:asset",
      name: "资产基础信息",
      targetType: "asset",
      filled: 0,
      total: 0,
      percent: 0,
      fields: [
        workBaseDetailField("资产名称", asset.asset_name || asset.name),
        workBaseDetailField("资产编号", asset.asset_no || asset.asset_code),
        workBaseDetailField("资产状态", asset.asset_status_name),
        workBaseDetailField("备注", asset.remark),
      ],
    });
  }
  if (!leadBaseSection) {
    return [...baseSections.map(workDetailSectionProgress), ...customerSections];
  }
  return [
    workDetailSectionProgress(leadBaseSection),
    ...sourceSections,
    ...baseSections.map(workDetailSectionProgress),
    ...customerSections,
  ];
}

function workBaseDetailField(label: string, value: unknown): WorkDetailField {
  const text = displayText(value, "");
  return {
    key: `base:${label}`,
    label,
    value: text,
    valueType: "text",
    empty: !text,
    files: [],
  };
}

function workDetailSectionProgress(section: WorkDetailSection): WorkDetailSection {
  const filled = section.fields.filter((field) => !field.empty).length;
  const total = section.fields.length;
  return {
    ...section,
    filled,
    total,
    percent: total > 0 ? Math.round((filled / total) * 100) : 0,
  };
}

function WorkCustomerDetailFieldGroups({
  fields,
  onPreviewFile,
}: {
  fields: WorkDetailField[];
  onPreviewFile: (file: UploadFileItem) => void;
}) {
  const groups = useMemo(() => {
    const result = new Map<string, WorkDetailField[]>();
    fields.forEach((field) => {
      const group = field.group || "基本字段";
      if (!result.has(group)) result.set(group, []);
      result.get(group)?.push(field);
    });
    return Array.from(result.entries());
  }, [fields]);

  return (
    <div className="grid gap-5 pt-4">
      {groups.map(([group, groupFields]) => (
        <section key={group}>
          {groups.length > 1 ? (
            <h4 className="mb-2 text-xs font-medium text-muted-foreground">
              {group}
            </h4>
          ) : null}
          <div className="crm-customer-detail-fields">
            {groupFields.map((field) => (
              <div
                key={field.key}
                className="min-w-0 border-b border-border/50 py-3"
              >
                <div className="text-xs text-muted-foreground">{field.label}</div>
                <div className="mt-1.5 min-h-5 text-sm font-medium">
                  <WorkCustomerDetailFieldValue
                    field={field}
                    onPreviewFile={onPreviewFile}
                  />
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

function WorkCustomerDetailFieldValue({
  field,
  onPreviewFile,
}: {
  field: WorkDetailField;
  onPreviewFile: (file: UploadFileItem) => void;
}) {
  if (field.empty) {
    return <span className="font-normal text-muted-foreground">未填写</span>;
  }
  const files = normalizeUploadItems(field.files);
  if (field.valueType === "files" && files.length > 0) {
    return (
      <div className="grid gap-1.5">
        {files.map((file) => (
          <div key={String(file.id || file.name)} className="flex min-w-0 items-center gap-2">
            <Paperclip className="h-4 w-4 shrink-0 text-muted-foreground" />
            <Button
              type="button"
              variant="ghost"
              className="h-auto min-w-0 flex-1 justify-start truncate px-0 py-0 text-left font-normal hover:bg-transparent hover:text-primary hover:underline"
              onClick={() => onPreviewFile(file)}
            >
              {file.name || "附件"}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0 text-muted-foreground"
              aria-label="下载附件"
              title="下载附件"
              onClick={() => void downloadUploadFile(file)}
            >
              <Download className="h-4 w-4" />
            </Button>
          </div>
        ))}
      </div>
    );
  }
  return <span className="break-words text-foreground">{displayText(field.value)}</span>;
}

export function WorkCustomerFlowTimeline({
  entries,
  loading,
  scope,
  onScopeChange,
  onOpen,
  loadingText = "正在加载流程记录",
  emptyText = "暂无流程记录",
}: {
  entries: WorkCustomerFlowEntryView[];
  loading: boolean;
  scope: WorkCustomerOperationScope;
  onScopeChange: (scope: WorkCustomerOperationScope) => void;
  onOpen: (entry: WorkCustomerFlowEntryView) => void;
  loadingText?: string;
  emptyText?: string;
}) {
  return (
    <div className="grid gap-4">
      <div className="inline-flex w-fit rounded-md border border-border/70 bg-muted/20 p-1">
        {operationScopeOptions.map((option) => (
          <Button
            type="button"
            key={option.value}
            variant="ghost"
            aria-pressed={scope === option.value}
            className={`h-auto rounded px-3 py-1.5 text-sm font-medium ${
              scope === option.value
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => onScopeChange(option.value)}
          >
            {option.label}
          </Button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          {loadingText}
        </div>
      ) : entries.length === 0 ? (
        <div className="py-10 text-center text-sm text-muted-foreground">
          {emptyText}
        </div>
      ) : (
        <div className="crm-customer-flow-timeline relative grid gap-3 border-l border-border/70 pl-5">
          {entries.map((entry) => (
            <Button
              type="button"
              key={entry.id}
              variant="outline"
              className="relative h-auto w-full flex-col items-stretch gap-0 rounded-md border-border/60 bg-background px-4 py-3 text-left font-normal hover:bg-muted/20"
              onClick={() => onOpen(entry)}
            >
              <span
                className={`crm-customer-flow-dot absolute top-4 h-3 w-3 rounded-full border-2 border-background ${entry.dotClassName}`}
              />
              <div className="flex min-w-0 items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="min-w-0 break-words text-sm font-semibold text-foreground">
                      {entry.title}
                    </span>
                    <span className={`rounded px-2 py-0.5 text-[11px] font-medium ${entry.badgeClassName}`}>
                      {entry.badge}
                    </span>
                  </div>
                  <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    {entry.stageName ? <span>{entry.stageName}</span> : null}
                    <span>操作人：{displayText(entry.operatorName)}</span>
                  </div>
                </div>
                <span className="shrink-0 whitespace-nowrap text-xs text-muted-foreground">
                  {entry.time}
                </span>
              </div>
              {entry.description ? (
                <p className="mt-2 text-sm leading-6 text-muted-foreground">
                  {entry.description}
                </p>
              ) : null}
            </Button>
          ))}
        </div>
      )}
    </div>
  );
}

export function WorkCustomerDetailSectionEmpty() {
  return (
    <div className="flex min-h-40 flex-col items-center justify-center text-center text-muted-foreground">
      <FileText className="h-5 w-5" />
      <span className="mt-2 text-sm">暂无可展示资料</span>
    </div>
  );
}

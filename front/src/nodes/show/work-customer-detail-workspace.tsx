import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  BriefcaseBusiness,
  Clock3,
  Download,
  FileText,
  FolderOpen,
  Image as ImageIcon,
  Loader2,
  RotateCcw,
  UserRound,
  Workflow,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { downloadUploadFile, type UploadFileItem } from "@/lib/upload";
import { normalizeUploadItems } from "@/lib/resource";

import { WorkCommunicationGroups } from "./work-communication-groups";
import {
  type WorkAsset,
  type WorkCommunicationGroup,
  type WorkCommunicationGroupType,
  type WorkCustomer,
  type WorkDetailSection,
  displayText,
  workIsRecord,
} from "./work-core";
import {
  WorkCustomerDetailData,
  WorkCustomerDetailStyles,
} from "./work-customer-detail";
import {
  resolveWorkUploadPreviewKind,
  WorkTaskUploadPreviewDialog,
  workUploadPreviewUrl,
} from "./work-upload";

export type WorkCustomerDetailWorkspaceSummary = {
  title: string;
  subtitle: string;
  identifiers: string[];
  statusName: string;
  workflowName: string;
  stageName: string;
  ownerName: string;
  updatedAt: string;
  flowStatus: string;
  flowStatusName: string;
  stageDays: string;
};

export type WorkCustomerDetailAttachment = {
  key: string;
  file: UploadFileItem;
  source: string;
  fieldLabel: string;
};

type WorkCustomerDetailWorkspaceProps = {
  customer: WorkCustomer;
  asset?: WorkAsset;
  summary: WorkCustomerDetailWorkspaceSummary;
  sections: WorkDetailSection[];
  attachments: WorkCustomerDetailAttachment[];
  profileLoading: boolean;
  profileError: string;
  attachmentsLoading: boolean;
  attachmentsError: string;
  timelineError: string;
  timelineHasData: boolean;
  timeline: ReactNode;
  communicationGroups: WorkCommunicationGroup[];
  communicationGroupTypes: WorkCommunicationGroupType[];
  communicationGroupWorkflowInstanceID: string;
  canManageCommunicationGroups: boolean;
  onReloadProfile: () => void;
  onReloadAttachments: () => void;
  onReloadTimeline: () => void;
};

export function WorkCustomerDetailWorkspace({
  customer,
  asset,
  summary,
  sections,
  attachments,
  profileLoading,
  profileError,
  attachmentsLoading,
  attachmentsError,
  timelineError,
  timelineHasData,
  timeline,
  communicationGroups,
  communicationGroupTypes,
  communicationGroupWorkflowInstanceID,
  canManageCommunicationGroups,
  onReloadProfile,
  onReloadAttachments,
  onReloadTimeline,
}: WorkCustomerDetailWorkspaceProps) {
  const [activeCenterTab, setActiveCenterTab] = useState<"info" | "groups">(
    "info",
  );
  return (
    <div
      data-crm-work-detail-workspace="true"
      className="crm-work-detail-workspace flex min-h-0 flex-col overflow-hidden bg-background"
    >
      <WorkCustomerDetailStyles />
      <WorkCustomerDetailWorkspaceStyles />
      <WorkDetailWorkspaceHeader summary={summary} />

      <div className="crm-work-detail-columns grid min-h-0 flex-1 overflow-hidden">
        <aside className="crm-work-detail-column crm-work-detail-timeline-column min-w-0 border-r border-border/70 bg-muted/5 px-4 py-4">
          <WorkDetailStageOverview summary={summary} />
          <div className="mt-4 border-t border-border/70 pt-4">
            {timelineError && !timelineHasData ? (
              <WorkDetailColumnMessage
                message={timelineError}
                onRetry={onReloadTimeline}
              />
            ) : (
              <>
                {timelineError ? (
                  <WorkDetailInlineError
                    message={timelineError}
                    onRetry={onReloadTimeline}
                  />
                ) : null}
                {timeline}
              </>
            )}
          </div>
        </aside>

        <main className="crm-work-detail-column crm-work-detail-info-column min-w-0 px-5 py-4">
          <WorkDetailWorkspaceTabs
            activeTab={activeCenterTab}
            groupCount={communicationGroups.length}
            onChange={setActiveCenterTab}
          />
          <div className="pt-4">
            {profileError && sections.length > 0 ? (
              <WorkDetailInlineError
                message={profileError}
                onRetry={onReloadProfile}
              />
            ) : null}
            {profileError && sections.length === 0 ? (
              <WorkDetailColumnMessage
                message={profileError}
                onRetry={onReloadProfile}
              />
            ) : profileLoading && sections.length === 0 ? (
              <WorkDetailColumnLoading label="正在加载详细信息" />
            ) : activeCenterTab === "groups" ? (
              <WorkCommunicationGroups
                groups={communicationGroups}
                groupTypes={communicationGroupTypes}
                workflowInstanceID={communicationGroupWorkflowInstanceID}
                canCreate={canManageCommunicationGroups}
              />
            ) : (
              <WorkCustomerDetailData
                customer={customer}
                asset={asset}
                sections={sections}
                navigation="tabs"
              />
            )}
          </div>
        </main>

        <WorkDetailAttachmentPanel
          attachments={attachments}
          loading={attachmentsLoading}
          error={attachmentsError}
          onRetry={onReloadAttachments}
        />
      </div>
    </div>
  );
}

function WorkDetailColumnLoading({ label }: { label: string }) {
  return (
    <div className="flex min-h-40 items-center justify-center gap-2 text-sm text-muted-foreground">
      <Loader2 className="h-4 w-4 animate-spin" />
      {label}
    </div>
  );
}

function WorkDetailColumnMessage({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  return (
    <div className="flex min-h-40 flex-col items-center justify-center gap-3 text-center text-sm text-muted-foreground">
      <span>{message}</span>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        <RotateCcw className="h-4 w-4" />
        重新加载
      </Button>
    </div>
  );
}

function WorkDetailInlineError({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  return (
    <div className="mb-3 flex items-center justify-between gap-3 rounded border border-border/70 bg-background px-3 py-2 text-xs text-muted-foreground">
      <span className="min-w-0 truncate">{message}</span>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-7 shrink-0 px-2"
        onClick={onRetry}
      >
        <RotateCcw className="h-3.5 w-3.5" />
        重试
      </Button>
    </div>
  );
}

function WorkCustomerDetailWorkspaceStyles() {
  return (
    <style>{`
      [role="dialog"]:has([data-crm-work-detail-workspace="true"]) {
        width: min(96vw, 1680px) !important;
        max-width: min(96vw, 1680px) !important;
        height: min(92vh, 960px) !important;
        max-height: 92vh !important;
        overflow: hidden !important;
      }

      [role="dialog"]:has([data-crm-work-detail-workspace="true"]) > [data-slot="dialog-header"] {
        position: absolute;
        top: 14px;
        right: 14px;
        z-index: 40;
        padding: 0;
      }

      [role="dialog"]:has([data-crm-work-detail-workspace="true"]) > [data-slot="dialog-header"] > div {
        justify-content: flex-end;
      }

      [role="dialog"]:has([data-crm-work-detail-workspace="true"]) > [data-slot="dialog-header"] > div > div:first-child {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
      }

      [role="dialog"]:has([data-crm-work-detail-workspace="true"]) > [data-slot="dialog-header"] button {
        margin-top: 0 !important;
        margin-right: 0 !important;
      }

      [role="dialog"]:has([data-crm-work-detail-workspace="true"]) > form,
      [role="dialog"]:has([data-crm-work-detail-workspace="true"]) .crm-customer-detail-modal-body {
        min-height: 0;
        height: 100%;
      }

      .crm-work-detail-workspace {
        height: 100%;
      }

      .crm-work-detail-columns {
        grid-template-columns: minmax(380px, 1.1fr) minmax(560px, 1.45fr) minmax(300px, .85fr);
      }

      .crm-work-detail-column {
        min-height: 0;
        overflow-y: auto;
        overscroll-behavior: contain;
      }

      .crm-work-detail-summary-header {
        display: grid;
        grid-template-columns: minmax(0, 1fr) minmax(640px, 52%);
        min-height: 96px;
        align-items: center;
        gap: 24px;
        padding: 16px 112px 16px 24px;
      }

      .crm-work-detail-summary-grid {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        overflow: hidden;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--background);
      }

      .crm-work-detail-summary-item {
        min-width: 0;
        padding: 10px 14px;
        border-left: 1px solid var(--border);
      }

      .crm-work-detail-summary-item:first-child {
        border-left: 0;
      }

      .crm-work-detail-workspace .crm-customer-detail-data-grid--tabs .crm-customer-detail-section-nav {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        overflow: visible;
        margin-bottom: 16px;
        padding: 6px;
        border: 1px solid #dce3ec;
        border-radius: 6px;
        background: #f6f8fb;
      }

      .crm-work-detail-workspace .crm-customer-detail-data-grid--tabs .crm-customer-detail-section-nav button {
        width: auto;
        min-width: 0;
        min-height: 34px;
        margin: 0;
        padding: 6px 11px;
        border: 1px solid transparent;
        border-radius: 4px;
        background: transparent;
        color: #667085;
      }

      .crm-work-detail-workspace .crm-customer-detail-data-grid--tabs .crm-customer-detail-section-nav button[aria-pressed="true"] {
        position: relative;
        z-index: 1;
        border-color: #cbd5e1;
        background: #ffffff;
        color: #172033;
        box-shadow: 0 1px 2px rgba(15, 23, 42, .08);
      }

      .crm-work-detail-workspace .crm-customer-detail-data-grid--tabs .crm-customer-detail-section-nav button > span:last-child {
        display: none;
      }

      .crm-work-detail-workspace .crm-customer-detail-data-grid--tabs > section {
        padding-top: 0;
      }

      .crm-work-detail-workspace .crm-customer-detail-fields {
        gap: 0;
        border-top: 1px solid #dce3ec;
        border-left: 1px solid #dce3ec;
      }

      .crm-work-detail-workspace .crm-customer-detail-fields > div {
        display: grid;
        grid-template-columns: minmax(108px, .8fr) minmax(0, 1.3fr);
        min-height: 48px;
        padding: 0;
        border-right: 1px solid #dce3ec;
        border-bottom: 1px solid #dce3ec;
      }

      .crm-work-detail-workspace .crm-customer-detail-fields > div > div:first-child {
        display: flex;
        align-items: center;
        padding: 9px 12px;
        background: #f6f8fb;
        color: #64748b;
        font-size: 12px;
        font-weight: 600;
      }

      .crm-work-detail-workspace .crm-customer-detail-fields > div > div:nth-child(2) {
        display: flex;
        min-width: 0;
        min-height: 48px;
        align-items: center;
        margin-top: 0;
        padding: 9px 12px;
        font-size: 13px;
        line-height: 20px;
        color: #1f2937;
        overflow-wrap: anywhere;
      }

      .crm-work-detail-info-column {
        background: #ffffff;
      }

      .crm-work-detail-view-tabs {
        min-height: 52px;
        padding: 8px 10px;
        border: 1px solid #dce3ec;
        border-radius: 6px;
        background: #f7f9fc;
      }

      .crm-work-detail-view-tab {
        min-height: 32px;
        padding: 6px 12px;
        border: 1px solid transparent;
        border-radius: 4px;
        color: #667085;
        font-size: 12px;
        font-weight: 600;
      }

      .crm-work-detail-view-tab--active {
        border-color: #cbd5e1;
        background: #ffffff;
        color: #172033;
        box-shadow: 0 1px 2px rgba(15, 23, 42, .08);
      }

      .crm-work-detail-section-header {
        min-height: 48px;
        align-items: center;
        margin-bottom: 0;
        padding: 10px 12px;
        border: 1px solid #dce3ec;
        border-bottom: 0;
        background: #fbfcfe;
      }

      .crm-work-detail-field-groups {
        padding-top: 0;
      }

      .crm-work-detail-field-group + .crm-work-detail-field-group {
        margin-top: 18px;
      }

      .crm-work-detail-field-group-title {
        margin: 0;
        padding: 9px 12px;
        border: 1px solid #dce3ec;
        border-bottom: 0;
        background: #eef3f8;
        color: #475569;
        font-size: 12px;
        font-weight: 700;
      }

      .crm-work-detail-attachment-grid {
        grid-template-columns: repeat(3, minmax(0, 1fr));
      }

      .crm-work-detail-stage-overview {
        border-color: var(--border);
        background: var(--background);
      }

      .crm-work-detail-stage-icon,
      .crm-work-detail-stage-status {
        background: var(--muted);
        color: var(--muted-foreground);
      }

      .crm-work-detail-stage-overview--active {
        border-color: #b9e7cd;
        background: #edfbf3;
      }

      .crm-work-detail-stage-overview--active .crm-work-detail-stage-icon,
      .crm-work-detail-stage-overview--active .crm-work-detail-stage-status {
        background: #d8f7e5;
        color: #087a53;
      }

      .crm-work-detail-stage-overview--completed {
        border-color: #cbdff5;
        background: #f0f7ff;
      }

      .crm-work-detail-stage-overview--completed .crm-work-detail-stage-icon,
      .crm-work-detail-stage-overview--completed .crm-work-detail-stage-status {
        background: #dcecff;
        color: #235f9e;
      }

      .crm-work-detail-stage-overview--terminated {
        border-color: #efcccc;
        background: #fff3f3;
      }

      .crm-work-detail-stage-overview--terminated .crm-work-detail-stage-icon,
      .crm-work-detail-stage-overview--terminated .crm-work-detail-stage-status {
        background: #fce0e0;
        color: #a82d2d;
      }

      @media (max-width: 1399px) {
        .crm-work-detail-columns {
          grid-template-columns: minmax(340px, 1fr) minmax(440px, 1.35fr) minmax(260px, .82fr);
        }

        .crm-work-detail-summary-header {
          grid-template-columns: minmax(0, 1fr) minmax(560px, 54%);
        }
      }

      @media (max-width: 1099px) {
        .crm-work-detail-columns {
          grid-template-columns: minmax(280px, 340px) minmax(0, 1fr);
        }

        .crm-work-detail-attachment-column {
          grid-column: 1 / -1;
          min-height: 360px;
          border-top: 1px solid var(--border);
          border-left: 0;
        }

        .crm-work-detail-summary-header {
          grid-template-columns: minmax(0, 1fr);
          gap: 12px;
          padding-right: 96px;
        }

        .crm-work-detail-summary-grid {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }

        .crm-work-detail-summary-item:nth-child(3) {
          border-left: 0;
        }

        .crm-work-detail-summary-item:nth-child(n + 3) {
          border-top: 1px solid var(--border);
        }
      }

      @media (max-width: 767px) {
        [role="dialog"]:has([data-crm-work-detail-workspace="true"]) {
          width: calc(100vw - 16px) !important;
          max-width: calc(100vw - 16px) !important;
        }

        .crm-work-detail-columns {
          display: block;
          overflow-y: auto;
        }

        .crm-work-detail-column {
          overflow: visible;
        }

        .crm-work-detail-column + .crm-work-detail-column {
          border-top: 1px solid var(--border);
          border-left: 0;
        }

        .crm-work-detail-summary-header {
          display: block;
          padding: 16px 88px 14px 16px;
        }

        .crm-work-detail-summary-grid {
          margin-top: 12px;
        }
      }
    `}</style>
  );
}

function WorkDetailWorkspaceHeader({
  summary,
}: {
  summary: WorkCustomerDetailWorkspaceSummary;
}) {
  return (
    <header className="crm-work-detail-summary-header shrink-0 border-b border-border/70 bg-muted/5">
      <div className="min-w-0">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h2 className="truncate text-lg font-semibold text-foreground">
            {displayText(summary.title)}
          </h2>
          {summary.statusName && summary.statusName !== "-" ? (
            <span className="rounded border border-border/70 bg-background px-2 py-0.5 text-xs font-medium text-muted-foreground">
              {summary.statusName}
            </span>
          ) : null}
        </div>
        <p className="mt-1 truncate text-sm font-medium text-foreground/75">
          {displayText(summary.subtitle)}
        </p>
        {summary.identifiers.length > 0 ? (
          <p className="mt-1 truncate text-xs text-muted-foreground">
            {summary.identifiers.join(" · ")}
          </p>
        ) : null}
      </div>

      <div className="crm-work-detail-summary-grid">
        <WorkDetailSummaryItem
          icon={<Workflow className="h-4 w-4" />}
          label="流程"
          value={summary.workflowName}
        />
        <WorkDetailSummaryItem
          icon={<BriefcaseBusiness className="h-4 w-4" />}
          label="当前阶段"
          value={summary.stageName}
        />
        <WorkDetailSummaryItem
          icon={<UserRound className="h-4 w-4" />}
          label="负责人"
          value={summary.ownerName}
        />
        <WorkDetailSummaryItem
          icon={<Clock3 className="h-4 w-4" />}
          label="更新时间"
          value={summary.updatedAt}
        />
      </div>
    </header>
  );
}

function WorkDetailSummaryItem({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="crm-work-detail-summary-item">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        {icon}
        <span>{label}</span>
      </div>
      <div className="mt-1 truncate text-sm font-semibold text-foreground">
        {displayText(value)}
      </div>
    </div>
  );
}

function WorkDetailWorkspaceTabs({
  activeTab,
  groupCount,
  onChange,
}: {
  activeTab: "info" | "groups";
  groupCount: number;
  onChange: (tab: "info" | "groups") => void;
}) {
  const tabs: Array<{ key: "info" | "groups"; label: string }> = [
    { key: "info", label: "客户资料" },
    { key: "groups", label: `沟通群 ${groupCount}` },
  ];
  return (
    <div className="crm-work-detail-view-tabs flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
        <FileText className="h-4 w-4 text-muted-foreground" />
        详细信息
      </div>
      <div className="flex items-center gap-1">
        {tabs.map((tab) => {
          const active = tab.key === activeTab;
          return (
            <Button
              type="button"
              key={tab.key}
              variant="ghost"
              className={`crm-work-detail-view-tab ${
                active ? "crm-work-detail-view-tab--active" : ""
              }`}
              aria-pressed={active}
              onClick={() => onChange(tab.key)}
            >
              {tab.label}
            </Button>
          );
        })}
      </div>
    </div>
  );
}

function WorkDetailStageOverview({
  summary,
}: {
  summary: WorkCustomerDetailWorkspaceSummary;
}) {
  const tone =
    summary.flowStatus === "active"
      ? "active"
      : summary.flowStatus === "completed"
        ? "completed"
        : summary.flowStatus === "terminated"
          ? "terminated"
          : "default";
  return (
    <section
      className={`crm-work-detail-stage-overview crm-work-detail-stage-overview--${tone} rounded-md border px-4 py-3.5`}
    >
      <div className="flex min-w-0 items-center gap-3">
        <div className="crm-work-detail-stage-icon flex h-9 w-9 shrink-0 items-center justify-center rounded-full">
          <Workflow className="h-4 w-4" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="text-xs text-muted-foreground">当前阶段</div>
              <div className="mt-0.5 truncate text-sm font-semibold text-foreground">
                {displayText(summary.stageName)}
              </div>
            </div>
            <span
              className="crm-work-detail-stage-status shrink-0 rounded px-2 py-0.5 text-xs font-medium"
            >
              {displayText(summary.flowStatusName, "未开始")}
            </span>
          </div>
          <div className="mt-1.5 text-xs text-muted-foreground">
            {summary.stageDays
              ? `本阶段已停留 ${summary.stageDays} 天`
              : "暂无阶段时长"}
          </div>
        </div>
      </div>
    </section>
  );
}

export function normalizeWorkCustomerDetailAttachments(
  value: unknown,
): WorkCustomerDetailAttachment[] {
  if (!Array.isArray(value)) return [];
  const attachments = new Map<string, WorkCustomerDetailAttachment>();
  value.forEach((entry, index) => {
    if (!workIsRecord(entry)) return;
    const [file] = normalizeUploadItems(entry.file);
    if (!file) return;
    const source = displayText(entry.source, "客户资料");
    const fieldLabel = displayText(entry.field_label, "附件");
    const key = displayText(
      entry.key || file.id || file.url || file.open_url || file.download,
      `${displayText(file.name, "附件")}:${source}:${fieldLabel}:${index}`,
    );
    if (!attachments.has(key)) {
      attachments.set(key, { key, file, source, fieldLabel });
    }
  });
  return Array.from(attachments.values());
}

function WorkDetailAttachmentPanel({
  attachments,
  loading,
  error,
  onRetry,
}: {
  attachments: WorkCustomerDetailAttachment[];
  loading: boolean;
  error: string;
  onRetry: () => void;
}) {
  const [previewFile, setPreviewFile] = useState<UploadFileItem | null>(null);
  return (
    <aside className="crm-work-detail-column crm-work-detail-attachment-column min-w-0 border-l border-border/70 bg-muted/10 px-4 py-4">
      <div className="-mx-4 -mt-4 flex min-h-14 items-center justify-between gap-3 border-b border-border/70 bg-muted/25 px-4 py-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-border/70 bg-background">
            <FolderOpen className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="min-w-0">
            <h3 className="truncate text-sm font-semibold text-foreground">
              资料附件
            </h3>
            <p className="mt-0.5 text-xs text-muted-foreground">
              共 {attachments.length} 个文件
            </p>
          </div>
        </div>
      </div>

      {error && attachments.length > 0 ? (
        <WorkDetailInlineError message={error} onRetry={onRetry} />
      ) : null}
      {error && attachments.length === 0 ? (
        <WorkDetailColumnMessage message={error} onRetry={onRetry} />
      ) : loading && attachments.length === 0 ? (
        <WorkDetailColumnLoading label="正在加载附件" />
      ) : attachments.length === 0 ? (
        <div className="mt-4 flex min-h-40 flex-col items-center justify-center text-center text-muted-foreground">
          <FileText className="h-6 w-6" />
          <span className="mt-2 text-sm">暂无附件</span>
        </div>
      ) : (
        <div className="crm-work-detail-attachment-grid mt-4 grid gap-2.5">
          {attachments.map((attachment) => (
            <WorkDetailAttachmentCard
              key={attachment.key}
              attachment={attachment}
              onPreview={() => setPreviewFile(attachment.file)}
            />
          ))}
        </div>
      )}

      <WorkTaskUploadPreviewDialog
        file={previewFile}
        onOpenChange={(open) => {
          if (!open) setPreviewFile(null);
        }}
      />
    </aside>
  );
}

function WorkDetailAttachmentCard({
  attachment,
  onPreview,
}: {
  attachment: WorkCustomerDetailAttachment;
  onPreview: () => void;
}) {
  return (
    <article className="group min-w-0 overflow-hidden rounded-md border border-border/70 bg-background">
      <Button
        type="button"
        variant="ghost"
        className="h-auto w-full rounded-none p-0 hover:bg-muted/20"
        title={`预览 ${displayText(attachment.file.name, "附件")}`}
        onClick={onPreview}
      >
        <WorkDetailAttachmentPreview file={attachment.file} />
      </Button>
      <div className="flex min-w-0 items-start gap-1.5 border-t border-border/60 px-2.5 py-2">
        <div className="min-w-0 flex-1">
          <div
            className="truncate text-xs font-medium text-foreground"
            title={displayText(attachment.file.name, "附件")}
          >
            {displayText(attachment.file.name, "附件")}
          </div>
          <div
            className="mt-0.5 truncate text-[11px] text-muted-foreground"
            title={`${attachment.source} · ${attachment.fieldLabel}`}
          >
            {attachment.source}
          </div>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0 text-muted-foreground"
          title="下载附件"
          aria-label="下载附件"
          onClick={() => void downloadUploadFile(attachment.file)}
        >
          <Download className="h-3.5 w-3.5" />
        </Button>
      </div>
    </article>
  );
}

function WorkDetailAttachmentPreview({ file }: { file: UploadFileItem }) {
  const [imageFailed, setImageFailed] = useState(false);
  const previewUrl = workUploadPreviewUrl(file);
  const isImage = resolveWorkUploadPreviewKind(file) === "image";

  useEffect(() => {
    setImageFailed(false);
  }, [file.id, previewUrl]);

  if (isImage && previewUrl && !imageFailed) {
    return (
      <img
        src={previewUrl}
        alt={displayText(file.name, "附件预览")}
        className="aspect-[4/3] w-full object-cover"
        onError={() => setImageFailed(true)}
      />
    );
  }
  return (
    <div className="flex aspect-[4/3] w-full items-center justify-center bg-muted/25">
      {isImage ? (
        <ImageIcon className="h-7 w-7 text-muted-foreground/70" />
      ) : (
        <FileText className="h-7 w-7 text-muted-foreground/70" />
      )}
    </div>
  );
}

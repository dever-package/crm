import { useState, type ComponentProps } from "react";
import { Download, LoaderCircle } from "lucide-react";
import { getCompatModule, joinFrontApi, request } from "@dever/front-plugin";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";

type ExportTask = {
  id: number;
  status: string;
  resultName: string;
  errorMessage: string;
};

type ExportTaskResponse = {
  code?: number;
  data?: unknown;
  message?: string;
};

type ExportButtonMeta = {
  path?: string;
  variant?: ComponentProps<typeof Button>["variant"];
  size?: ComponentProps<typeof Button>["size"];
  disabled?: boolean;
};

type ExportButtonProps = {
  item?: {
    name?: string;
    className?: string;
    meta?: ExportButtonMeta;
    action?: {
      click?: {
        exportKey?: string;
      };
    };
  };
};

type RequestBlob = (
  path: string,
  method?: string,
  params?: Record<string, unknown>,
) => Promise<Blob | null>;

const requestBlob = getCompatModule("@/lib/request")
  .requestBlob as RequestBlob;

const exportPollInterval = 500;
const exportPollLimit = 240;

export function ShowCrmExportButton({ item }: ExportButtonProps = {}) {
  const [exporting, setExporting] = useState(false);

  const exportFields = async () => {
    if (exporting) return;

    const exportKey = textValue(item?.action?.click?.exportKey);
    const pagePath = textValue(item?.meta?.path);
    if (!exportKey || !pagePath) {
      toast.error("导出配置不完整");
      return;
    }

    setExporting(true);
    try {
      const task = await createExportTask(pagePath, exportKey);
      const completedTask = await waitForExportTask(task);
      await downloadExportTask(completedTask);
      toast.success("字段导出完成");
    } catch (error) {
      toast.error(exportErrorMessage(error));
    } finally {
      setExporting(false);
    }
  };

  return (
    <Button
      type="button"
      variant={item?.meta?.variant || "outline"}
      size={item?.meta?.size || "sm"}
      className={item?.className}
      disabled={exporting || Boolean(item?.meta?.disabled)}
      onClick={() => void exportFields()}
    >
      {exporting ? (
        <LoaderCircle className="size-4 animate-spin" />
      ) : (
        <Download className="size-4" />
      )}
      <span>{exporting ? "导出中..." : item?.name || "导出"}</span>
    </Button>
  );
}

async function createExportTask(
  pagePath: string,
  exportKey: string,
): Promise<ExportTask> {
  const response = (await request(
    joinFrontApi("export/task_create"),
    "post",
    {
      path: pagePath,
      tableId: "",
      exportKey,
      query: "{}",
    },
  )) as ExportTaskResponse;

  return exportTaskFromResponse(response, "创建导出任务失败");
}

async function getExportTask(taskID: number): Promise<ExportTask> {
  const response = (await request(
    joinFrontApi("export/task_info"),
    "get",
    { id: taskID },
  )) as ExportTaskResponse;

  return exportTaskFromResponse(response, "获取导出任务失败");
}

async function waitForExportTask(initialTask: ExportTask): Promise<ExportTask> {
  let task = initialTask;

  for (let attempt = 0; attempt < exportPollLimit; attempt += 1) {
    if (task.status === "success") return task;
    if (task.status === "failed") {
      throw new Error(task.errorMessage || "字段导出失败");
    }

    await delay(exportPollInterval);
    task = await getExportTask(task.id);
  }

  throw new Error("字段导出超时，请稍后重试");
}

async function downloadExportTask(task: ExportTask): Promise<void> {
  const blob = await requestBlob(
    joinFrontApi("export/download"),
    "get",
    { id: task.id },
  );
  if (!blob) throw new Error("下载导出文件失败");

  const filename = task.resultName || `export-${task.id}.xlsx`;
  const objectURL = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = objectURL;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(objectURL);
}

function exportTaskFromResponse(
  response: ExportTaskResponse,
  fallbackMessage: string,
): ExportTask {
  if (Number(response?.code) !== 0 || !isRecord(response?.data)) {
    throw new Error(textValue(response?.message) || fallbackMessage);
  }

  const task = response.data;
  const id = Number(task.id) || 0;
  if (id <= 0) throw new Error(fallbackMessage);

  return {
    id,
    status: textValue(task.status),
    resultName: textValue(task.result_name),
    errorMessage: textValue(task.error_message),
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function textValue(value: unknown): string {
  return value === null || value === undefined ? "" : String(value).trim();
}

function exportErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : "字段导出失败";
}

function delay(milliseconds: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds));
}

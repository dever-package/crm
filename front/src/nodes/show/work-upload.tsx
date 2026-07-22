import { useCallback, useEffect, useRef, useState } from "react";
import type { ChangeEvent } from "react";
import { Download, FileText, Loader2, Trash2, Upload } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  downloadUploadFile,
  uploadFileByRule,
  type UploadFileItem,
} from "@/lib/upload";
import {
  formatUploadSize,
  normalizeUploadItems,
  resolveResourcePreviewKind,
} from "@/lib/resource";

import {
  errorMessage,
  positiveTextID,
  setWorkStoreValue,
  textValue,
  workImageExtensions,
  workStoreValue,
  workTaskFormDataPath,
  workTaskUploadFilesPath,
  workTaskUploadPendingPath,
  workUploadGridColumns,
  type WorkNodeProps,
  type WorkTaskUploadMeta,
  type WorkTaskUploadProgress,
} from "./work-core";

export function ShowCrmWorkTaskUpload({
  item,
  value,
  setValue,
  store,
}: WorkNodeProps & {
  value?: unknown;
  setValue?: (value: unknown) => void;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadMessage, setUploadMessage] = useState("");
  const [uploadProgress, setUploadProgress] =
    useState<WorkTaskUploadProgress | null>(null);
  const [localFiles, setLocalFiles] = useState<UploadFileItem[] | null>(null);
  const [previewFile, setPreviewFile] = useState<UploadFileItem | null>(null);
  const taskFormKey = inferWorkTaskFormKey(item?.value);
  const uploadStateKey = taskFormKey || textValue(item?.id) || "upload";
  const taskFilesPath = taskFormKey
    ? `${workTaskUploadFilesPath}.${taskFormKey}`
    : "";
  const relationPath = inferWorkRelationPath(item?.value);
  const relationValue =
    store && relationPath
      ? workStoreValue<unknown>(store, relationPath, undefined)
      : undefined;
  const taskFilesValue =
    store && taskFilesPath
      ? workStoreValue<unknown>(store, taskFilesPath, undefined)
      : undefined;
  const meta = resolveWorkTaskUploadMeta(item?.meta);
  const initialFiles = normalizeUploadItems(item?.meta?.["initialFiles"]);
  const readonly = Boolean(item?.meta?.["readonly"]);
  const files = normalizeWorkTaskUploadItems(
    taskFilesValue,
    relationValue,
    value,
    localFiles,
    initialFiles,
  );
  const remainingCount = Math.max(meta.maxCount - files.length, 0);
  const disabled = readonly || uploading || remainingCount <= 0;

  const syncFiles = useCallback(
    (nextFiles: UploadFileItem[]) => {
      const fileIDs = nextFiles.map((file) => file.id);
      setLocalFiles(nextFiles);
      if (store && taskFormKey) {
        const formValues = workStoreValue<Record<string, unknown>>(
          store,
          workTaskFormDataPath,
          {},
        );
        setWorkStoreValue(store, workTaskFormDataPath, {
          ...formValues,
          [taskFormKey]: fileIDs,
        });
        setWorkStoreValue(store, taskFilesPath, nextFiles);
      } else {
        setValue?.(fileIDs);
      }
      if (store && relationPath) {
        setWorkStoreValue(store, relationPath, nextFiles);
      }
    },
    [relationPath, setValue, store, taskFilesPath, taskFormKey],
  );

  const handleChooseFiles = useCallback(
    async (event: ChangeEvent<HTMLInputElement>) => {
      const selected = Array.from(event.target.files || []);
      event.target.value = "";
      if (selected.length === 0 || uploading) return;

      const nextSelected = selected.slice(
        0,
        Math.max(meta.maxCount - files.length, 0),
      );
      if (nextSelected.length === 0) {
        setUploadMessage(`最多只能上传 ${meta.maxCount} 个文件。`);
        return;
      }

      setUploading(true);
      setWorkTaskUploadPending(store, uploadStateKey, true);
      setUploadMessage("");
      setUploadProgress({
        fileName: nextSelected[0]?.name || "",
        percent: 0,
        currentIndex: 1,
        total: nextSelected.length,
      });
      try {
        let nextFiles = [...files];
        for (let index = 0; index < nextSelected.length; index += 1) {
          const file = nextSelected[index];
          if (!file) continue;
          const currentIndex = index + 1;
          setUploadProgress({
            fileName: file.name,
            percent: resolveWorkUploadOverallProgress(
              index,
              0,
              nextSelected.length,
            ),
            currentIndex,
            total: nextSelected.length,
          });
          const uploaded = await uploadFileByRule(meta.ruleId, file, {
            kind: meta.kind,
            bizKey: meta.bizKey,
            bizName: meta.bizName,
            onProgress: (loaded, total) => {
              setUploadProgress({
                fileName: file.name,
                percent: resolveWorkUploadOverallProgress(
                  index,
                  resolveWorkUploadFileProgress(loaded, total),
                  nextSelected.length,
                ),
                currentIndex,
                total: nextSelected.length,
              });
            },
          });
          const uploadedFile = normalizeUploadItems(uploaded)[0] || uploaded;
          if (
            !nextFiles.some(
              (current) => String(current.id) === String(uploadedFile.id),
            )
          ) {
            nextFiles = [...nextFiles, uploadedFile];
          }
        }
        syncFiles(nextFiles);
      } catch (uploadError) {
        setUploadMessage(errorMessage(uploadError) || "上传失败");
      } finally {
        setUploading(false);
        setWorkTaskUploadPending(store, uploadStateKey, false);
        setUploadProgress(null);
      }
    },
    [files, meta, store, syncFiles, uploadStateKey, uploading],
  );

  const removeFile = useCallback(
    (targetID: UploadFileItem["id"]) => {
      syncFiles(files.filter((file) => String(file.id) !== String(targetID)));
    },
    [files, syncFiles],
  );

  return (
    <div className="w-full space-y-3">
      <Input
        ref={inputRef}
        type="file"
        className="hidden"
        multiple
        disabled={readonly}
        onChange={handleChooseFiles}
      />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Button
          type="button"
          variant="outline"
          disabled={disabled}
          onClick={() => inputRef.current?.click()}
        >
          {uploading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Upload className="h-4 w-4" />
          )}
          {uploading ? "上传中..." : "上传文件"}
        </Button>
        <span className="text-xs text-muted-foreground">
          已选择 {files.length} 个文件
        </span>
      </div>
      {uploading && uploadProgress ? (
        <div className="rounded-lg border border-border/70 bg-muted/20 px-3 py-2">
          <div className="flex items-center justify-between gap-3 text-xs">
            <span
              className="min-w-0 truncate text-muted-foreground"
              title={uploadProgress.fileName}
            >
              正在上传 {uploadProgress.fileName}
              {uploadProgress.total > 1
                ? `（${uploadProgress.currentIndex}/${uploadProgress.total}）`
                : ""}
            </span>
            <span className="shrink-0 font-medium text-foreground">
              {uploadProgress.percent}%
            </span>
          </div>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary transition-all duration-200"
              style={{ width: `${uploadProgress.percent}%` }}
            />
          </div>
        </div>
      ) : null}
      <div className="overflow-hidden rounded-xl border border-border/70 bg-background text-sm shadow-xs">
        <div
          className="grid border-b bg-muted/30"
          style={{ gridTemplateColumns: workUploadGridColumns }}
        >
          <div className="flex h-12 min-w-0 items-center px-4 font-medium text-muted-foreground">
            文件名
          </div>
          <div className="flex h-12 items-center whitespace-nowrap px-4 font-medium text-muted-foreground">
            大小
          </div>
          <div className="flex h-12 items-center whitespace-nowrap px-4 font-medium text-muted-foreground">
            操作
          </div>
        </div>
        {files.length === 0 ? (
          <div className="py-6 text-center text-sm text-muted-foreground">
            暂无附件
          </div>
        ) : (
          files.map((file) => (
            <div
              key={String(file.id)}
              className="grid border-b last:border-b-0"
              style={{ gridTemplateColumns: workUploadGridColumns }}
            >
              <div className="flex min-w-0 items-center overflow-hidden px-4 py-3">
                <Button
                  type="button"
                  variant="ghost"
                  className="h-auto w-full min-w-0 justify-start overflow-hidden truncate whitespace-nowrap px-0 py-0 text-left text-sm font-medium text-foreground underline-offset-4 hover:bg-transparent hover:text-primary hover:underline"
                  title={file.name}
                  onClick={() => setPreviewFile(file)}
                >
                  {file.name}
                </Button>
              </div>
              <div className="flex items-center whitespace-nowrap px-4 py-3 text-sm">
                {formatUploadSize(Number(file.size || 0))}
              </div>
              <div className="flex items-center px-4 py-3">
                <div
                  className="flex items-center gap-1"
                  style={{ flexWrap: "nowrap" }}
                >
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0"
                    aria-label="下载附件"
                    onClick={() => void downloadUploadFile(file)}
                  >
                    <Download className="h-4 w-4" />
                  </Button>
                  {!readonly ? (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 shrink-0 text-destructive hover:text-destructive"
                      aria-label="删除附件"
                      disabled={uploading}
                      onClick={() => removeFile(file.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  ) : null}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
      {uploadMessage ? (
        <p className="text-xs text-destructive">{uploadMessage}</p>
      ) : null}
      <WorkTaskUploadPreviewDialog
        file={previewFile}
        onOpenChange={(open) => {
          if (!open) setPreviewFile(null);
        }}
      />
    </div>
  );
}

function setWorkTaskUploadPending(
  store: WorkNodeProps["store"],
  key: string,
  pending: boolean,
) {
  if (!store || !key) return;
  const current = workStoreValue<Record<string, boolean>>(
    store,
    workTaskUploadPendingPath,
    {},
  );
  setWorkStoreValue(store, workTaskUploadPendingPath, {
    ...current,
    [key]: pending,
  });
}

function resolveWorkTaskUploadMeta(
  meta?: Record<string, unknown>,
): WorkTaskUploadMeta {
  return {
    ruleId: Number(meta?.ruleId || 6),
    kind: textValue(meta?.kind) || "file",
    maxCount: Number(meta?.maxCount || 10),
    bizKey: textValue(meta?.bizKey) || "crm.work",
    bizName: textValue(meta?.bizName) || "CRM工作台",
  };
}

function normalizeWorkTaskUploadItems(
  taskFilesValue: unknown,
  relationValue: unknown,
  value: unknown,
  localFiles: UploadFileItem[] | null,
  initialFiles: UploadFileItem[],
): UploadFileItem[] {
  if (localFiles !== null) return localFiles;

  if (taskFilesValue !== undefined) {
    return normalizeUploadItems(taskFilesValue);
  }

  const relationItems = normalizeUploadItems(relationValue);
  if (relationItems.length > 0) return relationItems;

  if (initialFiles.length > 0) return initialFiles;

  const valueItems = normalizeUploadItems(value);
  if (valueItems.length > 0) return valueItems;

  if (Array.isArray(value)) {
    return value
      .filter((current) => current && typeof current === "object")
      .map((current) => normalizeUploadItems(current)[0])
      .filter((file): file is UploadFileItem => Boolean(file));
  }

  return [];
}

function resolveWorkUploadFileProgress(loaded: number, total: number): number {
  const totalValue = Number(total || 0);
  if (!Number.isFinite(totalValue) || totalValue <= 0) {
    return Number(loaded || 0) > 0 ? 100 : 0;
  }
  return clampWorkUploadPercent((Number(loaded || 0) / totalValue) * 100);
}

function resolveWorkUploadOverallProgress(
  completedFileCount: number,
  currentFilePercent: number,
  totalFileCount: number,
): number {
  const total = Math.max(Number(totalFileCount || 0), 1);
  const completed = Math.max(
    0,
    Math.min(Number(completedFileCount || 0), total),
  );
  const current = clampWorkUploadPercent(currentFilePercent) / 100;
  return clampWorkUploadPercent(((completed + current) / total) * 100);
}

function clampWorkUploadPercent(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, Math.round(value)));
}

export function WorkTaskUploadPreviewDialog({
  file,
  onOpenChange,
}: {
  file: UploadFileItem | null;
  onOpenChange: (open: boolean) => void;
}) {
  const [imageFailed, setImageFailed] = useState(false);
  const previewKind = resolveWorkUploadPreviewKind(file);
  const previewUrl = workUploadPreviewUrl(file);
  const canPreviewImage = previewKind === "image" && previewUrl && !imageFailed;

  useEffect(() => {
    setImageFailed(false);
  }, [file?.id, previewUrl]);

  return (
    <Dialog open={Boolean(file)} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[88vh] max-h-[88vh] max-w-5xl flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="border-b px-6 py-4">
          <DialogTitle>{file?.name || "资源详情"}</DialogTitle>
          <DialogDescription>
            可查看当前选中资源，支持图片预览与附件下载。
          </DialogDescription>
        </DialogHeader>
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex min-h-0 flex-1 items-center justify-center overflow-hidden bg-muted/30 px-6 py-6">
            {canPreviewImage ? (
              <img
                src={previewUrl}
                alt={file?.name || "附件预览"}
                className="max-h-full max-w-full rounded-xl object-contain shadow-sm"
                onError={() => setImageFailed(true)}
              />
            ) : (
              <div className="flex w-full max-w-2xl flex-col items-center gap-4 rounded-xl border bg-background px-6 py-8 text-center shadow-sm">
                <FileText className="h-10 w-10 text-muted-foreground" />
                <div className="max-w-full space-y-1">
                  <div className="truncate text-sm font-medium">
                    {file?.name || "未选择资源"}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    当前文件暂不支持直接预览，可以下载后查看。
                  </div>
                </div>
              </div>
            )}
          </div>
          <div className="flex flex-col gap-4 border-t bg-background px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0 flex-1 space-y-1">
              <div className="truncate text-sm font-medium">
                {file?.name || "未选择资源"}
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span>{formatUploadSize(Number(file?.size || 0))}</span>
                {file?.ext ? (
                  <span>
                    {textValue(file.ext).replace(/^\./, "").toUpperCase()}
                  </span>
                ) : null}
              </div>
            </div>
            {file ? (
              <Button
                type="button"
                onClick={() => void downloadUploadFile(file)}
              >
                <Download className="h-4 w-4" />
                下载
              </Button>
            ) : null}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function workUploadPreviewUrl(file?: UploadFileItem | null): string {
  const directUrl = textValue(
    file?.thumbnail || file?.url || file?.open_url || file?.download,
  );
  if (directUrl) return directUrl;

  const fileID = positiveTextID(file?.id);
  if (fileID) {
    return `/front/upload/open?id=${encodeURIComponent(fileID)}`;
  }
  return "";
}

export function resolveWorkUploadPreviewKind(
  file?: UploadFileItem | null,
): string {
  const resourceKind = resolveResourcePreviewKind(file);
  if (resourceKind) return resourceKind;
  if (!file) return "";

  const kind = textValue(file.kind).toLowerCase();
  if (kind === "image") return "image";

  const mime = textValue(file.mime).toLowerCase();
  if (mime.startsWith("image/")) return "image";

  const ext = workUploadFileExtension(file);
  return workImageExtensions.has(ext) ? "image" : "";
}

function workUploadFileExtension(file?: UploadFileItem | null): string {
  const explicitExt = normalizeWorkUploadExtension(file?.ext);
  if (explicitExt) return explicitExt;

  const name = textValue(file?.name).split(/[?#]/)[0];
  const dotIndex = name.lastIndexOf(".");
  if (dotIndex < 0) return "";
  return normalizeWorkUploadExtension(name.slice(dotIndex + 1));
}

function normalizeWorkUploadExtension(value: unknown): string {
  return textValue(value).replace(/^\./, "").toLowerCase();
}

function inferWorkRelationPath(valuePath?: string): string {
  const path = textValue(valuePath);
  if (!path) return "";
  if (path.endsWith("_ids")) return `${path.slice(0, -4)}s`;
  if (path.endsWith("_id")) return path.slice(0, -3);
  return "";
}

function inferWorkTaskFormKey(valuePath?: string): string {
  const path = textValue(valuePath);
  const prefix = "workTaskForm.";
  return path.startsWith(prefix) ? path.slice(prefix.length) : "";
}

import { useEffect, useState } from "react";
import { KeyRound, LoaderCircle } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

import { errorMessage, workApi } from "./work-core";
import { WorkFormField } from "./work-form-field";

const minimumPasswordLength = 6;

export function WorkPasswordDialog({
  open,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setFormError("");
  }, [open]);

  const submit = async () => {
    if (submitting) return;
    const validationError = workPasswordValidationError({
      currentPassword,
      newPassword,
      confirmPassword,
    });
    if (validationError) {
      setFormError(validationError);
      return;
    }
    setFormError("");
    setSubmitting(true);
    try {
      await workApi("/crm/work/change_password", {
        method: "POST",
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
          confirm_password: confirmPassword,
        }),
      });
      toast.success("密码修改成功，请重新登录");
      onOpenChange(false);
      onChanged();
    } catch (error) {
      setFormError(errorMessage(error, "密码修改失败"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => !submitting && onOpenChange(nextOpen)}
    >
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>修改密码</DialogTitle>
          <DialogDescription>
            修改成功后，当前账号需要重新登录。
          </DialogDescription>
        </DialogHeader>
        <form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <WorkFormField label="原密码" required>
            <Input
              type="password"
              autoComplete="current-password"
              value={currentPassword}
              disabled={submitting}
              aria-invalid={Boolean(formError)}
              onChange={(event) => {
                setCurrentPassword(event.currentTarget.value);
                setFormError("");
              }}
            />
          </WorkFormField>
          <WorkFormField
            label="新密码"
            required
            hint={`至少 ${minimumPasswordLength} 位`}
          >
            <Input
              type="password"
              autoComplete="new-password"
              value={newPassword}
              disabled={submitting}
              aria-invalid={Boolean(formError)}
              onChange={(event) => {
                setNewPassword(event.currentTarget.value);
                setFormError("");
              }}
            />
          </WorkFormField>
          <WorkFormField label="确认新密码" required>
            <Input
              type="password"
              autoComplete="new-password"
              value={confirmPassword}
              disabled={submitting}
              aria-invalid={Boolean(formError)}
              onChange={(event) => {
                setConfirmPassword(event.currentTarget.value);
                setFormError("");
              }}
            />
          </WorkFormField>
          {formError ? (
            <p className="text-sm text-destructive" role="alert">
              {formError}
            </p>
          ) : null}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={submitting}
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? (
                <LoaderCircle className="h-4 w-4 animate-spin" />
              ) : (
                <KeyRound className="h-4 w-4" />
              )}
              确认修改
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function workPasswordValidationError({
  currentPassword,
  newPassword,
  confirmPassword,
}: {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}): string {
  if (!currentPassword || !newPassword || !confirmPassword) {
    return "请完整填写原密码、新密码和确认密码";
  }
  if (Array.from(newPassword).length < minimumPasswordLength) {
    return `新密码至少需要 ${minimumPasswordLength} 位`;
  }
  if (newPassword !== confirmPassword) return "两次输入的新密码不一致";
  if (newPassword === currentPassword) return "新密码不能与原密码相同";
  return "";
}

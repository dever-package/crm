import { useEffect, useState, type RefObject } from "react";

export function useWorkFeedbackModalHeaderTarget(
  contentRef: RefObject<HTMLElement | null>,
  enabled = true,
): HTMLElement | null {
  const [target, setTarget] = useState<HTMLElement | null>(null);

  useEffect(() => {
    if (!enabled) {
      setTarget(null);
      return undefined;
    }

    const dialog = contentRef.current?.closest('[role="dialog"]');
    const header = dialog?.querySelector<HTMLElement>(
      '[data-slot="dialog-header"]',
    );
    const maximizeButton = header?.querySelector<HTMLButtonElement>(
      'button[title="最大化"], button[title="还原"]',
    );
    const headerActions = maximizeButton?.parentElement;
    if (!maximizeButton || !headerActions) {
      setTarget(null);
      return undefined;
    }

    const headerTarget = document.createElement("div");
    headerTarget.setAttribute("data-crm-work-modal-header-action", "true");
    headerTarget.className = "contents";
    headerActions.insertBefore(headerTarget, maximizeButton);
    setTarget(headerTarget);

    return () => {
      headerTarget.remove();
      setTarget((current) => (current === headerTarget ? null : current));
    };
  }, [contentRef, enabled]);

  return target;
}

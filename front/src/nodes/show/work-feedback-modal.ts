import { useEffect, useState, type RefObject } from "react";

export type WorkFeedbackModalFooterTargets = {
  left: HTMLElement;
  actions: HTMLElement;
};

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

export function useWorkFeedbackModalFooterTargets(
  contentRef: RefObject<HTMLElement | null>,
  enabled = true,
  replaceSubmit = false,
): WorkFeedbackModalFooterTargets | null {
  const [targets, setTargets] =
    useState<WorkFeedbackModalFooterTargets | null>(null);

  useEffect(() => {
    if (!enabled) {
      setTargets(null);
      return undefined;
    }

    const form = contentRef.current?.closest("form");
    const footer = findWorkFeedbackModalFooter(form || null);
    const submitButton =
      footer?.querySelector<HTMLButtonElement>('button[type="submit"]') || null;
    if (!footer) {
      setTargets(null);
      return undefined;
    }

    const left = document.createElement("div");
    left.setAttribute("data-crm-work-modal-footer-left", "true");
    left.className = "mr-auto flex items-center gap-2";

    const actions = document.createElement("div");
    actions.setAttribute("data-crm-work-modal-footer-actions", "true");
    actions.className = "flex items-center gap-2";

    const previousSubmitDisplay = submitButton?.style.display || "";
    footer.insertBefore(left, footer.firstChild);
    if (submitButton) {
      footer.insertBefore(actions, submitButton);
      if (replaceSubmit) submitButton.style.display = "none";
    } else {
      footer.appendChild(actions);
    }
    setTargets({ left, actions });

    return () => {
      left.remove();
      actions.remove();
      if (submitButton) submitButton.style.display = previousSubmitDisplay;
      setTargets(null);
    };
  }, [contentRef, enabled, replaceSubmit]);

  return targets;
}

function findWorkFeedbackModalFooter(form: Element | null): HTMLElement | null {
  if (!form) return null;
  const children = Array.from(form.children).filter(
    (child): child is HTMLElement => child instanceof HTMLElement,
  );
  for (const child of [...children].reverse()) {
    if (child.querySelector('button[type="submit"]')) return child;
  }
  return form.querySelector<HTMLButtonElement>('button[type="submit"]')
    ?.parentElement || null;
}

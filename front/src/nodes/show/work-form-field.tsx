import type { ReactNode } from "react";

export function WorkFormField({
  label,
  required = false,
  className = "",
  hint,
  error,
  children,
}: {
  label: string;
  required?: boolean;
  className?: string;
  hint?: ReactNode;
  error?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className={`block min-w-0 ${className}`.trim()}>
      <span className="mb-1.5 block text-sm font-medium">
        {label}
        {required ? <span className="ml-1 text-destructive">*</span> : null}
      </span>
      {children}
      {error ? (
        <p className="mt-1.5 text-xs text-destructive">{error}</p>
      ) : hint ? (
        <p className="mt-1.5 text-xs text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  );
}
